package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Download DownloadConfig `toml:"download"`
	Network  NetworkConfig  `toml:"network"`
}

type DownloadConfig struct {
	OutputDir           string `toml:"output_dir"`
	Quality             string `toml:"quality"`
	Format              string `toml:"format"`
	ConcurrentFragments int    `toml:"concurrent_fragments"`
	RateLimit           string `toml:"rate_limit"`
}

type NetworkConfig struct {
	Proxy              string `toml:"proxy"`
	CookiesFromBrowser string `toml:"cookies_from_browser"`
}

func Default() *Config {
	home, _ := os.UserHomeDir()
	outputDir := filepath.Join(home, "Downloads", "Neodlp")

	return &Config{
		Download: DownloadConfig{
			OutputDir:           outputDir,
			Quality:             "best",
			Format:              "auto",
			ConcurrentFragments: 5,
			RateLimit:           "",
		},
		Network: NetworkConfig{
			Proxy:              "",
			CookiesFromBrowser: "",
		},
	}
}

var configPathFn = func() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot get home dir: %w", err)
	}
	return filepath.Join(home, ".config", "neostore", "neodlp", "config.toml"), nil
}

func Path() (string, error) {
	return configPathFn()
}

func Dir() (string, error) {
	cfgPath, err := Path()
	if err != nil {
		return "", err
	}
	return filepath.Dir(cfgPath), nil
}

func Load() (*Config, error) {
	cfgPath, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			def := Default()
			if saveErr := def.Save(); saveErr != nil {
				return nil, fmt.Errorf("failed to save default config: %w", saveErr)
			}
			return def, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.Download.OutputDir == "" {
		cfg.Download.OutputDir = Default().Download.OutputDir
	}
	if cfg.Download.Quality == "" {
		cfg.Download.Quality = "best"
	}
	if cfg.Download.Format == "" {
		cfg.Download.Format = "auto"
	}
	if cfg.Download.ConcurrentFragments == 0 {
		cfg.Download.ConcurrentFragments = 5
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	cfgPath, _ := Path()
	data, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
