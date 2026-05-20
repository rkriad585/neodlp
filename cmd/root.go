package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/rkriad585/neodlp/internal/banner"
	"github.com/rkriad585/neodlp/internal/config"
	"github.com/rkriad585/neodlp/internal/detector"
	"github.com/rkriad585/neodlp/internal/downloader"
	"github.com/rkriad585/neodlp/internal/output"
	"github.com/rkriad585/neodlp/internal/version"
	"github.com/rkriad585/neodlp/pkg/types"
)

var (
	cfg         *config.Config
	outManager  *output.Manager
	downloaders map[types.Platform]downloader.Downloader
	noBanner    bool
	quality     string
	outputDir   string
)

var rootCmd = &cobra.Command{
	Use:   "neodlp",
	Short: "Multi-platform media downloader",
	Long:  `neodlp downloads videos, audio, and media from YouTube, Facebook, Instagram, X, and 1000+ more sites via yt-dlp.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return err
		}

		dir := cfg.OutputDir
		if outputDir != "" {
			dir = outputDir
		}

		outManager, err = output.New(dir)
		if err != nil {
			return err
		}

		downloaders = downloader.NewProvider(outManager, cfg.FFmpegPath)

		bannerStr := banner.Build(noBanner)
		if bannerStr != "" {
			color.Cyan(bannerStr)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var dlCmd = &cobra.Command{
	Use:   "dl [url...]",
	Short: "Download media from URLs",
	Long:  `Download one or more media URLs. Supports YouTube, Facebook, Instagram, X, and 1000+ more sites.`,
	Example: `  neodlp dl https://youtube.com/watch?v=abc123
  neodlp dl https://instagram.com/p/xyz https://x.com/user/status/123
  neodlp dl -f urls.txt`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")

		var urls []string

		if filePath != "" {
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read URL file: %w", err)
			}
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					urls = append(urls, line)
				}
			}
		}

		urls = append(urls, args...)

		if len(urls) == 0 {
			return fmt.Errorf("no URLs provided. Use -f <file> or pass URLs as arguments")
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			color.Yellow("\nDownload interrupted by user")
			cancel()
		}()

		for _, url := range urls {
			if ctx.Err() != nil {
				break
			}

			platform := detector.Detect(url)
			dl := downloaders[platform]

			color.Cyan("→ Detected: %s", platform)
			color.White("  URL: %s", url)

			q := quality
			if q == "" {
				if p, ok := cfg.Platforms[string(platform)]; ok {
					q = p.Quality
				}
				if q == "" {
					q = cfg.DefaultQuality
				}
			}

			result, err := dl.Download(ctx, &types.Media{
				URL:      url,
				Platform: platform,
				Quality:  q,
			})

			if err != nil {
				color.Red("  ✗ Error: %v", err)
				continue
			}

			if result.Success {
				color.Green("  ✓ Downloaded: %s", result.FilePath)
				if result.Size > 0 {
					size := float64(result.Size) / (1024 * 1024)
					color.White("    Size: %.2f MB", size)
				}
			}
		}

		return nil
	},
}

var infoCmd = &cobra.Command{
	Use:   "info <url>",
	Short: "Show media metadata",
	Long:  `Fetch and display metadata about a media URL without downloading.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		platform := detector.Detect(url)
		dl := downloaders[platform]

		info, err := dl.Info(context.Background(), url)
		if err != nil {
			return fmt.Errorf("get info: %w", err)
		}

		color.Cyan("\nMedia Info")
		color.White("  Platform : %s", info.Platform)
		color.White("  Title    : %s", info.Title)
		if info.Author != "" {
			color.White("  Author   : %s", info.Author)
		}
		if info.Duration > 0 {
			color.White("  Duration : %v", info.Duration)
		}

		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config [set <key> <value>]",
	Short: "View or modify configuration",
	Long: `Display current configuration or set a config value.
	
Available keys: output_dir, default_quality, ffmpeg_path, concurrent_downloads`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			color.Cyan("\nConfiguration (%s)", cfg.OutputDir)
			path, _ := config.ConfigPath()
			color.White("  Config file: %s", path)
			color.White("  Output directory : %s", cfg.OutputDir)
			color.White("  Default quality  : %s", cfg.DefaultQuality)
			color.White("  FFmpeg path      : %s", cfg.FFmpegPath)
			color.White("  Concurrent dl    : %d", cfg.ConcurrentDownloads)
			for platform, p := range cfg.Platforms {
				color.White("  %-17s: %s", platform, p.Quality)
			}
			return nil
		}

		if args[0] == "set" && len(args) == 3 {
			if err := cfg.Set(args[1], args[2]); err != nil {
				return err
			}
			color.Green("Config updated: %s = %s", args[1], args[2])
			return nil
		}

		return cmd.Help()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		color.Cyan("neodlp version %s", version.FullVersion())
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&noBanner, "no-banner", false, "Suppress banner")
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "", "Output directory (overrides config)")
	rootCmd.PersistentFlags().StringVarP(&quality, "quality", "q", "", "Download quality (overrides config)")

	dlCmd.Flags().StringP("file", "f", "", "File containing URLs to download")

	rootCmd.AddCommand(dlCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
