package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
)

type RSS struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title         string `xml:"title"`
	Description   string `xml:"description"`
	Link          string `xml:"link"`
	Items         []Item `xml:"item"`
	LastBuiltDate string `xml:"lastBuildDate"`
}

type Item struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
}

func main() {
	url := "https://koaning.io/feed.xml"

	response, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error fetching RSS feed: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	var rss RSS
	err = xml.Unmarshal(body, &rss)
	if err != nil {
		log.Fatalf("Error parsing XML: %v", err)
	}

	fmt.Println("Feed Title:", rss.Channel.Title)
	fmt.Println(rss.Channel)
	for i, item := range rss.Channel.Items {
		if i >= 5 {
			break
		}
		fmt.Printf("%d. %s\n", i+1, item.Title)
		fmt.Println("   Link:", item.Link)
		fmt.Println("   Published Date:", item.PubDate)
	}
}
