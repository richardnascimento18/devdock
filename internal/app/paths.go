// Package app coordinates workspace use cases and preference persistence.
// It has no terminal presentation dependencies.
package app

import (
	"os"
	"path/filepath"
)

type Paths struct {
	Home, Directory, Config, State, Presets, Templates, Journal string
}

func PathsAt(directory string) Paths {
	return Paths{Directory: directory, Config: filepath.Join(directory, "config.toml"),
		State: filepath.Join(directory, "state.json"), Presets: filepath.Join(directory, "presets.json"),
		Templates: filepath.Join(directory, "templates.json"), Journal: filepath.Join(directory, "journal")}
}

func ResolvePaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	paths := PathsAt(filepath.Join(home, ".config", "devdock"))
	paths.Home = home
	return paths, nil
}
