package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/config"
)

func (m model) updatePickRootForClone(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		if k.String() == "esc" {
			m.state = stateList
			return m, nil
		}
		if navPicker(&m.genericPicker, k.String()) {
			return m, nil
		}
		if k.String() == "enter" {
			if m.genericPicker.cursor < 0 || m.genericPicker.cursor >= len(m.cfg.ActiveRoots()) {
				m.state = stateList
				m.statusMsg = errorStyle.Render("no root selected")
				return m, nil
			}
			m.pendingRoot = m.cfg.ActiveRoots()[m.genericPicker.cursor]
			return m.openDomainPickerForClone(), nil
		}
	}
	return m, nil
}

func (m model) updatePickDomainForClone(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "up", "k":
			if m.genericPicker.cursor > 0 {
				m.genericPicker.cursor--
			}
			return m, nil
		case "down", "j":
			if m.genericPicker.cursor < len(m.genericPicker.options)-1 {
				m.genericPicker.cursor++
			}
			return m, nil
		case "enter":
			if len(m.genericPicker.options) == 0 {
				return m, nil
			}
			chosen := m.genericPicker.options[m.genericPicker.cursor]
			if chosen == createNewDomainOption {
				m.inputScr = newInputScreen(
					fmt.Sprintf("New Domain in \"%s\" — enter name:", config.RootName(m.pendingRoot)),
					"domain-name", "enter confirm  •  esc back",
				)
				m.state = stateNewDomainName
				m.pendingDomain = ""
				return m, nil
			}
			m.pendingDomain = chosen
			return m.openPlacement(m.pendingRoot, chosen, placeClone), nil
		}
	}
	return m, nil
}

func (m model) beginClone(domain string) (tea.Model, tea.Cmd) {
	m.pendingDomain = domain
	m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Cloning %s...", m.pendingGHRepo.Name))
	m.state = stateCloningRepo
	return m, cmdCloneRepo(m.context(), m.workspaces, m.pendingGHRepo.CloneURL, m.currentLocation(), m.pendingGHRepo.Name)
}
