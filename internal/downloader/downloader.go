package downloader

import (
	"context"

	"github.com/rkriad585/neodlp/pkg/types"
)

type Downloader interface {
	Download(ctx context.Context, media *types.Media) (*types.Result, error)
	Info(ctx context.Context, url string) (*types.MediaInfo, error)
}
