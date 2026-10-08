package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/app"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/preset"
	uistate "github.com/richardnascimento18/devdock/internal/state"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/tmux"
	"github.com/richardnascimento18/devdock/internal/ui"
)

type model struct {
	navigation   navigationState
	creation     creationState
	launch       launchState
	auth         authState
	scan         scanState
	repositories repositoriesState
	tmux         tmuxState
	query        queryState

	tmuxClient             tmux.Client
	workspaces             app.Workspaces
	appContext             context.Context
	preferences            app.Preferences
	selected               map[string]bool
	bulk                   bulkWorkflow
	bulkID                 uint64
	palette                paletteScreen
	helpScroll             int
	modalScroll            int
	filesystemStatePending bool
	motion                 animationClock
	moveFilesystem         core.Mover
	statusDetails          string
	rootIntent             rootPickerIntent
	domainIntent           domainIntent
	groupFlow              groupWorkflow
	operationID            uint64
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

	githubRepos []gh.Repo
	isFiltered  bool

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
	editorRootReturn bool
	confirmDelTmux   confirmDeleteTmuxScreen

	// tmux-sessions tab
	pendingDomainName string
	pendingRoot       string
	pendingDomain     string
	pendingLocation   core.Location
	placementIntent   placementIntent
	moveTarget        core.Project
	movePlacementOpts []movePlacementOption
	deleteTarget      core.Project
	diagnostic        app.Diagnostic
}

type movePlacementOption struct {
	label    string
	location core.Location
}

func newModelWithPreferences(projects []core.Project, cfg config.Config, presets []preset.Preset, templates []tmpl.Template, uiSt uistate.UIState, preferences app.Preferences) model {
	roots := cfg.ActiveRoots()

	m := model{
		preferences: preferences, workspaces: app.NewWorkspaces(), tmuxClient: tmux.NewClient(),
		cfg: cfg, motion: animationClock{generation: 1, reduced: reducedMotion()},
		navigation:   navigationState{focus: ui.Projects, projects: projects},
		repositories: repositoriesState{loading: cfg.IsGitHubConnected(), id: 1},
		tmux:         tmuxState{refreshing: uiSt.ActiveTab == TabTmux, id: 1},
		presets:      presets, templates: templates,
		uiState: uiSt.Clone(), committedState: uiSt.Clone(),
		rootSel: newRootSelector(roots), presetSel: newPresetSelector(presets, cfg.DefaultPreset),
		treeMode: uiSt.TreeMode, collapsedNodes: cloneCollapsed(uiSt.CollapsedNodes), activeTab: uiSt.ActiveTab,
		termW: 120, termH: 40,
	}

	if m.collapsedNodes == nil {
		m.collapsedNodes = make(map[core.NodeKey]bool)
	}
	if cfg.IsGitHubConnected() {
		m.diagnostic = app.Diagnostic{Severity: app.Info, Summary: "↻  loading GitHub repos..."}
	}

	delegate := projectDelegate{}

	items := m.buildListItems(projects, false)
	m.allItems = items

	l := list.New(items, delegate, 100, 30)
	l.Title = ""
	l.Styles.Title = promptStyle
	l.Styles.FilterPrompt = promptStyle
	l.Styles.FilterCursor = cursorStyle
	l.Styles.NoItems = dimStyle
	ui.ConfigureInput(&l.FilterInput)
	l.DisableQuitKeybindings()
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
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
	m.query.input = transparentInput()
	m.query.input.Prompt = "/ "
	m.refreshProjectDisambiguators()
	m.sizePresentation()
	m.motion.pending = !m.motion.reduced && m.activityLabel() != ""
	m.motion.label = m.activityLabel()
	return m
}

func (m model) Init() tea.Cmd {
	var commands []tea.Cmd
	if m.scan.startup != nil {
		commands = append(commands, m.scan.startup)
	}
	if m.motion.pending {
		commands = append(commands, animationTick(m.motion.generation, 320*time.Millisecond))
	}
	if m.cfg.IsGitHubConnected() {
		commands = append(commands, cmdFetchRepos(m.context(), m.cfg.GitHubToken, m.repositories.id))
	}
	if m.activeTab == TabTmux {
		id := m.tmux.id
		commands = append(commands, func() tea.Msg {
			sessions, err := m.tmuxClient.ListSessions()
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
	if err := m.preferences.State.Save(m.uiState); err != nil {
		m.persistenceErr = fmt.Errorf("save state: %w", err)
		if !m.filesystemStatePending {
			m.uiState = m.committedState.Clone()
		}
		m.collapsedNodes = m.uiState.CollapsedNodes
		m.treeMode = m.uiState.TreeMode
		m.activeTab = m.uiState.ActiveTab
		*m = m.rebuildList(m.cfg.IsGitHubConnected() && len(m.githubRepos) > 0)
		m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "save state: " + err.Error(), Cause: err, Operation: "save-state"}
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
			if !m.inWorkspaceScope(r.Location) {
				continue
			}
			items = append(items, recentItem{entry: r})
		}
		m.list.SetItems(items)
	case TabFavorites:
		var items []list.Item
		showRoot := m.isAllMode() && len(m.cfg.ActiveRoots()) > 1
		for _, p := range m.navigation.projects {
			if m.uiState.Favorites[p.Path] && m.inWorkspaceScope(p.Location) {
				items = append(items, favoriteItem{project: p, showRoot: showRoot, verified: verified})
			}
		}
		m.list.SetItems(items)
	case TabTmux:
		var items []list.Item
		for _, s := range m.tmux.sessions {
			items = append(items, m.sessionItem(s))
		}
		m.list.SetItems(items)
	default: // TabSearch
		if !m.isFiltered {
			m.list.SetItems(m.allItems)
		}
	}
	return m
}

func (m *model) sizePresentation() {
	l := m.dashboardLayout()
	m.list.SetDelegate(projectDelegate{projects: m.navigation.projectIndex, duplicates: m.navigation.duplicateLocations, sessions: m.tmux.cached, favorites: m.uiState.Favorites, selected: m.selected, focused: m.navigation.focus == ui.Projects})
	m.list.SetSize(max(l.Projects, 1), max(l.BodyHeight-1, 1))
	m.query.input.Width = max(min(m.termW-6, 72), 1)
}

func (m model) rebuildList(verified bool) model {
	m.refreshProjectDisambiguators()
	m.reconcileSelection()
	m.refreshWorkspaceRows()
	selected := rowIdentity(m.list.SelectedItem())
	items := m.buildListItems(m.navigation.projects, verified)
	if len(m.githubRepos) > 0 {
		items = m.appendGitHubItems(items, m.navigation.projects)
	}
	m.allItems = items
	m = m.refreshTabList()
	m = m.applySearch()
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

// Snapshot-derived metadata is cached when project/scope data changes, never
// rebuilt for every animation frame or terminal resize.
func (m *model) refreshProjectDisambiguators() {
	names := map[string]int{}
	m.navigation.projectIndex = make(map[string]core.Project, len(m.navigation.projects))
	m.navigation.duplicateLocations = map[string]bool{}
	for _, p := range m.navigation.projects {
		names[p.Name+"\x00"+compactProjectLocation(p)]++
		m.navigation.projectIndex[p.Path] = p
	}
	for _, p := range m.navigation.projects {
		m.navigation.duplicateLocations[p.Path] = names[p.Name+"\x00"+compactProjectLocation(p)] > 1
	}
}

func (m model) context() context.Context {
	if m.appContext == nil {
		return context.Background()
	}
	return m.appContext
}
