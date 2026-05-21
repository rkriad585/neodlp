package downloader

import (
	"testing"
)

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"https://www.instagram.com/reel/DYZ6ueYyQ3T/?utm_source=ig_web_copy_link&igsh=NTc4MTIwNjQ2YQ==",
			"https://www.instagram.com/reel/DYZ6ueYyQ3T/",
		},
		{
			"https://www.facebook.com/share/r/17ePCYyZ1m/",
			"https://www.facebook.com/share/r/17ePCYyZ1m/",
		},
		{
			"https://youtube.com/watch?v=dQw4w9WgXcQ&si=abc123&utm_source=share",
			"https://youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			"https://www.threads.com/@theshakibkhan/post/DYj6_WzgrXC?xmt=AQG0UXOhCOU2DxI-uuhuDq_htJzaq_ZLons2VKRVjAcwxg",
			"https://www.threads.com/@theshakibkhan/post/DYj6_WzgrXC",
		},
		{
			"https://example.com/video?ref=share&source=web",
			"https://example.com/video",
		},
		{
			"not-a-url",
			"not-a-url",
		},
		{
			"https://example.com?keep=this&utm_source=bad&fbclid=123",
			"https://example.com?keep=this",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
