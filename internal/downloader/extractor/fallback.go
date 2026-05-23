package extractor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lrstanley/go-ytdlp"
)

func TryNative(ctx context.Context, url string) (*NativeResult, error) {
	return Extract(ctx, url)
}

func DownloadNative(ctx context.Context, result *NativeResult, outputDir string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output dir: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, result.MediaURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", fmt.Sprintf("https://www.%s.com/", result.Platform))

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download media: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("media server returned HTTP %d", resp.StatusCode)
	}

	mime := resp.Header.Get("Content-Type")
	if idx := strings.Index(mime, ";"); idx != -1 {
		mime = mime[:idx]
	}
	mime = strings.TrimSpace(mime)
	ext := detectExt(mime)
	if ext == "" {
		ext = "mp4"
	}

	safeTitle := sanitizeFilename(result.Title)
	if safeTitle == "" {
		safeTitle = "download"
	}

	filename := fmt.Sprintf("%s [%s].%s", safeTitle, result.Platform, ext)
	outPath := filepath.Join(outputDir, filename)

	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outPath, nil
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)
	runes := []rune(name)
	if len(runes) > 200 {
		name = string(runes[:200])
	}
	return name
}

func IsNativeSupported(url string) bool {
	_, ok := Find(url)
	return ok
}

func isTimeoutError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "timed out") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "Client.Timeout")
}

func isExecError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "exec:") ||
		strings.Contains(msg, "executable file not found") ||
		strings.Contains(msg, "no command")
}

// installYtDlpWithRetry tries to download/install yt-dlp, retrying on
// transient network errors with backoff.
func installYtDlpWithRetry(ctx context.Context) error {
	const maxRetries = 3
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		// Use a generous per-attempt timeout (5 min) for the ~20 MB download
		installCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		_, err := ytdlp.Install(installCtx, nil)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		if isTimeoutError(err) && i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 3 * time.Second)
			continue
		}
		return err
	}
	return fmt.Errorf("yt-dlp download failed after %d attempts: %w", maxRetries, lastErr)
}

func ShouldFallback(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Unsupported URL") ||
		strings.Contains(msg, "Unable to extract") ||
		strings.Contains(msg, "Unsupported site") ||
		strings.Contains(msg, "not supported") ||
		strings.Contains(msg, "Please report this issue") ||
		isExecError(err)
}

func synthesizeInfo(r *NativeResult, url string) *ytdlp.ExtractedInfo {
	title := r.Title
	platform := r.Platform
	return &ytdlp.ExtractedInfo{
		Title:      &title,
		URL:        &url,
		Extractor:  &platform,
		WebpageURL: &url,
	}
}

func InfoWithFallback(ctx context.Context, url string) (*ytdlp.ExtractedInfo, error) {
	isNative := IsNativeSupported(url)

	info, ytdlpErr := func() (*ytdlp.ExtractedInfo, error) {
		if err := installYtDlpWithRetry(ctx); err != nil {
			if isNative {
				return nil, fmt.Errorf("yt-dlp unavailable (%w)", err)
			}
			return nil, err
		}
		result, err := ytdlp.New().
			DumpJSON().
			Quiet().
			NoWarnings().
			Run(ctx, url)
		if err != nil {
			return nil, err
		}
		infos, err := result.GetExtractedInfo()
		if err != nil {
			return nil, err
		}
		if len(infos) == 0 {
			return nil, errors.New("no info returned")
		}
		return infos[0], nil
	}()

	if ytdlpErr == nil {
		return info, nil
	}

	// yt-dlp unavailable (network/install failure) + native supported → try native
	if isNative && (isExecError(ytdlpErr) || isTimeoutError(ytdlpErr) ||
		strings.Contains(ytdlpErr.Error(), "yt-dlp unavailable")) {
		native, nativeErr := TryNative(ctx, url)
		if nativeErr == nil {
			return synthesizeInfo(native, url), nil
		}
		return nil, fmt.Errorf("yt-dlp not available and native extraction failed: %w", nativeErr)
	}

	if !ShouldFallback(ytdlpErr) {
		return nil, ytdlpErr
	}

	native, nativeErr := TryNative(ctx, url)
	if nativeErr != nil {
		return nil, fmt.Errorf("%w; native fallback also failed: %v", ytdlpErr, nativeErr)
	}

	return synthesizeInfo(native, url), nil
}

func DownloadWithFallback(ctx context.Context, url string, outputDir string, nativeOnly bool) error {
	if !nativeOnly {
		installErr := installYtDlpWithRetry(ctx)
		if installErr == nil {
			dl := ytdlp.New().
				Paths(outputDir).
				NoProgress().
				Output("%(extractor)s - %(title)s [%(id)s].%(ext)s")

			if _, err := dl.Run(ctx, url); err == nil {
				return nil
			}
		}
	}

	native, err := TryNative(ctx, url)
	if err != nil {
		return fmt.Errorf("native extraction: %w", err)
	}

	path, err := DownloadNative(ctx, native, outputDir)
	if err != nil {
		return fmt.Errorf("native download: %w", err)
	}

	fmt.Printf("  Downloaded: %s\n", path)
	return nil
}
