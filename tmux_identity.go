package main

import (
	"github.com/richardnascimento18/devdock/internal/tmux"
	"path/filepath"
)

func (m model) sessionItem(name string) tmuxSessionItem {
	row := tmuxSessionItem{name: name}
	for _, p := range m.navigation.projects {
		if tmux.SessionName(p) != name {
			continue
		}
		row.display = p.Name
		row.location = compactProjectLocation(p)
		row.confirmation = p.Name
		for _, other := range m.navigation.projects {
			if other.Path != p.Path && other.Name == p.Name {
				parts := append([]string{p.Root, p.Domain}, p.GroupPath...)
				row.confirmation = filepath.Join(append(parts, filepath.Base(p.Path))...)
				break
			}
		}
		break
	}
	return row
}
