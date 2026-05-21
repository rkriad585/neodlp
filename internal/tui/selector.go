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
	title    string
	helpText string
	items    []string
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

func SelectContainerFormat() (string, error) {
	containers := []string{"mp4", "mkv", "mp3", "m4a"}
	m := selectorModel{
		title:    " select container ",
		helpText: "Choose a container format (↑/↓, Enter to select, Esc to cancel)",
		items:    containers,
	}
	return runSelector(&m)
}

type resValue struct {
	label string
	value string
}

func SelectResolution(info *ytdlp.ExtractedInfo) (string, error) {
	resolutions := extractResolutions(info)
	if len(resolutions) == 0 {
		return "", fmt.Errorf("no video formats found")
	}

	var resValues []resValue
	for _, r := range resolutions {
		label := fmt.Sprintf("%dp", r.height)
		if r.note != "" && r.note != label {
			label += " (" + r.note + ")"
		}
		if r.filesize > 0 {
			label += " ~" + formatBytes(r.filesize)
		}
		resValues = append(resValues, resValue{
			label: label,
			value: fmt.Sprintf("%dp", r.height),
		})
	}

	var labels []string
	for _, rv := range resValues {
		labels = append(labels, rv.label)
	}

	m := selectorModel{
		title:    " select resolution ",
		helpText: "Choose a resolution (↑/↓, Enter to select, Esc to cancel)",
		items:    labels,
	}

	selected, err := runSelector(&m)
	if err != nil {
		return "", err
	}
	if selected == "" {
		return "", fmt.Errorf("no resolution selected")
	}

	for _, rv := range resValues {
		if rv.label == selected {
			return rv.value, nil
		}
	}
	return "", fmt.Errorf("no resolution selected")
}

func runSelector(m *selectorModel) (string, error) {
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	m, ok := final.(*selectorModel)
	if !ok {
		return "", fmt.Errorf("unexpected model type")
	}
	if m.err != nil {
		return "", m.err
	}
	if m.done && m.selected >= 0 && m.selected < len(m.items) {
		return m.items[m.selected], nil
	}
	return "", fmt.Errorf("no selection made")
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

func (m *selectorModel) Init() tea.Cmd {
	return nil
}

func (m *selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m *selectorModel) View() string {
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
		Render(titleStyle.Render(m.title)))
	b.WriteString("\n\n")

	b.WriteString("  " + dimStyle.Render(m.helpText))
	b.WriteString("\n\n")

	for i, item := range m.items {
		line := "  "
		if i == m.cursor {
			line += "▸ " + selectedStyle.Render(item)
		} else {
			line += "  " + itemStyle.Render(item)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %d option(s) available", len(m.items))))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
