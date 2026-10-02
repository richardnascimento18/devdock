/* Package config stores all config related functions and methods. */
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"

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
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "devdock")
}

func Path() string {
	if Dir() == "" {
		return ""
	}
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
	}
	if err := Validate(cfg); err != nil {
		return Config{}, fmt.Errorf("invalid configuration %q: %w; correct the roots/settings in this file and restart (file unchanged)", Path(), err)
	}
	return cfg, nil
}

func Save(cfg Config) error {
	if Dir() == "" {
		return fmt.Errorf("cannot determine configuration directory: HOME is unset")
	}
	if err := Validate(cfg); err != nil {
		return err
	}
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(Path(), data, 0o600)
}

func (c *Config) AddRoot(path string) error {
	roots := append(slices.Clone(c.ActiveRoots()), path)
	if _, err := ValidateRoots(roots); err != nil {
		return err
	}
	c.Roots, c.Root = roots, ""
	return nil
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

func Validate(cfg Config) error {
	if _, err := ValidateRoots(cfg.ActiveRoots()); err != nil {
		return err
	}
	if (cfg.GitHubToken == "") != (cfg.GitHubUsername == "") {
		return fmt.Errorf("GitHub token and username must be configured together")
	}
	return nil
}
func (c Config) Clone() Config { c.Roots = slices.Clone(c.Roots); return c }

// Commit validates and persists a detached proposal before replacing live config.
// An unchanged proposal succeeds without writing; errors return a copy of current.
func Commit(current, proposed Config) (Config, bool, error) {
	proposed = proposed.Clone()
	if err := Validate(proposed); err != nil {
		return current.Clone(), false, err
	}
	if reflect.DeepEqual(current, proposed) {
		return current.Clone(), false, nil
	}
	if err := Save(proposed); err != nil {
		return current.Clone(), false, err
	}
	return proposed, true, nil
}
