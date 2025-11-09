package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/aaronzipp/feeder/database"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Tokyo Night Dark color scheme
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7aa2f7")) // Tokyo Night blue

	feedNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#bb9af7")). // Tokyo Night purple
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9ece6a")) // Tokyo Night green

	dateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89")) // Tokyo Night comment

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89")) // Tokyo Night comment

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f7768e")). // Tokyo Night red
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#565f89")) // Tokyo Night comment

	postItemStyle = lipgloss.NewStyle().
			MarginBottom(1)
)

type model struct {
	posts   []database.ListPostsWithFeedRow
	cursor  int
	err     error
	width   int
	height  int
	lastKey string
}

func InitialModel(posts []database.ListPostsWithFeedRow) model {
	return model{
		posts:  posts,
		cursor: 0,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:

		key := msg.String()

		if key != "g" {
			defer func() {
				m.lastKey = ""
			}()
		}

		switch key {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "j", "down":
			if m.cursor < len(m.posts)-1 {
				m.cursor++
			}

		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "g":
			if m.lastKey == "g" {
				m.cursor = 0
				m.lastKey = ""
			} else {
				m.lastKey = "g"
			}

		case "G":
			m.cursor = len(m.posts) - 1

		case "enter":
			if len(m.posts) > 0 && m.cursor < len(m.posts) {
				go openBrowser(m.posts[m.cursor].Url)
			}
		}
	}

	return m, nil
}

func formatDate(dateStr string) string {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < 24*time.Hour:
		if now.Day() == t.Day() {
			return "today"
		}
		return "yesterday"
	case diff < 7*24*time.Hour:
		return t.Format("Monday")
	case t.Year() == now.Year():
		return t.Format("Jan 02")
	default:
		return t.Format("Jan 02, 2006")
	}
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.\n", m.err)
	}

	if len(m.posts) == 0 {
		return "No posts found.\n\nPress q to quit.\n"
	}

	var s string

	// Calculate how many posts can fit on screen
	// Header takes 3 lines, status takes 2 lines, each post takes 2 lines (with margin)
	availableHeight := m.height - 2
	windowSize := min(max(availableHeight/2, 1), len(m.posts))

	// Show a window of posts around the cursor
	start := 0
	end := len(m.posts)

	// If there are many posts, show a window around the cursor
	if len(m.posts) > windowSize {
		start = max(m.cursor-windowSize/2, 0)
		end = start + windowSize
		if end > len(m.posts) {
			end = len(m.posts)
			start = max(end-windowSize, 0)
		}
	} else {
		end = len(m.posts)
	}

	// Calculate column widths for tabular display
	maxTitleWidth := 0
	maxFeedWidth := 0
	maxDateWidth := 0

	for i := start; i < end; i++ {
		post := m.posts[i]
		titleLen := len(post.Title)
		feedLen := len(post.FeedName)
		dateLen := len(formatDate(post.PublishedAt))

		if titleLen > maxTitleWidth {
			maxTitleWidth = titleLen
		}
		if feedLen > maxFeedWidth {
			maxFeedWidth = feedLen
		}
		if dateLen > maxDateWidth {
			maxDateWidth = dateLen
		}
	}

	// Reserve space for cursor (2 chars) + column spacing (4 chars between columns)
	// and ensure title doesn't take up too much space
	availableWidth := m.width - 2 - 8 // cursor + spacing
	if maxTitleWidth > availableWidth-maxFeedWidth-maxDateWidth {
		maxTitleWidth = availableWidth - maxFeedWidth - maxDateWidth
		if maxTitleWidth < 20 {
			maxTitleWidth = 20 // minimum title width
		}
	}

	for i := start; i < end; i++ {
		post := m.posts[i]

		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("❯ ")
		}

		// Truncate title if needed
		title := post.Title
		if len(title) > maxTitleWidth {
			title = title[:maxTitleWidth-3] + "..."
		}

		// Format with fixed-width columns (left-aligned)
		titlePadded := fmt.Sprintf("%-*s", maxTitleWidth, title)
		feedPadded := fmt.Sprintf("%-*s", maxFeedWidth, post.FeedName)
		datePadded := fmt.Sprintf("%-*s", maxDateWidth, formatDate(post.PublishedAt))

		// Apply styles to padded strings
		var styledTitle string
		if m.cursor == i {
			styledTitle = selectedStyle.Render(titlePadded)
		} else {
			styledTitle = titleStyle.Render(titlePadded)
		}
		styledFeed := feedNameStyle.Render(feedPadded)
		styledDate := dateStyle.Render(datePadded)

		postContent := cursor + styledTitle + "  " + styledFeed + "  " + styledDate
		s += postItemStyle.Render(postContent) + "\n"
	}

	s += "\n" + statusStyle.Render(
		fmt.Sprintf("Showing %d-%d of %d posts", start+1, end, len(m.posts)),
	) + "\n"
	s += helpStyle.Render(
		"Navigate: j/k (or ↑/↓) • Open: Enter • To top: gg • To bottom: G • Quit: q",
	)

	return s
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	default:
		return fmt.Errorf("unsupported platform")
	}

	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// Run starts the TUI application
func Run(ctx context.Context, queries *database.Queries) error {
	posts, err := queries.ListPostsWithFeed(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch posts: %w", err)
	}

	p := tea.NewProgram(
		InitialModel(posts),
		tea.WithAltScreen(),
	)
	_, err = p.Run()
	return err
}
