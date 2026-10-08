package main

import (
	"strings"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/ui"
)

func (m model) switchScope(loc *core.Location) model {
	m.workspaceScope = loc
	m.rootSel.cursor = 0
	if loc != nil {
		copy := *loc
		copy.GroupPath = append([]string(nil), loc.GroupPath...)
		m.workspaceScope = &copy
		for i, root := range m.rootSel.roots {
			if root == loc.Root {
				m.rootSel.cursor = i + 1
				break
			}
		}
		m.workspaceSelection = loc.Key()
	} else {
		m.workspaceSelection = ""
	}
	m.focus = ui.Projects
	m.inspectorScroll = 0
	// Preserve the search query. The Update boundary clears multi-selection
	// when scope changes, preventing actions against an old scope.
	return m.rebuildList(m.cfg.IsGitHubConnected())
}
func (m model) navigateScope(key string) model {
	if key == "backspace" && m.workspaceScope == nil && m.activeRoot() == "" {
		return m
	}
	if key == "ctrl+a" {
		return m.switchScope(nil)
	}
	root := m.activeRoot()
	if m.workspaceScope != nil {
		root = m.workspaceScope.Root
	}
	if root == "" {
		if p, ok := projectFromItem(m.list.SelectedItem()); ok {
			root = p.Root
		}
	}
	if root == "" {
		return m.switchScope(nil)
	}
	loc := core.Location{Root: root}
	if key == "backspace" && m.workspaceScope != nil {
		loc = *m.workspaceScope
		if len(loc.GroupPath) > 0 {
			loc.GroupPath = loc.GroupPath[:len(loc.GroupPath)-1]
		} else if loc.Domain != "" {
			loc.Domain = ""
		} else {
			return m.switchScope(nil)
		}
	}
	return m.switchScope(&loc)
}
func (m model) scopeActions() []actionBinding {
	actions := []actionBinding{{"Parent workspace scope", "parent-scope", "", false}, {"Current root scope", "root-scope", "", false}, {"All workspaces scope", "all-scope", "", false}, {"Scope All workspaces", "scope:", "", false}}
	var visit func(*core.Node)
	visit = func(n *core.Node) {
		rootLabel := config.RootName(n.Root)
		for _, other := range m.workspaceTree.Roots {
			if other.Root != n.Root && config.RootName(other.Root) == rootLabel {
				rootLabel = n.Root
				break
			}
		}
		parts := []string{rootLabel}
		// Show complete readable ancestry for fuzzy queries and duplicate names.
		if n.Domain != "" {
			parts = append(parts, n.Domain)
		}
		parts = append(parts, n.GroupPath...)
		actions = append(actions, actionBinding{"Scope " + strings.Join(parts, " › "), "scope:" + string(n.Key()), "", false})
		for _, child := range n.Children {
			visit(child)
		}
	}
	for _, root := range m.workspaceTree.Roots {
		visit(root)
	}
	return actions
}
func (m model) chooseScope(key string) model {
	if key == "" {
		return m.switchScope(nil)
	}
	loc, err := core.ParseNodeKey(core.NodeKey(key))
	if err != nil {
		m.statusMsg = errorStyle.Render(ui.SafeBlock("Scope unavailable: " + err.Error()))
		return m
	}
	return m.switchScope(&loc)
}
