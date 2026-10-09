package main

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/app"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/preset"
)

// ---------------------------------------------------------------------------
// Async message types
// ---------------------------------------------------------------------------

type moveProjectDoneMsg struct {
	id         uint64
	newProject core.Project
	err        error
	rejected   bool
	status     core.MoveStatus
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
				next.navigation.inspectorScroll = 0
			}
			if next.scopeIdentity() != m.scopeIdentity() || next.activeTab != m.activeTab {
				next.selected = nil
			}
			next.sizePresentation()
			if next.editorRootReturn && (next.state == stateList || next.state == stateEditor) {
				next.editorRootReturn = false
				next.state = stateEditor
				next.editorScr.cfg = next.cfg.Clone()
				next.editorScr.diagnostic = next.diagnostic
			}
			var commands []tea.Cmd
			if cmd != nil {
				commands = append(commands, cmd)
			}
			if next.scan.requested {
				commands = append(commands, next.scanCommand())
			}
			if next.tmux.requested {
				commands = append(commands, next.tmuxRefreshCommand())
			}
			if tick := next.scheduleAnimation(); tick != nil {
				commands = append(commands, tick)
			}
			result = next
			cmd = tea.Batch(commands...)
		}
	}()

	if flow, ok := msg.(ptyFlowMsg); ok {
		if m.state != statePTYExecution || flow.id != m.ptyScr.operationID {
			if started, ok := flow.msg.(ptyStepStartMsg); ok {
				if err := started.session.Close(); err != nil {
					m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: err.Error()}
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
		if m.editorRootReturn && m.state != stateEditor {
			m.editorScr, _ = m.editorScr.Update(msg)
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
	case scopeDoneMsg:
		return m.handleScopeDone(msg)
	case createPreflightMsg:
		return m.handleCreatePreflight(msg)
	case createDoneMsg:
		return m.handleCreateDone(msg)
	case deleteDoneMsg:
		return m.handleDeleteDone(msg)
	case tmuxSessionsMsg:
		return m.handleTmuxSessions(msg)
	case tmuxKilledMsg:
		return m.handleTmuxKilled(msg)
	case scanResultMsg:
		return m.handleScanResult(msg)
	case ReposLoadedMsg:
		return m.handleGitHubReposLoaded(msg)
	case DeviceStartedMsg:
		return m.handleDeviceStarted(msg)
	case AuthDoneMsg:
		return m.handleGitHubAuthDone(msg)
	case RepoCreatedMsg:
		return m.handleGitHubRepoCreated(msg)
	case CloneDoneMsg:
		return m.handleGitCloneDone(msg)
	case bulkPreflightMsg:
		return m.handleBulkPreflight(msg)
	case bulkDoneMsg:
		return m.handleBulkDone(msg)
	case moveProjectDoneMsg:
		return m.handleMoveProjectDone(msg)
	case animationTickMsg:
		return m.handleAnimationTick(msg)
	}

	return m.routeInput(msg)
}

// ---------------------------------------------------------------------------
// Background message handlers
// ---------------------------------------------------------------------------

func (m model) handleGitHubReposLoaded(msg ReposLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.ID != m.repositories.id || !m.cfg.IsGitHubConnected() {
		return m, nil
	}
	m.repositories.loading = false
	if msg.Err != nil {
		m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "✗  GitHub: " + msg.Err.Error()}
		return m, nil
	}
	m.githubRepos = msg.Repos
	m = m.rescan()
	m.diagnostic = app.Diagnostic{Severity: app.Success, Summary: fmt.Sprintf("✓  GitHub: %d repos loaded", len(msg.Repos))}
	if m.state == stateGitHubAuth {
		m.state = stateList
	}
	return m, nil
}

func (m model) handleGitHubAuthDone(msg AuthDoneMsg) (tea.Model, tea.Cmd) {
	if m.state != stateGitHubAuth || msg.ID != m.auth.id {
		return m, nil
	}
	if msg.Err != nil {
		m.auth.screen.err = msg.Err.Error()
		return m, nil
	}
	proposed := m.cfg.Clone()
	proposed.GitHubToken = msg.Token
	proposed.GitHubUsername = msg.Username
	if err := m.preferences.Config.Save(proposed); err != nil {
		m.auth.screen.err = "save credentials: " + err.Error()
		return m, nil
	}
	m.cfg = proposed
	m.auth.screen.done = true
	m.cancelAuth()
	return m, m.fetchRepos()
}

func (m model) handleGitHubRepoCreated(msg RepoCreatedMsg) (tea.Model, tea.Cmd) {
	if m.state != stateCreatingGitHub {
		return m, nil
	}
	if msg.Err != nil {
		m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "✗  GitHub repo creation failed: " + msg.Err.Error()}
		m.state = stateList
		return m, nil
	}

	m.creation.repo = msg.Repo

	return m.beginLocalCreation()
}

func (m model) handleGitCloneDone(msg CloneDoneMsg) (tea.Model, tea.Cmd) {
	if m.state != stateCloningRepo {
		return m, nil
	}
	if msg.Err != nil {
		m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "✗  clone failed: " + msg.Err.Error()}
		m.state = stateList
		return m, nil
	}
	m = m.rescan()
	m.state = stateList
	m.uiState.AddRecent(msg.Project)
	m.saveState()
	m.launch.project = msg.Project
	m.launch.ready = true
	m.launch.preset = preset.ByName(m.presets, m.presetSel.SelectedName())
	return m, tea.Quit
}

func (m model) handleMoveProjectDone(msg moveProjectDoneMsg) (tea.Model, tea.Cmd) {
	if m.state != stateMovingProject || msg.id != m.operationID {
		return m, nil
	}
	if msg.rejected {
		m.state = stateMovePickPlacement
		m.genericPicker.err = fmt.Sprintf("%s: %v", msg.status, msg.err)
		return m, nil
	}
	if msg.err != nil {
		var partial *core.PartialMoveError
		if errors.As(msg.err, &partial) {
			m.uiState.MoveProject(m.moveTarget.Path, msg.newProject)
			m.saveFilesystemState()
			m = m.rescan()
		}
		m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "✗  move failed: " + msg.err.Error()}
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
	m.diagnostic = app.Diagnostic{Severity: app.Success, Summary: fmt.Sprintf("✓  moved \"%s\"", msg.newProject.Name)}
	m.state = stateList
	return m, nil
}

// ---------------------------------------------------------------------------
// Open action handlers
// ---------------------------------------------------------------------------

func (m model) openMoveDestDomainPicker() model {
	domains := m.navigation.domains[m.pendingRoot]
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
	domains := m.navigation.domains[m.pendingRoot]
	opts := append(append([]string{}, domains...), createNewDomainOption)
	m.genericPicker = newGenericPicker(
		fmt.Sprintf("Clone \"%s\" — select domain:", m.creation.repo.Name),
		opts, "↑/↓  •  enter  •  esc",
	)
	m.state = statePickDomainForClone
	return m
}

func (m model) openDomainPicker(root, projectName string) (model, tea.Cmd) {
	domains := m.navigation.domains[root]
	// Collisions belong to the selected location, not the entire domain.
	existing := map[string]bool{}
	m.creation.name = projectName
	m.pendingRoot = root
	m.domainPicker = newDomainPickerScreen(projectName, domains, existing)
	m.state = statePickDomain
	return m, nil
}

func (m model) openRootPickerForProject(projectName string) model {
	m.creation.name = projectName
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
		if k.String() == "enter" && m.editorScr.layer == editorLayerList && !m.editorScr.deleting && m.editorScr.tab == editorTabSettings && m.editorScr.cursor > 0 {
			m.editorRootReturn = true
			m.diagnostic = app.Diagnostic{}
			if m.editorScr.cursor == 1 {
				return m.startAddRoot()
			}
			return m.startRemoveRoot()
		}
		if k.String() == "esc" && m.editorScr.layer == editorLayerList && !m.editorScr.deleting {
			m.state = stateList
			return m, nil
		}
		if k.String() == "ctrl+c" && m.editorScr.layer == editorLayerList && !m.editorScr.deleting {
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
	if m.tmux.deleting {
		return m, nil
	}
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "enter":
			typed := strings.TrimSpace(m.confirmDelTmux.input.Value())
			if typed != m.confirmDelTmux.target() {
				m.confirmDelTmux.err = "name does not match — try again or esc to cancel"
				m.confirmDelTmux.input.SetValue("")
				return m, nil
			}
			name := m.confirmDelTmux.sessionName
			m.tmux.deleting = true
			return m, func() tea.Msg { return tmuxKilledMsg{name: name, err: m.tmuxClient.KillSession(name)} }
		}
	}
	var cmd tea.Cmd
	m.confirmDelTmux, cmd = m.confirmDelTmux.Update(msg)
	return m, cmd
}

func (m model) loading() bool { return m.state.kind() == routeOperation }
