package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"neodlp/internal/banner"
	"neodlp/internal/config"
	"neodlp/internal/downloader"
	"neodlp/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:   "neodlp",
	Short: "NeoDLP - Universal media downloader",
	Long:  `NeoDLP downloads video, audio, and media from YouTube, Instagram, Facebook, X, TikTok, and more.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return downloadRun(cmd, args)
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

var (
	downloadOpts struct {
		quality    string
		format     string
		outputDir  string
		noPlaylist bool
		audioOnly  bool
		rateLimit  string
		proxy      string
	}
)

var downloadCmd = &cobra.Command{
	Use:     "download [url...]",
	Aliases: []string{"dl"},
	Short:   "Download media from URL(s)",
	Args:    cobra.MinimumNArgs(1),
	RunE:    downloadRun,
}

var infoCmd = &cobra.Command{
	Use:   "info <url>",
	Short: "Show media metadata without downloading",
	Args:  cobra.ExactArgs(1),
	RunE:  infoRun,
}

var configCmd = &cobra.Command{
	Use:   "config [get|set|edit]",
	Short: "Manage configuration",
	Args:  cobra.MaximumNArgs(3),
	RunE:  configRun,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(banner.String())
		return nil
	},
}

func init() {
	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.Flags().SortFlags = false
	downloadCmd.Flags().SortFlags = false

	downloadCmd.Flags().StringVarP(&downloadOpts.quality, "quality", "q", "", "video quality (best, 1080p, 720p, audio-only)")
	downloadCmd.Flags().StringVarP(&downloadOpts.format, "format", "f", "", "output format (mp4, mkv, mp3, m4a)")
	downloadCmd.Flags().StringVarP(&downloadOpts.outputDir, "output-dir", "o", "", "custom output directory")
	downloadCmd.Flags().BoolVarP(&downloadOpts.noPlaylist, "no-playlist", "n", false, "download only single video, not playlist")
	downloadCmd.Flags().BoolVarP(&downloadOpts.audioOnly, "audio-only", "a", false, "extract audio only")
	downloadCmd.Flags().StringVarP(&downloadOpts.rateLimit, "rate-limit", "r", "", "download rate limit (e.g. 10M)")
	downloadCmd.Flags().StringVarP(&downloadOpts.proxy, "proxy", "p", "", "proxy URL")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func downloadRun(cmd *cobra.Command, args []string) error {
	outputDir := downloadOpts.outputDir

	if err := downloader.EnsureOutputDir(outputDir); err != nil {
		return friendlyError("failed to prepare output directory", err)
	}

	formatSet := cmd.Flags().Changed("format")

	opts := downloader.Options{
		Quality:    downloadOpts.quality,
		Format:     downloadOpts.format,
		OutputDir:  outputDir,
		NoPlaylist: downloadOpts.noPlaylist,
		AudioOnly:  downloadOpts.audioOnly,
		RateLimit:  downloadOpts.rateLimit,
		Proxy:      downloadOpts.proxy,
	}

	for _, url := range args {
		if formatSet {
			ctx := context.Background()
			info, err := downloader.Info(ctx, url)
			if err != nil {
				return friendlyError("failed to fetch available formats", err)
			}

			selected, err := tui.SelectFormat(info)
			if err != nil {
				return friendlyError("format selection failed", err)
			}
			opts.Quality = selected
		}

		if err := tui.Start(url, opts); err != nil {
			return friendlyError("download failed", err)
		}
	}

	return nil
}

func friendlyError(msg string, err error) error {
	var msgs []string
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "yt-dlp") && strings.Contains(errStr, "not found"):
		msgs = append(msgs, "yt-dlp binary not found. Run 'neodlp download' again to auto-install it.")
	case strings.Contains(errStr, "HTTP Error 403"):
		msgs = append(msgs, "Access denied (403). The video may be private or region-locked.")
	case strings.Contains(errStr, "HTTP Error 404"):
		msgs = append(msgs, "Video not found (404). Check the URL.")
	case strings.Contains(errStr, "Unable to extract"):
		msgs = append(msgs, "Could not extract video info. The platform may have changed its API.")
	case strings.Contains(errStr, "copyright"):
		msgs = append(msgs, "This video is blocked due to copyright claims.")
	case strings.Contains(errStr, "Private video"):
		msgs = append(msgs, "This video is private.")
	case strings.Contains(errStr, "download cancelled"):
		msgs = append(msgs, "Download was cancelled by the user.")
	case errors.Is(err, context.Canceled):
		msgs = append(msgs, "Download was cancelled.")
	}

	result := fmt.Sprintf("%s: %s", msg, errStr)
	if len(msgs) > 0 {
		result += "\n\n" + strings.Join(msgs, "\n")
	}
	return errors.New(result)
}

func infoRun(cmd *cobra.Command, args []string) error {
	url := args[0]
	ctx := context.Background()

	fmt.Println(banner.String())
	fmt.Println()
	fmt.Printf("Fetching info for: %s\n", url)
	fmt.Println()

	info, err := downloader.Info(ctx, url)
	if err != nil {
		return friendlyError("failed to get info", err)
	}

	if info.ID != "" {
		fmt.Printf("  ID           : %s\n", info.ID)
	}
	if info.Title != nil {
		fmt.Printf("  Title        : %s\n", *info.Title)
	}
	if info.Uploader != nil {
		fmt.Printf("  Uploader     : %s\n", *info.Uploader)
	}
	if info.Channel != nil {
		fmt.Printf("  Channel      : %s\n", *info.Channel)
	}
	if info.Duration != nil {
		d := time.Duration(*info.Duration) * time.Second
		fmt.Printf("  Duration     : %s\n", d.Round(time.Second))
	}
	if info.ViewCount != nil {
		fmt.Printf("  Views        : %.0f\n", *info.ViewCount)
	}
	if info.LikeCount != nil {
		fmt.Printf("  Likes        : %.0f\n", *info.LikeCount)
	}
	if info.UploadDate != nil {
		fmt.Printf("  Upload date  : %s\n", *info.UploadDate)
	}
	if info.Extractor != nil {
		fmt.Printf("  Platform     : %s\n", *info.Extractor)
	}
	if info.Format != "" {
		fmt.Printf("  Format       : %s\n", info.Format)
	}
	if len(info.Formats) > 0 {
		fmt.Printf("  Formats avail: %d\n", len(info.Formats))
	}

	return nil
}

func configRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		cfgPath, _ := config.Path()
		fmt.Printf("Config file: %s\n", cfgPath)
		fmt.Printf("\n[download]\n")
		fmt.Printf("  output_dir           = %s\n", cfg.Download.OutputDir)
		fmt.Printf("  quality              = %s\n", cfg.Download.Quality)
		fmt.Printf("  format               = %s\n", cfg.Download.Format)
		fmt.Printf("  concurrent_fragments = %d\n", cfg.Download.ConcurrentFragments)
		fmt.Printf("  rate_limit           = %s\n", cfg.Download.RateLimit)
		fmt.Printf("\n[network]\n")
		fmt.Printf("  proxy                = %s\n", cfg.Network.Proxy)
		fmt.Printf("  cookies_from_browser = %s\n", cfg.Network.CookiesFromBrowser)
		return nil
	}

	switch args[0] {
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: neodlp config get <key>")
		}
		return configGet(args[1])
	case "set":
		if len(args) < 3 {
			return fmt.Errorf("usage: neodlp config set <key> <value>")
		}
		return configSet(args[1], args[2])
	case "edit":
		return configEdit()
	default:
		return fmt.Errorf("unknown subcommand: %s (use get, set, or edit)", args[0])
	}
}

func configGet(key string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	val, ok := configLookup(cfg, key)
	if !ok {
		return fmt.Errorf("unknown config key: %s", key)
	}
	fmt.Println(val)
	return nil
}

func configSet(key, value string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if !configSetValue(cfg, key, value) {
		return fmt.Errorf("unknown config key: %s", key)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ %s set to %s\n", key, value)
	return nil
}

func configEdit() error {
	cfgPath, err := config.Path()
	if err != nil {
		return err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "nano"
		}
	}

	editCmd := exec.Command(editor, cfgPath)
	editCmd.Stdin = os.Stdin
	editCmd.Stdout = os.Stdout
	editCmd.Stderr = os.Stderr

	return editCmd.Run()
}

func configLookup(cfg *config.Config, key string) (string, bool) {
	m := configFlatten(cfg)
	val, ok := m[key]
	return val, ok
}

func configSetValue(cfg *config.Config, key, value string) bool {
	switch key {
	case "download.output_dir":
		cfg.Download.OutputDir = value
	case "download.quality":
		cfg.Download.Quality = value
	case "download.format":
		cfg.Download.Format = value
	case "download.concurrent_fragments":
		fmt.Sscanf(value, "%d", &cfg.Download.ConcurrentFragments)
	case "download.rate_limit":
		cfg.Download.RateLimit = value
	case "network.proxy":
		cfg.Network.Proxy = value
	case "network.cookies_from_browser":
		cfg.Network.CookiesFromBrowser = value
	default:
		return false
	}
	return true
}

func configFlatten(cfg *config.Config) map[string]string {
	return map[string]string{
		"download.output_dir":           cfg.Download.OutputDir,
		"download.quality":              cfg.Download.Quality,
		"download.format":               cfg.Download.Format,
		"download.concurrent_fragments": fmt.Sprintf("%d", cfg.Download.ConcurrentFragments),
		"download.rate_limit":           cfg.Download.RateLimit,
		"network.proxy":                 cfg.Network.Proxy,
		"network.cookies_from_browser":  cfg.Network.CookiesFromBrowser,
	}
}
