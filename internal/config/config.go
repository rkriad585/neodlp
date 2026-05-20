package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	OutputDir          string              `json:"output_dir"`
	DefaultQuality     string              `json:"default_quality"`
	FFmpegPath         string              `json:"ffmpeg_path"`
	ConcurrentDownloads int                `json:"concurrent_downloads"`
	Platforms          map[string]Platform `json:"platforms"`
}

type Platform struct {
	Quality string `json:"quality"`
}

func Default() *Config {
	return &Config{
		OutputDir:           "~/Downloads/Neodlp",
		DefaultQuality:      "best",
		FFmpegPath:          "",
		ConcurrentDownloads: 3,
		Platforms: map[string]Platform{
			"youtube":   {Quality: "bestvideo+bestaudio"},
			"instagram": {Quality: "best"},
			"facebook":  {Quality: "best"},
			"twitter":   {Quality: "best"},
		},
	}
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".config", "neostore", "neodlp"), nil
}

func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Default(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := Default()
			if saveErr := cfg.Save(); saveErr != nil {
				return cfg, nil
			}
			return cfg, nil
		}
		return Default(), nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), nil
	}

	if cfg.OutputDir == "" {
		cfg.OutputDir = Default().OutputDir
	}
	if cfg.DefaultQuality == "" {
		cfg.DefaultQuality = Default().DefaultQuality
	}
	if cfg.ConcurrentDownloads == 0 {
		cfg.ConcurrentDownloads = Default().ConcurrentDownloads
	}
	if cfg.Platforms == nil {
		cfg.Platforms = Default().Platforms
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir config: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func (c *Config) Set(key, value string) error {
	switch key {
	case "output_dir":
		c.OutputDir = value
	case "default_quality":
		c.DefaultQuality = value
	case "ffmpeg_path":
		c.FFmpegPath = value
	case "concurrent_downloads":
		var n int
		if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
			return fmt.Errorf("invalid number: %w", err)
		}
		c.ConcurrentDownloads = n
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return c.Save()
}
