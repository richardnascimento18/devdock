package main

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/richardnascimento18/devdock/internal/core"
)

func (m model) buildListItems(projects []core.Project, verified bool) []list.Item {
	showRoot := m.isAllMode() && len(m.cfg.ActiveRoots()) > 1
	if !m.treeMode || len(m.workspaceTree.Roots) == 0 {
		var items []list.Item
		for _, p := range projects {
			if m.isAllMode() || p.Root == m.activeRoot() {
				items = append(items, flatItem{item{project: p, showRoot: showRoot, verified: verified}})
			}
		}
		if !m.treeMode {
			for _, row := range m.workspaceTree.Flatten(nil, m.activeRoot(), false) {
				if row.Node != nil && row.Node.ProjectCount == 0 {
					items = append(items, groupItem{nodeKey: row.Node.Key(), name: row.Node.Name, location: row.Node.Location})
				}
			}
		}
		return items
	}
	// GitHub linkage is a flat-index annotation; tree membership remains canonical.
	linked := make(map[core.ProjectKey]core.Project, len(projects))
	for _, p := range projects {
		linked[p.Key()] = p
	}
	var items []list.Item
	for _, row := range m.workspaceTree.Flatten(m.collapsedNodes, m.activeRoot(), false) {
		if row.Project != nil {
			p := *row.Project
			if current, ok := linked[p.Key()]; ok {
				p = current
			}
			items = append(items, item{project: p, showRoot: showRoot, verified: verified, indent: row.Depth})
		} else {
			n := row.Node
			items = append(items, groupItem{nodeKey: n.Key(), name: n.Name, location: n.Location, depth: row.Depth, collapsed: m.collapsedNodes[n.Key()], totalCount: n.ProjectCount})
		}
	}
	return items
}
