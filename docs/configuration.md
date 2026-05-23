# Configuration

NeoDLP uses a TOML configuration file stored at:

| Platform | Path |
|----------|------|
| Windows | `%USERPROFILE%\.config\neostore\neodlp\config.toml` |
| Linux    | `~/.config/neostore/neodlp/config.toml` |
| macOS    | `~/.config/neostore/neodlp/config.toml` |

The file is auto-created with defaults on first run.

## Default config

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

## Reference

### `[download]` section

#### `output_dir`

Path where downloaded files are saved. Supports `~` for home directory.

```toml
output_dir = "~/Videos/Neodlp"
output_dir = "D:/Media/Downloads"
```

#### `quality`

Quality preset when `format` is `auto`. Values:

| Value | Behaviour |
|-------|-----------|
| `best` | Highest available resolution |
| `1080p` | Prefer 1080p |
| `720p` | Prefer 720p |
| `audio-only` | Extract audio only (same as `--audio-only` flag) |

```toml
quality = "1080p"
```

#### `format`

Output container format. Values: `auto`, `mp4`, `mkv`, `mp3`, `m4a`.

| Value | Behaviour |
|-------|-----------|
| `auto` | Let yt-dlp decide based on quality preference |
| `mp4` | Recode to MP4 |
| `mkv` | Recode to MKV |
| `mp3` | Audio-only MP3 (requires `quality = "audio-only"` or `--audio-only`) |
| `m4a` | Audio-only M4A |

```toml
format = "mkv"
```

#### `concurrent_fragments`

Number of fragments (chunks) to download simultaneously for DASH/hls
streams. Higher values may improve speed on fast connections.

```toml
concurrent_fragments = 10
```

#### `rate_limit`

Maximum bandwidth usage. Supports common suffixes: `K`, `M`, `G`.

```toml
rate_limit = "10M"
rate_limit = "5M"
```

### `[network]` section

#### `proxy`

HTTP/HTTPS/SOCKS proxy for all requests.

```toml
proxy = "http://127.0.0.1:8080"
proxy = "socks5://127.0.0.1:1080"
```

#### `cookies_from_browser`

Extract cookies from a browser to access restricted/age-gated content.

```toml
cookies_from_browser = "firefox"
cookies_from_browser = "chrome"
cookies_from_browser = "brave"
```

### `[upload]` section

Configure destinations for the `--upload` trigger.

#### `upload.telegram`
- `bot_token`: API token of your Telegram bot.
- `chat_id`: Telegram channel, group, or user ID to post the media file to.

#### `upload.discord`
- `webhook_url`: Discord channel integration webhook URL.

#### `upload.custom`
- `command`: Command template run on complete. Supports `%file%` (absolute path), `%filename%` (file name only), and `%dir%` (target directory) placeholders.

## CLI overrides

Every config key can be overridden at runtime with a command flag.
CLI flags take precedence over config file values.

| Config key | CLI flag |
|------------|----------|
| `download.output_dir` | `--output-dir` / `-o` |
| `download.quality` | `--quality` / `-q` |
| `download.format` | `--format` / `-f` |
| `download.rate_limit` | `--rate-limit` / `-r` |
| `network.proxy` | `--proxy` / `-p` |
| `upload.telegram.bot_token` | *(via --upload telegram)* |
| `upload.discord.webhook_url` | *(via --upload discord)* |
| `upload.custom.command` | *(via --upload custom)* |

## CLI management

```bash
# View current config
neodlp config

# Get specific value
neodlp config get download.quality

# Set a value
neodlp config set download.quality "1080p"

# Open in $EDITOR (or notepad/nano)
neodlp config edit
```
