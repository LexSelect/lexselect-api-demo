package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultAPIURL = "https://api.lexselect.io/api"
	APIVersion    = "2026-03-06"
	CLIVersion    = "0.1.0"
)

type Config struct {
	APIKey     string
	APIURL     string
	APIVersion string
	JSONOutput bool
}

// Load builds a Config from environment variables and .env file.
// Flag overrides are applied by cobra after this.
func Load() *Config {
	cfg := &Config{
		APIKey:     os.Getenv("LEXSELECT_API_KEY"),
		APIURL:     os.Getenv("LEXSELECT_API_URL"),
		APIVersion: APIVersion,
	}

	if cfg.APIURL == "" {
		cfg.APIURL = DefaultAPIURL
	}

	loadDotEnv(cfg)

	cfg.APIURL = strings.TrimRight(cfg.APIURL, "/")
	return cfg
}

func loadDotEnv(cfg *Config) {
	path := findDotEnv()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch key {
		case "LEXSELECT_API_KEY":
			if cfg.APIKey == "" {
				cfg.APIKey = val
			}
		case "LEXSELECT_API_URL":
			if cfg.APIURL == DefaultAPIURL {
				cfg.APIURL = val
			}
		}
	}
}

// findDotEnv looks for a .env file starting in the current working directory
// and walking up to parent directories. This lets the CLI be run from a
// subdirectory (e.g. cli/) while the .env lives at the repo root.
func findDotEnv() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, ".env")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
