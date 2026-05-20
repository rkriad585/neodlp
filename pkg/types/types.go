package types

import "time"

type Platform string

const (
	PlatformYouTube   Platform = "youtube"
	PlatformFacebook  Platform = "facebook"
	PlatformInstagram Platform = "instagram"
	PlatformTwitter   Platform = "twitter"
	PlatformUnknown   Platform = "unknown"
)

type MediaType string

const (
	MediaTypeVideo MediaType = "video"
	MediaTypeAudio MediaType = "audio"
	MediaTypeImage MediaType = "image"
	MediaTypeUnknown MediaType = "unknown"
)

type Media struct {
	URL      string
	Platform Platform
	MediaType MediaType
	Quality  string
}

type MediaInfo struct {
	Title       string
	Author      string
	Duration    time.Duration
	Platform    Platform
	MediaType   MediaType
	Quality     string
	Format      string
	Size        int64
	Thumbnail   string
	Description string
}

type Result struct {
	URL      string
	Title    string
	FilePath string
	Platform Platform
	Size     int64
	Success  bool
	Error    error
}
