package downloader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lrstanley/go-ytdlp"
	"neodlp/internal/config"
)

type Result struct {
	URL      string
	Title    string
	Filename string
	Platform string
}

type Options struct {
	Quality    string
	Format     string
	OutputDir  string
	NoPlaylist bool
	AudioOnly  bool
	RateLimit  string
	Proxy      string
}

func resolveOpts(opts Options) (*config.Config, string, string, string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, "", "", "", fmt.Errorf("failed to load config: %w", err)
	}

	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = cfg.Download.OutputDir
	}

	quality := opts.Quality
	if quality == "" {
		quality = cfg.Download.Quality
	}

	outFmt := opts.Format
	if outFmt == "" {
		outFmt = cfg.Download.Format
	}

	return cfg, outputDir, quality, outFmt, nil
}

func applyOpts(dl *ytdlp.Command, opts Options, cfg *config.Config, quality, outFmt string) *ytdlp.Command {
	if opts.NoPlaylist {
		dl = dl.NoPlaylist()
	}

	if quality == "audio-only" || opts.AudioOnly {
		dl = dl.ExtractAudio().
			AudioFormat("mp3").
			AudioQuality("0")
	} else if outFmt != "auto" {
		dl = dl.FormatSort(fmt.Sprintf("res,ext:%s:m4a", outFmt)).
			RecodeVideo(outFmt)
	} else if quality != "" && quality != "best" {
		height := strings.TrimSuffix(quality, "p")
		dl = dl.Format(fmt.Sprintf("bestvideo[height<=%s]+bestaudio/best[height<=%s]", height, height))
	} else {
		dl = dl.FormatSort("res,ext:mp4:m4a")
	}

	if opts.RateLimit != "" {
		dl = dl.LimitRate(opts.RateLimit)
	}

	proxy := opts.Proxy
	if proxy == "" {
		proxy = cfg.Network.Proxy
	}
	if proxy != "" {
		dl = dl.Proxy(proxy)
	}

	if cfg.Network.CookiesFromBrowser != "" {
		dl = dl.CookiesFromBrowser(cfg.Network.CookiesFromBrowser)
	}

	if cfg.Download.ConcurrentFragments > 0 {
		dl = dl.ConcurrentFragments(cfg.Download.ConcurrentFragments)
	}

	return dl
}

func Download(ctx context.Context, urls []string, opts Options) ([]Result, error) {
	return download(ctx, urls, opts, nil)
}

func DownloadWithProgress(ctx context.Context, urls []string, opts Options, progressFn ytdlp.ProgressCallbackFunc) ([]Result, error) {
	return download(ctx, urls, opts, progressFn)
}

func download(ctx context.Context, urls []string, opts Options, progressFn ytdlp.ProgressCallbackFunc) ([]Result, error) {
	cfg, outputDir, quality, outFmt, err := resolveOpts(opts)
	if err != nil {
		return nil, err
	}

	ytdlp.Install(ctx, nil)

	var results []Result

	for _, url := range urls {
		dl := ytdlp.New().
			Paths(outputDir).
			Output("%(extractor)s - %(title)s [%(id)s].%(ext)s")

		if progressFn != nil {
			dl = dl.ProgressFunc(100*time.Millisecond, progressFn)
		} else {
			dl = dl.NoProgress()
		}

		dl = applyOpts(dl, opts, cfg, quality, outFmt)

		if _, err := dl.Run(ctx, url); err != nil {
			return nil, fmt.Errorf("failed to download %s: %w", url, err)
		}

		platform := extractPlatform(url)
		results = append(results, Result{
			URL:      url,
			Platform: platform,
		})
	}

	return results, nil
}

func Info(ctx context.Context, url string) (*ytdlp.ExtractedInfo, error) {
	ytdlp.Install(ctx, nil)

	result, err := ytdlp.New().
		DumpJSON().
		Quiet().
		NoWarnings().
		Run(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to get info: %w", err)
	}

	info, err := result.GetExtractedInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to parse info: %w", err)
	}

	if len(info) > 0 {
		return info[0], nil
	}
	return nil, fmt.Errorf("no info returned")
}

func extractPlatform(url string) string {
	url = strings.ToLower(url)
	switch {
	case strings.Contains(url, "youtube.com"), strings.Contains(url, "youtu.be"), strings.Contains(url, "m.youtube.com"), strings.Contains(url, "music.youtube.com"), strings.Contains(url, "youtube-nocookie.com"), strings.Contains(url, "yt.be"):
		return "youtube"
	case strings.Contains(url, "instagram.com"):
		return "instagram"
	case strings.Contains(url, "facebook.com"), strings.Contains(url, "fb.watch"):
		return "facebook"
	case strings.Contains(url, "twitter.com"), strings.Contains(url, "x.com"):
		return "twitter"
	case strings.Contains(url, "tiktok.com"):
		return "tiktok"
	case strings.Contains(url, "t.me"), strings.Contains(url, "telegram.org"):
		return "telegram"
	case strings.Contains(url, "threads.net"), strings.Contains(url, "threads.com"):
		return "threads"
	case strings.Contains(url, "vimeo.com"):
		return "vimeo"
	case strings.Contains(url, "reddit.com"):
		return "reddit"
	case strings.Contains(url, "pinterest.com"):
		return "pinterest"
	case strings.Contains(url, "soundcloud.com"):
		return "soundcloud"
	case strings.Contains(url, "twitch.tv"), strings.Contains(url, "twitch.com"):
		return "twitch"
	case strings.Contains(url, "dailymotion.com"):
		return "dailymotion"
	case strings.Contains(url, "bandcamp.com"):
		return "bandcamp"
	case strings.Contains(url, "mixcloud.com"):
		return "mixcloud"
	case strings.Contains(url, "rumble.com"):
		return "rumble"
	case strings.Contains(url, "odysee.com"):
		return "odysee"
	case strings.Contains(url, "bilibili.com"):
		return "bilibili"
	case strings.Contains(url, "dailymail.co.uk"):
		return "dailymail"
	case strings.Contains(url, "abc.net.au"):
		return "abc_au"
	case strings.Contains(url, "nbc.com"):
		return "nbc"
	case strings.Contains(url, "cbssports.com"), strings.Contains(url, "cbsnews.com"):
		return "cbs"
	case strings.Contains(url, "bbc.co.uk"), strings.Contains(url, "bbc.com"):
		return "bbc"
	case strings.Contains(url, "patreon.com"):
		return "patreon"
	case strings.Contains(url, "linkedin.com"):
		return "linkedin"
	default:
		return "unknown"
	}
}

func EnsureOutputDir(path string) error {
	if path == "" {
		return nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	return os.MkdirAll(abs, 0755)
}
