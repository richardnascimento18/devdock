package main

import (
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/ui"
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

func (m model) rescan() model { m.scanRequested = true; return m }
func (m *model) scanCommand() tea.Cmd {
	m.scanRequested = false
	m.scanInFlight = true
	m.scanID++
	id := m.scanID
	roots := append([]string(nil), m.cfg.ActiveRoots()...)
	repos := append([]gh.Repo(nil), m.githubRepos...)
	return func() tea.Msg {
		return scanResultMsg{id: id, snapshot: m.workspaces.Refresh(m.context(), roots, repos)}
	}
}
func (m model) handleScanResult(msg scanResultMsg) (tea.Model, tea.Cmd) {
	if msg.id != m.scanID {
		return m, nil
	}
	m.scanInFlight = false
	m.startupCmd = nil
	m.rawProjects = msg.snapshot.Projects
	m.workspaceTree = msg.snapshot.Tree
	m.workspaceDomains = msg.snapshot.Domains
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
	m.scanWarnings = ""
	m.scanWarningCount = 0
	if msg.snapshot.Err != nil {
		m.scanWarnings = msg.snapshot.Err.Error()
		m.scanWarningCount = warningCount(msg.snapshot.Err)
		m.statusMsg = warningStyle.Render(ui.SafeBlock(fmt.Sprintf("⚠ Workspace scan completed with %d warnings · ! details", m.scanWarningCount)))
	} else if m.scanWarnings == "" && m.scanWarningCount == 0 && strings.Contains(ansi.Strip(m.statusMsg), "Workspace scan completed with") {
		m.statusMsg = ""
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
