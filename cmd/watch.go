package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"neodlp/internal/banner"
	"neodlp/internal/config"
	"neodlp/internal/downloader"
)

var (
	watchFile         string
	watchDir          string
	watchPollInterval time.Duration
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch a text file or directory for links to download",
	Long:  `Automatically monitors a specific text file or directory. As soon as you add URLs, it downloads them in the background.`,
	RunE:  runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)
	watchCmd.Flags().StringVarP(&watchFile, "file", "f", "", "Text file to watch for new URLs")
	watchCmd.Flags().StringVarP(&watchDir, "dir", "d", "", "Directory to watch for incoming *.txt / *.url files")
	watchCmd.Flags().DurationVar(&watchPollInterval, "poll-interval", 2*time.Second, "Interval to check for updates (e.g. 1s, 2s, 5s)")
}

func runWatch(cmd *cobra.Command, args []string) error {
	fmt.Println(banner.String())
	fmt.Println("\n⏱️ Starting NeoDLP Watcher Suite...")

	if watchFile == "" && watchDir == "" {
		// Default to watch.txt in config directory
		cfgDir, err := config.Dir()
		if err != nil {
			return err
		}
		watchFile = filepath.Join(cfgDir, "watch.txt")
		fmt.Printf("   Notice: No target provided. Defaulting to watch file.\n")
	}

	if watchFile != "" {
		absWatchFile, err := filepath.Abs(watchFile)
		if err != nil {
			absWatchFile = watchFile
		}
		fmt.Printf("   📁 Watching File: %s\n", absWatchFile)
		fmt.Printf("   Interval     : %s\n", watchPollInterval)
		fmt.Printf("   Status       : Active. Append a URL to download. (Press Ctrl+C to stop)\n\n")

		// Create file if it doesn't exist
		if _, err := os.Stat(absWatchFile); os.IsNotExist(err) {
			os.MkdirAll(filepath.Dir(absWatchFile), 0755)
			os.WriteFile(absWatchFile, []byte("# NeoDLP Watch List\n# Append a media URL on a new line to download it automatically.\n\n"), 0644)
		}

		return watchFileLoop(cmd.Context(), absWatchFile)
	}

	if watchDir != "" {
		absWatchDir, err := filepath.Abs(watchDir)
		if err != nil {
			absWatchDir = watchDir
		}
		fmt.Printf("   📂 Watching Folder: %s\n", absWatchDir)
		fmt.Printf("   Interval       : %s\n", watchPollInterval)
		fmt.Printf("   Status         : Active. Drop *.txt files with URLs. (Press Ctrl+C to stop)\n\n")

		if err := os.MkdirAll(absWatchDir, 0755); err != nil {
			return fmt.Errorf("failed to prepare watched directory: %w", err)
		}

		return watchFolderLoop(cmd.Context(), absWatchDir)
	}

	return nil
}

var urlRegex = regexp.MustCompile(`https?://[^\s]+`)

// watchFileLoop monitors a text file, downloads new URLs, and comments them out
func watchFileLoop(ctx context.Context, filePath string) error {
	ticker := time.NewTicker(watchPollInterval)
	defer ticker.Stop()

	// Initial size tracker to detect changes
	var lastSize int64
	if stat, err := os.Stat(filePath); err == nil {
		lastSize = stat.Size()
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			stat, err := os.Stat(filePath)
			if err != nil {
				continue
			}

			// Only read if the file size has changed
			if stat.Size() == lastSize {
				continue
			}
			lastSize = stat.Size()

			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			lines := strings.Split(string(data), "\n")
			modified := false

			for i, line := range lines {
				trimmed := strings.TrimSpace(line)
				// Skip comments, empty lines, or already marked
				if trimmed == "" || strings.HasPrefix(trimmed, "#") {
					continue
				}

				foundURL := urlRegex.FindString(trimmed)
				if foundURL != "" {
					fmt.Printf("\n⏱️ [watcher] New URL detected: %s\n", foundURL)

					// Build standard options
					opts := downloader.Options{
						Quality:       "best",
						Format:        "auto",
						EmbedMetadata: true,
					}

					// Execute download sequentially in watcher to keep TUI clear
					_, err := downloader.Download(ctx, []string{foundURL}, opts)

					// Comment out the processed line in the file
					if err != nil {
						fmt.Printf("  ✗ Download failed: %v\n", err)
						lines[i] = fmt.Sprintf("# failed (%s): %s", time.Now().Format("2006-01-02 15:04"), line)
					} else {
						fmt.Printf("  ✓ Download completed successfully!\n")
						lines[i] = fmt.Sprintf("# processed (%s): %s", time.Now().Format("2006-01-02 15:04"), line)
					}
					modified = true
				}
			}

			if modified {
				// Write updated list back to file
				newData := strings.Join(lines, "\n")
				err = os.WriteFile(filePath, []byte(newData), 0644)
				if err == nil {
					// Update lastSize with our own modification size
					if newStat, err := os.Stat(filePath); err == nil {
						lastSize = newStat.Size()
					}
				}
			}
		}
	}
}

// watchFolderLoop polls a directory for *.txt or *.url files, processes them, and moves them to processed/
func watchFolderLoop(ctx context.Context, dirPath string) error {
	ticker := time.NewTicker(watchPollInterval)
	defer ticker.Stop()

	processedDir := filepath.Join(dirPath, "processed")

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			entries, err := os.ReadDir(dirPath)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				name := entry.Name()
				ext := strings.ToLower(filepath.Ext(name))

				if ext == ".txt" || ext == ".url" {
					filePath := filepath.Join(dirPath, name)
					fmt.Printf("\n⏱️ [watcher] Found batch file: %s\n", name)

					data, err := os.ReadFile(filePath)
					if err != nil {
						fmt.Printf("  ! Failed to read batch file: %v\n", err)
						continue
					}

					lines := strings.Split(string(data), "\n")
					var urls []string
					for _, line := range lines {
						trimmed := strings.TrimSpace(line)
						if trimmed == "" || strings.HasPrefix(trimmed, "#") {
							continue
						}
						foundURL := urlRegex.FindString(trimmed)
						if foundURL != "" {
							urls = append(urls, foundURL)
						}
					}

					if len(urls) == 0 {
						fmt.Println("  ! No valid URLs found in file. Cleaning up...")
						os.Remove(filePath)
						continue
					}

					fmt.Printf("  Processing batch of %d URLs...\n", len(urls))

					opts := downloader.Options{
						Quality:       "best",
						Format:        "auto",
						EmbedMetadata: true,
					}

					// Process urls
					_, dlErr := downloader.Download(ctx, urls, opts)

					// Move file to processed/ or delete if directory create fails
					if err := os.MkdirAll(processedDir, 0755); err == nil {
						targetPath := filepath.Join(processedDir, name)
						// Remove if exists
						os.Remove(targetPath)
						if err := os.Rename(filePath, targetPath); err != nil {
							os.Remove(filePath)
						} else {
							fmt.Printf("  ✓ Completed! Batch file archived to processed/%s\n", name)
						}
					} else {
						os.Remove(filePath)
						fmt.Println("  ✓ Completed! Batch file deleted.")
					}

					if dlErr != nil {
						fmt.Printf("  ! Notice: Some downloads in batch failed: %v\n", dlErr)
					}
				}
			}
		}
	}
}
