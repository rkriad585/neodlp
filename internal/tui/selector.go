package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lrstanley/go-ytdlp"
)

type formatItem struct {
	height   int
	filesize int
	note     string
}

type selectorModel struct {
	items    []formatItem
	cursor   int
	selected int
	done     bool
	width    int
	err      error
}

var (
	itemStyle     = lipgloss.NewStyle().Padding(0, 2)
	selectedStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(accentColor)

	dimStyle = lipgloss.NewStyle().Foreground(mutedColor)
)

func SelectFormat(info *ytdlp.ExtractedInfo) (string, error) {
	heights := extractResolutions(info)
	if len(heights) == 0 {
		return "", fmt.Errorf("no video formats found")
	}

	m := selectorModel{items: heights}
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	m, ok := final.(selectorModel)
	if !ok {
		return "", fmt.Errorf("unexpected model type")
	}
	if m.err != nil {
		return "", m.err
	}
	if m.done && m.selected >= 0 && m.selected < len(heights) {
		return fmt.Sprintf("%dp", heights[m.selected].height), nil
	}
	return "", fmt.Errorf("no format selected")
}

func extractResolutions(info *ytdlp.ExtractedInfo) []formatItem {
	seen := make(map[int]bool)
	var items []formatItem

	for _, f := range info.Formats {
		if f.Height == nil || *f.Height == 0 {
			continue
		}
		if f.VCodec == nil || *f.VCodec == "none" {
			continue
		}
		h := int(*f.Height)
		if seen[h] {
			continue
		}
		seen[h] = true

		item := formatItem{height: h}
		if f.FileSize != nil {
			item.filesize = *f.FileSize
		}
		if f.FormatNote != nil {
			item.note = *f.FormatNote
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].height > items[j].height
	})

	return items
}

func (m selectorModel) Init() tea.Cmd {
	return nil
}

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

		case "enter":
			m.selected = m.cursor
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m selectorModel) View() string {
	var b strings.Builder

	width := m.width
	if width == 0 {
		width = 70
	}
	if width > 64 {
		width = 64
	}

	b.WriteString(lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(titleStyle.Render(" select format ")))
	b.WriteString("\n\n")

	b.WriteString("  " + dimStyle.Render("Choose a resolution (↑/↓, Enter to select, Esc to cancel)"))
	b.WriteString("\n\n")

	for i, item := range m.items {
		label := fmt.Sprintf("%dp", item.height)
		if item.note != "" && item.note != label {
			label += " (" + item.note + ")"
		}
		if item.filesize > 0 {
			label += dimStyle.Render(" ~" + formatBytes(item.filesize))
		}

		line := "  "
		if i == m.cursor {
			line += "▸ " + selectedStyle.Render(label)
		} else {
			line += "  " + itemStyle.Render(label)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %d resolution(s) available", len(m.items))))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
