package state

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/richardnascimento18/devdock/internal/core"
)

// Legacy compatibility exists only at the persisted input boundary.
func migrateLegacy(data []byte, s *UIState) error {
	var old struct {
		CollapsedGroups    map[string]bool `json:"collapsed_groups"`
		CollapsedSubgroups map[string]bool `json:"collapsed_subgroups"`
	}
	if err := json.Unmarshal(data, &old); err != nil {
		return err
	}
	migrate := func(entries map[string]bool, components int) error {
		for key, value := range entries {
			parts := strings.Split(key, "::")
			if len(parts) != components {
				return fmt.Errorf("ambiguous legacy collapse key %q; state file retained", key)
			}
			location := core.Location{Root: parts[0], Domain: parts[1], GroupPath: parts[2:]}
			if err := location.Validate(); err != nil {
				return fmt.Errorf("legacy collapse key: %w", err)
			}
			s.CollapsedNodes[location.Key()] = value
		}
		return nil
	}
	if err := migrate(old.CollapsedGroups, 3); err != nil {
		return err
	}
	if err := migrate(old.CollapsedSubgroups, 4); err != nil {
		return err
	}
	for i := range s.Recents {
		r := &s.Recents[i]
		location, err := core.LocationForProject(r.Root, r.Domain, r.Path)
		if err != nil {
			return fmt.Errorf("legacy recent %q: %w", r.Path, err)
		}
		r.Location = location
	}
	s.deduplicateRecents()
	return nil
}
func (s *UIState) deduplicateRecents() {
	seen := map[string]bool{}
	entries := make([]RecentEntry, 0, len(s.Recents))
	for _, entry := range s.Recents {
		if !seen[entry.Path] {
			entries = append(entries, entry)
			seen[entry.Path] = true
		}
	}
	s.Recents = entries
}
