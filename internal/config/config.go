/* Package config stores all config related functions and methods. */
package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/richardnascimento18/devdock/internal/fileutil"
)

type Config struct {
	Roots          []string `toml:"roots"`
	Root           string   `toml:"root,omitempty"`
	DefaultPreset  string   `toml:"default_preset,omitempty"`
	GitHubToken    string   `toml:"github_token,omitempty"`
	GitHubUsername string   `toml:"github_username,omitempty"`
}

func (c Config) ActiveRoots() []string {
	if len(c.Roots) > 0 {
		return c.Roots
	}
	if c.Root != "" {
		return []string{c.Root}
	}
	return nil
}

func (c Config) IsGitHubConnected() bool {
	return c.GitHubToken != "" && c.GitHubUsername != ""
}

func RootName(root string) string {
	return filepath.Base(root)
}

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "devdock")
}

func Path() string {
	return filepath.Join(Dir(), "config.toml")
}

func Load() (Config, error) {
	var cfg Config
	data, err := os.ReadFile(Path())
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Root != "" && len(cfg.Roots) == 0 {
		cfg.Roots = []string{cfg.Root}
		cfg.Root = ""
		_ = Save(cfg)
	}
	return cfg, nil
}

func Save(cfg Config) error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(Path(), data, 0o600)
}

func (c *Config) AddRoot(path string) {
	for _, r := range c.Roots {
		if r == path {
			return
		}
	}
	c.Roots = append(c.Roots, path)
}

func (c *Config) RemoveRoot(path string) {
	filtered := c.Roots[:0]
	for _, r := range c.Roots {
		if r != path {
			filtered = append(filtered, r)
		}
	}
	c.Roots = filtered
}
