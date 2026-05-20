package banner

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/rkriad585/neodlp/internal/version"
)

var tips = []string{
	"Use -o to set custom output folder",
	"Pass multiple URLs to batch download",
	"Use -f urls.txt to download from a file",
	"Use --quality to override quality settings",
	"Use neodlp info <url> to preview media",
}

var commands = []string{
	"neodlp dl <url>",
	"neodlp dl <url1> <url2> <url3>",
	"neodlp dl -f urls.txt",
	"neodlp info <url>",
	"neodlp config",
	"neodlp version",
}

func Build(noBanner bool) string {
	if noBanner {
		return ""
	}

	tip := tips[rand.Intn(len(tips))]
	cmd := commands[rand.Intn(len(commands))]

	ver := version.Version()
	commit := version.CommitShort()

	lines := []string{
		fmt.Sprintf("╭──────────────── neodlp ───────────────────╮"),
		fmt.Sprintf("│      Author : RK Riad Khan                │"),
		fmt.Sprintf("│      Version: %-39s│", ver),
		fmt.Sprintf("│      Commit : %-39s│", commit),
		fmt.Sprintf("│      GitHub : rkriad585/neodlp            │"),
		fmt.Sprintf("│      [TIP]: %-38s│", tip),
		fmt.Sprintf("│      %-42s│", cmd),
		fmt.Sprintf("╰───────────────────────────────────────────╯"),
	}

	return strings.Join(lines, "\n")
}
