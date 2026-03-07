package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	GitHub GitHubConfig `toml:"github"`
	Orgs   []OrgConfig  `toml:"orgs"`
}

type GitHubConfig struct {
	Host string `toml:"host"`
}

type OrgConfig struct {
	Name  string   `toml:"name"`
	Repos []string `toml:"repos"`
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	return loadFromFile(path)
}

func loadFromFile(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at %s\n\nCreate it with:\n\n  [github]\n  host = \"github.com\"\n\n  [[orgs]]\n  name = \"my-org\"", path)
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.GitHub.Host == "" {
		cfg.GitHub.Host = "github.com"
	}

	return &cfg, nil
}

func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "gh-dashboard")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}
