package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/preset"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/tmux"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Async message types
// ---------------------------------------------------------------------------

type moveProjectDoneMsg struct {
	newProject core.Project
	err        error
}

// ---------------------------------------------------------------------------
// Update dispatcher
// ---------------------------------------------------------------------------

func (m model) Update(msg tea.Msg) (result tea.Model, cmd tea.Cmd) {
	defer func() {
		if next, ok := result.(model); ok {
			if next.state != m.state {
				next.modalScroll = 0
			}
			if rowIdentity(next.list.SelectedItem()) != rowIdentity(m.list.SelectedItem()) {
				next.inspectorScroll = 0
			}
			if next.scopeIdentity() != m.scopeIdentity() || next.activeTab != m.activeTab {
				next.selected = nil
			}
			next.sizePresentation()
			if next.editorRootReturn && (next.state == stateList || next.state == stateEditor) {
				next.editorRootReturn = false
				next.state = stateEditor
				next.editorScr.cfg = next.cfg.Clone()
				next.editorScr.statusMsg = next.statusMsg
			}
			var commands []tea.Cmd
			if cmd != nil {
				commands = append(commands, cmd)
			}
			if next.scanRequested {
				commands = append(commands, next.scanCommand())
			}
			if next.tmuxRefreshRequested {
				commands = append(commands, next.tmuxRefreshCommand())
			}
			result = next
			cmd = tea.Batch(commands...)
		}
	}()

	if flow, ok := msg.(ptyFlowMsg); ok {
		if m.state != statePTYExecution || flow.id != m.ptyScr.operationID {
			if started, ok := flow.msg.(ptyStepStartMsg); ok {
				if err := started.session.Close(); err != nil {
					m.statusMsg = errorStyle.Render(err.Error())
				}
			}
			return m, nil
		}
		msg = flow.msg
	}

	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.termW = sz.Width
		m.termH = sz.Height
		m.sizePresentation()
		if m.flowModal() {
			m.modalScroll = min(m.modalScroll, m.modalScrollLimit())
		}
		if m.state == statePTYExecution {
			return m.updatePTYExecution(msg)
		}
		if m.state == stateEditor {
			return m.updateEditor(msg)
		}
		return m, nil
	}
	if key, ok := msg.(tea.KeyMsg); ok && m.flowModal() {
		switch key.String() {
		case "pgdown":
			m.modalScroll = min(m.modalScroll+max(m.termH-7, 1), m.modalScrollLimit())
			return m, nil
		case "pgup":
			m.modalScroll = max(m.modalScroll-max(m.termH-7, 1), 0)
			return m, nil
		default:
			m.modalScroll = 0
		}
	}

	switch msg := msg.(type) {
	case deleteDoneMsg:
		return m.handleDeleteDone(msg)
	case tmuxSessionsMsg:
		return m.handleTmuxSessions(msg)
	case tmuxKilledMsg:
		return m.handleTmuxKilled(msg)
	case scanResultMsg:
		return m.handleScanResult(msg)
	case gh.ReposLoadedMsg:
		return m.handleGitHubReposLoaded(msg)
	case gh.DeviceStartedMsg:
		return m.handleDeviceStarted(msg)
	case gh.AuthDoneMsg:
		return m.handleGitHubAuthDone(msg)
	case gh.RepoCreatedMsg:
		return m.handleGitHubRepoCreated(msg)
	case gitLinkedMsg:
		return m.handleGitLinked(msg)
	case gh.CloneDoneMsg:
		return m.handleGitCloneDone(msg)
	case bulkPreflightMsg:
		return m.handleBulkPreflight(msg)
	case bulkDoneMsg:
		return m.handleBulkDone(msg)
	case moveProjectDoneMsg:
		return m.handleMoveProjectDone(msg)
	case spinnerTickMsg:
		if msg.id != m.spinnerID || !m.loading() || len(m.spinnerScr.frames) == 0 {
			return m, nil
		}
		m.spinnerScr.frame = (m.spinnerScr.frame + 1) % len(m.spinnerScr.frames)
		return m, spinnerTick(m.spinnerID)
	}

	switch m.state {
	case stateNewProjectName:
		return m.updateNewProjectName(msg)
	case statePickRoot:
		return m.updatePickRoot(msg)
	case statePickDomain:
		return m.updateDomainPicker(msg)
	case stateNewDomainName:
		return m.updateNewDomainName(msg)
	case stateCreateDomainOnly:
		return m.updateCreateDomainOnly(msg)
	case statePickRootForDomain:
		return m.updatePickRootForDomain(msg)
	case statePickPreset:
		return m.updatePickPreset(msg)
	case stateAskCreateGitHub:
		return m.updateAskCreateGitHub(msg)
	case stateAskRepoPrivacy:
		return m.updateAskRepoPrivacy(msg)
	case stateBulkPreflight:
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
			m.state = stateList
			m.bulk = bulkWorkflow{}
		}
		return m, nil
	case stateBulkMoving, stateCreatingGitHub, stateCloningRepo, stateMovingProject, stateDeletingWorkspace:
		return m, nil
	case statePickTemplate:
		return m.updatePickTemplate(msg)
	case statePTYExecution:
		return m.updatePTYExecution(msg)
	case stateDeleteProject:
		return m.updateDeleteProject(msg)
	case stateDeleteDomain:
		return m.updateDeleteDomain(msg)
	case stateAddRoot:
		return m.updateAddRoot(msg)
	case stateRemoveRoot:
		return m.updateRemoveRoot(msg)
	case stateGitHubAuth:
		return m.updateGitHubAuth(msg)
	case statePickRootForClone:
		return m.updatePickRootForClone(msg)
	case statePickDomainForClone:
		return m.updatePickDomainForClone(msg)
	case stateMovePickRoot:
		return m.updateMovePickRoot(msg)
	case stateMovePickDomain:
		return m.updateMovePickDomain(msg)
	case stateMovePickPlacement:
		return m.updatePlacement(msg)
	case stateBulkConfirm, stateBulkResult:
		return m.updateBulk(msg)
	case stateHelp:
		return m.updateHelp(msg)
	case statePalette:
		return m.updatePalette(msg)
	case stateCreateGroup:
		return m.updateCreateGroup(msg)
	case stateConfirmDeleteGroup:
		return m.updateConfirmDeleteGroup(msg)
	case stateDeleteGroup:
		return m.updateDeleteGroup(msg)
	case stateEditor:
		return m.updateEditor(msg)
	case stateDeleteTmuxSession:
		return m.updateDeleteTmuxSession(msg)
	default:
		return m.updateList(msg)
	}
}

// ---------------------------------------------------------------------------
// Background message handlers
// ---------------------------------------------------------------------------

func (m model) handleGitHubReposLoaded(msg gh.ReposLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.ID != m.repoLoadID || !m.cfg.IsGitHubConnected() {
		return m, nil
	}
	if msg.Err != nil {
		m.statusMsg = errorStyle.Render("✗  GitHub: " + msg.Err.Error())
		return m, nil
	}
	m.githubRepos = msg.Repos
	m = m.rescan()
	m.statusMsg = successStyle.Render(fmt.Sprintf("✓  GitHub: %d repos loaded", len(msg.Repos)))
	if m.state == stateGitHubAuth {
		m.state = stateList
	}
	return m, nil
}

func (m model) handleGitHubAuthDone(msg gh.AuthDoneMsg) (tea.Model, tea.Cmd) {
	if m.state != stateGitHubAuth || msg.ID != m.authID {
		return m, nil
	}
	if msg.Err != nil {
		m.githubAuthScr.err = msg.Err.Error()
		return m, nil
	}
	proposed := m.cfg.Clone()
	proposed.GitHubToken = msg.Token
	proposed.GitHubUsername = msg.Username
	if err := config.Save(proposed); err != nil {
		m.githubAuthScr.err = "save credentials: " + err.Error()
		return m, nil
	}
	m.cfg = proposed
	m.githubAuthScr.done = true
	m.cancelAuth()
	return m, m.fetchRepos()
}

func (m model) handleGitHubRepoCreated(msg gh.RepoCreatedMsg) (tea.Model, tea.Cmd) {
	if m.state != stateCreatingGitHub {
		return m, nil
	}
	if msg.Err != nil {
		m.statusMsg = errorStyle.Render("✗  GitHub repo creation failed: " + msg.Err.Error())
		m.state = stateList
		return m, nil
	}

	m.pendingGHRepo = msg.Repo

	if m.pendingTemplate != nil {
		vars := tmpl.Vars{
			ProjectName: m.pendingProjectName,
			Domain:      m.pendingDomain,
			Root:        m.pendingRoot,
		}
		projectPath, workDir, err := core.PrepareProject(m.currentLocation(), vars.ProjectName, m.pendingTemplate.CreatesProjectFolder)
		if err != nil {
			m.statusMsg = errorStyle.Render("create project: " + err.Error())
			m.state = stateList
			return m, nil
		}
		vars.ProjectPath = projectPath
		m.ptyScr = newPTYScreen(m.termW, m.termH, m.pendingTemplate, projectPath, vars,
			m.pendingTemplate.Steps, workDir, msg.Repo)
		m.pendingLaunchPreset = m.pendingPreset
		m.operationID++
		m.ptyScr.operationID = m.operationID
		m.state = statePTYExecution
		return m, m.ptyScr.startNextStep()
	}

	p, err := core.CreateProject(m.currentLocation(), m.pendingProjectName)
	if err != nil {
		m.statusMsg = errorStyle.Render("✗  project create error: " + err.Error())
		m.state = stateList
		return m, nil
	}
	if err := tmpl.WriteDevDockMarkerFile(p.Path); err != nil {
		m.statusMsg = errorStyle.Render(err.Error())
		m.state = stateList
		return m, nil
	}
	p.GitHubRepo = msg.Repo.FullName
	return m, func() tea.Msg { return gitLinkedMsg{project: p, err: gh.InitRepoWithRemote(p.Path, msg.Repo.CloneURL)} }
}

func (m model) handleGitCloneDone(msg gh.CloneDoneMsg) (tea.Model, tea.Cmd) {
	if m.state != stateCloningRepo {
		return m, nil
	}
	if msg.Err != nil {
		m.statusMsg = errorStyle.Render("✗  clone failed: " + msg.Err.Error())
		m.state = stateList
		return m, nil
	}
	m = m.rescan()
	m.state = stateList
	m.uiState.AddRecent(msg.Project)
	m.saveState()
	m.pendingLaunch = msg.Project
	m.pendingLaunchReady = true
	m.pendingLaunchPreset = preset.ByName(m.presets, m.presetSel.SelectedName())
	return m, tea.Quit
}

func (m model) handleMoveProjectDone(msg moveProjectDoneMsg) (tea.Model, tea.Cmd) {
	if m.state != stateMovingProject {
		return m, nil
	}
	if msg.err != nil {
		var partial *core.PartialMoveError
		if errors.As(msg.err, &partial) {
			m.uiState.MoveProject(m.moveTarget.Path, msg.newProject)
			m.saveFilesystemState()
			m = m.rescan()
		}
		m.statusMsg = errorStyle.Render("✗  move failed: " + msg.err.Error())
		m.state = stateList
		return m, nil
	}
	m.uiState.MoveProject(m.moveTarget.Path, msg.newProject)
	if !m.saveFilesystemState() {
		m = m.rescan()
		m.state = stateList
		return m, nil
	}
	m = m.rescan()
	m.statusMsg = successStyle.Render(fmt.Sprintf("✓  moved \"%s\"", msg.newProject.Name))
	m.state = stateList
	return m, nil
}

// ---------------------------------------------------------------------------
// Open action handlers
// ---------------------------------------------------------------------------

func (m model) openMoveDestDomainPicker() model {
	domains := m.workspaceDomains[m.pendingRoot]
	opts := append(append([]string{}, domains...), createNewDomainOption)
	m.genericPicker = newGenericPicker(
		fmt.Sprintf("Move \"%s\" — destination domain:", m.moveTarget.Name),
		opts, "↑/↓  •  enter  •  esc",
	)
	if len(m.bulk.projects) > 0 {
		m.genericPicker.title = fmt.Sprintf("Move %d selected projects — destination domain", len(m.bulk.projects))
	}
	m.state = stateMovePickDomain
	return m
}

func (m model) openMovePlacementPicker(destRoot, destDomain string) model {
	return m.openPlacement(destRoot, destDomain, placeMove)
}

func (m model) openDomainPickerForClone() model {
	domains := m.workspaceDomains[m.pendingRoot]
	opts := append(append([]string{}, domains...), createNewDomainOption)
	m.genericPicker = newGenericPicker(
		fmt.Sprintf("Clone \"%s\" — select domain:", m.pendingGHRepo.Name),
		opts, "↑/↓  •  enter  •  esc",
	)
	m.state = statePickDomainForClone
	return m
}

func (m model) openDomainPicker(root, projectName string) (model, tea.Cmd) {
	domains := m.workspaceDomains[root]
	// Collisions belong to the selected location, not the entire domain.
	existing := map[string]bool{}
	m.pendingProjectName = projectName
	m.pendingRoot = root
	m.domainPicker = newDomainPickerScreen(projectName, domains, existing)
	m.state = statePickDomain
	return m, nil
}

func (m model) openRootPickerForProject(projectName string) model {
	m.pendingProjectName = projectName
	m.genericPicker = newRootPicker(fmt.Sprintf("Select root for %q:", projectName), m.cfg.ActiveRoots(), "↑/↓ • enter • esc")
	m.state = statePickRoot
	return m
}

func (m model) openPresetPicker() model {
	m.presetPicker = newPresetPicker(m.presets, m.presetSel.SelectedName())
	m.state = statePickPreset
	return m
}

func navPicker(g *genericPickerScreen, key string) bool {
	switch key {
	case "up", "k":
		if g.cursor > 0 {
			g.cursor--
		}
		return true
	case "down", "j":
		if g.cursor < len(g.options)-1 {
			g.cursor++
		}
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Editor state handler
// ---------------------------------------------------------------------------

func (m model) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	// ESC at the list layer returns to stateList
	if k, ok := msg.(tea.KeyMsg); ok {
		if k.String() == "enter" && m.editorScr.layer == editorLayerList && m.editorScr.tab == editorTabSettings && m.editorScr.cursor > 0 {
			m.editorRootReturn = true
			m.statusMsg = ""
			if m.editorScr.cursor == 1 {
				return m.startAddRoot()
			}
			return m.startRemoveRoot()
		}
		if k.String() == "esc" && m.editorScr.layer == editorLayerList {
			m.state = stateList
			return m, nil
		}
		if k.String() == "ctrl+c" {
			m.state = stateList
			return m, nil
		}
	}

	var cmd tea.Cmd
	revision := m.editorScr.revision
	m.editorScr, cmd = m.editorScr.Update(msg)
	if m.editorScr.revision != revision {
		oldDefault := m.cfg.DefaultPreset
		m.cfg = m.editorScr.cfg.Clone()
		m.presets = deepCopyPresets(m.editorScr.presets)
		m.templates = deepCopyTemplates(m.editorScr.tmpls)
		m.presetSel.SetPresets(m.presets)
		if oldDefault != m.cfg.DefaultPreset {
			m.presetSel = newPresetSelector(m.presets, m.cfg.DefaultPreset)
		}
	}

	return m, cmd
}

// isSafePathName returns true if name is safe to use as a filesystem path component.
// It rejects empty names, dot-only names, and anything containing path separators
// or backslashes that could be used to escape the intended directory.
func isSafePathName(name string) bool { return core.ValidName(name) }

// ---------------------------------------------------------------------------
// Tmux-sessions tab handlers
// ---------------------------------------------------------------------------

func (m model) updateDeleteTmuxSession(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.tmuxDeleting {
		return m, nil
	}
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "enter":
			typed := strings.TrimSpace(m.confirmDelTmux.input.Value())
			if typed != m.confirmDelTmux.sessionName {
				m.confirmDelTmux.err = "name does not match — try again or esc to cancel"
				m.confirmDelTmux.input.SetValue("")
				return m, nil
			}
			name := m.confirmDelTmux.sessionName
			m.tmuxDeleting = true
			return m, func() tea.Msg { return tmuxKilledMsg{name: name, err: tmux.KillSession(name)} }
		}
	}
	var cmd tea.Cmd
	m.confirmDelTmux, cmd = m.confirmDelTmux.Update(msg)
	return m, cmd
}

func (m model) loading() bool {
	return m.state == stateBulkPreflight || m.state == stateBulkMoving || m.state == stateCreatingGitHub || m.state == stateCloningRepo || m.state == stateMovingProject || m.state == stateDeletingWorkspace
}

type gitLinkedMsg struct {
	project core.Project
	err     error
}

func (m model) handleGitLinked(msg gitLinkedMsg) (tea.Model, tea.Cmd) {
	if m.state != stateCreatingGitHub {
		return m, nil
	}
	if msg.err != nil {
		m.statusMsg = errorStyle.Render("git setup failed: " + msg.err.Error())
		m.state = stateList
		return m.rescan(), nil
	}
	m.uiState.AddRecent(msg.project)
	m.saveState()
	m.pendingLaunch = msg.project
	m.pendingLaunchReady = true
	m.pendingLaunchPreset = m.pendingPreset
	return m, tea.Quit
}
