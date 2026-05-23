package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type imdbExtractor struct{}

var reIMDBVideoID = regexp.MustCompile(`vi(\d+)`)

func init() {
	Register(&imdbExtractor{})
}

func (e *imdbExtractor) Match(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if !strings.Contains(u.Host, "imdb.com") {
		return false
	}
	return reIMDBVideoID.MatchString(u.Path)
}

func (e *imdbExtractor) Extract(ctx context.Context, rawURL string) (*NativeResult, error) {
	matches := reIMDBVideoID.FindStringSubmatch(rawURL)
	if len(matches) < 1 {
		return nil, fmt.Errorf("could not parse IMDB video URL: %s", rawURL)
	}
	fullID := matches[0]

	strategies := []struct {
		name string
		fn   func(ctx context.Context, videoID string) (string, string, error)
	}{
		{"page_html", e.extractFromPage},
		{"embed", e.extractFromEmbed},
		{"jsonld", e.extractJSONLD},
	}

	for _, s := range strategies {
		mediaURL, title, err := s.fn(ctx, fullID)
		if err == nil && mediaURL != "" {
			return &NativeResult{
				Title:    truncateTitle(title),
				MediaURL: mediaURL,
				Platform: "imdb",
			}, nil
		}
	}

	videoURL, title, err := e.extractDirectURL(ctx, fullID)
	if err == nil && videoURL != "" {
		return &NativeResult{
			Title:    truncateTitle(title),
			MediaURL: videoURL,
			Platform: "imdb",
		}, nil
	}

	return nil, fmt.Errorf("could not extract IMDB video from %s", rawURL)
}

func (e *imdbExtractor) fetchPage(ctx context.Context, pageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.imdb.com/")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

var reVideoInHTML = regexp.MustCompile(`(https?://[^"'\s<>]+\.(?:mp4|m3u8|webm)[^"'\s<>]*)`)
var reVideoInQuotes = regexp.MustCompile(`["'](https?://[^"']+/(?:video|media|cdn)[^"']+)["']`)

func (e *imdbExtractor) extractFromPage(ctx context.Context, videoID string) (string, string, error) {
	pageURL := fmt.Sprintf("https://www.imdb.com/video/%s/", videoID)
	body, err := e.fetchPage(ctx, pageURL)
	if err != nil {
		return "", "", err
	}

	html := string(body)

	urls := reVideoInHTML.FindAllString(html, -1)
	for _, u := range urls {
		if strings.Contains(u, "imdb") || strings.Contains(u, "media-imdb") || strings.Contains(u, "amazon") {
			return u, e.extractTitle(html), nil
		}
	}
	quoted := reVideoInQuotes.FindAllStringSubmatch(html, -1)
	for _, m := range quoted {
		if len(m) > 1 {
			u := m[1]
			if strings.Contains(u, "imdb") || strings.Contains(u, "media-imdb") || strings.Contains(u, "video") {
				if strings.HasSuffix(u, ".mp4") || strings.HasSuffix(u, ".m3u8") {
					return u, e.extractTitle(html), nil
				}
			}
		}
	}

	meta := regexp.MustCompile(`<meta[^>]+property=["']og:video["'][^>]+content=["']([^"']+)["']`)
	if m := meta.FindStringSubmatch(html); len(m) > 1 {
		return m[1], e.extractTitle(html), nil
	}

	return "", "", fmt.Errorf("no video URL in page")
}

func (e *imdbExtractor) extractFromEmbed(ctx context.Context, videoID string) (string, string, error) {
	embedURL := fmt.Sprintf("https://www.imdb.com/video/%s/imdb/embed", videoID)
	body, err := e.fetchPage(ctx, embedURL)
	if err != nil {
		return "", "", err
	}

	html := string(body)

	urls := reVideoInHTML.FindAllString(html, -1)
	for _, u := range urls {
		if strings.Contains(u, "imdb") || strings.Contains(u, "media-imdb") || strings.Contains(u, "amazon") {
			return u, e.extractTitle(html), nil
		}
	}

	quoted := reVideoInQuotes.FindAllStringSubmatch(html, -1)
	for _, m := range quoted {
		if len(m) > 1 {
			u := m[1]
			if strings.Contains(u, "imdb") || strings.Contains(u, "media-imdb") || strings.Contains(u, "video") || strings.Contains(u, ".mp4") {
				return u, e.extractTitle(html), nil
			}
		}
	}

	return "", "", fmt.Errorf("no video URL in embed page")
}

func (e *imdbExtractor) extractJSONLD(ctx context.Context, videoID string) (string, string, error) {
	pageURL := fmt.Sprintf("https://www.imdb.com/video/%s/", videoID)
	body, err := e.fetchPage(ctx, pageURL)
	if err != nil {
		return "", "", err
	}

	html := string(body)

	reJSONLD := regexp.MustCompile(`<script[^>]*type="application/ld\+json"[^>]*>(.*?)</script>`)
	matches := reJSONLD.FindAllStringSubmatch(html, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(m[1]), &data); err != nil {
			continue
		}

		if url, ok := data["contentUrl"].(string); ok && url != "" {
			title, _ := data["name"].(string)
			return url, title, nil
		}

		if video, ok := data["video"].(map[string]any); ok {
			if url, ok := video["contentUrl"].(string); ok && url != "" {
				title, _ := video["name"].(string)
				return url, title, nil
			}
		}
	}

	return "", "", fmt.Errorf("no JSON-LD video data")
}

func (e *imdbExtractor) extractDirectURL(ctx context.Context, videoID string) (string, string, error) {
	possibleURLs := []string{
		fmt.Sprintf("https://imdb-video.media-imdb.com/%s/1435-720p.mp4", videoID),
		fmt.Sprintf("https://imdb-video.media-imdb.com/%s/1435-480p.mp4", videoID),
	}

	for _, u := range possibleURLs {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Referer", "https://www.imdb.com/")

		resp, err := httpClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent {
				return u, "", nil
			}
		}
	}

	return "", "", fmt.Errorf("no direct video URL found")
}

func (e *imdbExtractor) extractTitle(html string) string {
	re := regexp.MustCompile(`<title[^>]*>(.*?)</title>`)
	if m := re.FindStringSubmatch(html); len(m) > 1 {
		title := strings.TrimSpace(m[1])
		title = strings.ReplaceAll(title, " - IMDb", "")
		return strings.TrimSpace(title)
	}

	re = regexp.MustCompile(`<meta[^>]+property=["']og:title["'][^>]+content=["']([^"']+)["']`)
	if m := re.FindStringSubmatch(html); len(m) > 1 {
		return m[1]
	}

	return ""
}
