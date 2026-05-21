package banner

import (
	"strings"
	"testing"

	"neodlp/internal/version"
)

func TestStringContainsAuthor(t *testing.T) {
	s := String()
	if !strings.Contains(s, "RK Riad Khan") {
		t.Errorf("banner should contain author name")
	}
}

func TestStringContainsVersion(t *testing.T) {
	s := String()
	if !strings.Contains(s, version.Version) {
		t.Errorf("banner should contain version %s", version.Version)
	}
}

func TestStringContainsGitHub(t *testing.T) {
	s := String()
	if !strings.Contains(s, "rkriad585/neodlp") {
		t.Errorf("banner should contain GitHub URL")
	}
}

func TestStringContainsCommit(t *testing.T) {
	s := String()
	if !strings.Contains(s, version.Commit) {
		t.Errorf("banner should contain commit %s", version.Commit)
	}
}

func TestBoxDrawingChars(t *testing.T) {
	s := String()
	if !strings.HasPrefix(s, "╭") {
		t.Errorf("banner should start with top-left corner")
	}
	if !strings.Contains(s, "neodlp") {
		t.Errorf("banner should contain project name")
	}
	if !strings.HasSuffix(s, "╯") {
		t.Errorf("banner should end with bottom-right corner")
	}
}

func TestPrintDoesNotPanic(t *testing.T) {
	Print()
}
