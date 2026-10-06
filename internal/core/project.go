package core

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
)

// ProjectKind describes a local discovery boundary.
type ProjectKind string

const (
	KindProject ProjectKind = "project"
	KindGroup   ProjectKind = "group"
)

// Project separates the display name from the directory and workspace location.
type Project struct {
	Location
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	LocalGit   bool        `json:"local_git,omitempty"`
	GitHubRepo string      `json:"github_repo,omitempty"`
	Languages  []string    `json:"languages,omitempty"`
	Kind       ProjectKind `json:"kind,omitempty"`
}

type ProjectKey string

func (p Project) Key() ProjectKey { return ProjectKey(p.Path) }

// PathID is a compact display/session discriminator, not a permanent project ID.
func PathID(path string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(path)))
	return fmt.Sprintf("%x", sum[:8])
}
