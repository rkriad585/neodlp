package downloader

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractPlatform(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://youtube.com/watch?v=abc123", "youtube"},
		{"https://youtu.be/abc123", "youtube"},
		{"https://instagram.com/p/abc123", "instagram"},
		{"https://facebook.com/watch?v=abc123", "facebook"},
		{"https://fb.watch/abc123", "facebook"},
		{"https://twitter.com/user/status/123", "twitter"},
		{"https://x.com/user/status/123", "twitter"},
		{"https://tiktok.com/@user/video/123", "tiktok"},
		{"https://vimeo.com/123456", "vimeo"},
		{"https://reddit.com/r/videos/comments/abc", "reddit"},
		{"https://t.me/somechannel", "telegram"},
		{"https://threads.net/@user/post/abc", "threads"},
		{"https://pinterest.com/pin/abc", "pinterest"},
		{"https://soundcloud.com/user/track", "soundcloud"},
		{"https://twitch.tv/streamer", "twitch"},
		{"https://dailymotion.com/video/abc", "dailymotion"},
		{"https://bandcamp.com/track/abc", "bandcamp"},
		{"https://mixcloud.com/user/mix", "mixcloud"},
		{"https://rumble.com/video/abc", "rumble"},
		{"https://odysee.com/video/abc", "odysee"},
		{"https://bilibili.com/video/abc", "bilibili"},
		{"https://patreon.com/user", "patreon"},
		{"https://linkedin.com/video/abc", "linkedin"},
		{"https://bbc.co.uk/video/abc", "bbc"},
		{"https://cbsnews.com/video/abc", "cbs"},
		{"https://abc.net.au/video/abc", "abc_au"},
		{"https://threads.com/@user/post/abc", "threads"},
		{"https://m.youtube.com/watch?v=abc", "youtube"},
		{"https://music.youtube.com/watch?v=abc", "youtube"},
		{"https://youtube-nocookie.com/watch?v=abc", "youtube"},
		{"https://yt.be/abc", "youtube"},
		{"https://example.com/video", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := extractPlatform(tt.url)
			if result != tt.expected {
				t.Errorf("extractPlatform(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestOptionsDefaults(t *testing.T) {
	opts := Options{}
	if opts.Quality != "" {
		t.Errorf("expected empty quality")
	}
	if opts.Format != "" {
		t.Errorf("expected empty format")
	}
}

func TestEnsureOutputDir(t *testing.T) {
	dir := t.TempDir()
	testDir := filepath.Join(dir, "nested", "test", "dir")

	if err := EnsureOutputDir(testDir); err != nil {
		t.Fatalf("EnsureOutputDir() failed: %v", err)
	}

	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Error("directory was not created")
	}
}

func TestEnsureOutputDirEmpty(t *testing.T) {
	if err := EnsureOutputDir(""); err != nil {
		t.Errorf("empty path should not error, got: %v", err)
	}
}

func TestExtractPlatformCaseInsensitive(t *testing.T) {
	result := extractPlatform("HTTPS://YOUTUBE.COM/WATCH?V=ABC")
	if result != "youtube" {
		t.Errorf("expected youtube, got: %s", result)
	}
}

func TestExtractPlatformSubdomain(t *testing.T) {
	result := extractPlatform("https://www.instagram.com/p/abc123")
	if result != "instagram" {
		t.Errorf("expected instagram, got: %s", result)
	}
}

func TestExtractPlatformEdgeCases(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"", "unknown"},
		{"not-a-url", "unknown"},
		{"youtube.com", "youtube"},
		{"facebook.com/some/video", "facebook"},
		{"twitter.com", "twitter"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := extractPlatform(tt.url)
			if result != tt.expected {
				t.Errorf("extractPlatform(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestInfoRequiresYtDlp(t *testing.T) {
	ctx := context.Background()
	_, err := Info(ctx, "https://youtube.com/watch?v=invalid")
	if err == nil {
		t.Skip("skipping: yt-dlp may be installed")
	}
	if strings.Contains(err.Error(), "failed to get info") {
		// expected when yt-dlp not installed
	}
}
