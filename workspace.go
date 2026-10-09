package main

import (
	"fmt"
	"reflect"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/app"
	gh "github.com/richardnascimento18/devdock/internal/github"
)

type scanResultMsg struct {
	id       uint64
	snapshot app.Snapshot
}

func (m model) rescan() model { m.scan.requested = true; return m }
func (m *model) scanCommand() tea.Cmd {
	m.scan.requested = false
	m.scan.inFlight = true
	m.scan.id++
	id := m.scan.id
	roots := append([]string(nil), m.cfg.ActiveRoots()...)
	repos := append([]gh.Repo(nil), m.githubRepos...)
	return func() tea.Msg {
		return scanResultMsg{id: id, snapshot: m.workspaces.Refresh(m.context(), roots, repos)}
	}
}
func (m model) handleScanResult(msg scanResultMsg) (tea.Model, tea.Cmd) {
	if msg.id != m.scan.id {
		return m, nil
	}
	m.scan.inFlight = false
	m.scan.startup = nil
	m.navigation.projects = msg.snapshot.Projects
	m.navigation.tree = msg.snapshot.Tree
	m.navigation.domains = msg.snapshot.Domains
	availableRoots := make([]string, 0, len(msg.snapshot.Domains))
	for root := range msg.snapshot.Domains {
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
	m.scan.warnings = ""
	m.scan.warningCount = 0
	if msg.snapshot.Err != nil {
		m.scan.warnings = msg.snapshot.Err.Error()
		m.scan.warningCount = warningCount(msg.snapshot.Err)
		m.diagnostic = app.Diagnostic{Severity: app.Warning, Summary: fmt.Sprintf("⚠ Workspace scan completed with %d warnings · ! details", m.scan.warningCount), Cause: msg.snapshot.Err, Operation: "scan", Details: m.scan.warnings}
	} else if m.scan.warnings == "" && m.scan.warningCount == 0 && strings.Contains(m.diagnostic.Summary, "Workspace scan completed with") {
		m.diagnostic = app.Diagnostic{}
	}
	m = m.rebuildList(m.cfg.IsGitHubConnected() && len(m.githubRepos) > 0)
	return m, nil
}

// Count joined scan failures without counting contextual wrappers twice.
func warningCount(err error) int {
	if err == nil {
		return 0
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		count := 0
		for _, child := range joined.Unwrap() {
			count += warningCount(child)
		}
		return count
	}
	return 1
}
