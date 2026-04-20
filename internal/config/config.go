package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	appDir        = "linear_tui"
	envAPIKey     = "LINEAR_API_KEY"
	apiKeyPrefix  = "lin_api_"
	cacheFileName = "cache.db"
	stateFileName = "state.json"
)

var ErrMissingAPIKey = errors.New(envAPIKey + " is not set; export LINEAR_API_KEY=lin_api_...")

type Config struct {
	APIKey    string
	ConfigDir string
	DataDir   string
	StateDir  string
}

func Load() (*Config, error) {
	key := os.Getenv(envAPIKey)
	if key == "" {
		return nil, ErrMissingAPIKey
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}
	return &Config{
		APIKey:    key,
		ConfigDir: resolveXDG("XDG_CONFIG_HOME", filepath.Join(home, ".config")),
		DataDir:   resolveXDG("XDG_DATA_HOME", filepath.Join(home, ".local", "share")),
		StateDir:  resolveXDG("XDG_STATE_HOME", filepath.Join(home, ".local", "state")),
	}, nil
}

func (c *Config) DBPath() string {
	return filepath.Join(c.DataDir, appDir, cacheFileName)
}

func (c *Config) StatePath() string {
	return filepath.Join(c.StateDir, appDir, stateFileName)
}

func (c *Config) EnsureDirs() error {
	for _, p := range []string{
		filepath.Dir(c.DBPath()),
		filepath.Dir(c.StatePath()),
	} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func resolveXDG(envVar, fallback string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return fallback
}

func ValidAPIKeyPrefix(key string) bool {
	return len(key) > len(apiKeyPrefix) && key[:len(apiKeyPrefix)] == apiKeyPrefix
}
