package extractor

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

type NativeResult struct {
	Title    string
	MediaURL string
	Ext      string
	Platform string
}

type Extractor interface {
	Match(url string) bool
	Extract(ctx context.Context, url string) (*NativeResult, error)
}

var extractors []Extractor

func Register(e Extractor) {
	extractors = append(extractors, e)
}

func Find(url string) (Extractor, bool) {
	for _, e := range extractors {
		if e.Match(url) {
			return e, true
		}
	}
	return nil, false
}

func Extract(ctx context.Context, url string) (*NativeResult, error) {
	e, ok := Find(url)
	if !ok {
		return nil, fmt.Errorf("no native extractor found for url")
	}
	return e.Extract(ctx, url)
}

func truncateTitle(t string) string {
	cleaned := strings.TrimSpace(t)
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	runes := []rune(cleaned)
	if len(runes) > 120 {
		cleaned = string(runes[:117]) + "..."
	}
	return cleaned
}

func pickBestVideo(videos []any) string {
	var bestURL string
	var bestHeight float64

	for _, v := range videos {
		vm, ok := v.(map[string]any)
		if !ok {
			continue
		}
		u, _ := vm["url"].(string)
		if u == "" {
			continue
		}
		height, _ := vm["height"].(float64)
		if height > bestHeight {
			bestHeight = height
			bestURL = u
		}
	}
	return bestURL
}

func pickBestImage(candidates []any) string {
	var bestURL string
	var bestArea float64

	for _, c := range candidates {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		u, _ := cm["url"].(string)
		if u == "" {
			continue
		}
		w, _ := cm["width"].(float64)
		h, _ := cm["height"].(float64)
		area := w * h
		if area > bestArea {
			bestArea = area
			bestURL = u
		}
	}
	return bestURL
}

func deepFind(data any, key string) []any {
	var results []any

	var walk func(any)
	walk = func(v any) {
		switch val := v.(type) {
		case map[string]any:
			for k, child := range val {
				if k == key {
					results = append(results, child)
				}
				walk(child)
			}
		case []any:
			for _, child := range val {
				walk(child)
			}
		}
	}

	walk(data)
	return results
}

func detectExt(mime string) string {
	switch mime {
	case "video/mp4", "video/mpeg4":
		return "mp4"
	case "video/webm":
		return "webm"
	case "video/x-matroska":
		return "mkv"
	case "video/quicktime":
		return "mov"
	case "video/x-msvideo":
		return "avi"
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		return ""
	}
}
