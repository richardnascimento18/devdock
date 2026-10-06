package main

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/ui"
	"github.com/sahilm/fuzzy"
	"strings"
)

func (m model) applySearch() model {
	selected := rowIdentity(m.list.SelectedItem())
	base := m.allItems
	if m.activeTab != TabSearch {
		query, active := m.lastFilter, m.isFiltered
		m.lastFilter = ""
		m.isFiltered = false
		m = m.refreshTabList()
		base = m.list.Items()
		m.lastFilter, m.isFiltered = query, active
	}
	query := strings.TrimSpace(m.lastFilter)
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
		if !m.searching && len(filtered) == 0 && m.activeTab == TabSearch && isSafePathName(query) {
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
	m.searchRestore = m.lastFilter
	m.searchInput.SetValue(m.lastFilter)
	m.searching = true
	m.focus = ui.Projects
	return m, m.searchInput.Focus()
}

func (m model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "alt+1", "alt+2", "alt+3":
			m.searching = false
			m.searchInput.Blur()
			m.isFiltered = strings.TrimSpace(m.lastFilter) != ""
			next, cmd, _ := m.updateDashboardNavigation(key)
			return next, cmd
		case "ctrl+p":
			m.searching = false
			m.searchInput.Blur()
			return m.openPalette()
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.lastFilter = m.searchRestore
			m.searching = false
			m.searchInput.Blur()
			m.isFiltered = m.lastFilter != ""
			return m.applySearch(), nil
		case "enter":
			m.searching = false
			m.searchInput.Blur()
			m.isFiltered = strings.TrimSpace(m.lastFilter) != ""
			return m.applySearch(), nil
		case "up", "down":
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.lastFilter = m.searchInput.Value()
	m.isFiltered = strings.TrimSpace(m.lastFilter) != ""
	return m.applySearch(), cmd
}
