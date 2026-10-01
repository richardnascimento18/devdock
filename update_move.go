package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/config"
)

func (m model) updateMovePickRoot(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m.openMoveDestDomainPicker(), nil
		}
	}
	return m, nil
}

func (m model) updateMovePickDomain(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				m.domainIntent = domainForMove
				m.state = stateCreateDomainOnly
				return m, nil
			}
			return m.openMovePlacementPicker(m.pendingRoot, chosen), nil
		}
	}
	return m, nil
}

func (m model) updateMovePickPlacement(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			opt := m.movePlacementOpts[m.genericPicker.cursor]
			m.spinnerID++
			m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Moving \"%s\"...", m.moveTarget.Name))
			m.state = stateMovingProject
			return m, tea.Batch(
				CmdMoveProjectToPath(m.moveTarget, opt.destPath, m.pendingRoot, opt.domain),
				spinnerTick(m.spinnerID),
			)
		}
	}
	return m, nil
}
