package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/preset"
	tmpl "github.com/richardnascimento18/devdock/internal/template"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Sub-state handlers
// ---------------------------------------------------------------------------

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
			root := m.cfg.ActiveRoots()[m.genericPicker.cursor]
			if m.pendingProjectName == "__creategroup__" {
				m.pendingProjectName = ""
				m.pendingRoot = root
				return m.openDomainPickerForGroup()
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
			return m.openPresetPicker(), nil
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
			return m.openPresetPicker(), nil
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
			if m.pendingProjectName == "__move__" {
				m.pendingProjectName = ""
				return m.openMovePlacementPicker(m.pendingRoot, name), nil
			}
			m.state = stateList
			return m, nil
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
			m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Creating GitHub repo \"%s\"...", m.pendingProjectName))
			m.state = stateCreatingGitHub
			return m, tea.Batch(
				gh.CmdCreateRepo(m.cfg.GitHubToken, m.pendingProjectName, m.pendingGHPrivate),
				spinnerTick(),
			)
		}
	}
	return m, nil
}

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
			m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Cloning %s...", m.pendingGHRepo.Name))
			m.state = stateCloningRepo
			return m, tea.Batch(
				gh.CmdCloneRepo(m.pendingGHRepo.CloneURL, m.pendingRoot, chosen, m.pendingGHRepo.Name),
				spinnerTick(),
			)
		}
	}
	return m, nil
}

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
			chosen := m.genericPicker.options[m.genericPicker.cursor]
			if chosen == createNewDomainOption {
				m.inputScr = newInputScreen(
					fmt.Sprintf("New Domain in \"%s\" — enter name:", config.RootName(m.pendingRoot)),
					"domain-name", "enter confirm  •  esc back",
				)
				m.pendingProjectName = "__move__"
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
			m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Moving \"%s\"...", m.moveTarget.Name))
			m.state = stateCloningRepo
			return m, tea.Batch(
				CmdMoveProjectToPath(m.moveTarget, opt.destPath, m.pendingRoot, opt.domain),
				spinnerTick(),
			)
		}
	}
	return m, nil
}

func (m model) updateDeleteProject(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "enter":
			typed := strings.TrimSpace(m.inputScr.input.Value())
			if typed != m.deleteTarget.Path {
				m.inputScr.err = "path does not match — try again or esc to cancel"
				m.inputScr.input.SetValue("")
				return m, nil
			}
			if err := core.DeleteProject(m.deleteTarget); err != nil {
				m.inputScr.err = fmt.Sprintf("error: %v", err)
				return m, nil
			}
			m = m.rescan()
			m.state = stateList
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.inputScr, cmd = m.inputScr.Update(msg)
	return m, cmd
}

func (m model) updateDeleteDomain(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.pendingDomainName = ""
			m.pendingRoot = ""
			m.state = stateList
			return m, nil
		case "enter":
			if m.pendingDomainName == "" {
				typed := strings.TrimSpace(m.inputScr.input.Value())
				if typed == "" {
					m.inputScr.err = "name cannot be empty"
					return m, nil
				}
				roots := m.cfg.ActiveRoots()
				if !m.isAllMode() {
					roots = []string{m.activeRoot()}
				}
				foundRoot := ""
				for _, root := range roots {
					domains, _ := core.ScanDomainsInRoot(root)
					for _, d := range domains {
						if d == typed {
							foundRoot = root
							break
						}
					}
					if foundRoot != "" {
						break
					}
				}
				if foundRoot == "" {
					m.inputScr.err = fmt.Sprintf("domain \"%s\" not found", typed)
					return m, nil
				}
				m.pendingDomainName = typed
				m.pendingRoot = foundRoot
				m.confirmDelDomain = newConfirmDeleteDomainScreen(typed)
				return m, nil
			}
			typed := strings.TrimSpace(m.confirmDelDomain.input.Value())
			if typed != m.pendingDomainName {
				m.confirmDelDomain.err = "name does not match — type exactly or esc to cancel"
				m.confirmDelDomain.input.SetValue("")
				return m, nil
			}
			if err := core.DeleteDomain(m.pendingRoot, m.pendingDomainName); err != nil {
				m.confirmDelDomain.err = fmt.Sprintf("error: %v", err)
				return m, nil
			}
			m.pendingDomainName = ""
			m.pendingRoot = ""
			m = m.rescan()
			m.state = stateList
			return m, nil
		}
	}
	if m.pendingDomainName == "" {
		var cmd tea.Cmd
		m.inputScr, cmd = m.inputScr.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.confirmDelDomain, cmd = m.confirmDelDomain.Update(msg)
	return m, cmd
}

func (m model) updateAddRoot(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "enter":
			path := expandTilde(strings.TrimSpace(m.inputScr.input.Value()))
			if path == "" {
				m.inputScr.err = "path cannot be empty"
				return m, nil
			}
			path, err := filepath.Abs(path)
			if err != nil {
				m.inputScr.err = err.Error()
				return m, nil
			}
			info, err := os.Stat(path)
			if err != nil || !info.IsDir() {
				m.inputScr.err = "root must be an existing directory"
				return m, nil
			}
			proposed := m.cfg.Clone()
			proposed.AddRoot(path)
			if err := config.Save(proposed); err != nil {
				m.inputScr.err = fmt.Sprintf("error saving config: %v", err)
				return m, nil
			}
			m.cfg = proposed
			m.rootSel.SetRoots(m.cfg.ActiveRoots())
			m = m.rescan()
			m.state = stateList
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.inputScr, cmd = m.inputScr.Update(msg)
	return m, cmd
}

func expandTilde(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func (m model) updateRemoveRoot(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		if k.String() == "esc" {
			m.state = stateList
			return m, nil
		}
		if navPicker(&m.genericPicker, k.String()) {
			return m, nil
		}
		if k.String() == "enter" {
			root := m.cfg.ActiveRoots()[m.genericPicker.cursor]
			proposed := m.cfg.Clone()
			proposed.RemoveRoot(root)
			if err := config.Save(proposed); err != nil {
				m.genericPicker.err = fmt.Sprintf("error saving config: %v", err)
				return m, nil
			}
			m.cfg = proposed
			m.rootSel.SetRoots(m.cfg.ActiveRoots())
			m = m.rescan()
			m.state = stateList
			return m, nil
		}
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

// ---------------------------------------------------------------------------
// finishCreateProject
// ---------------------------------------------------------------------------

func (m model) finishCreateProject(root, domainName string, ps preset.Preset) (tea.Model, tea.Cmd) {
	domainPath := filepath.Join(root, domainName)

	if m.pendingCreateGH && m.cfg.IsGitHubConnected() {
		m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Creating GitHub repo \"%s\"...", m.pendingProjectName))
		m.state = stateCreatingGitHub
		return m, tea.Batch(
			gh.CmdCreateRepo(m.cfg.GitHubToken, m.pendingProjectName, m.pendingGHPrivate),
			spinnerTick(),
		)
	}

	if m.pendingTemplate != nil {
		vars := tmpl.Vars{
			ProjectName: m.pendingProjectName,
			Domain:      domainName,
			Root:        root,
		}
		var projectPath, workDir string
		if m.pendingTemplate.CreatesProjectFolder {
			vars.ProjectPath = filepath.Join(domainPath, m.pendingProjectName)
			projectPath = vars.ProjectPath
			workDir = domainPath
		} else {
			projectPath = filepath.Join(domainPath, m.pendingProjectName)
			if err := os.MkdirAll(projectPath, 0o755); err != nil {
				m.inputScr.err = fmt.Sprintf("error: %v", err)
				return m, nil
			}
			vars.ProjectPath = projectPath
			workDir = projectPath
		}
		m.ptyScr = newPTYScreen(m.termW, m.termH, m.pendingTemplate, projectPath, vars,
			m.pendingTemplate.Steps, workDir, m.pendingGHRepo)
		m.operationID++
		m.ptyScr.operationID = m.operationID
		m.state = statePTYExecution
		m.pendingLaunchPreset = ps
		return m, m.ptyScr.startNextStep()
	}

	p, err := core.CreateProject(root, domainName, m.pendingProjectName)
	if err != nil {
		m.inputScr.err = fmt.Sprintf("error: %v", err)
		return m, nil
	}
	if err := tmpl.WriteDevDockMarkerFile(p.Path); err != nil {
		m.statusMsg = dimStyle.Render(fmt.Sprintf("note: could not write .devdock marker: %v", err))
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
			m.statusMsg = errorStyle.Render(fmt.Sprintf("✗  Template setup failed: %v", m.ptyScr.exitErr))
			m.state = stateList
			return m, nil
		}
		projectPath := m.ptyScr.projectPath
		p := core.Project{
			Name:   filepath.Base(projectPath),
			Path:   projectPath,
			Domain: m.pendingDomain,
			Root:   m.pendingRoot,
		}
		if m.pendingGHRepo.FullName != "" {
			p.GitHubRepo = m.pendingGHRepo.FullName
		}
		m.pendingLaunch = p
		m.pendingLaunchReady = true
		m.statusMsg = successStyle.Render(fmt.Sprintf("✓  Project \"%s\" created successfully", p.Name))
		return m, tea.Quit
	}

	return m, cmd
}
