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
	"neodlp/internal/downloader/extractor"
	"neodlp/internal/uploader"
)

type Result struct {
	URL      string
	Title    string
	Filename string
	Platform string
}

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

	isAudio := quality == "audio-only" || opts.AudioOnly

	if isAudio {
		dl = dl.ExtractAudio().
			AudioFormat("mp3").
			AudioQuality("0")
	} else {
		if quality != "" && quality != "best" {
			height := strings.TrimSuffix(quality, "p")
			dl = dl.Format(fmt.Sprintf("bestvideo[height<=%s]+bestaudio/best[height<=%s]", height, height))
		}

		if outFmt != "auto" {
			dl = dl.FormatSort(fmt.Sprintf("res,ext:%s:m4a", outFmt)).
				RecodeVideo(outFmt)
		} else {
			dl = dl.FormatSort("res,ext:mp4:m4a")
		}
	}

	if opts.WriteThumbnail || isAudio {
		dl = dl.WriteThumbnail().EmbedThumbnail()
	}

	if opts.EmbedMetadata || isAudio {
		dl = dl.EmbedMetadata()
	}

	if opts.WriteSubs != "" {
		dl = dl.WriteSubs().SubLangs(opts.WriteSubs)
	}

	if opts.WriteAutoSubs {
		dl = dl.WriteAutoSubs()
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

func findDownloadedFile(outputDir string, rawURL string, capturedFilename string) string {
	if capturedFilename != "" {
		if _, err := os.Stat(capturedFilename); err == nil {
			return capturedFilename
		}
		// If capturedFilename doesn't exist, try resolving relative to outputDir
		fullPath := filepath.Join(outputDir, filepath.Base(capturedFilename))
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	// Fallback 1: Extract files matching basename without extension
	if capturedFilename != "" {
		baseWithoutExt := strings.TrimSuffix(filepath.Base(capturedFilename), filepath.Ext(capturedFilename))
		files, _ := filepath.Glob(filepath.Join(outputDir, baseWithoutExt+".*"))
		if len(files) > 0 {
			return files[0]
		}
	}

	// Fallback 2: Look up video info to get ID and search outputDir for [ID].ext or [ID] in name
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	info, err := Info(ctx, rawURL)
	if err == nil && info.ID != "" {
		files, _ := os.ReadDir(outputDir)
		for _, f := range files {
			if !f.IsDir() && strings.Contains(f.Name(), "["+info.ID+"]") {
				return filepath.Join(outputDir, f.Name())
			}
		}
		// Try without brackets
		for _, f := range files {
			if !f.IsDir() && strings.Contains(f.Name(), info.ID) {
				return filepath.Join(outputDir, f.Name())
			}
		}
	}

	return capturedFilename
}

func download(ctx context.Context, urls []string, opts Options, progressFn ytdlp.ProgressCallbackFunc) ([]Result, error) {
	cfg, outputDir, quality, outFmt, err := resolveOpts(opts)
	if err != nil {
		return nil, err
	}

	proxy := opts.Proxy
	if proxy == "" {
		proxy = cfg.Network.Proxy
	}
	if proxy != "" {
		os.Setenv("HTTP_PROXY", proxy)
		os.Setenv("HTTPS_PROXY", proxy)
	}

	if _, installErr := ytdlp.Install(ctx, nil); installErr != nil {
		fmt.Fprintf(os.Stderr, "  ! yt-dlp not available (%v); will try native fallback\n", installErr)
	}

	var results []Result

	for _, url := range urls {
		var capturedFilename string

		dl := ytdlp.New().
			Paths(outputDir).
			Output("%(extractor)s - %(title)s [%(id)s].%(ext)s")

		// Create a local wrapper callback to capture the filename
		innerProgressFn := func(prog ytdlp.ProgressUpdate) {
			if prog.Filename != "" {
				capturedFilename = prog.Filename
			}
			if progressFn != nil {
				progressFn(prog)
			}
		}

		dl = dl.ProgressFunc(100*time.Millisecond, innerProgressFn)

		dl = applyOpts(dl, opts, cfg, quality, outFmt)

		if _, err := dl.Run(ctx, url); err != nil {
			if extractor.ShouldFallback(err) {
				if fbErr := downloadWithFallback(ctx, url, opts); fbErr != nil {
					return nil, fmt.Errorf("yt-dlp: %w; native fallback: %v", err, fbErr)
				}
				results = append(results, Result{
					URL:      url,
					Platform: extractPlatform(url),
				})
				continue
			}
			return nil, fmt.Errorf("failed to download %s: %w", url, err)
		}

		// Resolve correct final post-processed file
		resolvedFile := findDownloadedFile(outputDir, url, capturedFilename)

		// Post-download cloud uploader hook
		if opts.UploadTarget != "" && resolvedFile != "" {
			if err := uploader.Upload(opts.UploadTarget, resolvedFile); err != nil {
				fmt.Fprintf(os.Stderr, "  ! Cloud upload error: %v\n", err)
			}
		}

		platform := extractPlatform(url)
		results = append(results, Result{
			URL:      url,
			Platform: platform,
			Filename: resolvedFile,
		})
	}

	return results, nil
}

func Info(ctx context.Context, url string) (*ytdlp.ExtractedInfo, error) {
	return InfoWithFallback(ctx, url)
}

func InfoWithFallback(ctx context.Context, url string) (*ytdlp.ExtractedInfo, error) {
	return extractor.InfoWithFallback(ctx, url)
}

func downloadWithFallback(ctx context.Context, url string, opts Options) error {
	outputDir := opts.OutputDir
	if outputDir == "" {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		outputDir = cfg.Download.OutputDir
	}
	return extractor.DownloadWithFallback(ctx, url, outputDir, false)
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

func ReadURLsFromFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading URL file: %w", err)
	}

	var urls []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("no URLs found in %s", path)
	}

	return urls, nil
}
