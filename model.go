package main

import (
	"strings"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/preset"
	uistate "github.com/richardnascimento18/devdock/internal/state"
	tmpl "github.com/richardnascimento18/devdock/internal/template"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	cfg       config.Config
	state     appState
	list      list.Model
	allItems  []list.Item
	rootSel   rootSelectorWidget
	presetSel presetSelectorWidget
	presets   []preset.Preset
	templates []tmpl.Template
	termW     int
	termH     int

	activeTab int

	uiState            uistate.UIState
	treeMode           bool
	collapsedGroups    map[string]bool
	collapsedSubgroups map[string]bool

	githubRepos   []gh.Repo
	githubIndex   map[string]core.Project
	nameIndex     map[string]core.Project
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
	pendingPreset      preset.Preset
	pendingTemplate    *tmpl.Template
	pendingGHRepo      gh.Repo
	pendingCreateGH    bool
	pendingGHPrivate   bool
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
	destPath string
	domain   string
}

func newModel(projects []core.Project, cfg config.Config, presets []preset.Preset, templates []tmpl.Template, uiSt uistate.UIState) model {
	roots := cfg.ActiveRoots()

	m := model{
		cfg:                cfg,
		presets:            presets,
		templates:          templates,
		uiState:            uiSt,
		rootSel:            newRootSelector(roots),
		presetSel:          newPresetSelector(presets, cfg.DefaultPreset),
		treeMode:           uiSt.TreeMode,
		collapsedGroups:    uiSt.CollapsedGroups,
		collapsedSubgroups: uiSt.CollapsedSubgroups,
		activeTab:          uiSt.ActiveTab,
		rawProjects:        projects,
		termW:              120,
		termH:              40,
		nameIndex:          core.BuildNameIndex(projects),
	}
	if m.collapsedGroups == nil {
		m.collapsedGroups = make(map[string]bool)
	}
	if m.collapsedSubgroups == nil {
		m.collapsedSubgroups = make(map[string]bool)
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
	if m.cfg.IsGitHubConnected() {
		return gh.CmdFetchRepos(m.cfg.GitHubToken)
	}
	return nil
}

func (m model) activeRoot() string { return m.rootSel.Selected() }
func (m model) isAllMode() bool    { return m.activeRoot() == "" }

func (m model) saveState() {
	m.uiState.CollapsedGroups = m.collapsedGroups
	m.uiState.CollapsedSubgroups = m.collapsedSubgroups
	m.uiState.TreeMode = m.treeMode
	m.uiState.ActiveTab = m.activeTab
	uistate.Save(config.Dir(), m.uiState)
}

func (m model) rescan() model {
	roots := m.cfg.ActiveRoots()
	if len(roots) == 0 {
		return m
	}
	var projects []core.Project
	var err error
	if m.isAllMode() {
		projects, err = core.ScanRoots(config.Dir(), roots, collectFn)
	} else {
		projects, err = core.ScanRoot(m.activeRoot(), collectFn)
	}
	if err != nil {
		return m
	}
	verified := false
	if m.cfg.IsGitHubConnected() && len(m.githubRepos) > 0 {
		projects = gh.LinkProjectsToRepos(projects, m.githubRepos)
		verified = true
	}
	m.rawProjects = projects
	m.nameIndex = core.BuildNameIndex(projects)
	m.githubIndex = core.BuildGitHubIndex(projects)
	m.isFiltered = false

	items := m.buildListItems(projects, verified)
	if len(m.githubRepos) > 0 {
		items = m.appendGitHubItems(items, projects)
	}
	m.allItems = items
	m = m.refreshTabList()
	return m
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
			items = append(items, recentItem{entry: r})
		}
		m.list.SetItems(items)
	case TabFavorites:
		var items []list.Item
		showRoot := m.isAllMode() && len(m.cfg.ActiveRoots()) > 1
		for _, p := range m.rawProjects {
			if m.uiState.Favorites[p.Path] {
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
	items := m.buildListItems(m.rawProjects, verified)
	if len(m.githubRepos) > 0 {
		items = m.appendGitHubItems(items, m.rawProjects)
	}
	m.allItems = items
	m = m.refreshTabList()
	return m
}
