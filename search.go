package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/ui"
	"github.com/sahilm/fuzzy"
)

func (m model) applySearch() model {
	selected := rowIdentity(m.list.SelectedItem())
	base := m.allItems
	if m.activeTab != TabSearch {
		query, active := m.query.value, m.isFiltered
		m.query.value = ""
		m.isFiltered = false
		m = m.refreshTabList()
		base = m.list.Items()
		m.query.value, m.isFiltered = query, active
	}
	query := strings.TrimSpace(m.query.value)
	if query == "" {
		m.list.SetItems(base)
	} else {
		values := make([]string, len(base))
		for i, it := range base {
			values[i] = it.FilterValue()
		}
		matches := fuzzy.Find(query, values)
		filtered := make([]list.Item, 0, len(matches))
		for _, match := range matches {
			filtered = append(filtered, base[match.Index])
		}
		if !m.query.editing && len(filtered) == 0 && m.activeTab == TabSearch && isSafePathName(query) {
			filtered = append(filtered, createProjectItem{name: query})
		}
		m.list.SetItems(filtered)
	}
	for i, it := range m.list.Items() {
		if selected != "" && rowIdentity(it) == selected {
			m.list.Select(i)
			break
		}
	}
	return m
}

func (m model) beginSearch() (tea.Model, tea.Cmd) {
	m.query.restore = m.query.value
	m.query.input.SetValue(m.query.value)
	m.query.editing = true
	m.navigation.focus = ui.Projects
	return m, m.query.input.Focus()
}

func (m model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "alt+1", "alt+2", "alt+3":
			m.query.editing = false
			m.query.input.Blur()
			m.isFiltered = strings.TrimSpace(m.query.value) != ""
			next, cmd, _ := m.updateDashboardNavigation(key)
			return next, cmd
		case "ctrl+p":
			m.query.editing = false
			m.query.input.Blur()
			return m.openPalette()
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.query.value = m.query.restore
			m.query.editing = false
			m.query.input.Blur()
			m.isFiltered = m.query.value != ""
			return m.applySearch(), nil
		case "enter":
			m.query.editing = false
			m.query.input.Blur()
			m.isFiltered = strings.TrimSpace(m.query.value) != ""
			return m.applySearch(), nil
		case "up", "down":
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
	}
	var cmd tea.Cmd
	m.query.input, cmd = m.query.input.Update(msg)
	m.query.value = m.query.input.Value()
	m.isFiltered = strings.TrimSpace(m.query.value) != ""
	return m.applySearch(), cmd
}
