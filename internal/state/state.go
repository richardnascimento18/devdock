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
	Path string `json:"path"`
	Name string `json:"name"`
	core.Location
	OpenedAt time.Time `json:"opened_at"`
}

type UIState struct {
	Version        int                   `json:"version"`
	CollapsedNodes map[core.NodeKey]bool `json:"collapsed_nodes"`
	loadErr        error
	Favorites      map[string]bool `json:"favorites"`
	Recents        []RecentEntry   `json:"recents"`
	ActiveTab      int             `json:"active_tab"`
	TreeMode       bool            `json:"tree_mode"`
}

const maxRecents = 50
const SchemaVersion = 2

func statePath(configDir string) string {
	return filepath.Join(configDir, "state.json")
}

func Load(configDir string) (UIState, error) {
	defaults := UIState{Version: SchemaVersion, CollapsedNodes: make(map[core.NodeKey]bool), Favorites: make(map[string]bool), TreeMode: true}
	fail := func(err error) (UIState, error) { defaults.loadErr = err; return defaults, err }
	data, err := os.ReadFile(statePath(configDir))
	if os.IsNotExist(err) {
		return defaults, nil
	}
	if err != nil {
		return fail(err)
	}
	var loaded UIState
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fail(fmt.Errorf("invalid state.json: %w", err))
	}
	if loaded.Version != 0 && loaded.Version != SchemaVersion {
		return fail(fmt.Errorf("unsupported state schema %d", loaded.Version))
	}
	if loaded.ActiveTab < 0 || loaded.ActiveTab > 3 {
		return fail(fmt.Errorf("state.json: invalid active tab"))
	}
	if loaded.CollapsedNodes == nil {
		loaded.CollapsedNodes = make(map[core.NodeKey]bool)
	}
	if loaded.Favorites == nil {
		loaded.Favorites = make(map[string]bool)
	}
	if loaded.Version == 0 {
		if err := migrateLegacy(data, &loaded); err != nil {
			return fail(err)
		}
	}
	for key := range loaded.CollapsedNodes {
		if _, err := core.ParseNodeKey(key); err != nil {
			return fail(err)
		}
	}
	loaded.Version = SchemaVersion
	return loaded, nil
}

func Save(configDir string, s UIState) error {
	if s.loadErr != nil {
		return fmt.Errorf("state was not loaded safely; original file retained: %w", s.loadErr)
	}
	s.Version = SchemaVersion
	for key := range s.CollapsedNodes {
		if _, err := core.ParseNodeKey(key); err != nil {
			return err
		}
	}
	if s.ActiveTab < 0 || s.ActiveTab > 3 {
		return fmt.Errorf("state.json: invalid active tab")
	}
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
		Location: core.Location{Root: p.Root, Domain: p.Domain, GroupPath: append([]string(nil), p.GroupPath...)},
		OpenedAt: time.Now(),
	}
	s.Recents = append([]RecentEntry{entry}, filtered...)
	if len(s.Recents) > maxRecents {
		s.Recents = s.Recents[:maxRecents]
	}
}

// ToggleFavorite has no product limit.
func (s *UIState) ToggleFavorite(path string) bool {
	if s.Favorites == nil {
		s.Favorites = make(map[string]bool)
	}
	if s.Favorites[path] {
		delete(s.Favorites, path)
	} else {
		s.Favorites[path] = true
	}
	return true
}

func (s UIState) Clone() UIState {
	s.CollapsedNodes = cloneMap(s.CollapsedNodes)
	s.Favorites = cloneMap(s.Favorites)
	s.Recents = append([]RecentEntry(nil), s.Recents...)
	for i := range s.Recents {
		s.Recents[i].GroupPath = append([]string(nil), s.Recents[i].GroupPath...)
	}
	return s
}
func cloneMap[K comparable](src map[K]bool) map[K]bool {
	dst := make(map[K]bool, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

// RemovePath prunes the removed directory and its descendants without touching
// favorites belonging to other or temporarily unavailable roots.
func (s *UIState) RemovePath(path string) {
	for key := range s.CollapsedNodes {
		location, err := core.ParseNodeKey(key)
		if err == nil && core.IsDescendant(path, location.Path()) {
			delete(s.CollapsedNodes, key)
		}
	}
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
			s.Recents[i].Location = p.Location
		}
	}
	s.deduplicateRecents()
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
