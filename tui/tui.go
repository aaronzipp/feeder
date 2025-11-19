package tui

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"time"

	"github.com/aaronzipp/feeder/database"
	"github.com/charmbracelet/bubbles/list"
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

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f7768e")). // Tokyo Night red
			Bold(true)
)

// postItem implements list.Item and list.DefaultItem interfaces
type postItem struct {
	post database.PostWithFeed
}

func (i postItem) FilterValue() string {
	return i.post.Title + " " + i.post.FeedName
}

func (i postItem) Title() string {
	return i.post.Title
}

func (i postItem) Description() string {
	return i.post.FeedName + " • " + formatDate(i.post.PublishedAt)
}

// customDelegate renders items with Tokyo Night colors and tabular format
type customDelegate struct {
	list.DefaultDelegate
}

func (d customDelegate) Height() int {
	return 1
}

func (d customDelegate) Spacing() int {
	return 1
}

func (d customDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d customDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(postItem)
	if !ok {
		return
	}

	// Calculate column widths based on visible items
	maxTitleWidth := 0
	maxFeedWidth := 0
	maxDateWidth := 0

	for _, visibleItem := range m.VisibleItems() {
		if vi, ok := visibleItem.(postItem); ok {
			titleLen := len(vi.post.Title)
			feedLen := len(vi.post.FeedName)
			dateLen := len(formatDate(vi.post.PublishedAt))

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
	}

	// Reserve space for cursor and spacing
	availableWidth := m.Width() - 2 - 8
	if maxTitleWidth > availableWidth-maxFeedWidth-maxDateWidth {
		maxTitleWidth = availableWidth - maxFeedWidth - maxDateWidth
		if maxTitleWidth < 20 {
			maxTitleWidth = 20
		}
	}

	cursor := "  "
	if index == m.Index() {
		cursor = cursorStyle.Render("❯ ")
	}

	// Truncate title if needed
	title := i.post.Title
	if len(title) > maxTitleWidth {
		title = title[:maxTitleWidth-3] + "..."
	}

	// Format with fixed-width columns
	titlePadded := fmt.Sprintf("%-*s", maxTitleWidth, title)
	feedPadded := fmt.Sprintf("%-*s", maxFeedWidth, i.post.FeedName)
	datePadded := fmt.Sprintf("%-*s", maxDateWidth, formatDate(i.post.PublishedAt))

	// Apply styles
	var styledTitle string
	if index == m.Index() {
		styledTitle = selectedStyle.Render(titlePadded)
	} else {
		styledTitle = titleStyle.Render(titlePadded)
	}
	styledFeed := feedNameStyle.Render(feedPadded)
	styledDate := dateStyle.Render(datePadded)

	fmt.Fprint(w, cursor+styledTitle+"  "+styledFeed+"  "+styledDate)
}

type model struct {
	list    list.Model
	lastKey string
}

func InitialModel(posts []database.PostWithFeed) model {
	items := make([]list.Item, len(posts))
	for i, post := range posts {
		items[i] = postItem{post: post}
	}

	delegate := customDelegate{}

	l := list.New(items, delegate, 0, 0)
	l.Title = ""
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	return model{
		list:    l,
		lastKey: "",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Calculate height needed for exactly 10 items per page
		itemHeight := 1  // from delegate.Height()
		itemSpacing := 1 // from delegate.Spacing()
		desiredItems := 10

		// Calculate chrome height based on enabled components
		// The list component's updatePagination() subtracts these from available height:
		// - statusView: typically 1 line when shown
		// - paginationView: typically 1 line when shown
		// - helpView: typically 2-3 lines when shown
		// We add 1 extra for safe padding
		chromeHeight := 0
		if m.list.ShowStatusBar() {
			chromeHeight += 1
		}
		// Pagination is shown by default
		chromeHeight += 1
		if m.list.ShowHelp() {
			chromeHeight += 3 // Help view can be multiple lines
		}
		chromeHeight += 1 // Extra padding for safety

		// Calculate the height that would give us exactly 10 items
		// The formula mirrors what updatePagination does:
		// availHeight = height - chrome
		// PerPage = availHeight / (itemHeight + spacing)
		// So: height = (PerPage * (itemHeight + spacing)) + chrome
		itemsHeight := desiredItems * (itemHeight + itemSpacing)
		constrainedHeight := itemsHeight + chromeHeight

		// Use the smaller of terminal height or our constrained height
		height := msg.Height
		if constrainedHeight < msg.Height {
			height = constrainedHeight
		}

		m.list.SetSize(msg.Width, height)

	case tea.KeyMsg:
		key := msg.String()

		// Filter guard: only intercept keys when NOT filtering
		if !m.list.SettingFilter() {
			// Reset lastKey for non-g keys
			if key != "g" {
				defer func() {
					m.lastKey = ""
				}()
			}

			switch key {
			case "ctrl+c", "q":
				return m, tea.Quit

			case "g":
				if m.lastKey == "g" {
					m.list.Select(0)
					m.lastKey = ""
					return m, nil
				} else {
					m.lastKey = "g"
					return m, nil
				}

			case "G":
				m.list.Select(len(m.list.Items()) - 1)
				return m, nil

			case "enter":
				if item, ok := m.list.SelectedItem().(postItem); ok {
					go openBrowser(item.post.Url)
				}
				return m, nil
			}
		}
	}

	// Let the list handle all other keys
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
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
	return m.list.View()
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
	posts, err := queries.ListInbox(ctx)
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
