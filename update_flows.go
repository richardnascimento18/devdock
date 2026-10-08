package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateGitHubAuth(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc", "enter", "ctrl+c":
			m.cancelAuth()
			m.state = stateList
			return m, nil
		}
	}
	if _, ok := msg.(ReposLoadedMsg); ok {
		m.state = stateList
	}
	return m, nil
}

func (m model) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "j", "down":
			m.helpScroll++
			return m, nil
		case "k", "up":
			m.helpScroll = max(m.helpScroll-1, 0)
			return m, nil
		case "pgdown":
			m.helpScroll += max(m.termH-8, 1)
			return m, nil
		case "pgup":
			m.helpScroll = max(m.helpScroll-max(m.termH-8, 1), 0)
			return m, nil
		}
	}
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc", "?", "q":
			m.state = stateList
			return m, nil
		}
	}
	return m, nil
}
