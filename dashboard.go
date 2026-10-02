package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/ui"
	"strings"
)

// The foundation frame hosts the existing snapshot list. Dashboard panes extend
// this composition in 3B without changing the operation dispatchers.
func (m model) viewList(w, h int) string {
	context := "all roots"
	if root := m.activeRoot(); root != "" {
		context = root
	}
	context += " · " + tabNames[m.activeTab] + " · preset " + m.presetSel.SelectedName()
	rootLine := ui.PaneTitle("Workspace", m.focus == ui.Workspace, 14) + " " + m.rootSel.View()
	listTitle := ui.PaneTitle("Projects", m.focus == ui.Projects, 14)
	if m.activeTab == TabTmux {
		listTitle = ui.PaneTitle("Tmux sessions", m.focus == ui.Projects, 18)
	}
	if m.list.SettingFilter() {
		listTitle += " / searching"
	} else if m.isFiltered {
		listTitle += " / " + m.lastFilter
	}
	listTitle += dimStyle.Render(fmt.Sprintf("  %d items", len(m.list.VisibleItems())))
	listContent := m.list.View()
	if len(m.list.Items()) == 0 {
		empty := "No projects here yet.\nPress n to create a project or g to connect GitHub."
		switch m.activeTab {
		case TabFavorites:
			empty = "No favorites yet.\nPress f on a project to add one."
		case TabRecents:
			empty = "No recent projects.\nOpen a project to find it here."
		case TabTmux:
			empty = "No tmux sessions.\nOpen a project to start a workspace."
		}
		if len(m.cfg.ActiveRoots()) == 0 {
			empty = "No workspace roots configured.\nPress a to add a root."
		}
		if m.isFiltered {
			empty = fmt.Sprintf("No projects match %q.\nPress esc to clear search.", m.lastFilter)
		}
		listContent = dimStyle.Render(empty)
	}
	l := ui.Measure(w, h)
	body := lipgloss.NewStyle().Width(w).Height(max(l.BodyHeight-2, 1)).Render(ui.Fit(listContent, w, max(l.BodyHeight-2, 1)))
	location := rowIdentity(m.list.SelectedItem())
	if group, ok := m.list.SelectedItem().(groupItem); ok {
		location = group.location.Path()
	}
	status := m.statusMsg
	if status == "" && location != "" {
		status = dimStyle.Render(core.PathID(location) + "  " + location)
	}
	hint := "/ search · enter open · tab focus · ? help · q quit"
	if m.focus == ui.Workspace {
		hint = "j/k roots · enter projects · tab focus · { / } root · ? help"
	}
	return strings.Join([]string{
		ui.Header(context, w), ui.Fit(rootLine, w, 1), ui.Fit(listTitle, w, 1),
		body, ui.Footer(hint, status, w),
	}, "\n")
}
