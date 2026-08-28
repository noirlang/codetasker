package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Config holds user-specific CLI settings.
type Config struct {
	ServerURL         string  `json:"server_url"`
	Token             string  `json:"token"`
	GitHubToken       string  `json:"github_token"`
	DefaultRepo       string  `json:"default_repo"`
	DefaultBranch     string  `json:"default_branch"`
	DefaultDays       int     `json:"default_days"`
	DefaultHourlyCost float64 `json:"default_hourly_cost"`
}

// DefaultConfig returns default initial configuration.
func DefaultConfig() *Config {
	return &Config{
		ServerURL:         "http://localhost:8080",
		Token:             "",
		GitHubToken:       "",
		DefaultRepo:       "",
		DefaultBranch:     "main",
		DefaultDays:       90,
		DefaultHourlyCost: 35.0,
	}
}

// ConfigDir returns the configuration directory path based on the OS.
func ConfigDir() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(home, "AppData", "Roaming", "codetasker"), nil
		}
		return filepath.Join(appData, "codetasker"), nil
	}

	// Unix-like systems (Linux, macOS)
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome != "" {
		return filepath.Join(configHome, "codetasker"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "codetasker"), nil
}

// ConfigPath returns the full path to config.json.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads and parses the configuration file. If it doesn't exist, returns default config.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	cfgPath, err := ConfigPath()
	if err == nil {
		if data, err := os.ReadFile(cfgPath); err == nil {
			_ = json.Unmarshal(data, cfg)
		}
	}

	// Environment variable overrides
	if envServer := os.Getenv("CODETASKER_SERVER"); envServer != "" {
		cfg.ServerURL = envServer
	} else if envServerURL := os.Getenv("CODETASKER_SERVER_URL"); envServerURL != "" {
		cfg.ServerURL = envServerURL
	}

	if envToken := os.Getenv("CODETASKER_TOKEN"); envToken != "" {
		cfg.Token = envToken
	} else if envAuthToken := os.Getenv("CODETASKER_AUTH_TOKEN"); envAuthToken != "" {
		cfg.Token = envAuthToken
	}

	if envRepo := os.Getenv("CODETASKER_REPO"); envRepo != "" {
		cfg.DefaultRepo = envRepo
	}

	return cfg, nil
}

// Save writes the configuration struct to the config file.
func Save(cfg *Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	cfgPath := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(cfgPath, data, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
