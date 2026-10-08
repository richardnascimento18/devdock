package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/preset"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/ui"
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
			m.pendingProjectName = name
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
				m.statusMsg = errorStyle.Render("no root selected")
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
			return m.openDomainPicker(root, m.pendingProjectName)
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
				if err := core.CreateDomain(m.pendingRoot, name); err != nil {
					m.inputScr.err = err.Error()
					return m, nil
				}
				return m.openPlacement(m.pendingRoot, name, placeClone), nil
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
				m.statusMsg = errorStyle.Render("no root selected")
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
			if err := core.CreateDomain(m.pendingRoot, name); err != nil {
				m.inputScr.err = fmt.Sprintf("error: %v", err)
				return m, nil
			}
			if m.domainIntent == domainForMove {
				return m.openMovePlacementPicker(m.pendingRoot, name), nil
			}
			m.state = stateList
			return m.rescan(), nil
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
			m.pendingPreset = m.presetPicker.Selected()
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
			m.pendingTemplate = m.templatePicker.Selected()
			if m.cfg.IsGitHubConnected() {
				m.yesNoScr = newYesNoScreen("Create a GitHub repository for this project?")
				m.state = stateAskCreateGitHub
				return m, nil
			}
			return m.finishCreateProject(m.pendingRoot, m.pendingDomain, m.pendingPreset)
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
				m.pendingCreateGH = true
				m.yesNoScr = newYesNoScreen("Make the repository private?")
				m.state = stateAskRepoPrivacy
				return m, nil
			}
			m.pendingCreateGH = false
			return m.finishCreateProject(m.pendingRoot, m.pendingDomain, m.pendingPreset)
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
			m.pendingGHPrivate = m.yesNoScr.IsYes()
			return m.finishCreateProject(m.pendingRoot, m.pendingDomain, m.pendingPreset)
		}
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// finishCreateProject
// ---------------------------------------------------------------------------

func (m model) finishCreateProject(root, domainName string, ps preset.Preset) (tea.Model, tea.Cmd) {
	if !core.ValidName(domainName) || !core.ValidName(m.pendingProjectName) {
		m.statusMsg = errorStyle.Render("invalid project or domain name")
		m.state = stateList
		return m, nil
	}
	location := m.currentLocation()
	path, err := location.ProjectPath(m.pendingProjectName)
	if err == nil {
		err = core.CheckDestination(location.Root, path)
	}
	if err != nil {
		m.statusMsg = errorStyle.Render(ui.SafeBlock(err.Error()))
		m.state = stateList
		return m, nil
	}

	if m.pendingCreateGH && m.cfg.IsGitHubConnected() {
		m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Creating GitHub repo \"%s\"...", m.pendingProjectName))
		m.state = stateCreatingGitHub
		return m, cmdCreateRepo(m.context(), m.cfg.GitHubToken, m.pendingProjectName, m.pendingGHPrivate)
	}

	if m.pendingTemplate != nil {
		vars := tmpl.Vars{
			ProjectName: m.pendingProjectName,
			Domain:      domainName,
			Root:        root,
		}
		projectPath, workDir, err := core.PrepareProject(location, vars.ProjectName, m.pendingTemplate.CreatesProjectFolder)
		if err != nil {
			m.statusMsg = errorStyle.Render(ui.SafeBlock("create project: " + err.Error()))
			m.state = stateList
			return m, nil
		}
		vars.ProjectPath = projectPath
		m.ptyScr = newPTYScreen(m.termW, m.termH, m.pendingTemplate, projectPath, vars,
			m.pendingTemplate.Steps, workDir, m.pendingGHRepo)
		m.operationID++
		m.ptyScr.operationID = m.operationID
		m.state = statePTYExecution
		m.pendingLaunchPreset = ps
		return m, m.ptyScr.startNextStep()
	}

	p, err := core.CreateProject(location, m.pendingProjectName)
	if err != nil {
		m.inputScr.err = fmt.Sprintf("error: %v", err)
		return m, nil
	}
	if err := tmpl.WriteDevDockMarkerFile(p.Path); err != nil {
		m.statusMsg = dimStyle.Render(ui.SafeBlock(fmt.Sprintf("note: could not write .devdock marker: %v", err)))
	}
	m = m.rescan()
	m.state = stateList
	m.uiState.AddRecent(p)
	m.saveState()
	m.pendingLaunch = p
	m.pendingLaunchReady = true
	m.pendingLaunchPreset = ps
	return m, tea.Quit
}

// ---------------------------------------------------------------------------
// updatePTYExecution
// ---------------------------------------------------------------------------

func (m model) updatePTYExecution(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(ptyInterruptMsg); ok {
		m.state = statePickTemplate
		m.statusMsg = dimStyle.Render("interrupted — choose a template")
		return m, nil
	}

	var cmd tea.Cmd
	m.ptyScr, cmd = m.ptyScr.Update(msg)

	if m.ptyScr.completed {
		if m.ptyScr.interrupted {
			return m, cmd
		}
		if m.ptyScr.exitErr != nil {
			m.statusMsg = errorStyle.Render(ui.SafeBlock(fmt.Sprintf("✗  Template setup failed: %v", m.ptyScr.exitErr)))
			m.state = stateList
			return m, nil
		}
		projectPath := m.ptyScr.projectPath
		p := core.Project{Location: m.currentLocation(), Name: filepath.Base(projectPath),
			Path: projectPath,
		}
		if m.pendingGHRepo.FullName != "" {
			p.GitHubRepo = m.pendingGHRepo.FullName
		}
		m.uiState.AddRecent(p)
		m.saveState()
		m.pendingLaunch = p
		m.pendingLaunchReady = true
		m.statusMsg = successStyle.Render(ui.SafeBlock(fmt.Sprintf("✓  Project \"%s\" created successfully", p.Name)))
		return m, tea.Quit
	}

	return m, cmd
}
