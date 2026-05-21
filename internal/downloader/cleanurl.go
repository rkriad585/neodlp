package downloader

import (
	"net/url"
	"strings"
)

var trackingParams = map[string]bool{
	"utm_source":   true,
	"utm_medium":   true,
	"utm_campaign": true,
	"utm_term":     true,
	"utm_content":  true,
	"fbclid":       true,
	"gclid":        true,
	"igsh":         true,
	"xmt":          true,
	"ref":          true,
	"source":       true,
	"si":           true,
	"feature":      true,
	"_ga":          true,
	"_gl":          true,
	"mc_cid":       true,
	"mc_eid":       true,
}

func SanitizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	q := u.Query()
	for key := range q {
		if trackingParams[strings.ToLower(key)] {
			q.Del(key)
		}
	}

	u.RawQuery = q.Encode()

	cleaned := u.String()
	if cleaned != rawURL {
		return cleaned
	}
	return rawURL
}
