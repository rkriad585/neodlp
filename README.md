# NeoDLP

**Universal media downloader** — YouTube, Instagram, Facebook, X/Twitter, TikTok, and 1000+ sites.

Powered by [yt-dlp](https://github.com/yt-dlp/yt-dlp) under the hood. Built with Go. Ships with a real-time TUI.

```
╭────────────────────── neodlp ──────────────────────╮
│      Author : RK Riad Khan                         │
│      Version: v0.2.1                               │
│      Commit : 2b4b657                              │
│      GitHub : rkriad585/neodlp                     │
╰────────────────────────────────────────────────────╯
```

---

## Features

- **1000+ sites** — YouTube, Instagram, Facebook, X/Twitter, TikTok, SoundCloud, Twitch, Vimeo, Reddit, Telegram, Dailymotion, Bandcamp, Mixcloud, Rumble, Bilibili, and more
- **Parallel TUI Queue** — Concurrent multi-download dashboard with individual progress bars, speed, and ETA rendering using BubbleTea
- **REST API Daemon (`neodlp serve`)** — Run a background HTTP JSON service to trigger/track downloads remotely
- **File/Folder Watcher (`neodlp watch`)** — Automate downloads by watching folders or text files for new links
- **Metadata & Album Art Embedding** — Embed high-res thumbnails and ID3 tags natively with `--embed-metadata`
- **Push-to-Cloud Uploaders** — Auto-upload finished downloads to Discord, Telegram, or custom shell scripts, clearing local files afterwards
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

### Automatic remote installation

You can install `neodlp` instantly on your system globally using the following one-liners:

* **Windows (PowerShell)**:
  ```powershell
  irm https://raw.githubusercontent.com/rkriad585/neodlp/main/install.ps1 | iex
  ```

* **Linux / macOS (Bash)**:
  ```bash
  curl -fsSL https://raw.githubusercontent.com/rkriad585/neodlp/main/installer.sh | sh
  ```

### Pre-built binaries

Alternatively, download the latest compiled binary for your specific platform and architecture from the [Releases page](https://github.com/rkriad585/neodlp/releases).

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

NeoDLP installers automatically pre-download the official `yt-dlp` binary (v2026.03.17) during the installation process so that the application works out-of-the-box. If it is ever missing or deleted, NeoDLP will still auto-download it on first run.


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

# Multiple URLs (downloads in parallel, default 3 concurrent)
neodlp dl "https://youtu.be/abc" "https://youtu.be/xyz" "https://youtu.be/123"

# Concurrency control
neodlp dl -c 5 "https://youtu.be/abc" "https://youtu.be/xyz"

# Smart Metadata & ID3 tags embedding
neodlp dl --embed-metadata "https://youtu.be/abc"

# Post-download cloud upload (telegram/discord/custom)
neodlp dl -u discord "https://youtu.be/abc"
```

### Background REST API Daemon

Start the local REST API daemon server to queue and track downloads asynchronously from external apps, tools, or browser extensions:

```bash
# Start daemon on default host and port (127.0.0.1:12121)
neodlp serve

# Custom host and port
neodlp serve --host 0.0.0.0 --port 8080
```

### Real-Time Folder/File Watcher

Automatically monitor a file or directory for newly added links to download:

```bash
# Watch a specific text file (polls for URL appends, downloads, comments out)
neodlp watch -f ~/watch_downloads.txt

# Watch a directory for incoming *.txt / *.url batch files
neodlp watch -d ~/watch_folder
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

### Self-update

```bash
# Update neodlp binary itself to the latest version automatically
neodlp self-update
```

### Self-uninstall

To completely uninstall `neodlp`, remove all its files, config directories, and the binary from your system instantly:

```bash
# Call the self-uninstall flag directly on the executable
neodlp --selfuninstall
```

Alternatively, you can run the remote uninstaller script directly:

* **Windows (PowerShell)**:
  ```powershell
  Invoke-RestMethod -Uri "https://raw.githubusercontent.com/rkriad585/neodlp/main/installer.ps1" | Invoke-Expression -ArgumentList "--selfuninstall"
  ```

* **Linux / macOS (Bash)**:
  ```bash
  curl -fsSL https://raw.githubusercontent.com/rkriad585/neodlp/main/installer.sh | sh -s -- --selfuninstall
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

[upload.telegram]
bot_token = ""
chat_id = ""

[upload.discord]
webhook_url = ""

[upload.custom]
command = ""
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
| `upload.telegram.bot_token` | string | Telegram bot credentials |
| `upload.telegram.chat_id` | string | Target Telegram chat/channel destination |
| `upload.discord.webhook_url` | string | Webhook URL for Discord attachments |
| `upload.custom.command` | string | Shell command line containing `%file%` template variable |

---

## How it works

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│   CLI    │────▶│Downloader│────▶│ yt-dlp  │────▶│  Media   │
│  (Cobra) │     │  (Go)   │     │ (Binary) │     │  File    │
└──────────┘     └──────────┘     └──────────┘     └──────────┘
      │                │                                │
      │                ▼                                ▼
      │         ┌──────────┐                     ┌──────────┐
      └────────▶│TUI / Queue│                    │ Uploader │
                │(BubbleTea)│                    │ (Cloud)  │
                └──────────┘                     └──────────┘
```

1. **Cobra CLI** parses commands and flags (`download`, `search`, `serve`, `watch`, `config`)
2. **Parallel Queue** (`internal/queue/`) schedules enqueued jobs concurrently inside goroutine worker pools
3. **Downloader** (`internal/downloader/`) wraps `go-ytdlp` — handles metadata embedding and triggers hooks
4. **TUI** (`internal/tui/`) renders single URL progress bars or multi-item dashboard views
5. **Uploader** (`internal/uploader/`) executes Telegram/Discord API transfers or runs shell execution pipelines
6. **Config** (`internal/config/`) handles configuration loading, saving, and command modifications

### Project structure

```
├── main.go                        # Entry point
├── cmd/
│   ├── root.go                    # CLI command registry & download command
│   ├── serve.go                   # REST API daemon (neodlp serve)
│   ├── watch.go                   # Folder/file watcher (neodlp watch)
│   ├── self_update.go             # Binary auto-update command
│   └── uninstall.go               # Self-uninstall flag actions
├── internal/
│   ├── version/version.go         # Version metadata definitions
│   ├── banner/banner.go           # ASCII banner builder
│   ├── config/config.go           # Configuration TOML engine
│   ├── queue/queue.go             # Concurrent orchestrator queue
│   ├── uploader/uploader.go       # Cloud uploader targets
│   ├── downloader/                # Downloader logic & extractors
│   └── tui/                       # BubbleTea dashboards (single & multi)
├── .version                       # Version definition file (v0.2.1)
├── Makefile                       # Unix build automation rules
└── build.ps1                      # Windows build automation script
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
