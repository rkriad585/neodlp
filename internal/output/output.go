package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rkriad585/neodlp/pkg/types"
)

type Manager struct {
	BaseDir string
}

func New(baseDir string) (*Manager, error) {
	expanded := expandHome(baseDir)

	abs, err := filepath.Abs(expanded)
	if err != nil {
		return nil, fmt.Errorf("resolve output path: %w", err)
	}

	return &Manager{BaseDir: abs}, nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func (m *Manager) FullPath(platform types.Platform) string {
	return filepath.Join(m.BaseDir, string(platform))
}

func (m *Manager) EnsureDir(platform types.Platform) (string, error) {
	dir := m.FullPath(platform)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return dir, nil
}

func Template(platform types.Platform) string {
	return filepath.Join(string(platform), "%(title)s_%(id)s.%(ext)s")
}
