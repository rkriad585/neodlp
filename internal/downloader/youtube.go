package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	yt "github.com/kkdai/youtube/v2"

	"github.com/rkriad585/neodlp/internal/output"
	"github.com/rkriad585/neodlp/pkg/types"
)

type YouTube struct {
	OutputManager *output.Manager
}

func (y *YouTube) Download(ctx context.Context, media *types.Media) (*types.Result, error) {
	client := yt.Client{}

	video, err := client.GetVideo(media.URL)
	if err != nil {
		return nil, fmt.Errorf("get video info: %w", err)
	}

	title := sanitizeFilename(video.Title)
	ext := "mp4"

	var format *yt.Format
	for _, f := range video.Formats {
		if f.AudioChannels > 0 && f.QualityLabel != "" {
			format = &f
			break
		}
	}
	if format == nil && len(video.Formats) > 0 {
		format = &video.Formats[0]
	}
	if format == nil {
		return nil, fmt.Errorf("no suitable format found")
	}

	outDir := y.OutputManager.FullPath(types.PlatformYouTube)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir output: %w", err)
	}

	filePath := filepath.Join(outDir, title+"."+ext)

	if _, err := os.Stat(filePath); err == nil {
		return &types.Result{
			URL:      media.URL,
			Title:    title,
			FilePath: filePath,
			Platform: types.PlatformYouTube,
			Success:  true,
		}, nil
	}

	stream, _, err := client.GetStream(video, format)
	if err != nil {
		return nil, fmt.Errorf("get stream: %w", err)
	}
	defer stream.Close()

	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, stream)
	if err != nil {
		return nil, fmt.Errorf("download stream: %w", err)
	}

	return &types.Result{
		URL:      media.URL,
		Title:    title,
		FilePath: filePath,
		Platform: types.PlatformYouTube,
		Size:     written,
		Success:  true,
	}, nil
}

func (y *YouTube) Info(ctx context.Context, url string) (*types.MediaInfo, error) {
	client := yt.Client{}
	video, err := client.GetVideo(url)
	if err != nil {
		return nil, fmt.Errorf("get video info: %w", err)
	}

	return &types.MediaInfo{
		Title:    video.Title,
		Author:   video.Author,
		Duration: video.Duration,
		Platform: types.PlatformYouTube,
	}, nil
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, "\"", "_")
	name = strings.ReplaceAll(name, "<", "_")
	name = strings.ReplaceAll(name, ">", "_")
	name = strings.ReplaceAll(name, "|", "_")
	return strings.TrimSpace(name)
}

func init() {
	http.DefaultClient = &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
}

var _ Downloader = (*YouTube)(nil)
