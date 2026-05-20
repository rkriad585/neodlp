package downloader

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rkriad585/neodlp/internal/output"
	"github.com/rkriad585/neodlp/pkg/types"
)

type YTDLP struct {
	OutputManager *output.Manager
	FFmpegPath    string
}

func (y *YTDLP) Download(ctx context.Context, media *types.Media) (*types.Result, error) {
	if !y.checkInstalled() {
		return nil, fmt.Errorf("yt-dlp not found in PATH. Install from https://github.com/yt-dlp/yt-dlp")
	}

	args := []string{
		"--print", "filename",
		"-o", output.Template(media.Platform),
		"--no-warnings",
		"--no-playlist",
	}

	if media.Quality != "" {
		args = append(args, "-f", media.Quality)
	} else {
		args = append(args, "-f", "best")
	}

	if y.FFmpegPath != "" {
		args = append(args, "--ffmpeg-location", y.FFmpegPath)
	}

	args = append(args, media.URL)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	outDir := y.OutputManager.FullPath(media.Platform)
	cmd.Dir = outDir

	outputBytes, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp info: %w", err)
	}

	filename := strings.TrimSpace(string(outputBytes))
	filePath := filepath.Join(outDir, filename)

	dlArgs := []string{
		"-o", output.Template(media.Platform),
		"--no-warnings",
		"--no-playlist",
		"--print", "after_move:filepath",
	}

	if media.Quality != "" {
		dlArgs = append(dlArgs, "-f", media.Quality)
	} else {
		dlArgs = append(dlArgs, "-f", "best")
	}

	if y.FFmpegPath != "" {
		dlArgs = append(dlArgs, "--ffmpeg-location", y.FFmpegPath)
	}

	dlArgs = append(dlArgs, media.URL)

	dlCmd := exec.CommandContext(ctx, "yt-dlp", dlArgs...)
	dlCmd.Dir = outDir

	dlOutput, err := dlCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp download: %w", err)
	}

	finalPath := strings.TrimSpace(string(dlOutput))
	if finalPath == "" {
		finalPath = filePath
	}

	return &types.Result{
		URL:      media.URL,
		Title:    filename,
		FilePath: finalPath,
		Platform: media.Platform,
		Success:  true,
	}, nil
}

func (y *YTDLP) Info(ctx context.Context, url string) (*types.MediaInfo, error) {
	if !y.checkInstalled() {
		return nil, fmt.Errorf("yt-dlp not found in PATH")
	}

	args := []string{
		"--dump-json",
		"--no-warnings",
		"--no-playlist",
		url,
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp info: %w", err)
	}

	var info types.MediaInfo
	if err := parseJSON(output, &info); err != nil {
		return nil, fmt.Errorf("parse info: %w", err)
	}
	return &info, nil
}

func (y *YTDLP) checkInstalled() bool {
	cmd := exec.Command("yt-dlp", "--version")
	return cmd.Run() == nil
}

var _ Downloader = (*YTDLP)(nil)
