package main

import (
	"fmt"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/preset"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "?":
			if !m.list.SettingFilter() {
				m.state = stateHelp
				return m, nil
			}

		case "[":
			if !m.list.SettingFilter() {
				m.activeTab = (m.activeTab + len(tabNames) - 1) % len(tabNames)
				m.uiState.ActiveTab = m.activeTab
				m.saveState()
				if m.activeTab == TabTmux {
					m = m.refreshTmuxSessions()
				}
				m = m.refreshTabList()
				return m, nil
			}

		case "]":
			if !m.list.SettingFilter() {
				m.activeTab = (m.activeTab + 1) % len(tabNames)
				m.uiState.ActiveTab = m.activeTab
				m.saveState()
				if m.activeTab == TabTmux {
					m = m.refreshTmuxSessions()
				}
				m = m.refreshTabList()
				return m, nil
			}

		case "f":
			if !m.list.SettingFilter() {
				sel := m.list.SelectedItem()
				var projPath string
				switch s := sel.(type) {
				case item:
					projPath = s.project.Path
				case flatItem:
					projPath = s.item.project.Path
				case favoriteItem:
					projPath = s.project.Path
				}
				if projPath != "" {
					m.uiState.ToggleFavorite(projPath)
					if !m.saveState() {
						return m, nil
					}
					if m.uiState.Favorites[projPath] {
						m.statusMsg = successStyle.Render("★  added to favorites")
					} else {
						m.statusMsg = dimStyle.Render("☆  removed from favorites")
					}
					if m.activeTab == TabFavorites {
						m = m.refreshTabList()
					}

				}
				return m, nil
			}

		case "tab":
			m.rootSel.Next()
			m = m.rebuildList(m.cfg.IsGitHubConnected() && len(m.githubRepos) > 0)
			return m, nil
		case "shift+tab":
			m.rootSel.Prev()
			m = m.rebuildList(m.cfg.IsGitHubConnected() && len(m.githubRepos) > 0)
			return m, nil

		case "p", "P":
			if !m.list.SettingFilter() {
				proposedSel := m.presetSel
				if msg.String() == "p" {
					proposedSel.Next()
				} else {
					proposedSel.Prev()
				}
				proposed := m.cfg.Clone()
				proposed.DefaultPreset = proposedSel.SelectedName()
				if err := config.Save(proposed); err != nil {
					m.statusMsg = errorStyle.Render("save config: " + err.Error())
					return m, nil
				}
				m.cfg = proposed
				m.presetSel = proposedSel
				return m, nil
			}

		case "v":
			if !m.list.SettingFilter() {
				m.treeMode = !m.treeMode
				m = m.rebuildList(m.cfg.IsGitHubConnected() && len(m.githubRepos) > 0)
				mode := "flat"
				if m.treeMode {
					mode = "tree"
				}
				m.statusMsg = dimStyle.Render("view: " + mode)
				m.saveState()
				return m, nil
			}

		case " ":
			if !m.list.SettingFilter() {
				sel := m.list.SelectedItem()
				if gi, ok := sel.(groupItem); ok {
					m.collapsedNodes[gi.nodeKey] = !m.collapsedNodes[gi.nodeKey]
					m = m.rebuildList(m.cfg.IsGitHubConnected() && len(m.githubRepos) > 0)
					m.saveState()
					return m, nil
				}
			}

		case "g":
			if !m.list.SettingFilter() {
				return m.startGitHubAuth()
			}
		case "G":
			if !m.list.SettingFilter() {
				return m.startCreateGroup()
			}
		case "ctrl+g":
			if !m.list.SettingFilter() {
				return m.startDeleteGroup()
			}
		case "e":
			if !m.list.SettingFilter() {
				return m.startEditor()
			}

		case "r":
			if !m.list.SettingFilter() {
				m = m.rescan()
				var cmd tea.Cmd
				if m.cfg.IsGitHubConnected() {
					cmd = m.fetchRepos()
					m.statusMsg = dimStyle.Render("↻  refreshing GitHub repos...")
				}
				return m, cmd
			}

		case "c":
			if !m.list.SettingFilter() && m.isFiltered && m.activeTab == TabSearch {
				m.isFiltered = false
				m = m.refreshTabList()
				m.statusMsg = ""
				return m, nil
			}

		case "m":
			if !m.list.SettingFilter() {
				return m.startMoveProject(), nil
			}

		case "n":
			if !m.list.SettingFilter() {
				return m.startNewProject("")
			}
		case "N":
			if !m.list.SettingFilter() {
				return m.startNewDomainOnly()
			}
		case "x":
			if !m.list.SettingFilter() {
				if m.activeTab == TabTmux {
					return m.startDeleteTmuxSession()
				}
				return m.startDeleteProject()
			}
		case "X":
			if !m.list.SettingFilter() {
				return m.startDeleteDomain()
			}
		case "a":
			if !m.list.SettingFilter() {
				return m.startAddRoot()
			}
		case "A":
			if !m.list.SettingFilter() {
				return m.startRemoveRoot()
			}

		case "/":
			if !m.list.SettingFilter() && m.activeTab == TabSearch {
				searchModel := m
				searchModel.treeMode = false
				searchItems := searchModel.buildListItems(m.rawProjects, m.cfg.IsGitHubConnected())
				searchItems = m.appendGitHubItems(searchItems, m.rawProjects)
				m.list.SetItems(searchItems)
				m.list.SetFilteringEnabled(true)
			}

		case "esc":
			if !m.list.SettingFilter() && m.isFiltered && m.activeTab == TabSearch {
				m.isFiltered = false
				m = m.refreshTabList()
				m.statusMsg = ""
				return m, nil
			}

		case "enter":
			if m.list.SettingFilter() {
				query := m.list.FilterValue()
				m.lastFilter = query
				filtered := m.list.VisibleItems()
				var withCreate []list.Item
				withCreate = append(withCreate, filtered...)
				if m.activeTab == TabSearch {
					withCreate = append(withCreate, createProjectItem{name: query})
				}
				m.list.ResetFilter()
				m.list.SetItems(withCreate)
				m.isFiltered = true
				m.statusMsg = dimStyle.Render(fmt.Sprintf("filter: \"%s\"  •  c or esc to clear", query))
				return m, nil
			}
			sel := m.list.SelectedItem()
			if sel == nil {
				return m, nil
			}
			if gi, ok := sel.(groupItem); ok {
				m.collapsedNodes[gi.nodeKey] = !m.collapsedNodes[gi.nodeKey]
				m = m.rebuildList(m.cfg.IsGitHubConnected() && len(m.githubRepos) > 0)
				m.saveState()
				return m, nil
			}
			if cp, ok := sel.(createProjectItem); ok {
				return m.startNewProject(cp.name)
			}
			if it, ok := sel.(item); ok {
				ps := preset.ByName(m.presets, m.presetSel.SelectedName())
				m.uiState.AddRecent(it.project)
				m.saveState()
				m.pendingLaunch = it.project
				m.pendingLaunchReady = true
				m.pendingLaunchPreset = ps
				return m, tea.Quit
			}
			if fi, ok := sel.(flatItem); ok {
				ps := preset.ByName(m.presets, m.presetSel.SelectedName())
				m.uiState.AddRecent(fi.item.project)
				m.saveState()
				m.pendingLaunch = fi.item.project
				m.pendingLaunchReady = true
				m.pendingLaunchPreset = ps
				return m, tea.Quit
			}
			if fav, ok := sel.(favoriteItem); ok {
				ps := preset.ByName(m.presets, m.presetSel.SelectedName())
				m.uiState.AddRecent(fav.project)
				m.saveState()
				m.pendingLaunch = fav.project
				m.pendingLaunchReady = true
				m.pendingLaunchPreset = ps
				return m, tea.Quit
			}
			if re, ok := sel.(recentItem); ok {
				proj := core.Project{Location: re.entry.Location, Name: re.entry.Name, Path: re.entry.Path}
				ps := preset.ByName(m.presets, m.presetSel.SelectedName())
				m.uiState.AddRecent(proj)
				m.saveState()
				m.pendingLaunch = proj
				m.pendingLaunchReady = true
				m.pendingLaunchPreset = ps
				return m, tea.Quit
			}
			if gi, ok := sel.(githubItem); ok {
				return m.startCloneFlow(gi.repo), nil
			}
			if ts, ok := sel.(tmuxSessionItem); ok {
				m.pendingTmuxAttach = ts.name
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) startNewProject(name string) (tea.Model, tea.Cmd) {
	m.rootIntent = rootForProject
	m.domainIntent = domainForProject
	m.pendingGHRepo = gh.Repo{}
	m.pendingDomain = ""
	m.pendingLocation = core.Location{}
	m.pendingProjectName = ""
	m.pendingTemplate = nil
	m.pendingCreateGH = false
	m.pendingGHPrivate = false
	if name == "" {
		m.inputScr = newInputScreen("New Project — enter name:", "project-name", "enter confirm  •  esc cancel")
		m.state = stateNewProjectName
		return m, nil
	}
	if !isSafePathName(name) {
		m.inputScr = newInputScreen("New Project — enter name:", "project-name", "enter confirm  •  esc cancel")
		m.inputScr.err = "name contains invalid characters (/ and \\ are not allowed)"
		m.state = stateNewProjectName
		return m, nil
	}
	m.pendingProjectName = name
	roots := m.cfg.ActiveRoots()
	if len(roots) == 1 {
		return m.openDomainPicker(roots[0], name)
	}
	return m.openRootPickerForProject(name), nil
}

func (m model) startNewDomainOnly() (tea.Model, tea.Cmd) {
	m.domainIntent = domainOnly
	roots := m.cfg.ActiveRoots()
	if len(roots) == 1 {
		m.pendingRoot = roots[0]
		m.state = stateCreateDomainOnly
		m.inputScr = newInputScreen(
			fmt.Sprintf("New Domain in \"%s\" — enter name:", config.RootName(roots[0])),
			"domain-name", "enter confirm  •  esc cancel",
		)
		return m, nil
	}
	m.genericPicker = newRootPicker("Select root for new domain:", roots, "↑/↓  •  enter  •  esc")
	m.state = statePickRootForDomain
	return m, nil
}

func (m model) startDeleteProject() (tea.Model, tea.Cmd) {
	sel := m.list.SelectedItem()
	if _, ok := sel.(githubItem); ok {
		m.statusMsg = dimStyle.Render("Repository not cloned locally - nothing to delete")
		return m, nil
	}
	var proj core.Project
	switch s := sel.(type) {
	case item:
		proj = s.project
	case flatItem:
		proj = s.item.project
	case favoriteItem:
		proj = s.project
	default:
		return m, nil
	}
	m.deleteTarget = proj
	m.inputScr = newInputScreen(
		fmt.Sprintf("Delete \"%s\"? Type the path to confirm:", proj.Name),
		proj.Path,
		"type full path  •  enter confirm  •  esc cancel",
	)
	m.state = stateDeleteProject
	return m, nil
}

func (m model) startDeleteDomain() (tea.Model, tea.Cmd) {
	m.pendingDomainName = ""
	m.pendingRoot = ""
	m.inputScr = newInputScreen("Delete Domain — enter name:", "domain-name", "enter confirm  •  esc cancel")
	m.state = stateDeleteDomain
	return m, nil
}

func (m model) startAddRoot() (tea.Model, tea.Cmd) {
	m.inputScr = newInputScreen("Add Root Directory — enter path:", "~/projects", "enter confirm  •  esc cancel")
	m.state = stateAddRoot
	return m, nil
}

func (m model) startRemoveRoot() (tea.Model, tea.Cmd) {
	roots := m.cfg.ActiveRoots()
	if len(roots) == 0 {
		m.statusMsg = errorStyle.Render("no roots to remove")
		return m, nil
	}
	m.genericPicker = newRootPicker("Select root to remove:", roots, "↑/↓  •  enter  •  esc")
	m.state = stateRemoveRoot
	return m, nil
}

func (m model) startMoveProject() model {
	sel := m.list.SelectedItem()
	var proj core.Project
	switch s := sel.(type) {
	case item:
		proj = s.project
	case flatItem:
		proj = s.item.project
	case favoriteItem:
		proj = s.project
	default:
		return m
	}
	m.domainIntent = domainForMove
	m.moveTarget = proj
	roots := m.cfg.ActiveRoots()
	if len(roots) == 1 {
		m.pendingRoot = roots[0]
		return m.openMoveDestDomainPicker()
	}
	m.genericPicker = newRootPicker(
		fmt.Sprintf("Move \"%s\" — destination root:", proj.Name),
		roots, "↑/↓  •  enter  •  esc",
	)
	m.state = stateMovePickRoot
	return m
}

func (m model) startCloneFlow(repo gh.Repo) model {
	m.domainIntent = domainForClone
	m.pendingGHRepo = repo
	m.pendingLocation = core.Location{}
	roots := m.cfg.ActiveRoots()
	if len(roots) == 1 {
		m.pendingRoot = roots[0]
		return m.openDomainPickerForClone()
	}
	m.genericPicker = newRootPicker(
		fmt.Sprintf("Clone \"%s\" — select root:", repo.Name),
		roots, "↑/↓  •  enter  •  esc",
	)
	m.state = statePickRootForClone
	return m
}

func (m model) startEditor() (tea.Model, tea.Cmd) {
	m.editorScr = newEditorScreen(m.presets, m.templates, m.termW, m.termH)
	m.editorScr.cfg = m.cfg.Clone()
	m.state = stateEditor
	return m, nil
}

func (m model) startDeleteTmuxSession() (tea.Model, tea.Cmd) {
	sel := m.list.SelectedItem()
	ts, ok := sel.(tmuxSessionItem)
	if !ok {
		return m, nil
	}
	m.confirmDelTmux = newConfirmDeleteTmuxScreen(ts.name)
	m.state = stateDeleteTmuxSession
	return m, nil
}
