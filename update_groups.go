package main

import (
	"github.com/richardnascimento18/devdock/internal/app"

	"strings"

	"github.com/richardnascimento18/devdock/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) startCreateGroup() (tea.Model, tea.Cmd) {
	m.groupFlow = groupWorkflow{}
	roots := m.cfg.ActiveRoots()
	if len(roots) == 0 {
		m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "no roots configured"}
		return m, nil
	}
	// Determine which root/domain to use from context
	if len(roots) == 1 {
		m.pendingRoot = roots[0]
		return m.openDomainPickerForGroup()
	}
	m.genericPicker = newRootPicker("Create Group — select root:", roots, "↑/↓  •  enter  •  esc")
	m.state = statePickRoot
	m.rootIntent = rootForCreateGroup
	return m, nil
}

func (m model) openDomainPickerForGroup() (tea.Model, tea.Cmd) {
	domains := m.navigation.domains[m.pendingRoot]
	if len(domains) == 0 {
		m.state = stateList
		m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "no domains in this root — create a domain first (N)"}
		return m, nil
	}
	opts := make([]string, len(domains))
	copy(opts, domains)
	m.genericPicker = newGenericPicker("Create Group — select domain:", opts, "↑/↓  •  enter  •  esc")
	m.state = stateCreateGroup
	return m, nil
}

func (m model) updateCreateGroup(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.pendingDomain = ""
			m.state = stateList
			return m, nil
		}

		// Phase 1: picking domain (genericPicker is active)
		if m.groupFlow.phase == groupPickDomain {
			switch k.String() {
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
				m.groupFlow.domain = m.genericPicker.options[m.genericPicker.cursor]
				return m.openPlacement(m.pendingRoot, m.groupFlow.domain, placeGroup), nil
			}
			return m, nil
		}

		// Phase 2: entering group name (inputScr is active)
		if k.String() == "enter" {
			name := strings.TrimSpace(m.inputScr.input.Value())
			if name == "" {
				m.inputScr.err = "name cannot be empty"
				return m, nil
			}
			return m.beginScopeMutation(mutationGroup, name)
		}
	}

	// Phase 2: delegate to inputScr
	if m.groupFlow.phase == groupEnterName {
		var cmd tea.Cmd
		m.inputScr, cmd = m.inputScr.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) startDeleteGroup() (tea.Model, tea.Cmd) {
	m.groupFlow = groupWorkflow{}
	roots := m.cfg.ActiveRoots()
	if len(roots) == 0 {
		m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "no roots configured"}
		return m, nil
	}
	if len(roots) == 1 {
		m.pendingRoot = roots[0]
		return m.openGroupPickerForDelete()
	}
	m.genericPicker = newRootPicker("Delete Group — select root:", roots, "↑/↓  •  enter  •  esc")
	m.rootIntent = rootForDeleteGroup
	m.state = statePickRoot
	return m, nil
}

func (m model) openGroupPickerForDelete() (tea.Model, tea.Cmd) {
	domains := m.navigation.domains[m.pendingRoot]
	var groups []string
	m.groupFlow.deleteLocations = nil
	for _, d := range domains {
		for _, loc := range m.navigation.tree.Locations(m.pendingRoot, d) {
			if len(loc.GroupPath) > 0 {
				groups = append(groups, loc.Breadcrumb())
				m.groupFlow.deleteLocations = append(m.groupFlow.deleteLocations, loc)
			}
		}
	}
	if len(groups) == 0 {
		m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "no groups found in this root"}
		m.state = stateList
		return m, nil
	}
	m.genericPicker = newLocationPicker("Delete Group — select group:", m.groupFlow.deleteLocations, groups, "↑/↓  •  enter  •  esc cancel")
	m.state = stateDeleteGroup
	return m, nil
}

func (m model) updateDeleteGroup(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			location := m.groupFlow.deleteLocations[m.genericPicker.cursor]
			path, err := location.Resolve()
			if err != nil {
				m.genericPicker.err = err.Error()
				return m, nil
			}
			m.groupFlow.deletePath = path
			if err := core.ValidateDescendant(m.pendingRoot, m.groupFlow.deletePath); err != nil {
				m.genericPicker.err = err.Error()
				return m, nil
			}
			m.inputScr = newInputScreen("Permanently delete group and ALL its projects? Type the full path:", m.groupFlow.deletePath, "type full path • enter confirm • esc cancel")
			m.state = stateConfirmDeleteGroup
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateConfirmDeleteGroup(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c":
			m.state = stateList
			m.groupFlow = groupWorkflow{}
			return m, nil
		case "enter":
			if strings.TrimSpace(m.inputScr.input.Value()) != m.groupFlow.deletePath || m.groupFlow.deletePath == "" {
				m.inputScr.err = "full path does not match"
				return m, nil
			}
			return m.beginDelete(m.pendingRoot, m.groupFlow.deletePath)
		}
	}
	var cmd tea.Cmd
	m.inputScr, cmd = m.inputScr.Update(msg)
	return m, cmd
}
