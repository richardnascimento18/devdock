package main

import (
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/detect"
	gh "github.com/richardnascimento18/devdock/internal/github"
)

type workspaceSnapshot struct {
	projects []core.Project
	tree     core.Workspace
	domains  map[string][]string
	err      error
}
type scanResultMsg struct {
	id       uint64
	snapshot workspaceSnapshot
}

func scanWorkspace(roots []string, repos []gh.Repo) workspaceSnapshot {
	detector := detect.New()
	s := workspaceSnapshot{domains: map[string][]string{}}
	s.tree, s.err = core.ScanWorkspace(roots, detector.Inspect)
	s.projects = s.tree.Projects
	for _, root := range s.tree.Roots {
		s.domains[root.Root] = s.tree.Domains(root.Root)
	}
	if len(repos) > 0 {
		s.projects = gh.LinkProjectsToRepos(s.projects, repos)
	}
	return s
}
func (m model) rescan() model { m.scanRequested = true; return m }
func (m *model) scanCommand() tea.Cmd {
	m.scanRequested = false
	m.scanInFlight = true
	m.scanID++
	id := m.scanID
	roots := append([]string(nil), m.cfg.ActiveRoots()...)
	repos := append([]gh.Repo(nil), m.githubRepos...)
	return func() tea.Msg { return scanResultMsg{id: id, snapshot: scanWorkspace(roots, repos)} }
}
func (m model) handleScanResult(msg scanResultMsg) (tea.Model, tea.Cmd) {
	if msg.id != m.scanID {
		return m, nil
	}
	m.scanInFlight = false
	m.startupCmd = nil
	m.rawProjects = msg.snapshot.projects
	m.workspaceTree = msg.snapshot.tree
	m.workspaceDomains = msg.snapshot.domains
	availableRoots := make([]string, 0, len(msg.snapshot.domains))
	for root := range msg.snapshot.domains {
		availableRoots = append(availableRoots, root)
	}
	if len(availableRoots) > 0 {
		before := m.uiState.Clone()
		m.uiState.ReconcileMissing(availableRoots...)
		if !reflect.DeepEqual(before, m.uiState) {
			m.saveState()
		}
	}
	m.isFiltered = false
	if msg.snapshot.err != nil {
		m.statusMsg = errorStyle.Render("workspace scan incomplete: " + msg.snapshot.err.Error())
	}
	m = m.rebuildList(m.cfg.IsGitHubConnected() && len(m.githubRepos) > 0)
	return m, nil
}
