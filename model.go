package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/preset"
	uistate "github.com/richardnascimento18/devdock/internal/state"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/tmux"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	filesystemStatePending bool
	tmuxRefreshRequested   bool
	tmuxRefreshID          uint64
	tmuxDeleting           bool
	spinnerID              uint64
	scanRequested          bool
	scanID                 uint64
	workspaceTree          core.Workspace
	workspaceDomains       map[string][]string
	rootIntent             rootPickerIntent
	domainIntent           domainIntent
	groupFlow              groupWorkflow
	operationID            uint64
	authID                 uint64
	repoLoadID             uint64
	authContext            context.Context
	authCancel             context.CancelFunc
	authClient             authClient
	cfg                    config.Config
	state                  appState
	list                   list.Model
	allItems               []list.Item
	rootSel                rootSelectorWidget
	presetSel              presetSelectorWidget
	presets                []preset.Preset
	templates              []tmpl.Template
	termW                  int
	termH                  int

	activeTab int

	uiState        uistate.UIState
	committedState uistate.UIState
	persistenceErr error
	treeMode       bool
	collapsedNodes map[core.NodeKey]bool

	githubRepos   []gh.Repo
	githubAuthScr githubAuthScreen
	deviceCode    gh.DeviceCodeResponse
	isFiltered    bool

	inputScr         inputScreen
	domainPicker     domainPickerScreen
	genericPicker    genericPickerScreen
	presetPicker     presetPickerScreen
	templatePicker   templatePickerScreen
	yesNoScr         yesNoScreen
	spinnerScr       spinnerScreen
	ptyScr           ptyScreen
	confirmDelDomain confirmDeleteDomainScreen
	editorScr        editorScreen
	confirmDelTmux   confirmDeleteTmuxScreen

	// tmux-sessions tab
	tmuxSessions      []string
	pendingTmuxAttach string // session name to attach after TUI exits

	pendingProjectName string
	pendingDomainName  string
	pendingRoot        string
	pendingDomain      string
	pendingLocation    core.Location
	pendingPreset      preset.Preset
	pendingTemplate    *tmpl.Template
	pendingGHRepo      gh.Repo
	pendingCreateGH    bool
	pendingGHPrivate   bool
	placementIntent    placementIntent
	moveTarget         core.Project
	movePlacementOpts  []movePlacementOption
	deleteTarget       core.Project
	lastFilter         string
	statusMsg          string

	pendingLaunch       core.Project
	pendingLaunchReady  bool
	pendingLaunchPreset preset.Preset

	rawProjects []core.Project
}

type movePlacementOption struct {
	label    string
	location core.Location
}

func newModel(projects []core.Project, cfg config.Config, presets []preset.Preset, templates []tmpl.Template, uiSt uistate.UIState) model {
	roots := cfg.ActiveRoots()

	m := model{
		cfg:            cfg,
		repoLoadID:     1,
		tmuxRefreshID:  1,
		presets:        presets,
		templates:      templates,
		uiState:        uiSt.Clone(),
		committedState: uiSt.Clone(),
		rootSel:        newRootSelector(roots),
		presetSel:      newPresetSelector(presets, cfg.DefaultPreset),
		treeMode:       uiSt.TreeMode,
		collapsedNodes: cloneCollapsed(uiSt.CollapsedNodes),
		activeTab:      uiSt.ActiveTab,
		rawProjects:    projects,
		termW:          120,
		termH:          40,
	}
	if m.collapsedNodes == nil {
		m.collapsedNodes = make(map[core.NodeKey]bool)
	}
	if cfg.IsGitHubConnected() {
		m.statusMsg = dimStyle.Render("↻  loading GitHub repos...")
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetHeight(1)
	delegate.Styles.NormalTitle = listNormalStyle
	delegate.Styles.SelectedTitle = selectedStyle

	items := m.buildListItems(projects, false)
	m.allItems = items

	l := list.New(items, delegate, 100, 30)
	l.Title = ""
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new project")),
			key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "move project")),
			key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "del project")),
			key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "del domain")),
			key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "collapse/expand")),
			key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "toggle tree/flat")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add root")),
			key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "rem root")),
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clear filter")),
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "cycle root")),
			key.NewBinding(key.WithKeys("p"), key.WithHelp("p/P", "preset")),
			key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "github")),
			key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "editor")),
		}
	}
	m.list = l
	return m
}

func (m model) Init() tea.Cmd {
	var commands []tea.Cmd
	if m.cfg.IsGitHubConnected() {
		commands = append(commands, gh.CmdFetchRepos(m.cfg.GitHubToken, m.repoLoadID))
	}
	if m.activeTab == TabTmux {
		id := m.tmuxRefreshID
		commands = append(commands, func() tea.Msg {
			sessions, err := tmux.ListSessions()
			return tmuxSessionsMsg{id: id, sessions: sessions, err: err}
		})
	}
	return tea.Batch(commands...)
}

func (m model) activeRoot() string { return m.rootSel.Selected() }
func (m model) isAllMode() bool    { return m.activeRoot() == "" }

func (m *model) saveState() bool {
	m.uiState.CollapsedNodes = m.collapsedNodes
	m.uiState.TreeMode = m.treeMode
	m.uiState.ActiveTab = m.activeTab
	if err := uistate.Save(config.Dir(), m.uiState); err != nil {
		m.persistenceErr = fmt.Errorf("save state: %w", err)
		if !m.filesystemStatePending {
			m.uiState = m.committedState.Clone()
		}
		m.collapsedNodes = m.uiState.CollapsedNodes
		m.treeMode = m.uiState.TreeMode
		m.activeTab = m.uiState.ActiveTab
		*m = m.rebuildList(m.cfg.IsGitHubConnected() && len(m.githubRepos) > 0)
		m.statusMsg = errorStyle.Render("save state: " + err.Error())
		return false
	}
	m.committedState = m.uiState.Clone()
	m.persistenceErr = nil
	m.filesystemStatePending = false
	return true
}

func (m model) appendGitHubItems(items []list.Item, projects []core.Project) []list.Item {
	linkedRepos := make(map[string]bool)
	for _, p := range projects {
		if p.GitHubRepo != "" {
			linkedRepos[strings.ToLower(p.GitHubRepo)] = true
		}
	}
	for _, repo := range m.githubRepos {
		if !linkedRepos[strings.ToLower(repo.FullName)] {
			items = append(items, githubItem{repo: repo})
		}
	}
	return items
}

func (m model) refreshTabList() model {
	verified := m.cfg.IsGitHubConnected() && len(m.githubRepos) > 0
	switch m.activeTab {
	case TabRecents:
		var items []list.Item
		for _, r := range m.uiState.Recents {
			if !m.isAllMode() && r.Root != m.activeRoot() {
				continue
			}
			items = append(items, recentItem{entry: r})
		}
		m.list.SetItems(items)
	case TabFavorites:
		var items []list.Item
		showRoot := m.isAllMode() && len(m.cfg.ActiveRoots()) > 1
		for _, p := range m.rawProjects {
			if m.uiState.Favorites[p.Path] && (m.isAllMode() || p.Root == m.activeRoot()) {
				items = append(items, favoriteItem{project: p, showRoot: showRoot, verified: verified})
			}
		}
		m.list.SetItems(items)
	case TabTmux:
		var items []list.Item
		for _, s := range m.tmuxSessions {
			items = append(items, tmuxSessionItem{name: s})
		}
		m.list.SetItems(items)
	default: // TabSearch
		if !m.isFiltered {
			m.list.SetItems(m.allItems)
		}
	}
	return m
}

func (m model) listHeight() int {
	reserved := 22
	h := m.termH - reserved
	if h < 5 {
		h = 5
	}
	if h > 40 {
		h = 40
	}
	return h
}

func (m model) rebuildList(verified bool) model {
	selected := rowIdentity(m.list.SelectedItem())
	items := m.buildListItems(m.rawProjects, verified)
	if len(m.githubRepos) > 0 {
		items = m.appendGitHubItems(items, m.rawProjects)
	}
	m.allItems = items
	m = m.refreshTabList()
	if selected != "" {
		for i, it := range m.list.Items() {
			if rowIdentity(it) == selected {
				m.list.Select(i)
				break
			}
		}
	}
	return m
}

func cloneCollapsed(src map[core.NodeKey]bool) map[core.NodeKey]bool {
	dst := make(map[core.NodeKey]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// Filesystem changes cannot be rolled back by reverting UI state. Retain their
// reconciliation in memory on write failure so the next successful save retries.
func (m *model) saveFilesystemState() bool {
	m.filesystemStatePending = true
	return m.saveState()
}

func rowIdentity(it list.Item) string {
	switch x := it.(type) {
	case item:
		return string(x.project.Key())
	case flatItem:
		return string(x.project.Key())
	case favoriteItem:
		return string(x.project.Key())
	case groupItem:
		return string(x.nodeKey)
	case recentItem:
		return x.entry.Path
	case githubItem:
		return x.repo.FullName
	case tmuxSessionItem:
		return x.name
	}
	return ""
}
