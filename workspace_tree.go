package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/ui"
	"strings"
)

type workspaceRow struct {
	node  *core.Node
	depth int
	guide string
}

// Walk only loaded container nodes. Project count does not affect this cache.
func (m *model) refreshWorkspaceRows() {
	rows := []workspaceRow{{}}
	scopeFound := m.workspaceScope == nil
	var visit func(*core.Node, int, []bool, bool)
	visit = func(n *core.Node, depth int, parents []bool, last bool) {
		if m.workspaceScope != nil && n.Key() == m.workspaceScope.Key() {
			scopeFound = true
		}
		guide := ""
		if depth > 0 {
			for _, following := range parents {
				if following {
					guide += "│ "
				} else {
					guide += "  "
				}
			}
			if depth > 4 {
				guide += "… "
			}
			if last {
				guide += "└─"
			} else {
				guide += "├─"
			}
		}
		rows = append(rows, workspaceRow{node: n, depth: depth, guide: guide})
		if m.collapsedNodes[n.Key()] {
			return
		}
		nextParents := append([]bool(nil), parents...)
		if depth > 0 && len(nextParents) < 3 {
			nextParents = append(nextParents, !last)
		}
		for i, child := range n.Children {
			visit(child, depth+1, nextParents, i == len(n.Children)-1)
		}
	}
	for _, root := range m.workspaceTree.Roots {
		visit(root, 0, nil, true)
	}
	// A hidden descendant may still be the active scope. Validate against the
	// loaded tree, independent of expansion, before clearing a removed location.
	if !scopeFound {
		var exists func(*core.Node) bool
		exists = func(n *core.Node) bool {
			if n.Key() == m.workspaceScope.Key() {
				return true
			}
			for _, child := range n.Children {
				if exists(child) {
					return true
				}
			}
			return false
		}
		for _, root := range m.workspaceTree.Roots {
			if exists(root) {
				scopeFound = true
				break
			}
		}
		if !scopeFound {
			m.workspaceScope = nil
		}
	}
	m.workspaceRows = rows
	m.workspaceCursor = 0
	for i, row := range rows {
		if row.node != nil && row.node.Key() == m.workspaceSelection {
			m.workspaceCursor = i
			break
		}
	}
	// Fall back to the nearest visible ancestor when the selected branch closes.
	if m.workspaceSelection != "" && m.workspaceCursor == 0 {
		loc, err := core.ParseNodeKey(m.workspaceSelection)
		if err == nil {
			for i, row := range rows {
				if row.node != nil && row.node.Root == loc.Root && core.IsDescendant(row.node.Path(), loc.Path()) {
					m.workspaceCursor = i
				}
			}
		}
	}
	m.rememberWorkspaceSelection()
}

func (m *model) rememberWorkspaceSelection() {
	m.workspaceSelection = ""
	if m.workspaceCursor > 0 && m.workspaceCursor < len(m.workspaceRows) {
		m.workspaceSelection = m.workspaceRows[m.workspaceCursor].node.Key()
	}
}

func (m model) viewWorkspace(width, height int) string {
	lines := []string{ui.PaneTitle("Workspace", m.focus == ui.Workspace, width)}
	rows := max(height-1, 1)
	start := max(0, m.workspaceCursor-rows/2)
	start = min(start, max(len(m.workspaceRows)-rows, 0))
	for i := start; i < min(start+rows, len(m.workspaceRows)); i++ {
		row := m.workspaceRows[i]
		label := "All workspaces"
		if row.node != nil {
			n := row.node
			guide := row.guide
			glyph := "▾"
			if len(n.Children) == 0 {
				glyph = "·"
			} else if m.collapsedNodes[n.Key()] {
				glyph = "▸"
			}
			label = guide + glyph + " " + n.Name + dimStyle.Render(fmt.Sprintf("  %d", n.ProjectCount))
			if n.Kind == core.NodeDomain {
				label = guide + glyph + " /" + n.Name + dimStyle.Render(fmt.Sprintf("  %d", n.ProjectCount))
			}
			if n.Kind == core.NodeRoot {
				label = glyph + " " + n.Name + dimStyle.Render("  "+core.PathID(n.Root)[:4])
			}
		}
		prefix := "  "
		if i == m.workspaceCursor {
			prefix = "> "
			label = activeStyle.Render(ansi.Strip(label))
		}
		lines = append(lines, ui.Fit(prefix+label, width, 1))
	}
	if len(m.workspaceRows) == 1 {
		lines = append(lines, dimStyle.Render("No roots found.\nPress a to add a root."))
	}
	return paneContent(strings.Join(lines, "\n"), width, height)
}

func (m model) updateWorkspace(key tea.KeyMsg) (model, bool) {
	if len(m.workspaceRows) == 0 {
		m.refreshWorkspaceRows()
	}
	switch key.String() {
	case "j", "down":
		m.workspaceCursor = min(m.workspaceCursor+1, len(m.workspaceRows)-1)
	case "k", "up":
		m.workspaceCursor = max(m.workspaceCursor-1, 0)
	case "home":
		m.workspaceCursor = 0
	case "end":
		m.workspaceCursor = len(m.workspaceRows) - 1
	case "enter":
		m.workspaceScope = nil
		m.rootSel.cursor = 0
		if m.workspaceCursor > 0 {
			loc := m.workspaceRows[m.workspaceCursor].node.Location
			loc.GroupPath = append([]string(nil), loc.GroupPath...)
			m.workspaceScope = &loc
			for i, root := range m.rootSel.roots {
				if root == loc.Root {
					m.rootSel.cursor = i + 1
					break
				}
			}
		}
		m.lastFilter = ""
		m.isFiltered = false
		m = m.rebuildList(m.cfg.IsGitHubConnected())
		m.focus = ui.Projects
	case " ", "right", "l", "left", "h":
		if m.workspaceCursor == 0 {
			return m, true
		}
		n := m.workspaceRows[m.workspaceCursor].node
		if key.String() == "left" || key.String() == "h" {
			m.collapsedNodes[n.Key()] = true
		} else if key.String() == "right" || key.String() == "l" {
			m.collapsedNodes[n.Key()] = false
		} else {
			m.collapsedNodes[n.Key()] = !m.collapsedNodes[n.Key()]
		}
		m.rememberWorkspaceSelection()
		m.refreshWorkspaceRows()
		m.saveState()
	case "esc":
		m.focus = ui.Projects
	default:
		return m, false
	}
	m.rememberWorkspaceSelection()
	return m, true
}
