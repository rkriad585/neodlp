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

	"github.com/lrstanley/go-ytdlp"
	"github.com/spf13/cobra"

	"neodlp/internal/banner"
	"neodlp/internal/config"
	"neodlp/internal/downloader"
	"neodlp/internal/tui"
)

var selfUninstallFlag bool

var rootCmd = &cobra.Command{
	Use:   "neodlp",
	Short: "NeoDLP - Universal media downloader",
	Long:  `NeoDLP downloads video, audio, and media from YouTube, Instagram, Facebook, X, TikTok, and more.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if selfUninstallFlag {
			os.Exit(selfUninstall())
		}
		return nil
	},
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
	searchOpts struct {
		limit int
	}

	downloadOpts struct {
		quality        string
		format         string
		outputDir      string
		noPlaylist     bool
		audioOnly      bool
		rateLimit      string
		proxy          string
		pickFormat     bool
		fromFile       string
		writeThumbnail bool
		writeSubs      string
		writeAutoSubs  bool
	}
)

var downloadCmd = &cobra.Command{
	Use:     "download [url...]",
	Aliases: []string{"dl"},
	Short:   "Download media from URL(s)",
	Args:    cobra.ArbitraryArgs,
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

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search and download media",
	Long:  "Search YouTube and other platforms. Select from results to download.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  searchRun,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(banner.String())
		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update yt-dlp to the latest version",
	Long:  "Clears the cached yt-dlp binary and downloads the latest version from GitHub.",
	RunE:  updateRun,
}

func init() {
	cobra.EnableCommandSorting = false

	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.Flags().SortFlags = false
	downloadCmd.Flags().SortFlags = false

	downloadCmd.Flags().BoolVarP(&downloadOpts.pickFormat, "format", "f", false, "interactively select output format and resolution")
	downloadCmd.Flags().StringVarP(&downloadOpts.quality, "quality", "q", "", "video quality (best, 1080p, 720p, audio-only)")
	downloadCmd.Flags().StringVarP(&downloadOpts.format, "format-type", "", "", "output container (mp4, mkv, mp3, m4a, webm, mov, avi, flv, opus, wav)")
	downloadCmd.Flags().StringVarP(&downloadOpts.outputDir, "output-dir", "o", "", "custom output directory")
	downloadCmd.Flags().BoolVarP(&downloadOpts.noPlaylist, "no-playlist", "n", false, "download only single video, not playlist")
	downloadCmd.Flags().BoolVarP(&downloadOpts.audioOnly, "audio-only", "a", false, "extract audio only")
	downloadCmd.Flags().StringVarP(&downloadOpts.rateLimit, "rate-limit", "r", "", "download rate limit (e.g. 10M)")
	downloadCmd.Flags().StringVarP(&downloadOpts.proxy, "proxy", "p", "", "proxy URL")
	downloadCmd.Flags().StringVarP(&downloadOpts.fromFile, "from-file", "", "", "read URLs from a file (one per line)")
	downloadCmd.Flags().BoolVarP(&downloadOpts.writeThumbnail, "write-thumbnail", "", false, "write thumbnail image and embed it")
	downloadCmd.Flags().StringVarP(&downloadOpts.writeSubs, "write-subs", "", "", "write subtitles for given language(s) (e.g. en,es)")
	downloadCmd.Flags().BoolVarP(&downloadOpts.writeAutoSubs, "write-auto-subs", "", false, "write auto-generated subtitles")

	searchCmd.Flags().IntVarP(&searchOpts.limit, "limit", "l", 10, "max search results")

	updateCmd.Flags().StringP("proxy", "p", "", "proxy URL for downloading yt-dlp")

	rootCmd.PersistentFlags().BoolVar(&selfUninstallFlag, "selfuninstall", false, "Uninstall neodlp from the system")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func downloadRun(cmd *cobra.Command, args []string) error {
	outputDir := downloadOpts.outputDir

	if downloadOpts.fromFile != "" {
		fileURLs, err := downloader.ReadURLsFromFile(downloadOpts.fromFile)
		if err != nil {
			return friendlyError("failed to read URL file", err)
		}
		args = append(args, fileURLs...)
	}

	if len(args) == 0 {
		return friendlyError("no URLs provided", fmt.Errorf("provide URLs as arguments or use --from-file"))
	}

	if err := downloader.EnsureOutputDir(outputDir); err != nil {
		return friendlyError("failed to prepare output directory", err)
	}

	interactive := downloadOpts.pickFormat || downloadOpts.format != ""
	containerFmt := downloadOpts.format

	opts := downloader.Options{
		Quality:        downloadOpts.quality,
		Format:         containerFmt,
		OutputDir:      outputDir,
		NoPlaylist:     downloadOpts.noPlaylist,
		AudioOnly:      downloadOpts.audioOnly,
		RateLimit:      downloadOpts.rateLimit,
		Proxy:          downloadOpts.proxy,
		WriteThumbnail: downloadOpts.writeThumbnail,
		WriteSubs:      downloadOpts.writeSubs,
		WriteAutoSubs:  downloadOpts.writeAutoSubs,
	}

	for _, rawURL := range args {
		url := downloader.SanitizeURL(rawURL)
		if url != rawURL {
			fmt.Printf("  Cleaned URL: %s\n", url)
		}

		if interactive {
			if containerFmt == "" {
				selected, err := tui.SelectContainerFormat()
				if err != nil {
					return friendlyError("format selection cancelled", err)
				}
				containerFmt = selected
				opts.Format = containerFmt
			}

			ctx := context.Background()
			info, err := downloader.Info(ctx, url)
			if err != nil {
				return friendlyError("failed to fetch available formats", err)
			}

			quality, err := tui.SelectResolution(info)
			if err != nil {
				return friendlyError("resolution selection cancelled", err)
			}
			opts.Quality = quality
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
		msgs = append(msgs, "yt-dlp binary not found. Run 'neodlp update' to retry the download (it will retry 3 times with extended timeouts).")
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
	case strings.Contains(errStr, "deadline") || strings.Contains(errStr, "Client.Timeout"):
		msgs = append(msgs, "yt-dlp download timed out. If you are behind a proxy, set 'network.proxy' in the config file.")
	case strings.Contains(errStr, "threads"):
		msgs = append(msgs, "Threads support may require a newer yt-dlp version. Run 'neodlp download' to auto-update yt-dlp.")
	case strings.Contains(errStr, "Unsupported URL"):
		msgs = append(msgs, "This URL is not supported by yt-dlp. Check that the platform is supported and the URL is correct.")
	case strings.Contains(errStr, "Unsupported site"):
		msgs = append(msgs, "This site is not supported by yt-dlp.")
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
	ctx := context.Background()
	url := downloader.SanitizeURL(args[0])

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

func searchRun(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")
	ctx := context.Background()

	fmt.Println(banner.String())
	fmt.Println()
	fmt.Printf("Searching for: %s\n", query)
	fmt.Println()

	results, err := downloader.Search(ctx, query, searchOpts.limit)
	if err != nil {
		return friendlyError("search failed", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("no results found for: %s", query)
	}

	var entries []tui.SearchEntry
	for _, r := range results {
		entries = append(entries, tui.SearchEntry{
			Title:    r.Title,
			URL:      r.URL,
			Uploader: r.Uploader,
			Duration: r.Duration,
			Views:    r.Views,
			Platform: r.Platform,
		})
	}

	rawURL, err := tui.SelectSearchResult(query, entries)
	if err != nil {
		return friendlyError("search selection failed", err)
	}

	selectedURL := downloader.SanitizeURL(rawURL)
	if selectedURL != rawURL {
		fmt.Printf("  Cleaned URL: %s\n", selectedURL)
	}
	fmt.Printf("\nDownloading: %s\n\n", selectedURL)

	outputDir := downloadOpts.outputDir
	if err := downloader.EnsureOutputDir(outputDir); err != nil {
		return friendlyError("failed to prepare output directory", err)
	}

	opts := downloader.Options{
		OutputDir:  outputDir,
		RateLimit:  downloadOpts.rateLimit,
		Proxy:      downloadOpts.proxy,
	}

	if err := tui.Start(selectedURL, opts); err != nil {
		return friendlyError("download failed", err)
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

func updateRun(cmd *cobra.Command, args []string) error {
	fmt.Println(banner.String())
	fmt.Println()
	fmt.Println("Updating yt-dlp to the latest version...")
	fmt.Println()

	// Apply proxy from config or --update-proxy flag
	var proxyURL string
	if p, _ := cmd.Flags().GetString("proxy"); p != "" {
		proxyURL = p
	} else {
		cfg, err := config.Load()
		if err == nil {
			proxyURL = cfg.Network.Proxy
		}
	}
	if proxyURL != "" {
		fmt.Printf("  Using proxy: %s\n", proxyURL)
		os.Setenv("HTTP_PROXY", proxyURL)
		os.Setenv("HTTPS_PROXY", proxyURL)
	}

	if err := ytdlp.RemoveInstallCache(); err != nil {
		return friendlyError("failed to clear yt-dlp cache", err)
	}

	const maxRetries = 3
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		fmt.Printf("  Attempt %d/%d", i+1, maxRetries)
		if maxRetries > 1 {
			fmt.Print("...")
		}
		fmt.Println()

		installCtx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
		resolved, err := ytdlp.Install(installCtx, nil)
		cancel()
		if err == nil {
			fmt.Printf("  ✓ yt-dlp updated to %s\n", resolved.Version)
			fmt.Printf("  Location: %s\n", resolved.Executable)
			return nil
		}
		lastErr = err
		if isRetryableNetError(err) && i < maxRetries-1 {
			fmt.Printf("  ✗ download failed (timeout), retrying in %ds...\n", (i+1)*3)
			time.Sleep(time.Duration(i+1) * 3 * time.Second)
			continue
		}
		return friendlyError("update failed", err)
	}
	return friendlyError("update failed", lastErr)
}

func isRetryableNetError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "Client.Timeout") ||
		strings.Contains(msg, "connection") ||
		strings.Contains(msg, "reset") ||
		strings.Contains(msg, "EOF")
}
