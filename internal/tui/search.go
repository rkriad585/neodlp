package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	searchTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(accentColor).
			Padding(0, 3).
			Bold(true)

	searchItemStyle = lipgloss.NewStyle().Padding(0, 2)

	searchSelectedStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7C3AED"))

	searchMetaStyle = lipgloss.NewStyle().Foreground(mutedColor)
)

type searchItem struct {
	label string
	url   string
}

type searchModel struct {
	items    []searchItem
	cursor   int
	selected map[int]bool
	done     bool
	width    int
	err      error
	query    string
}

type SearchEntry struct {
	Title    string
	URL      string
	Uploader string
	Duration int
	Views    float64
	Platform string
}

func SelectSearchResult(query string, results []SearchEntry) (string, error) {
	var items []searchItem
	for _, r := range results {
		title := r.Title
		if len(title) > 55 {
			title = title[:52] + "..."
		}

		var parts []string
		if r.Uploader != "" {
			uploader := r.Uploader
			if len(uploader) > 18 {
				uploader = uploader[:15] + "..."
			}
			parts = append(parts, uploader)
		}
		if r.Duration > 0 {
			d := time.Duration(r.Duration) * time.Second
			parts = append(parts, d.Round(time.Second).String())
		}
		if r.Views > 0 {
			parts = append(parts, formatViews(r.Views))
		}
		meta := strings.Join(parts, "  •  ")

		label := title
		if meta != "" {
			label += "\n  " + searchMetaStyle.Render(meta)
		}

		items = append(items, searchItem{label: label, url: r.URL})
	}

	m := searchModel{
		items:    items,
		selected: make(map[int]bool),
		query:    query,
	}

	p := tea.NewProgram(&m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	m = *final.(*searchModel)
	if m.err != nil {
		return "", m.err
	}
	if !m.done || len(m.selected) == 0 {
		return "", fmt.Errorf("no result selected")
	}

	// Return the URL of the first selected item
	for idx := range m.selected {
		return items[idx].url, nil
	}
	return "", fmt.Errorf("no result selected")
}

func (m *searchModel) Init() tea.Cmd {
	return nil
}

func (m *searchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.err = fmt.Errorf("cancelled")
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]

		case "enter":
			if len(m.selected) == 0 {
				m.selected[m.cursor] = true
			}
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *searchModel) View() string {
	var b strings.Builder

	width := m.width
	if width == 0 {
		width = 70
	}
	if width > 80 {
		width = 80
	}

	title := " search results "
	if m.query != "" {
		q := m.query
		if len(q) > 25 {
			q = q[:22] + "..."
		}
		title = fmt.Sprintf(` search "%s" `, q)
	}

	b.WriteString(lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(searchTitleStyle.Render(title)))
	b.WriteString("\n\n")

	b.WriteString("  " + searchMetaStyle.Render("↑/↓ navigate • Space toggle • Enter select • Esc cancel"))
	b.WriteString("\n\n")

	for i, item := range m.items {
		prefix := "  "
		if m.selected[i] {
			prefix = "✓ "
		}

		line := "  " + prefix
		if i == m.cursor {
			line += searchSelectedStyle.Render(item.label)
		} else {
			line += searchItemStyle.Render(item.label)
		}
		b.WriteString(line)
		b.WriteString("\n\n")
	}

	b.WriteString("\n")
	selCount := len(m.selected)
	b.WriteString(searchMetaStyle.Render(
		fmt.Sprintf("  %d result(s) • %d selected", len(m.items), selCount)))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func formatViews(n float64) string {
	if n <= 0 {
		return ""
	}
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%sB", strconv.FormatFloat(n/1_000_000_000, 'f', 1, 64))
	case n >= 1_000_000:
		return fmt.Sprintf("%sM", strconv.FormatFloat(n/1_000_000, 'f', 1, 64))
	case n >= 1_000:
		return fmt.Sprintf("%sK", strconv.FormatFloat(n/1_000, 'f', 1, 64))
	}
	return strconv.FormatFloat(n, 'f', 0, 64)
}
