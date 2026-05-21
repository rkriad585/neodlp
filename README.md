# NeoDLP

**Universal media downloader** — YouTube, Instagram, Facebook, X/Twitter, TikTok, and 1000+ sites.

Powered by [yt-dlp](https://github.com/yt-dlp/yt-dlp) under the hood. Built with Go. Ships with a real-time TUI.

```
╭────────────────────── neodlp ──────────────────────╮
│      Author : RK Riad Khan                         │
│      Version: v0.1.0                               │
│      Commit : 2b4b657                              │
│      GitHub : rkriad585/neodlp                     │
╰────────────────────────────────────────────────────╯
```

---

## Features

- **1000+ sites** — YouTube, Instagram, Facebook, X/Twitter, TikTok, SoundCloud, Twitch, Vimeo, Reddit, Telegram, Dailymotion, Bandcamp, Mixcloud, Rumble, Bilibili, and more
- **Interactive TUI** — real-time progress bar, speed, ETA, file size with BubbleTea
- **Audio extraction** — download audio-only as MP3
- **Format selection** — MP4, MKV, WebM, MOV, AVI, FLV, MP3, M4A, Opus, WAV; quality presets (best, 1080p, 720p)
- **Playlist control** — download single video or full playlist
- **Rate limiting** — throttle bandwidth (e.g. `10M`)
- **Proxy support** — HTTP/HTTPS/SOCKS proxy
- **Configurable** — TOML config file with `get/set/edit` commands
- **Cross-platform** — Windows, Linux, macOS
- **Metadata inspection** — view title, uploader, duration, views, likes, formats before downloading

---

## Installation

### Pre-built binaries

Download the latest release for your platform from the [Releases page](https://github.com/rkriad585/neodlp/releases).

### Build from source

```bash
git clone https://github.com/rkriad585/neodlp.git
cd neodlp
make build   # or: go build -o neodlp .
```

On Windows:

```powershell
.\build.ps1
```

### Dependencies

NeoDLP auto-downloads the `yt-dlp` binary on first run — no manual installation required.

---

## Usage

### Download media

```bash
# Download best quality (default)
neodlp dl "https://youtu.be/dQw4w9WgXcQ"

# Interactive format + resolution selection
neodlp dl -f "https://youtu.be/dQw4w9WgXcQ"

# Interactive resolution with specific format
neodlp dl -f --format-type mp4 "https://youtu.be/dQw4w9WgXcQ"

# Download with specific quality and format
neodlp dl -q 1080p --format-type mp4 "https://youtu.be/dQw4w9WgXcQ"

# Extract audio only
neodlp dl -a "https://youtu.be/dQw4w9WgXcQ"

# Download without playlist (single video)
neodlp dl -n "https://youtube.com/playlist?list=PL..."

# Limit bandwidth to 5 MB/s
neodlp dl -r 5M "https://youtu.be/dQw4w9WgXcQ"

# Use proxy
neodlp dl -p "http://127.0.0.1:8080" "https://youtu.be/dQw4w9WgXcQ"

# Multiple URLs
neodlp dl "https://youtu.be/abc" "https://youtu.be/xyz"
```

### Search and download

```bash
# Search YouTube, pick from results interactively
neodlp search "never gonna give you up"

# Limit search results
neodlp search -l 5 "lofi hip hop music"
```

### View media info

```bash
neodlp info "https://youtu.be/dQw4w9WgXcQ"
```

### Manage configuration

```bash
# Show current config
neodlp config

# Get a specific value
neodlp config get download.output_dir

# Set a value
neodlp config set download.quality "1080p"

# Open config in editor
neodlp config edit
```

### Show version

```bash
neodlp version
```

---

## Configuration

Config file location:

| Platform | Path |
|----------|------|
| Windows | `%USERPROFILE%\.config\neostore\neodlp\config.toml` |
| Linux/macOS | `~/.config/neostore/neodlp/config.toml` |

### Default config

```toml
[download]
output_dir = "~/Downloads/Neodlp"
quality = "best"
format = "auto"
concurrent_fragments = 5
rate_limit = ""

[network]
proxy = ""
cookies_from_browser = ""
```

### Options

| Key | Type | Description |
|-----|------|-------------|
| `download.output_dir` | string | Output directory for downloads |
| `download.quality` | string | Quality preset: `best`, `1080p`, `720p`, `audio-only` |
| `download.format` | string | Output format: `auto`, `mp4`, `mkv`, `webm`, `mov`, `avi`, `flv`, `mp3`, `m4a`, `opus`, `wav` |
| `download.concurrent_fragments` | int | Number of concurrent fragment downloads |
| `download.rate_limit` | string | Bandwidth limit (e.g. `10M`, `5M`) |
| `network.proxy` | string | Proxy URL (HTTP/HTTPS/SOCKS) |
| `network.cookies_from_browser` | string | Browser to extract cookies from (e.g. `firefox`, `chrome`) |

---

## How it works

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│   CLI    │────▶│Downloader│────▶│ yt-dlp  │────▶│  Media   │
│  (Cobra) │     │  (Go)   │     │ (Binary) │     │  File    │
└──────────┘     └──────────┘     └──────────┘     └──────────┘
      │                │
      │                ▼
      │         ┌──────────┐
      └────────▶│   TUI    │
                │(BubbleTea)│
                └──────────┘
```

1. **Cobra CLI** parses commands and flags in `cmd/root.go`
2. **Downloader** (`internal/downloader/`) wraps `go-ytdlp` — handles config resolution, option mapping, and execution
3. **yt-dlp** binary does the actual download work — auto-installed on first run
4. **TUI** (`internal/tui/`) uses BubbleTea to render real-time progress via callback
5. **Config** (`internal/config/`) manages TOML settings with `get/set/edit` commands
6. **Banner** (`internal/banner/`) renders the ASCII identity banner

### Project structure

```
├── main.go                        # Entry point
├── cmd/root.go                    # CLI commands & flags
├── internal/
│   ├── version/version.go         # Version & commit vars
│   ├── banner/banner.go           # ASCII banner
│   ├── config/config.go           # TOML config load/save
│   ├── downloader/downloader.go   # Download engine
│   └── tui/tui.go                 # BubbleTea real-time TUI
├── .version                       # Version file (v0.1.0)
├── Makefile                       # Unix build targets
└── build.ps1                      # Windows build script
```

---

## Building

### Unix (Linux/macOS)

```bash
make build          # Build for current platform
make build-all      # Cross-compile for all platforms
make clean          # Remove build artifacts
```

### Windows

```powershell
.\build.ps1                     # Build for current platform
.\build.ps1 -All                # Cross-compile for all platforms
.\build.ps1 -Clean              # Remove build artifacts
```

Builds inject the current git commit into the binary via `-ldflags`:

```bash
neodlp version
# v0.1.0 (2b4b657)
```

---

## Author

**RK Riad Khan**  
Email: rkriad585@gmail.com  
GitHub: [rkriad585](https://github.com/rkriad585)

---

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
