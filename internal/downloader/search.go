package downloader

import (
	"context"
	"fmt"
	"os"

	"github.com/lrstanley/go-ytdlp"
	"neodlp/internal/config"
)

type SearchResult struct {
	ID       string
	Title    string
	URL      string
	Uploader string
	Duration int
	Views    float64
	Platform string
}

func Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	cfg, err := config.Load()
	var proxy string
	if err == nil && cfg != nil {
		proxy = cfg.Network.Proxy
	}

	if proxy != "" {
		os.Setenv("HTTP_PROXY", proxy)
		os.Setenv("HTTPS_PROXY", proxy)
	}

	ytdlp.Install(ctx, nil)

	searchQuery := fmt.Sprintf("ytsearch%d:%s", limit, query)

	cmd := ytdlp.New().
		DumpJSON().
		FlatPlaylist().
		Quiet().
		NoWarnings()

	if proxy != "" {
		cmd = cmd.Proxy(proxy)
	}

	result, err := cmd.Run(ctx, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	entries, err := result.GetExtractedInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	var results []SearchResult
	for _, e := range entries {
		platform := "youtube"
		if e.Extractor != nil && *e.Extractor != "" {
			platform = *e.Extractor
		}

		title := ""
		if e.Title != nil {
			title = *e.Title
		}

		url := ""
		if e.WebpageURL != nil {
			url = *e.WebpageURL
		} else if e.URL != nil {
			url = *e.URL
		} else {
			url = fmt.Sprintf("https://youtube.com/watch?v=%s", e.ID)
		}

		duration := 0
		if e.Duration != nil {
			duration = int(*e.Duration)
		}

		var views float64
		if e.ViewCount != nil {
			views = *e.ViewCount
		}

		uploader := ""
		if e.Uploader != nil {
			uploader = *e.Uploader
		} else if e.Channel != nil {
			uploader = *e.Channel
		}

		results = append(results, SearchResult{
			ID:       e.ID,
			Title:    title,
			URL:      url,
			Uploader: uploader,
			Duration: duration,
			Views:    views,
			Platform: platform,
		})
	}

	return results, nil
}
