package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	Firecrawl FirecrawlConfig `yaml:"firecrawl"`
}

// FirecrawlConfig holds Firecrawl-specific settings.
type FirecrawlConfig struct {
	APIKey string `yaml:"api_key"`
	APIURL string `yaml:"api_url"`
}

const appName = "trbooksearch"

// Load reads the config file from the XDG config directory.
// Returns a zero Config if the file doesn't exist (not an error).
func Load() (Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return Config{}, fmt.Errorf("config dir: %w", err)
	}

	path := filepath.Join(configDir, appName, "config.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil // no config file is fine
		}
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config %s: %w", path, err)
	}

	// Default API URL
	if cfg.Firecrawl.APIURL == "" {
		cfg.Firecrawl.APIURL = "https://api.firecrawl.dev"
	}

	return cfg, nil
}

// ConfigPath returns the expected config file path for display in error messages.
func ConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "~/.config/" + appName + "/config.yaml"
	}
	return filepath.Join(configDir, appName, "config.yaml")
}
