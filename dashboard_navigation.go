package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/ui"
)

func (m model) updateDashboardNavigation(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	// Terminal input can coalesce the search trigger and its first characters.
	// Treat that burst as a query, preserving the same cancellation semantics.
	if key.Type == tea.KeyRunes && len(key.Runes) > 1 && key.Runes[0] == '/' {
		next, cmd := m.beginSearch()
		m = next.(model)
		m.lastFilter = string(key.Runes[1:])
		m.searchInput.SetValue(m.lastFilter)
		m.isFiltered = true
		return m.applySearch(), cmd, true
	}
	switch key.String() {
	case "alt+1", "alt+2", "alt+3":
		switch key.String() {
		case "alt+1":
			m.focus = ui.Workspace
		case "alt+2":
			m.focus = ui.Projects
		case "alt+3":
			m.focus = ui.Inspector
		}
		m.inspectorScroll = 0
		return m, nil, true
	case "backspace", "ctrl+u", "ctrl+a":
		return m.navigateScope(key.String()), nil, true
	case "esc":
		if m.isFiltered {
			m.lastFilter = ""
			m.isFiltered = false
			return m.applySearch(), nil, true
		}
		if len(m.selected) > 0 {
			m.selected = nil
			m.statusMsg = dimStyle.Render("Selection cleared")
			return m, nil, true
		}
	case "tab", "shift+tab":
		delta := 1
		if key.String() == "shift+tab" {
			delta = -1
		}
		m.focus = m.focus.Cycle(delta, true)
		m.inspectorScroll = 0
		return m, nil, true
	case "ctrl+p":
		next, cmd := m.openPalette()
		return next, cmd, true
	case "!":
		m.statusDetails = m.statusMsg
		if m.scanWarnings != "" {
			m.statusDetails += "\n\nWorkspace scan warnings:\n" + m.scanWarnings
		}
		if m.statusDetails == "" {
			m.statusDetails = "No status message. Select a project to inspect its location."
		}
		m.state = stateStatusDetails
		return m, nil, true
	case "q":
		return m, tea.Quit, true
	case "/":
		next, cmd := m.beginSearch()
		return next, cmd, true
	}
	if key.String() == "esc" && !m.isFiltered && m.focus != ui.Projects {
		m.focus = ui.Projects
		return m, nil, true
	}
	if m.focus == ui.Workspace {
		next, handled := m.updateWorkspace(key)
		return next, nil, handled
	}
	if m.focus == ui.Inspector {
		limit := max(len(m.inspectorLines(m.dashboardLayout().Inspector))-max(m.dashboardLayout().BodyHeight-1, 1), 0)
		switch key.String() {
		case "j", "down":
			m.inspectorScroll = min(m.inspectorScroll+1, limit)
			return m, nil, true
		case "k", "up":
			m.inspectorScroll = max(m.inspectorScroll-1, 0)
			return m, nil, true
		case "pgdown":
			m.inspectorScroll = min(m.inspectorScroll+max(m.termH-7, 1), limit)
			return m, nil, true
		case "pgup":
			m.inspectorScroll = max(m.inspectorScroll-max(m.termH-7, 1), 0)
			return m, nil, true
		}
	}
	m.inspectorScroll = 0
	if key.String() == " " && m.focus == ui.Projects {
		return m.toggleSelection(), nil, true
	}
	if len(m.selected) > 0 {
		switch key.String() {
		case "m":
			return m.startBulkMove(), nil, true
		case "f":
			return m.bulkFavorites(), nil, true
		case "x", "X", "ctrl+g", "A":
			m.statusMsg = warningStyle.Render("! Clear selection with esc before deleting or removing.")
			return m, nil, true
		}
	}
	return m, nil, false
}
