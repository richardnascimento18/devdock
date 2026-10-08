package main

import (
	"github.com/richardnascimento18/devdock/internal/app"

	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/preset"

	"path/filepath"
	"strings"
)

func (m model) updateNewProjectName(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.inputScr.input.Value())
			if name == "" {
				m.inputScr.err = "name cannot be empty"
				return m, nil
			}
			if !isSafePathName(name) {
				m.inputScr.err = "name contains invalid characters (/ and \\ are not allowed)"
				return m, nil
			}
			m.creation.name = name
			roots := m.cfg.ActiveRoots()
			if len(roots) == 1 {
				return m.openDomainPicker(roots[0], name)
			}
			return m.openRootPickerForProject(name), nil
		}
	}
	var cmd tea.Cmd
	m.inputScr, cmd = m.inputScr.Update(msg)
	return m, cmd
}

func (m model) updatePickRoot(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "no root selected"}
				return m, nil
			}
			root := m.cfg.ActiveRoots()[m.genericPicker.cursor]
			m.pendingRoot = root
			switch m.rootIntent {
			case rootForCreateGroup:
				return m.openDomainPickerForGroup()
			case rootForDeleteGroup:
				return m.openGroupPickerForDelete()
			}
			return m.openDomainPicker(root, m.creation.name)
		}
	}
	return m, nil
}

func (m model) updateDomainPicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "up", "k":
			if m.domainPicker.cursor > 0 {
				m.domainPicker.cursor--
			}
			return m, nil
		case "down", "j":
			if m.domainPicker.cursor < len(m.domainPicker.options)-1 {
				m.domainPicker.cursor++
			}
			return m, nil
		case "enter":
			chosen := m.domainPicker.options[m.domainPicker.cursor]
			if chosen == createNewDomainOption {
				m.inputScr = newInputScreen(
					fmt.Sprintf("New Domain in \"%s\" — enter name:", config.RootName(m.pendingRoot)),
					"domain-name", "enter confirm  •  esc back",
				)
				m.state = stateNewDomainName
				return m, nil
			}
			if m.domainPicker.disabled[chosen] {
				return m, nil
			}
			m.pendingDomain = chosen
			return m.openPlacement(m.pendingRoot, chosen, placeProject), nil
		}
	}
	return m, nil
}

func (m model) updateNewDomainName(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = statePickDomain
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.inputScr.input.Value())
			if name == "" {
				m.inputScr.err = "name cannot be empty"
				return m, nil
			}
			if !isSafePathName(name) {
				m.inputScr.err = "name contains invalid characters (/ and \\ are not allowed)"
				return m, nil
			}
			m.pendingDomain = name
			if m.domainIntent == domainForClone {
				return m.beginScopeMutation(mutationCloneDomain, name)
			}
			return m.openPlacement(m.pendingRoot, name, placeProject), nil
		}
	}
	var cmd tea.Cmd
	m.inputScr, cmd = m.inputScr.Update(msg)
	return m, cmd
}

func (m model) updatePickRootForDomain(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "no root selected"}
				return m, nil
			}
			root := m.cfg.ActiveRoots()[m.genericPicker.cursor]
			m.pendingRoot = root
			m.state = stateCreateDomainOnly
			m.inputScr = newInputScreen(
				fmt.Sprintf("New Domain in \"%s\" — enter name:", config.RootName(root)),
				"domain-name", "enter confirm  •  esc cancel",
			)
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateCreateDomainOnly(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "enter":
			name := strings.TrimSpace(m.inputScr.input.Value())
			if name == "" {
				m.inputScr.err = "name cannot be empty"
				return m, nil
			}
			if !isSafePathName(name) {
				m.inputScr.err = "name contains invalid characters (/ and \\ are not allowed)"
				return m, nil
			}
			if m.domainIntent == domainForMove {
				return m.beginScopeMutation(mutationMoveDomain, name)
			}
			return m.beginScopeMutation(mutationDomain, name)
		}
	}
	var cmd tea.Cmd
	m.inputScr, cmd = m.inputScr.Update(msg)
	return m, cmd
}

func (m model) updatePickPreset(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = statePickDomain
			return m, nil
		case "up", "k":
			if m.presetPicker.cursor > 0 {
				m.presetPicker.cursor--
			}
			return m, nil
		case "down", "j":
			if m.presetPicker.cursor < len(m.presetPicker.presets)-1 {
				m.presetPicker.cursor++
			}
			return m, nil
		case "enter":
			m.creation.preset = m.presetPicker.Selected()
			m.templatePicker = newTemplatePicker(m.templates)
			m.state = statePickTemplate
			return m, nil
		}
	}
	return m, nil
}

func (m model) updatePickTemplate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = statePickPreset
			return m, nil
		case "up", "k":
			if m.templatePicker.cursor > 0 {
				m.templatePicker.cursor--
			}
			return m, nil
		case "down", "j":
			max := len(m.templatePicker.templates)
			if m.templatePicker.cursor < max {
				m.templatePicker.cursor++
			}
			return m, nil
		case "enter":
			m.creation.template = m.templatePicker.Selected()
			if m.cfg.IsGitHubConnected() {
				m.yesNoScr = newYesNoScreen("Create a GitHub repository for this project?")
				m.state = stateAskCreateGitHub
				return m, nil
			}
			return m.finishCreateProject(m.pendingRoot, m.pendingDomain, m.creation.preset)
		}
	}
	return m, nil
}

func (m model) updateAskCreateGitHub(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = statePickPreset
			return m, nil
		case "up", "k":
			m.yesNoScr.cursor = 0
		case "down", "j":
			m.yesNoScr.cursor = 1
		case "enter":
			if m.yesNoScr.IsYes() {
				m.creation.createGitHub = true
				m.yesNoScr = newYesNoScreen("Make the repository private?")
				m.state = stateAskRepoPrivacy
				return m, nil
			}
			m.creation.createGitHub = false
			return m.finishCreateProject(m.pendingRoot, m.pendingDomain, m.creation.preset)
		}
	}
	return m, nil
}

func (m model) updateAskRepoPrivacy(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = stateAskCreateGitHub
			return m, nil
		case "up", "k":
			m.yesNoScr.cursor = 0
		case "down", "j":
			m.yesNoScr.cursor = 1
		case "enter":
			m.creation.private = m.yesNoScr.IsYes()
			return m.finishCreateProject(m.pendingRoot, m.pendingDomain, m.creation.preset)
		}
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// finishCreateProject
// ---------------------------------------------------------------------------

func (m model) finishCreateProject(root, domainName string, ps preset.Preset) (tea.Model, tea.Cmd) {
	if !core.ValidName(domainName) || !core.ValidName(m.creation.name) {
		m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "invalid project or domain name"}
		m.state = stateList
		return m, nil
	}
	m.creation.preset = ps
	return m.preflightCreation()
}

// ---------------------------------------------------------------------------
// updatePTYExecution
// ---------------------------------------------------------------------------

func (m model) updatePTYExecution(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(ptyInterruptMsg); ok {
		m.state = statePickTemplate
		m.diagnostic = app.Diagnostic{Severity: app.Info, Summary: "interrupted — choose a template"}
		return m, nil
	}

	var cmd tea.Cmd
	m.ptyScr, cmd = m.ptyScr.Update(msg)

	if m.ptyScr.completed {
		if m.ptyScr.interrupted {
			return m, cmd
		}
		if m.ptyScr.exitErr != nil {
			m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: fmt.Sprintf("✗  Template setup failed: %v", m.ptyScr.exitErr)}
			m.state = stateList
			return m, nil
		}
		projectPath := m.ptyScr.scaffold.ProjectPath
		p := core.Project{Location: m.currentLocation(), Name: filepath.Base(projectPath),
			Path: projectPath,
		}
		if m.creation.repo.FullName != "" {
			p.GitHubRepo = m.creation.repo.FullName
		}
		m.uiState.AddRecent(p)
		m.saveState()
		m.launch.project = p
		m.launch.ready = true
		m.diagnostic = app.Diagnostic{Severity: app.Success, Summary: fmt.Sprintf("✓  Project \"%s\" created successfully", p.Name)}
		return m, tea.Quit
	}

	return m, cmd
}
