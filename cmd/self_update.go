package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"neodlp/internal/banner"
	"neodlp/internal/config"
	"neodlp/internal/version"
)

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Self update neodlp to the latest version",
	Long:  "Fetches the latest version info from GitHub and updates the neodlp binary if a new version is available.",
	RunE:  selfUpdateRun,
}

func init() {
	rootCmd.AddCommand(selfUpdateCmd)
	selfUpdateCmd.Flags().StringP("proxy", "p", "", "proxy URL for self-update")
}

func selfUpdateRun(cmd *cobra.Command, args []string) error {
	fmt.Println(banner.String())
	fmt.Println()
	fmt.Println("Checking for neodlp updates...")
	fmt.Println()

	// Apply proxy configuration
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

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 1. Fetch remote version
	versionURL := "https://raw.githubusercontent.com/rkriad585/neodlp/main/.version"
	req, err := http.NewRequestWithContext(cmd.Context(), "GET", versionURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check remote version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to check remote version: server returned status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read version info: %w", err)
	}

	remoteVersion := strings.TrimSpace(string(body))
	if remoteVersion == "" {
		return fmt.Errorf("received empty version info from server")
	}

	fmt.Printf("  Current version: %s\n", version.Version)
	fmt.Printf("  Latest version : %s\n", remoteVersion)
	fmt.Println()

	if remoteVersion == version.Version {
		fmt.Println("  ✓ neodlp is already up to date!")
		return nil
	}

	fmt.Printf("  Updating neodlp to %s...\n", remoteVersion)

	// 2. Resolve running executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate running executable: %w", err)
	}

	// 3. Resolve target platform/architecture binary
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	fileName := fmt.Sprintf("neodlp-%s-%s%s", runtime.GOOS, runtime.GOARCH, ext)
	downloadURL := fmt.Sprintf("https://github.com/rkriad585/neodlp/releases/download/%s/%s", remoteVersion, fileName)

	fmt.Printf("  Downloading from: %s\n", downloadURL)

	// Create request with longer timeout for download
	downloadClient := &http.Client{
		Timeout: 5 * time.Minute,
	}

	dlReq, err := http.NewRequestWithContext(cmd.Context(), "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	dlResp, err := downloadClient.Do(dlReq)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download release binary: server returned %s", dlResp.Status)
	}

	// 4. Perform cross-platform binary swap/update
	tempPath := exePath + ".tmp"
	// Ensure temp file doesn't exist
	os.Remove(tempPath)

	tempFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	cleanup := true
	defer func() {
		if cleanup {
			tempFile.Close()
			os.Remove(tempPath)
		}
	}()

	// Copy content with simple progress indicator
	progressWriter := &WriteCounter{}
	teeReader := io.TeeReader(dlResp.Body, progressWriter)

	_, err = io.Copy(tempFile, teeReader)
	if err != nil {
		return fmt.Errorf("failed to write binary content: %w", err)
	}
	tempFile.Close()
	fmt.Println()

	// Rename running binary
	oldPath := exePath + ".old"
	os.Remove(oldPath)

	if runtime.GOOS == "windows" {
		// On Windows, rename current running binary to .old first
		err = os.Rename(exePath, oldPath)
		if err != nil {
			return fmt.Errorf("failed to rename running executable: %w", err)
		}
		// Rename new binary to running binary name
		err = os.Rename(tempPath, exePath)
		if err != nil {
			// Restore old binary if this failed
			os.Rename(oldPath, exePath)
			return fmt.Errorf("failed to install new executable: %w", err)
		}
		cleanup = false // Keep the newly installed file
		fmt.Printf("  ✓ Success! neodlp has been updated to %s.\n", remoteVersion)
		fmt.Println("  Note: You can safely delete the old executable (neodlp.exe.old) after closing this session.")
	} else {
		// On Unix, just rename tempPath directly over exePath
		err = os.Rename(tempPath, exePath)
		if err != nil {
			return fmt.Errorf("failed to install new executable: %w", err)
		}
		cleanup = false // Keep the newly installed file
		// Ensure correct permissions
		os.Chmod(exePath, 0755)
		fmt.Printf("  ✓ Success! neodlp has been updated to %s.\n", remoteVersion)
	}

	return nil
}

// WriteCounter counts bytes written for basic progress indicator
type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	fmt.Printf("\r  Downloaded: %.2f MB", float64(wc.Total)/1024/1024)
	return n, nil
}
