# NeoDLP Commands

## Overview

NeoDLP is built with [Cobra](https://github.com/spf13/cobra) and exposes four
commands plus the root command.

## Root

```bash
neodlp [flags]
neodlp [command]
```

Running `neodlp` with no arguments shows help. Passing URLs directly downloads
them (equivalent to `neodlp dl`).

## download / dl

Download media from one or more URLs. Uses an interactive TUI by default.

```bash
neodlp download [flags] <url> [url...]
neodlp dl [flags] <url> [url...]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--quality` | `-q` | `best` | Video quality: `best`, `1080p`, `720p`, `audio-only` |
| `--format` | `-f` | `auto` | Output format: `auto`, `mp4`, `mkv`, `mp3`, `m4a` |
| `--output-dir` | `-o` | *config value* | Custom output directory |
| `--no-playlist` | `-n` | `false` | Download only single video, not playlist |
| `--audio-only` | `-a` | `false` | Extract audio only (overrides `--quality`) |
| `--rate-limit` | `-r` | *none* | Bandwidth limit (e.g. `10M`, `5M`) |
| `--proxy` | `-p` | *none* | Proxy URL |

### Examples

```bash
neodlp dl "https://youtu.be/dQw4w9WgXcQ"
neodlp download -q 1080p -f mp4 "https://youtu.be/dQw4w9WgXcQ"
neodlp dl -a "https://youtu.be/dQw4w9WgXcQ"
neodlp dl -r 5M "https://youtu.be/dQw4w9WgXcQ"
neodlp dl -p "http://127.0.0.1:8080" "https://youtu.be/dQw4w9WgXcQ"
neodlp dl -n "https://youtube.com/playlist?list=PL..."
```

## info

Show media metadata without downloading.

```bash
neodlp info <url>
```

### Output

```
╭────────────────────── neodlp ──────────────────────╮
│      Author : RK Riad Khan                         │
│      Version: v0.1.0                               │
│      Commit : 2b4b657                              │
│      GitHub : rkriad585/neodlp                     │
╰────────────────────────────────────────────────────╯

Fetching info for: https://youtu.be/dQw4w9WgXcQ

  ID           : dQw4w9WgXcQ
  Title        : Rick Astley - Never Gonna Give You Up
  Uploader     : Rick Astley
  Duration     : 3m32s
  Views        : 1500000000
  Likes        : 20000000
  Upload date  : 20091025
  Platform     : youtube
  Formats avail: 18
```

### Fields

| Field | Description |
|-------|-------------|
| `ID` | Media ID on the platform |
| `Title` | Media title |
| `Uploader` | Channel or uploader name |
| `Channel` | Channel name (if different from uploader) |
| `Duration` | Length in seconds |
| `Views` | View count |
| `Likes` | Like count |
| `Upload date` | Date in YYYYMMDD format |
| `Platform` | Source platform (youtube, instagram, etc.) |
| `Format` | Default format string |
| `Formats avail` | Number of available formats |

## config

Manage the TOML configuration file.

```bash
neodlp config           # Show current config
neodlp config get <key> # Get a specific value
neodlp config set <key> <value>  # Set a value
neodlp config edit      # Open in editor
```

### Keys

| Key | Example |
|-----|---------|
| `download.output_dir` | `~/Downloads/Neodlp` |
| `download.quality` | `best` |
| `download.format` | `auto` |
| `download.concurrent_fragments` | `5` |
| `download.rate_limit` | `10M` |
| `network.proxy` | `http://127.0.0.1:8080` |
| `network.cookies_from_browser` | `firefox` |

## version

Show version and commit information.

```bash
neodlp version
```

Output includes the ASCII banner with version and git commit hash.
