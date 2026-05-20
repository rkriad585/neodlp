package version

import (
	_ "embed"
	"strings"
)

//go:embed .version
var versionFile string

var Commit = "dev"

func Version() string {
	return strings.TrimSpace(versionFile)
}

func FullVersion() string {
	return Version() + " (commit: " + Commit + ")"
}

func CommitShort() string {
	if len(Commit) > 7 {
		return Commit[:7]
	}
	return Commit
}
