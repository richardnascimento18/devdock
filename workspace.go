package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/detect"
	gh "github.com/richardnascimento18/devdock/internal/github"
)

type workspaceSnapshot struct {
	projects []core.Project
	groups   map[string][]core.GroupInfo
	domains  map[string][]string
	err      error
}
type scanResultMsg struct {
	id       uint64
	snapshot workspaceSnapshot
}

func scanWorkspace(roots []string, repos []gh.Repo) workspaceSnapshot {
	detector := detect.New()
	s := workspaceSnapshot{groups: map[string][]core.GroupInfo{}, domains: map[string][]string{}}
	var failures []error
	s.projects, s.err = core.ScanRoots(roots, detector.CollectProjects)
	failures = append(failures, s.err)
	for _, root := range roots {
		domains, err := core.ScanDomainsInRoot(root)
		if err != nil {
			failures = append(failures, fmt.Errorf("scan domains in %q: %w", root, err))
			continue
		}
		s.domains[root] = domains
		for _, domain := range domains {
			path := filepath.Join(root, domain)
			groups, err := core.ScanGroupsInDomain(path, detector.ClassifyDir)
			s.groups[path] = groups
			if err != nil {
				failures = append(failures, fmt.Errorf("scan groups in %q: %w", path, err))
			}
		}
	}
	if len(repos) > 0 {
		s.projects = gh.LinkProjectsToRepos(s.projects, repos)
	}
	s.err = errors.Join(failures...)
	return s
}
func (m model) rescan() model { m.scanRequested = true; return m }
func (m *model) scanCommand() tea.Cmd {
	m.scanRequested = false
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
	m.rawProjects = msg.snapshot.projects
	m.workspaceGroups = msg.snapshot.groups
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
