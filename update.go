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

func CmdMoveProjectToPath(p core.Project, destPath, destRoot, destDomain string) tea.Cmd {
	return func() tea.Msg {
		next, err := core.MoveProject(p, destPath, destRoot, destDomain)
		return moveProjectDoneMsg{newProject: next, err: err}
	}
}

// ---------------------------------------------------------------------------
// Update dispatcher
// ---------------------------------------------------------------------------

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.list.SetWidth(min(sz.Width-10, 100))
		m.list.SetHeight(m.listHeight())
		return m, nil
	}

	switch msg := msg.(type) {
	case gh.ReposLoadedMsg:
		return m.handleGitHubReposLoaded(msg)
	case gh.DeviceStartedMsg:
		return m.handleDeviceStarted(msg)
	case gh.AuthDoneMsg:
		return m.handleGitHubAuthDone(msg)
	case gh.RepoCreatedMsg:
		return m.handleGitHubRepoCreated(msg)
	case gh.CloneDoneMsg:
		return m.handleGitCloneDone(msg)
	case moveProjectDoneMsg:
		return m.handleMoveProjectDone(msg)
	case spinnerTickMsg:
		m.spinnerScr.frame = (m.spinnerScr.frame + 1) % len(m.spinnerScr.frames)
		return m, spinnerTick()
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
	case stateCreatingGitHub, stateCloningRepo:
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
		return m.updateMovePickPlacement(msg)
	case stateHelp:
		return m.updateHelp(msg)
	case stateCreateGroup:
		return m.updateCreateGroup(msg)
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
	linked := gh.LinkProjectsToRepos(m.rawProjects, msg.Repos)
	m.rawProjects = linked
	m = m.rebuildList(true)
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
	if msg.Err != nil {
		m.statusMsg = errorStyle.Render("✗  GitHub repo creation failed: " + msg.Err.Error())
		m.state = stateList
		return m, nil
	}

	m.pendingGHRepo = msg.Repo

	if m.pendingTemplate != nil {
		domainPath := filepath.Join(m.pendingRoot, m.pendingDomain)
		vars := tmpl.Vars{
			ProjectName: m.pendingProjectName,
			Domain:      m.pendingDomain,
			Root:        m.pendingRoot,
		}
		var projectPath, workDir string
		if m.pendingTemplate.CreatesProjectFolder {
			vars.ProjectPath = filepath.Join(domainPath, m.pendingProjectName)
			projectPath = vars.ProjectPath
			workDir = domainPath
		} else {
			projectPath = filepath.Join(domainPath, m.pendingProjectName)
			if err := os.MkdirAll(projectPath, 0o755); err != nil {
				m.statusMsg = errorStyle.Render(fmt.Sprintf("✗  Could not create project directory: %v", err))
				m.state = stateList
				return m, nil
			}
			vars.ProjectPath = projectPath
			workDir = projectPath
		}
		m.ptyScr = newPTYScreen(m.termW, m.termH, m.pendingTemplate, projectPath, vars,
			m.pendingTemplate.Steps, workDir, msg.Repo)
		m.operationID++
		m.ptyScr.operationID = m.operationID
		m.state = statePTYExecution
		return m, m.ptyScr.startNextStep()
	}

	p, err := core.CreateProject(m.pendingRoot, m.pendingDomain, m.pendingProjectName)
	if err != nil {
		m.statusMsg = errorStyle.Render("✗  project create error: " + err.Error())
		m.state = stateList
		return m, nil
	}
	if err := gh.InitRepoWithRemote(p.Path, msg.Repo.CloneURL); err != nil {
		m.statusMsg = errorStyle.Render("✗  git init error: " + err.Error())
	}
	p.GitHubRepo = msg.Repo.FullName
	m = m.rescan()
	m.state = stateList
	m.uiState.AddRecent(p)
	m.saveState()
	m.pendingLaunch = p
	m.pendingLaunchReady = true
	m.pendingLaunchPreset = m.pendingPreset
	return m, tea.Quit
}

func (m model) handleGitCloneDone(msg gh.CloneDoneMsg) (tea.Model, tea.Cmd) {
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
	if msg.err != nil {
		m.statusMsg = errorStyle.Render("✗  move failed: " + msg.err.Error())
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
	domains, _ := core.ScanDomainsInRoot(m.pendingRoot)
	opts := append(append([]string{}, domains...), createNewDomainOption)
	m.genericPicker = newGenericPicker(
		fmt.Sprintf("Move \"%s\" — destination domain:", m.moveTarget.Name),
		opts, "↑/↓  •  enter  •  esc",
	)
	m.state = stateMovePickDomain
	return m
}

func (m model) openMovePlacementPicker(destRoot, destDomain string) model {
	domainPath := filepath.Join(destRoot, destDomain)
	groups := core.ScanGroupsInDomain(domainPath, classifyFn)

	var opts []movePlacementOption
	opts = append(opts, movePlacementOption{
		label:    "(place directly in domain)",
		destPath: filepath.Join(domainPath, m.moveTarget.Name),
		domain:   destDomain,
	})
	for _, g := range groups {
		opts = append(opts, movePlacementOption{
			label:    g.Name,
			destPath: filepath.Join(g.Path, m.moveTarget.Name),
			domain:   destDomain,
		})
		for _, sg := range g.Subgroups {
			opts = append(opts, movePlacementOption{
				label:    g.Name + " > " + sg.Name,
				destPath: filepath.Join(sg.Path, m.moveTarget.Name),
				domain:   destDomain,
			})
			for _, nested := range sg.Subgroups {
				opts = append(opts, movePlacementOption{
					label:    g.Name + " > " + sg.Name + " > " + nested.Name,
					destPath: filepath.Join(nested.Path, m.moveTarget.Name),
					domain:   destDomain,
				})
			}
		}
	}
	labels := make([]string, len(opts))
	for i, o := range opts {
		labels[i] = o.label
	}
	m.movePlacementOpts = opts
	m.genericPicker = newGenericPicker(
		fmt.Sprintf("Move \"%s\" — place in:", m.moveTarget.Name),
		labels, "↑/↓  •  enter  •  esc",
	)
	m.state = stateMovePickPlacement
	return m
}

func (m model) openDomainPickerForClone() model {
	domains, _ := core.ScanDomainsInRoot(m.pendingRoot)
	opts := append(append([]string{}, domains...), createNewDomainOption)
	m.genericPicker = newGenericPicker(
		fmt.Sprintf("Clone \"%s\" — select domain:", m.pendingGHRepo.Name),
		opts, "↑/↓  •  enter  •  esc",
	)
	m.state = statePickDomainForClone
	return m
}

func (m model) openDomainPicker(root, projectName string) (model, tea.Cmd) {
	domains, _ := core.ScanDomainsInRoot(root)
	existing := map[string]bool{}
	for _, it := range m.allItems {
		if p, ok := it.(item); ok && p.project.Name == projectName && p.project.Root == root {
			existing[p.project.Domain] = true
		}
	}
	m.pendingProjectName = projectName
	m.pendingRoot = root
	m.domainPicker = newDomainPickerScreen(projectName, domains, existing)
	m.state = statePickDomain
	return m, nil
}

func (m model) openRootPickerForProject(projectName string) model {
	m.pendingProjectName = projectName
	opts := make([]string, len(m.cfg.ActiveRoots()))
	for i, r := range m.cfg.ActiveRoots() {
		opts[i] = config.RootName(r)
	}
	m.genericPicker = newGenericPicker(
		fmt.Sprintf("Select root for \"%s\":", projectName),
		opts, "↑/↓  •  enter  •  esc",
	)
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
		m.presets = deepCopyPresets(m.editorScr.presets)
		m.templates = deepCopyTemplates(m.editorScr.tmpls)
		m.presetSel.SetPresets(m.presets)
	}

	return m, cmd
}

// isSafePathName returns true if name is safe to use as a filesystem path component.
// It rejects empty names, dot-only names, and anything containing path separators
// or backslashes that could be used to escape the intended directory.
func isSafePathName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.ContainsAny(name, "/\\") {
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Tmux-sessions tab handlers
// ---------------------------------------------------------------------------

func (m model) refreshTmuxSessions() model {
	sessions, err := tmux.ListSessions()
	if err != nil {
		m.statusMsg = errorStyle.Render(err.Error())
	} else {
		m.tmuxSessions = sessions
	}
	return m
}

func (m model) updateDeleteTmuxSession(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if err := tmux.KillSession(m.confirmDelTmux.sessionName); err != nil {
				m.confirmDelTmux.err = fmt.Sprintf("error: %v", err)
				return m, nil
			}
			m = m.refreshTmuxSessions()
			m = m.refreshTabList()
			m.statusMsg = successStyle.Render(fmt.Sprintf("✓  session \"%s\" killed", m.confirmDelTmux.sessionName))
			m.state = stateList
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.confirmDelTmux, cmd = m.confirmDelTmux.Update(msg)
	return m, cmd
}
