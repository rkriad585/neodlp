package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lrstanley/go-ytdlp"

	"neodlp/internal/downloader"
)

var (
	Program *tea.Program

	accentColor  = lipgloss.Color("#7C3AED")
	successColor = lipgloss.Color("#10B981")
	errorColor   = lipgloss.Color("#EF4444")
	mutedColor   = lipgloss.Color("#6B7280")

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(accentColor).
			Padding(0, 3).
			Bold(true)

	infoKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA")).
			Width(14)

	infoVal = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))

	separator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#374151")).
			Render(strings.Repeat("─", 50))
)

type progressMsg struct {
	Progress ytdlp.ProgressUpdate
}

type finishedMsg struct {
	Err error
}

type readyMsg struct{}

type model struct {
	url      string
	opts     downloader.Options
	spinner  spinner.Model
	progress progress.Model
	prog     ytdlp.ProgressUpdate
	status   string
	err      error
	done     bool
	width    int
	started  time.Time
	ctx      context.Context
	cancel   context.CancelFunc
}

func initialModel(url string, opts downloader.Options, ctx context.Context, cancel context.CancelFunc) model {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(accentColor)
	s.Spinner = spinner.Moon

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithGradient("#7C3AED", "#10B981"),
	)

	return model{
		url:      url,
		opts:     opts,
		spinner:  s,
		progress: p,
		status:   "Preparing...",
		started:  time.Now(),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func Start(url string, opts downloader.Options) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := initialModel(url, opts, ctx, cancel)
	Program = tea.NewProgram(m, tea.WithAltScreen())
	final, err := Program.Run()
	if err != nil {
		return err
	}
	if m, ok := final.(model); ok && m.err != nil {
		return m.err
	}
	return nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg { return readyMsg{} })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.progress.Width = min(msg.Width-8, 60)

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" || msg.String() == "esc" {
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}
		if m.done {
			return m, tea.Quit
		}

	case readyMsg:
		m.status = "Starting download..."
		return m, launchDownload(m.ctx, m.url, m.opts)

	case progressMsg:
		m.prog = msg.Progress
		switch msg.Progress.Status {
		case "downloading":
			m.status = "Downloading..."
		case "processing":
			m.status = "Processing..."
		case "finished":
			m.status = "Finished"
		case "error":
			m.status = "Error"
		}
		return m, m.progress.SetPercent(msg.Progress.Percent() / 100)

	case finishedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.status = "Failed"
		} else {
			m.status = "Completed"
		}
		m.done = true
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		pm, cmd := m.progress.Update(msg)
		m.progress = pm.(progress.Model)
		return m, cmd
	}

	return m, nil
}

func launchDownload(ctx context.Context, url string, opts downloader.Options) tea.Cmd {
	return func() tea.Msg {
		if Program == nil {
			return finishedMsg{Err: fmt.Errorf("program not initialized")}
		}

		_, err := downloader.DownloadWithProgress(
			ctx,
			[]string{url},
			opts,
			func(prog ytdlp.ProgressUpdate) {
				Program.Send(progressMsg{Progress: prog})
			},
		)
		if err != nil {
			if ctx.Err() != nil {
				return finishedMsg{Err: fmt.Errorf("download cancelled")}
			}
			return finishedMsg{Err: err}
		}
		return finishedMsg{}
	}
}

func formatBytes(b int) string {
	if b == 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func speedLine(prog ytdlp.ProgressUpdate) string {
	speed := prog.DownloadedBytes
	elapsed := time.Since(prog.Started)
	if elapsed < time.Second {
		elapsed = time.Second
	}
	rate := float64(speed) / elapsed.Seconds()
	return formatBytes(int(rate)) + "/s"
}

func etaLine(prog ytdlp.ProgressUpdate) string {
	eta := prog.ETA()
	if eta <= 0 || eta > 24*time.Hour {
		return "—"
	}
	if eta >= time.Hour {
		return fmt.Sprintf("%dh%dm", int(eta.Hours()), int(eta.Minutes())%60)
	}
	if eta >= time.Minute {
		return fmt.Sprintf("%dm%ds", int(eta.Minutes()), int(eta.Seconds())%60)
	}
	return fmt.Sprintf("%ds", int(eta.Seconds()))
}

func (m model) View() string {
	var b strings.Builder

	width := m.width
	if width == 0 {
		width = 70
	}

	boxWidth := width
	if boxWidth > 72 {
		boxWidth = 72
	}

	b.WriteString(lipgloss.NewStyle().
		Width(boxWidth).
		Align(lipgloss.Center).
		Render(titleStyle.Render(" neodlp ")))
	b.WriteString("\n\n")

	elapsed := time.Since(m.started).Round(time.Second)

	info := [][2]string{
		{"URL", m.url},
		{"Status", m.status},
		{"Elapsed", elapsed.String()},
	}

	if !m.done && m.prog.Status != "" {
		prog := m.prog
		info = append(info, [2]string{"Speed", speedLine(prog)})
		info = append(info, [2]string{"ETA", etaLine(prog)})

		if prog.DownloadedBytes > 0 {
			downloaded := formatBytes(prog.DownloadedBytes)
			total := formatBytes(prog.TotalBytes)
			info = append(info, [2]string{"Size", downloaded + " / " + total})
		}

		if prog.Filename != "" {
			fn := prog.Filename
			if len(fn) > boxWidth-20 {
				fn = "..." + fn[len(fn)-boxWidth+23:]
			}
			info = append(info, [2]string{"File", fn})
		}
	}

	for _, row := range info {
		b.WriteString("  ")
		b.WriteString(infoKey.Render(row[0]))
		b.WriteString(infoVal.Render(row[1]))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString("  " + separator)
	b.WriteString("\n\n")

	if m.done {
		style := lipgloss.NewStyle().Bold(true)
		if m.err != nil {
			style = style.Foreground(errorColor)
			b.WriteString("  " + style.Render("✗ " + m.err.Error()))
		} else {
			style = style.Foreground(successColor)
			b.WriteString("  " + style.Render("✓ Download completed!"))
		}
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("  Press any key to exit"))
	} else {
		percent := m.prog.Percent()
		b.WriteString("  " + m.spinner.View() + " ")

		if percent > 0 {
			bar := m.progress.View()
			b.WriteString(bar)
			b.WriteString(fmt.Sprintf(" %.1f%%", percent))
		} else {
			b.WriteString(lipgloss.NewStyle().
				Foreground(mutedColor).
				Render("waiting..."))
		}
	}

	b.WriteString("\n")
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
