package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg == nil {
		t.Fatal("Default() should not return nil")
	}
	if cfg.Download.Quality != "best" {
		t.Errorf("expected best, got: %s", cfg.Download.Quality)
	}
	if cfg.Download.Format != "auto" {
		t.Errorf("expected auto, got: %s", cfg.Download.Format)
	}
	if cfg.Download.ConcurrentFragments != 5 {
		t.Errorf("expected 5, got: %d", cfg.Download.ConcurrentFragments)
	}
	if cfg.Network.Proxy != "" {
		t.Errorf("expected empty proxy, got: %s", cfg.Network.Proxy)
	}
}

func TestDefaultOutputDir(t *testing.T) {
	cfg := Default()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "Downloads", "Neodlp")
	if cfg.Download.OutputDir != expected {
		t.Errorf("expected %s, got: %s", expected, cfg.Download.OutputDir)
	}
}

func TestSaveAndLoad(t *testing.T) {
	origPath := configPathFn
	t.Cleanup(func() { configPathFn = origPath })

	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	configPathFn = func() (string, error) { return p, nil }

	cfg := Default()
	cfg.Download.Quality = "1080p"
	cfg.Network.Proxy = "http://proxy:8080"

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loaded.Download.Quality != "1080p" {
		t.Errorf("expected 1080p, got: %s", loaded.Download.Quality)
	}
	if loaded.Network.Proxy != "http://proxy:8080" {
		t.Errorf("expected http://proxy:8080, got: %s", loaded.Network.Proxy)
	}
}

func TestLoadFillsDefaults(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")

	// Write partial config
	if err := os.WriteFile(p, []byte(`[download]
output_dir = "/custom/path"
`), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	origPath := configPathFn
	t.Cleanup(func() { configPathFn = origPath })
	configPathFn = func() (string, error) { return p, nil }

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Download.OutputDir != "/custom/path" {
		t.Errorf("expected /custom/path, got: %s", cfg.Download.OutputDir)
	}
	if cfg.Download.Quality != "best" {
		t.Errorf("expected best default, got: %s", cfg.Download.Quality)
	}
	if cfg.Download.ConcurrentFragments != 5 {
		t.Errorf("expected 5 default, got: %d", cfg.Download.ConcurrentFragments)
	}
}

func TestDir(t *testing.T) {
	origPath := configPathFn
	t.Cleanup(func() { configPathFn = origPath })

	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "config.toml")
	configPathFn = func() (string, error) { return p, nil }

	d, err := Dir()
	if err != nil {
		t.Fatalf("Dir() failed: %v", err)
	}

	expected := filepath.Join(dir, "sub")
	if d != expected {
		t.Errorf("expected %s, got: %s", expected, d)
	}
}
