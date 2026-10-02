package main

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/richardnascimento18/devdock/internal/core"
)

// Projects are a flat action list. Hierarchy and collapsed state belong to the
// workspace pane, so collapsing navigation never hides a selected project.
func (m model) buildListItems(projects []core.Project, verified bool) []list.Item {
	var items []list.Item
	for _, p := range projects {
		if m.inWorkspaceScope(p.Location) {
			items = append(items, item{project: p, showRoot: m.isAllMode(), verified: verified})
		}
	}
	return items
}

func (m model) inWorkspaceScope(location core.Location) bool {
	if !m.isAllMode() && location.Root != m.activeRoot() {
		return false
	}
	if m.workspaceScope == nil {
		return true
	}
	scope := *m.workspaceScope
	if location.Root != scope.Root || scope.Domain != "" && location.Domain != scope.Domain || len(location.GroupPath) < len(scope.GroupPath) {
		return false
	}
	for i, name := range scope.GroupPath {
		if location.GroupPath[i] != name {
			return false
		}
	}
	return true
}
