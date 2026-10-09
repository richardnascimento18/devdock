package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/app"
	"github.com/richardnascimento18/devdock/internal/ui"
)

func (m model) updateDashboardNavigation(key tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	// Terminal input can coalesce the search trigger and its first characters.
	// Treat that burst as a query, preserving the same cancellation semantics.
	if key.Type == tea.KeyRunes && len(key.Runes) > 1 && key.Runes[0] == '/' {
		next, cmd := m.beginSearch()
		m = next.(model)
		m.query.value = string(key.Runes[1:])
		m.query.input.SetValue(m.query.value)
		m.isFiltered = true
		return m.applySearch(), cmd, true
	}
	switch key.String() {
	case "alt+1", "alt+2", "alt+3":
		switch key.String() {
		case "alt+1":
			m.navigation.focus = ui.Workspace
		case "alt+2":
			m.navigation.focus = ui.Projects
		case "alt+3":
			m.navigation.focus = ui.Inspector
		}
		m.navigation.inspectorScroll = 0
		return m, nil, true
	case "backspace", "ctrl+u", "ctrl+a":
		return m.navigateScope(key.String()), nil, true
	case "esc":
		if m.isFiltered {
			m.query.value = ""
			m.isFiltered = false
			return m.applySearch(), nil, true
		}
		if len(m.selected) > 0 {
			m.selected = nil
			m.diagnostic = app.Diagnostic{Severity: app.Info, Summary: "Selection cleared"}
			return m, nil, true
		}
	case "tab", "shift+tab":
		delta := 1
		if key.String() == "shift+tab" {
			delta = -1
		}
		m.navigation.focus = m.navigation.focus.Cycle(delta, true)
		m.navigation.inspectorScroll = 0
		return m, nil, true
	case "ctrl+p":
		next, cmd := m.openPalette()
		return next, cmd, true
	case "!":
		m.statusDetails = m.diagnostic.Summary
		if m.scan.warnings != "" {
			m.statusDetails += "\n\nWorkspace scan warnings:\n" + m.scan.warnings
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
	if key.String() == "esc" && !m.isFiltered && m.navigation.focus != ui.Projects {
		m.navigation.focus = ui.Projects
		return m, nil, true
	}
	if m.navigation.focus == ui.Workspace {
		next, handled := m.updateWorkspace(key)
		return next, nil, handled
	}
	if m.navigation.focus == ui.Inspector {
		limit := max(len(m.inspectorLines(m.dashboardLayout().Inspector))-max(m.dashboardLayout().BodyHeight-1, 1), 0)
		switch key.String() {
		case "j", "down":
			m.navigation.inspectorScroll = min(m.navigation.inspectorScroll+1, limit)
			return m, nil, true
		case "k", "up":
			m.navigation.inspectorScroll = max(m.navigation.inspectorScroll-1, 0)
			return m, nil, true
		case "pgdown":
			m.navigation.inspectorScroll = min(m.navigation.inspectorScroll+max(m.termH-7, 1), limit)
			return m, nil, true
		case "pgup":
			m.navigation.inspectorScroll = max(m.navigation.inspectorScroll-max(m.termH-7, 1), 0)
			return m, nil, true
		}
	}
	m.navigation.inspectorScroll = 0
	if key.String() == " " && m.navigation.focus == ui.Projects {
		return m.toggleSelection(), nil, true
	}
	if len(m.selected) > 0 {
		switch key.String() {
		case "m":
			return m.startBulkMove(), nil, true
		case "f":
			return m.bulkFavorites(), nil, true
		case "x", "X", "ctrl+g", "A":
			m.diagnostic = app.Diagnostic{Severity: app.Warning, Summary: "! Clear selection with esc before deleting or removing."}
			return m, nil, true
		}
	}
	return m, nil, false
}
