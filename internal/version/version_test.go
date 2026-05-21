package version

import (
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Fatal("Version should not be empty")
	}
}

func TestCommit(t *testing.T) {
	if Commit == "" {
		t.Fatal("Commit should not be empty")
	}
}

func TestStringWithCommit(t *testing.T) {
	originalCommit := Commit
	Commit = "abc1234"
	defer func() { Commit = originalCommit }()

	s := String()
	if !strings.Contains(s, "v0.1.0") {
		t.Errorf("expected version in string, got: %s", s)
	}
	if !strings.Contains(s, "abc1234") {
		t.Errorf("expected commit in string, got: %s", s)
	}
}

func TestStringWithoutCommit(t *testing.T) {
	originalCommit := Commit
	Commit = "unknown"
	defer func() { Commit = originalCommit }()

	s := String()
	if s != "v0.1.0" {
		t.Errorf("expected just version, got: %s", s)
	}
}


