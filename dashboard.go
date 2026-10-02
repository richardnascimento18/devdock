package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/ui"
	"strings"
)

func (m model) dashboardLayout() ui.Layout {
	l := ui.Measure(m.termW, m.termH)
	if l.Mode == ui.Narrow || l.Mode == ui.Medium && m.focus == ui.Inspector || !m.treeMode && m.focus == ui.Workspace {
		l.Workspace, l.Projects, l.Inspector = l.Width, l.Width, l.Width
	} else if !m.treeMode {
		l.Projects += l.Workspace + ui.PaneGap
		l.Workspace = 0
	}
	return l
}

func paneContent(content string, width, height int) string {
	return lipgloss.NewStyle().Width(max(width, 1)).Height(max(height, 1)).Render(ui.Fit(content, width, height))
}

func (m model) viewList(w, h int) string {
	l := m.dashboardLayout()
	context := "all workspaces"
	if m.workspaceScope != nil {
		context = config.RootName(m.workspaceScope.Root) + " › " + m.workspaceScope.Domain
		if len(m.workspaceScope.GroupPath) > 0 {
			context += " › " + compactBreadcrumb(m.workspaceScope.GroupPath)
		}
		if context == "" {
			context = m.activeRoot()
		}
	}
	if m.workspaceScope == nil && m.activeRoot() != "" {
		context = m.activeRoot()
	}
	header := ui.Header(context+" · preset "+m.presetSel.SelectedName(), w)
	var tabs []string
	for i, name := range tabNames {
		if i == m.activeTab {
			tabs = append(tabs, activeStyle.Render(name))
		} else {
			tabs = append(tabs, dimStyle.Render(name))
		}
	}
	collectionLine := strings.Join(tabs, "  ·  ")
	if w < 60 {
		collectionLine = activeStyle.Render(tabNames[m.activeTab]) + dimStyle.Render("  [ / ] switch")
	}
	collectionLine += "  " + dimStyle.Render(fmt.Sprintf("%d projects", len(m.list.Items())))
	var body string
	switch {
	case l.Mode == ui.Narrow || l.Mode == ui.Medium && m.focus == ui.Inspector || !m.treeMode && m.focus == ui.Workspace:
		switch m.focus {
		case ui.Workspace:
			body = m.viewWorkspace(w, l.BodyHeight)
		case ui.Inspector:
			body = m.viewInspector(w, l.BodyHeight)
		default:
			body = m.viewProjects(w, l.BodyHeight)
		}
	case l.Mode == ui.Wide:
		panes := []string{}
		if m.treeMode {
			panes = append(panes, m.viewWorkspace(l.Workspace, l.BodyHeight), strings.Repeat(" ", ui.PaneGap))
		}
		panes = append(panes, m.viewProjects(l.Projects, l.BodyHeight), strings.Repeat(" ", ui.PaneGap), m.viewInspector(l.Inspector, l.BodyHeight))
		body = lipgloss.JoinHorizontal(lipgloss.Top, panes...)
	default:
		if m.treeMode {
			body = lipgloss.JoinHorizontal(lipgloss.Top, m.viewWorkspace(l.Workspace, l.BodyHeight), strings.Repeat(" ", ui.PaneGap), m.viewProjects(l.Projects, l.BodyHeight))
		} else {
			body = m.viewProjects(l.Projects, l.BodyHeight)
		}
	}
	status := m.statusMsg
	if m.activityLabel() != "" {
		status = m.activityView()
	}
	if len(m.selected) > 0 {
		status = m.selectionStatus()
		if m.statusMsg != "" {
			status += " · " + m.statusMsg
		}
	}
	if status == "" {
		status = dimStyle.Render(m.currentLocationLabel())
	}
	return strings.Join([]string{header, ui.Fit(collectionLine, w, 1), "", body, ui.Footer(m.dashboardHint(w), status, w)}, "\n")
}

func (m model) dashboardHint(width int) string {
	if m.searching {
		return "enter apply · esc cancel · ↑/↓ results"
	}
	if len(m.selected) > 0 {
		if width < 60 {
			return "space · m move · f fav · esc clear"
		}
		return "space toggle · m move · f favorites · esc clear · ctrl+p actions"
	}
	if width < 60 {
		return "tab " + m.focus.String() + " · ctrl+p actions · ? help"
	}
	if m.focus == ui.Workspace {
		return "j/k navigate · ←/→ fold · enter scope · tab focus · ctrl+p actions · ? help"
	}
	if m.focus == ui.Inspector {
		return "j/k scroll · enter open · f favorite · m move · esc projects · ctrl+p actions"
	}
	return "/ search · space select · enter open · f favorite · tab focus · ctrl+p actions · ? help"
}

func (m model) viewProjects(width, height int) string {
	title := ui.PaneTitle("Projects", m.focus == ui.Projects, width)
	if m.activeTab == TabTmux {
		title = ui.PaneTitle("Tmux sessions", m.focus == ui.Projects, width)
	}
	var content string
	switch {
	case m.searching:
		title = ui.Fit(m.searchInput.View(), width, 1)
	case m.isFiltered:
		title = ui.Fit(title+dimStyle.Render(" / "+m.lastFilter), width, 1)
	}
	if len(m.list.Items()) > 0 {
		content = m.list.View()
	} else {
		content = "No projects here yet.\nPress n to create or g to connect GitHub."
		switch m.activeTab {
		case TabFavorites:
			content = "No favorites yet.\nPress f on a project to add one."
		case TabRecents:
			content = "No recent projects.\nOpen a project to find it here."
		case TabTmux:
			content = "No tmux sessions.\nOpen a project to start a workspace."
		}
		if len(m.cfg.ActiveRoots()) == 0 {
			content = "No workspace roots configured.\nPress a to add a root."
		}
		if m.lastFilter != "" {
			content = fmt.Sprintf("No projects match %q.\nEsc clears search; n creates a project.", m.lastFilter)
		}
		if m.scanInFlight && m.lastFilter == "" {
			content = "Loading projects from configured roots."
		}
		content = dimStyle.Render(content)
	}
	return paneContent(title+"\n"+content, width, height)
}
