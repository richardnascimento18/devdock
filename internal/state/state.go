package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/fileutil"
)

type RecentEntry struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Domain   string    `json:"domain"`
	Root     string    `json:"root"`
	OpenedAt time.Time `json:"opened_at"`
}

type UIState struct {
	CollapsedGroups    map[string]bool `json:"collapsed_groups"`
	CollapsedSubgroups map[string]bool `json:"collapsed_subgroups"`
	Favorites          map[string]bool `json:"favorites"`
	Recents            []RecentEntry   `json:"recents"`
	ActiveTab          int             `json:"active_tab"`
	TreeMode           bool            `json:"tree_mode"`
}

const (
	maxRecents   = 50
	MaxFavorites = 5
)

func statePath(configDir string) string {
	return filepath.Join(configDir, "state.json")
}

func Load(configDir string) (UIState, error) {
	s := UIState{
		CollapsedGroups:    make(map[string]bool),
		CollapsedSubgroups: make(map[string]bool),
		Favorites:          make(map[string]bool),
		TreeMode:           true,
	}
	data, err := os.ReadFile(statePath(configDir))
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	defaults := s.Clone()
	var loaded UIState
	if err := json.Unmarshal(data, &loaded); err != nil {
		return s, fmt.Errorf("invalid state.json: %w", err)
	}
	s = loaded
	if s.ActiveTab < 0 || s.ActiveTab > 3 {
		return defaults, fmt.Errorf("state.json: invalid active tab")
	}
	if s.CollapsedGroups == nil {
		s.CollapsedGroups = make(map[string]bool)
	}
	if s.CollapsedSubgroups == nil {
		s.CollapsedSubgroups = make(map[string]bool)
	}
	if s.Favorites == nil {
		s.Favorites = make(map[string]bool)
	}
	return s, nil
}

func Save(configDir string, s UIState) error {
	if !filepath.IsAbs(configDir) {
		return fmt.Errorf("configuration directory must be absolute")
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(statePath(configDir), data, 0o600)
}

func (s *UIState) AddRecent(p core.Project) {
	filtered := s.Recents[:0]
	for _, r := range s.Recents {
		if r.Path != p.Path {
			filtered = append(filtered, r)
		}
	}
	entry := RecentEntry{
		Path:     p.Path,
		Name:     p.Name,
		Domain:   p.Domain,
		Root:     p.Root,
		OpenedAt: time.Now(),
	}
	s.Recents = append([]RecentEntry{entry}, filtered...)
	if len(s.Recents) > maxRecents {
		s.Recents = s.Recents[:maxRecents]
	}
}

// ToggleFavorite adds or removes path from favorites.
// Returns false (and does nothing) if adding would exceed MaxFavorites.
func (s *UIState) ToggleFavorite(path string) bool {
	if s.Favorites[path] {
		delete(s.Favorites, path)
		return true
	}
	if len(s.Favorites) >= MaxFavorites {
		return false
	}
	s.Favorites[path] = true
	return true
}

func (s UIState) Clone() UIState {
	s.CollapsedGroups = cloneMap(s.CollapsedGroups)
	s.CollapsedSubgroups = cloneMap(s.CollapsedSubgroups)
	s.Favorites = cloneMap(s.Favorites)
	s.Recents = append([]RecentEntry(nil), s.Recents...)
	return s
}
func cloneMap(src map[string]bool) map[string]bool {
	dst := make(map[string]bool, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

// RemovePath prunes the removed directory and its descendants without touching
// favorites belonging to other or temporarily unavailable roots.
func (s *UIState) RemovePath(path string) {
	for key := range s.Favorites {
		if core.IsDescendant(path, key) {
			delete(s.Favorites, key)
		}
	}
	recents := make([]RecentEntry, 0, len(s.Recents))
	for _, r := range s.Recents {
		if !core.IsDescendant(path, r.Path) {
			recents = append(recents, r)
		}
	}
	s.Recents = recents
}
func (s *UIState) MoveProject(oldPath string, p core.Project) {
	if s.Favorites[oldPath] {
		delete(s.Favorites, oldPath)
		s.Favorites[p.Path] = true
	}
	for i := range s.Recents {
		if s.Recents[i].Path == oldPath {
			s.Recents[i].Path = p.Path
			s.Recents[i].Name = p.Name
			s.Recents[i].Domain = p.Domain
			s.Recents[i].Root = p.Root
		}
	}
}

// ReconcileMissing only removes known missing paths. Permission and I/O errors
// retain entries; root filtering must never prune another root's state.
func (s *UIState) ReconcileMissing(roots ...string) {
	missing := func(path string) bool {
		if len(roots) > 0 {
			scoped := false
			for _, root := range roots {
				if core.IsDescendant(root, path) {
					scoped = true
					break
				}
			}
			if !scoped {
				return false
			}
		}
		_, err := os.Stat(path)
		return os.IsNotExist(err)
	}
	for path, value := range s.Favorites {
		if !value || missing(path) {
			delete(s.Favorites, path)
		}
	}
	recents := make([]RecentEntry, 0, len(s.Recents))
	for _, r := range s.Recents {
		if !missing(r.Path) {
			recents = append(recents, r)
		}
	}
	s.Recents = recents
}
