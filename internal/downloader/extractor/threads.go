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

type threadsExtractor struct{}

var reThreadsURL = regexp.MustCompile(`threads\.(?:net|com)/@([^/]+)/post/([^/?]+)`)

func init() {
	Register(&threadsExtractor{})
}

func (e *threadsExtractor) Match(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.Contains(u.Host, "threads.net") || strings.Contains(u.Host, "threads.com")
}

func (e *threadsExtractor) Extract(ctx context.Context, rawURL string) (*NativeResult, error) {
	matches := reThreadsURL.FindStringSubmatch(rawURL)
	if len(matches) < 3 {
		return nil, fmt.Errorf("could not parse threads post URL: %s", rawURL)
	}
	username, postCode := matches[1], matches[2]

	pageURL := fmt.Sprintf("https://www.threads.net/@%s/post/%s", username, postCode)
	body, err := e.fetchPage(ctx, pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch threads page: %w", err)
	}

	data, err := e.extractJSON(body)
	if err != nil {
		return nil, fmt.Errorf("threads post data requires JavaScript rendering; try updating yt-dlp with 'neodlp update' or use a browser to get the media URL: %w", err)
	}

	mediaURL, title, err := e.parseMedia(data)
	if err != nil {
		return nil, fmt.Errorf("no media found in thread: %w", err)
	}

	return &NativeResult{
		Title:    title,
		MediaURL: mediaURL,
		Platform: "threads",
	}, nil
}

func (e *threadsExtractor) fetchPage(ctx context.Context, pageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 14; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Mobile Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.threads.net/")
	req.Header.Set("Origin", "https://www.threads.net")

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

var reScriptData = regexp.MustCompile(`<script[^>]*type="application/json"[^>]*data-sjs[^>]*>(.*?)</script>`)

func (e *threadsExtractor) extractJSON(body []byte) (any, error) {
	scripts := reScriptData.FindAllStringSubmatch(string(body), -1)
	if len(scripts) == 0 {
		return nil, fmt.Errorf("no data script tags found")
	}

	for _, s := range scripts {
		if len(s) < 2 {
			continue
		}
		raw := strings.TrimSpace(s[1])
		if !strings.Contains(raw, "ScheduledServerJS") {
			continue
		}
		if !strings.Contains(raw, "thread_items") {
			continue
		}

		var data any
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			continue
		}
		return data, nil
	}

	return nil, fmt.Errorf("no thread_items found in page data")
}

func (e *threadsExtractor) parseMedia(data any) (string, string, error) {
	items := deepFind(data, "thread_items")
	if len(items) == 0 {
		return "", "", fmt.Errorf("no thread_items found")
	}

	threadList, ok := items[0].([]any)
	if !ok || len(threadList) == 0 {
		return "", "", fmt.Errorf("no threads in thread_items")
	}

	firstThread, ok := threadList[0].(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("invalid thread item")
	}

	posts := deepFind(firstThread, "post")
	if len(posts) == 0 {
		return "", "", fmt.Errorf("no post data found")
	}
	post, ok := posts[0].(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("no post data found")
	}

	var title string
	if captionTexts := deepFind(post, "text"); len(captionTexts) > 0 {
		if t, ok := captionTexts[0].(string); ok {
			title = t
		}
	}

	mediaURL := e.extractBestMedia(post)
	if mediaURL == "" {
		return "", "", fmt.Errorf("no media found in post")
	}

	return mediaURL, truncateTitle(title), nil
}

func (e *threadsExtractor) extractBestMedia(data any) string {
	videos := deepFind(data, "video_versions")
	for _, vv := range videos {
		if list, ok := vv.([]any); ok && len(list) > 0 {
			if url := pickBestVideo(list); url != "" {
				return url
			}
		}
	}

	images := deepFind(data, "image_versions2")
	for _, iv := range images {
		if m, ok := iv.(map[string]any); ok {
			if candidates, ok := m["candidates"].([]any); ok && len(candidates) > 0 {
				if url := pickBestImage(candidates); url != "" {
					return url
				}
			}
		}
	}

	return ""
}
