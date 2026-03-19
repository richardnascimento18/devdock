package state

import (
	"encoding/json"
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

func Load(configDir string) UIState {
	s := UIState{
		CollapsedGroups:    make(map[string]bool),
		CollapsedSubgroups: make(map[string]bool),
		Favorites:          make(map[string]bool),
		TreeMode:           true,
	}
	data, err := os.ReadFile(statePath(configDir))
	if err != nil {
		return s
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s
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
	return s
}

func Save(configDir string, s UIState) {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = fileutil.WriteFileAtomic(statePath(configDir), data, 0o600)
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
