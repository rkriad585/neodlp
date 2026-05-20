package downloader

import (
	"github.com/rkriad585/neodlp/internal/output"
	"github.com/rkriad585/neodlp/pkg/types"
)

func NewProvider(out *output.Manager, ffmpegPath string) map[types.Platform]Downloader {
	ytdlp := &YTDLP{OutputManager: out, FFmpegPath: ffmpegPath}

	return map[types.Platform]Downloader{
		types.PlatformYouTube:   &YouTube{OutputManager: out},
		types.PlatformFacebook:  ytdlp,
		types.PlatformInstagram: ytdlp,
		types.PlatformTwitter:   ytdlp,
		types.PlatformUnknown:   ytdlp,
	}
}
