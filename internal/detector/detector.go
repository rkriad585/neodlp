package detector

import (
	"regexp"

	"github.com/rkriad585/neodlp/pkg/types"
)

var patterns = []struct {
	Platform types.Platform
	Regex    *regexp.Regexp
}{
	{types.PlatformYouTube, regexp.MustCompile(`(?:youtube\.com|youtu\.be)`)},
	{types.PlatformFacebook, regexp.MustCompile(`(?:facebook\.com|fb\.watch|fb\.com)`)},
	{types.PlatformInstagram, regexp.MustCompile(`(?:instagram\.com|instagr\.am)`)},
	{types.PlatformTwitter, regexp.MustCompile(`(?:twitter\.com|x\.com)`)},
}

func Detect(url string) types.Platform {
	for _, p := range patterns {
		if p.Regex.MatchString(url) {
			return p.Platform
		}
	}
	return types.PlatformUnknown
}
