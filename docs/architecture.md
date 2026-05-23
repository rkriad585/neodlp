# Architecture

NeoDLP is a layered Go application. This document describes how the pieces fit
together.

## Overview

```
┌─────────────────────────────────────────────────────┐
│                   main.go                           │
│                   cmd.Execute()                     │
└───────────────────┬─────────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────────┐
│                  cmd/root.go                        │
│  ┌─────────┐ ┌──────────┐ ┌────────┐ ┌─────────┐  │
│  │download │ │   info   │ │ config │ │ version │  │
│  │  / dl   │ │          │ │get/set │ │         │  │
│  └────┬────┘ └────┬─────┘ └────────┘ └─────────┘  │
└───────┼───────────┼────────────────────────────────┘
        │           │
┌───────▼───────────▼────────────────────────────────┐
│             internal/downloader/                    │
│  ┌──────────┐  ┌─────────────┐  ┌───────────────┐  │
│  │Download()│  │DownloadWith │  │    Info()     │  │
│  │          │  │ Progress()  │  │               │  │
│  └────┬─────┘  └──────┬──────┘  └──────┬────────┘  │
│       │               │                │           │
│       └───────┬───────┘                │           │
│               │                        │           │
│        resolveOpts()                   │           │
│         + applyOpts()                  │           │
└───────────────┼────────────────────────┼───────────┘
                │                        │
┌───────────────▼────────────────────────▼───────────┐
│              go-ytdlp library                       │
│  ┌──────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ ytdlp.New│  │ ProgressFunc │  │ DumpJSON()   │  │
│  │ .Run()   │  │ .Run()       │  │ .Run()       │  │
│  └────┬─────┘  └──────┬───────┘  └──────┬───────┘  │
└───────┼───────────────┼─────────────────┼──────────┘
        │               │                 │
┌───────▼───────────────▼─────────────────▼──────────┐
│                 yt-dlp binary                       │
│  Arguments: Paths, Output, FormatSort, NoPlaylist  │
│            ExtractAudio, LimitRate, Proxy, etc.     │
└────────────────────────────────────────────────────┘
```

## Package responsibilities

### `main.go`

Tiny entry point. Calls `cmd.Execute()`.

### `cmd/`

This package houses subcommands:

- `root.go`: Standard root command, flags, and `download` / `dl` subcommand execution.
- `serve.go`: Start the local HTTP REST API daemon daemonizing queue operations.
- `watch.go`: Start the folder/file watcher suite.
- `self_update.go`: Command to auto-update the binary from GitHub.
- `uninstall.go`: Uninstallation utility flags.

The `downloadRun` function:
1. Ensures the output directory exists
2. Resolves CLI flags into `downloader.Options`
3. Checks URL arguments count:
   - **Single URL**: Launches the classic TUI (`tui.Start(url, opts)`)
   - **Multiple URLs**: Launches the parallel dashboard (`tui.StartMulti(urls, opts, maxConcurrent)`)

### `internal/version/version.go`

Holds `Version` and `Commit` string variables. `Commit` is injected at build
time via `-ldflags`:

```bash
go build -ldflags="-X neodlp/internal/version.Commit=$(git rev-parse --short HEAD)" -o neodlp .
```

### `internal/banner/banner.go`

Renders the ASCII identity banner using Unicode box-drawing characters.
Contains author name, version, commit hash, and GitHub URL.

### `internal/config/config.go`

Manages the TOML config file at `~/.config/neostore/neodlp/config.toml`.

Key functions:

| Function | Description |
|----------|-------------|
| `Load()` | Reads config from disk; auto-creates with defaults if missing |
| `Default()` | Returns default config values |
| `Save()` | Writes config to disk, creating parent directories |
| `Path()` | Returns the config file path |
| `Dir()` | Returns the config directory path |

### `internal/downloader/downloader.go`

Core download engine wrapping `go-ytdlp`.

Key types:

```go
type Options struct {
    Quality        string
    Format         string
    OutputDir      string
    NoPlaylist     bool
    AudioOnly      bool
    RateLimit      string
    Proxy          string
    WriteThumbnail bool
    WriteSubs      string
    WriteAutoSubs  bool
    EmbedMetadata  bool
    UploadTarget   string
}

type Result struct {
    URL      string
    Title    string
    Filename string
    Platform string
}
```

**Config resolution order:**
1. Command-line flag (highest priority)
2. Config file value
3. Default value (lowest priority)

### `internal/queue/`

Orchestrates concurrent downloads:
- Bounded workers execute downloads using goroutines.
- Communicates updates via progress channels to BubbleTea dashboards or REST API responses.

### `internal/uploader/`

Contains cloud sync providers:
- **Telegram bot uploader**: Uploads files directly as documents using form requests.
- **Discord webhook uploader**: Pushes attachments to channel webhooks.
- **Custom command uploader**: Spawns shell processes replacing `%file%` placeholders.

### `internal/tui/`

BubbleTea TUI components. Renders either the single-URL layout (`tui.go`) or the parallel multi-item dashboard queue view (`multi.go`).

**TUI Architecture pattern:**

```
Update() receives readyMsg
  └─▶ starts queue.Run() worker pool
       └─▶ workers invoke downloader.DownloadWithProgress()
            └─▶ progress callback dispatches Program.Send(multiProgressMsg)
                 └─▶ Update() updates UI row metrics → View() re-renders
```

## Data flow: download

```
User runs: neodlp dl "https://youtu.be/abc"
  ├─ cmd/root.go: downloadRun()
  │   └─ tui.Start(url, opts)
  │       └─ tea.NewProgram(model, tea.WithAltScreen())
  │           └─ Program.Run() blocks
  │               ├─ model.Init() → readyMsg
  │               ├─ model.Update(readyMsg) → launchDownload()
  │               │   └─ downloader.DownloadWithProgress(ctx, urls, opts, callback)
  │               │       └─ ytdlp.New()
  │               │           .Paths(outputDir)
  │               │           .Output(template)
  │               │           .ProgressFunc(100ms, callback)
  │               │           .Run(ctx, url)
  │               │               └─ yt-dlp process
  │               │                   ├─ progress callback → Program.Send(progressMsg)
  │               │                   └─ process completes → finishedMsg
  │               ├─ model.Update(progressMsg) → update bar/speed/ETA
  │               └─ model.Update(finishedMsg) → tea.Quit
  └─ Program.Run() returns
```

## Build system

Two build scripts:

| Script | Platform | Purpose |
|--------|----------|---------|
| `Makefile` | Unix | `make build`, `make build-all`, `make clean`, `make tag` |
| `build.ps1` | Windows | `.\build.ps1`, `.\build.ps1 -All`, `.\build.ps1 -Clean`, `.\build.ps1 -Tag` |

Both inject the git commit hash via `-ldflags` and read the version from
`.version`.

## Dependencies

| Library | Purpose |
|---------|---------|
| `go-ytdlp` | Go bindings for yt-dlp binary |
| `cobra` | CLI framework |
| `go-toml/v2` | TOML config parsing |
| `bubbletea` | TUI framework |
| `bubbles` | TUI components (progress bar, spinner) |
| `lipgloss` | Terminal styling |
