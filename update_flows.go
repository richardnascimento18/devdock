package main

import (
	tea "github.com/charmbracelet/bubbletea"
	gh "github.com/richardnascimento18/devdock/internal/github"
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
	if _, ok := msg.(gh.ReposLoadedMsg); ok {
		m.state = stateList
	}
	return m, nil
}

func (m model) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc", "?", "q":
			m.state = stateList
			return m, nil
		}
	}
	return m, nil
}
