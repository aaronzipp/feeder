package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/aaronzipp/feeder/database"

	_ "modernc.org/sqlite"
)

type RawFeed interface {
	RSS | Atom
}

type RSS struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Items       []RSSItem `xml:"item"`
	LastUpdated string    `xml:"lastBuildDate"`
}

type RSSItem struct {
	Title     string `xml:"title"`
	Link      string `xml:"link"`
	Published string `xml:"pubDate"`
}

type Atom struct {
	Items       []AtomItem `xml:"entry"`
	LastUpdated string     `xml:"updated"`
}

type AtomItem struct {
	Title     string   `xml:"title"`
	Link      AtomLink `xml:"link"`
	Published string   `xml:"published"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
}

type NormalizedItem struct {
	Title     string
	URL       string
	Published string
}

func parseDate(dateStr string) (time.Time, string, error) {
	formats := []string{
		// Atom format
		time.RFC3339,
		// RSS formats
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, format, nil
		}
	}
	return time.Time{}, "", fmt.Errorf("unable to parse date: %s", dateStr)
}

func parseDateWithFormat(dateStr string, knownFormat sql.NullString) (time.Time, string, error) {
	if knownFormat.Valid && knownFormat.String != "" {
		if t, err := time.Parse(knownFormat.String, dateStr); err == nil {
			return t, knownFormat.String, nil
		}
	}

	return parseDate(dateStr)
}

func parseFeed[T RawFeed](url string, feed *T) error {
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error fetching feed %s: %v", url, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	return xml.Unmarshal(body, &feed)
}

func getRSSFeed(url string) (string, []NormalizedItem, error) {
	var rss RSS
	err := parseFeed(url, &rss)
	if err != nil {
		return "", nil, fmt.Errorf("error parsing XML: %v", err)
	}

	items := make([]NormalizedItem, len(rss.Channel.Items))
	for i, item := range rss.Channel.Items {
		items[i] = NormalizedItem{
			Title:     item.Title,
			URL:       item.Link,
			Published: item.Published,
		}
	}
	return rss.Channel.LastUpdated, items, nil
}

func getAtomFeed(url string) (string, []NormalizedItem, error) {
	var atom Atom
	err := parseFeed(url, &atom)
	if err != nil {
		return "", nil, fmt.Errorf("error parsing XML: %v", err)
	}

	items := make([]NormalizedItem, len(atom.Items))
	for i, item := range atom.Items {
		items[i] = NormalizedItem{
			Title:     item.Title,
			URL:       item.Link.Href,
			Published: item.Published,
		}
	}
	return atom.LastUpdated, items, nil
}

func openDB() (*database.Queries, func()) {
	db, err := sql.Open("sqlite", "database/feeder.db")
	if err != nil {
		log.Fatal(err)
	}

	cleanup := func() { db.Close() }
	return database.New(db), cleanup
}

func main() {
	ctx := context.Background()
	queries, cleanup := openDB()
	defer cleanup()

	feeds, err := queries.ListFeeds(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, feed := range feeds {
		var lastUpdatedAt string
		var items []NormalizedItem
		var err error

		switch feed.FeedType {
		case "rss":
			lastUpdatedAt, items, err = getRSSFeed(feed.Url)
		case "atom":
			lastUpdatedAt, items, err = getAtomFeed(feed.Url)
		case "custom":
			log.Fatal("'custom' option is not implemented yet.")
		default:
			continue
		}

		if err != nil {
			fmt.Printf("Can't parse feed %s: %v\n", feed.Name, err)
			continue
		}

		var detectedFormat string
		needsFormatUpdate := false

		for _, item := range items {
			parsedTime, usedFormat, err := parseDateWithFormat(item.Published, feed.DateFormat)
			if err != nil {
				fmt.Printf("Failed parsing date for post '%s': %v\n", item.Title, err)
				continue
			}

			if detectedFormat == "" && usedFormat != "" {
				detectedFormat = usedFormat
				if !feed.DateFormat.Valid || feed.DateFormat.String != usedFormat {
					needsFormatUpdate = true
				}
			}

			unifiedDate := parsedTime.Format(time.RFC3339)

			err = queries.CreatePost(ctx, database.CreatePostParams{
				Title:       item.Title,
				Url:         item.URL,
				PublishedAt: unifiedDate,
				FeedID:      feed.ID,
			})
			if err != nil {
				fmt.Printf("Failed writing post: %v\n", err)
			}
		}

		if needsFormatUpdate && detectedFormat != "" {
			err = queries.UpdateFeedFormat(
				ctx,
				database.UpdateFeedFormatParams{
					DateFormat: sql.NullString{String: detectedFormat, Valid: true},
					ID:         feed.ID,
				},
			)
			if err != nil {
				fmt.Printf("Failed updating feed format: %v\n", err)
			}
		}

		if lastUpdatedAt != "" {
			parsedTime, _, err := parseDateWithFormat(lastUpdatedAt, feed.DateFormat)
			if err == nil {
				lastUpdatedAt = parsedTime.Format(time.RFC3339)
			}
		}

		err = queries.UpdateFeedDate(
			ctx,
			database.UpdateFeedDateParams{
				LastUpdatedAt: sql.NullString{String: lastUpdatedAt, Valid: true},
				ID:            feed.ID,
			},
		)
		if err != nil {
			fmt.Printf("Failed updating feed date: %v\n", err)
		}
	}
}
