package tui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lrstanley/go-ytdlp"

	"neodlp/internal/downloader"
	"neodlp/internal/queue"
)

// ── styles ──────────────────────────────────────────────────────────────────

var (
	multiTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7C3AED")).
			Padding(0, 3).
			Bold(true)

	jobQueuedStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	jobDownloadingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	jobProcessingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true)
	jobDoneStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
	jobFailedStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)
	jobURLStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB"))
	jobStatStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	footerStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	footerDimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	multiSeparator      = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#374151")).
				Render(strings.Repeat("─", 60))
)

// ── job state ───────────────────────────────────────────────────────────────

type jobStatus int

const (
	statusQueued jobStatus = iota
	statusDownloading
	statusProcessing
	statusDone
	statusFailed
)

type jobState struct {
	url       string
	status    jobStatus
	percent   float64
	speed     string
	eta       string
	size      string
	err       error
	startedAt time.Time
	elapsed   time.Duration
}

func (j jobState) statusIcon() string {
	switch j.status {
	case statusQueued:
		return "⏳"
	case statusDownloading:
		return "⬇"
	case statusProcessing:
		return "⚙"
	case statusDone:
		return "✓"
	case statusFailed:
		return "✗"
	}
	return " "
}

func (j jobState) statusStyle() lipgloss.Style {
	switch j.status {
	case statusQueued:
		return jobQueuedStyle
	case statusDownloading:
		return jobDownloadingStyle
	case statusProcessing:
		return jobProcessingStyle
	case statusDone:
		return jobDoneStyle
	case statusFailed:
		return jobFailedStyle
	}
	return jobQueuedStyle
}

// ── messages ────────────────────────────────────────────────────────────────

type multiProgressMsg struct {
	jobIndex int
	progress ytdlp.ProgressUpdate
}

type multiFinishedMsg struct {
	results []queue.Result
}

// ── model ───────────────────────────────────────────────────────────────────

type multiModel struct {
	jobs          []jobState
	progressBars  []progress.Model
	spinner       spinner.Model
	urls          []string
	opts          downloader.Options
	maxConcurrent int
	width         int
	height        int
	scrollOffset  int
	done          bool
	err           error
	started       time.Time
	completed     int
	failed        int
	active        int
	ctx           context.Context
	cancel        context.CancelFunc
}

func newMultiModel(urls []string, opts downloader.Options, maxConcurrent int, ctx context.Context, cancel context.CancelFunc) multiModel {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(accentColor)
	s.Spinner = spinner.Dot

	jobs := make([]jobState, len(urls))
	bars := make([]progress.Model, len(urls))
	for i, u := range urls {
		jobs[i] = jobState{
			url:    u,
			status: statusQueued,
		}
		bars[i] = progress.New(
			progress.WithDefaultGradient(),
			progress.WithGradient("#7C3AED", "#10B981"),
			progress.WithoutPercentage(),
		)
		bars[i].Width = 20
	}

	return multiModel{
		jobs:          jobs,
		progressBars:  bars,
		spinner:       s,
		urls:          urls,
		opts:          opts,
		maxConcurrent: maxConcurrent,
		started:       time.Now(),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// StartMulti launches the multi-download TUI for parallel downloads.
func StartMulti(urls []string, opts downloader.Options, maxConcurrent int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := newMultiModel(urls, opts, maxConcurrent, ctx, cancel)
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Store the program so progress callbacks can send messages
	Program = p

	final, err := p.Run()
	if err != nil {
		return err
	}
	if fm, ok := final.(multiModel); ok && fm.err != nil {
		return fm.err
	}
	return nil
}

func (m multiModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.launchQueue())
}

func (m multiModel) launchQueue() tea.Cmd {
	return func() tea.Msg {
		q := queue.NewQueue(m.maxConcurrent)
		for _, u := range m.urls {
			q.Add(u, m.opts)
		}

		results := q.Run(m.ctx, func(update queue.ProgressUpdate) {
			if Program != nil {
				Program.Send(multiProgressMsg{
					jobIndex: update.JobIndex,
					progress: update.Progress,
				})
			}
		})

		return multiFinishedMsg{results: results}
	}
}

func (m multiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		barWidth := min(msg.Width-45, 30)
		if barWidth < 10 {
			barWidth = 10
		}
		for i := range m.progressBars {
			m.progressBars[i].Width = barWidth
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		case "up", "k":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "down", "j":
			maxScroll := len(m.jobs) - m.visibleRows()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scrollOffset < maxScroll {
				m.scrollOffset++
			}
		}
		if m.done {
			return m, tea.Quit
		}

	case multiProgressMsg:
		idx := msg.jobIndex
		if idx >= 0 && idx < len(m.jobs) {
			job := &m.jobs[idx]
			prog := msg.progress

			switch prog.Status {
			case "downloading":
				if job.status != statusDownloading {
					job.status = statusDownloading
					job.startedAt = time.Now()
					m.active++
				}
			case "processing":
				if job.status == statusDownloading {
					m.active--
				}
				job.status = statusProcessing
			case "finished":
				job.status = statusDone
			case "error":
				job.status = statusFailed
			}

			job.percent = prog.Percent()

			// Calculate speed
			elapsed := time.Since(job.startedAt)
			if elapsed > 0 && prog.DownloadedBytes > 0 {
				rate := float64(prog.DownloadedBytes) / elapsed.Seconds()
				job.speed = formatBytes(int(rate)) + "/s"
			}

			// ETA
			eta := prog.ETA()
			if eta > 0 && eta < 24*time.Hour {
				if eta >= time.Minute {
					job.eta = fmt.Sprintf("%dm%ds", int(eta.Minutes()), int(eta.Seconds())%60)
				} else {
					job.eta = fmt.Sprintf("%ds", int(eta.Seconds()))
				}
			} else {
				job.eta = ""
			}

			// Size
			if prog.DownloadedBytes > 0 {
				job.size = formatBytes(prog.DownloadedBytes)
				if prog.TotalBytes > 0 {
					job.size += "/" + formatBytes(prog.TotalBytes)
				}
			}

			return m, m.progressBars[idx].SetPercent(prog.Percent() / 100)
		}

	case multiFinishedMsg:
		m.done = true
		m.completed = 0
		m.failed = 0
		m.active = 0
		for _, r := range msg.results {
			idx := r.Job.Index
			if idx >= 0 && idx < len(m.jobs) {
				if r.Err != nil {
					m.jobs[idx].status = statusFailed
					m.jobs[idx].err = r.Err
					m.failed++
				} else {
					m.jobs[idx].status = statusDone
					m.jobs[idx].percent = 100
					m.completed++
				}
				m.jobs[idx].elapsed = r.FinishedAt.Sub(r.StartedAt)
			}
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		// Route frame messages to all active progress bars
		var cmds []tea.Cmd
		for i := range m.progressBars {
			pm, cmd := m.progressBars[i].Update(msg)
			m.progressBars[i] = pm.(progress.Model)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m multiModel) visibleRows() int {
	// Reserve lines for header (4), footer (5), padding (4)
	available := m.height - 13
	if available < 3 {
		available = 3
	}
	// Each job takes 2 lines (content + spacing)
	return available / 2
}

func (m multiModel) View() string {
	var b strings.Builder

	width := m.width
	if width == 0 {
		width = 80
	}
	boxWidth := width
	if boxWidth > 90 {
		boxWidth = 90
	}

	// ── Title ──
	elapsed := time.Since(m.started).Round(time.Second)
	titleText := fmt.Sprintf(" neodlp — %d downloads ", len(m.jobs))
	b.WriteString(lipgloss.NewStyle().
		Width(boxWidth).
		Align(lipgloss.Center).
		Render(multiTitleStyle.Render(titleText)))
	b.WriteString("\n\n")

	// ── Stats line ──
	activeCount := 0
	doneCount := 0
	failCount := 0
	queuedCount := 0
	for _, j := range m.jobs {
		switch j.status {
		case statusDownloading, statusProcessing:
			activeCount++
		case statusDone:
			doneCount++
		case statusFailed:
			failCount++
		case statusQueued:
			queuedCount++
		}
	}
	statsLine := fmt.Sprintf("  Active: %d  •  Done: %d  •  Failed: %d  •  Queued: %d  •  Elapsed: %s",
		activeCount, doneCount, failCount, queuedCount, elapsed)
	b.WriteString(footerDimStyle.Render(statsLine))
	b.WriteString("\n\n")
	b.WriteString("  " + multiSeparator)
	b.WriteString("\n\n")

	// ── Job rows ──
	visRows := m.visibleRows()
	startIdx := m.scrollOffset
	endIdx := startIdx + visRows
	if endIdx > len(m.jobs) {
		endIdx = len(m.jobs)
	}

	urlMaxLen := boxWidth - 50
	if urlMaxLen < 15 {
		urlMaxLen = 15
	}

	for i := startIdx; i < endIdx; i++ {
		job := m.jobs[i]

		// Status icon + index
		icon := job.statusIcon()
		stStyle := job.statusStyle()

		// Truncate URL
		displayURL := truncateURL(job.url, urlMaxLen)

		// Build the row
		prefix := fmt.Sprintf("  %s %s ", stStyle.Render(icon), stStyle.Render(fmt.Sprintf("[%d]", i+1)))
		b.WriteString(prefix)
		b.WriteString(jobURLStyle.Render(displayURL))
		b.WriteString("\n")

		// Second line: progress bar + stats
		b.WriteString("       ")
		switch job.status {
		case statusQueued:
			b.WriteString(jobQueuedStyle.Render("waiting..."))

		case statusDownloading:
			b.WriteString(m.spinner.View() + " ")
			b.WriteString(m.progressBars[i].View())
			b.WriteString(fmt.Sprintf(" %.0f%%", job.percent))
			if job.speed != "" {
				b.WriteString(jobStatStyle.Render("  " + job.speed))
			}
			if job.eta != "" {
				b.WriteString(jobStatStyle.Render("  ETA " + job.eta))
			}

		case statusProcessing:
			b.WriteString(m.spinner.View() + " ")
			b.WriteString(jobProcessingStyle.Render("processing..."))
			if job.size != "" {
				b.WriteString(jobStatStyle.Render("  " + job.size))
			}

		case statusDone:
			b.WriteString(jobDoneStyle.Render("✓ completed"))
			if job.elapsed > 0 {
				b.WriteString(jobStatStyle.Render(fmt.Sprintf("  in %s", job.elapsed.Round(time.Second))))
			}
			if job.size != "" {
				b.WriteString(jobStatStyle.Render("  " + job.size))
			}

		case statusFailed:
			errMsg := "download failed"
			if job.err != nil {
				errMsg = job.err.Error()
				errRunes := []rune(errMsg)
				if len(errRunes) > boxWidth-20 {
					errMsg = string(errRunes[:boxWidth-23]) + "..."
				}
			}
			b.WriteString(jobFailedStyle.Render("✗ " + errMsg))
		}
		b.WriteString("\n\n")
	}

	// Scroll indicator
	if len(m.jobs) > visRows {
		scrollInfo := fmt.Sprintf("  showing %d-%d of %d (↑/↓ to scroll)", startIdx+1, endIdx, len(m.jobs))
		b.WriteString(footerDimStyle.Render(scrollInfo))
		b.WriteString("\n")
	}

	// ── Footer ──
	b.WriteString("  " + multiSeparator)
	b.WriteString("\n\n")

	if m.done {
		summary := fmt.Sprintf("  ✓ All downloads finished — %d succeeded, %d failed", doneCount, failCount)
		if failCount > 0 {
			b.WriteString(jobFailedStyle.Render(summary))
		} else {
			b.WriteString(footerStyle.Render(summary))
		}
		b.WriteString("\n")
		b.WriteString(footerDimStyle.Render("  Press any key to exit"))
	} else {
		b.WriteString(footerDimStyle.Render(fmt.Sprintf("  max-concurrent: %d  •  q/Esc to cancel", m.maxConcurrent)))
	}

	b.WriteString("\n")
	return lipgloss.NewStyle().Padding(1, 1).Render(b.String())
}

// truncateURL shortens a URL for display, keeping the domain visible.
func truncateURL(u string, maxLen int) string {
	runeCount := utf8.RuneCountInString(u)
	if runeCount <= maxLen {
		return u
	}

	// Try to keep the domain part: strip protocol, show domain + truncated path
	stripped := u
	if idx := strings.Index(u, "://"); idx >= 0 {
		stripped = u[idx+3:]
	}

	strippedRunes := []rune(stripped)
	if len(strippedRunes) <= maxLen {
		return stripped
	}

	return string(strippedRunes[:maxLen-3]) + "..."
}
