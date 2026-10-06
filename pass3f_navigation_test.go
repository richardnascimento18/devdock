package main

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/tmux"
	"github.com/richardnascimento18/devdock/internal/ui"
)

func altKey(value string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value), Alt: true}
}
func TestDirectFocusAndTabCycle(t *testing.T) {
	m := renderFixture()
	for _, test := range []struct {
		key  string
		pane ui.Pane
	}{{"1", ui.Workspace}, {"3", ui.Inspector}, {"2", ui.Projects}, {"1", ui.Workspace}} {
		m = dashboardKey(t, m, altKey(test.key))
		if m.focus != test.pane {
			t.Fatalf("alt+%s: %v", test.key, m.focus)
		}
	}
	for _, pane := range []ui.Pane{ui.Projects, ui.Inspector, ui.Workspace} {
		m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyTab})
		if m.focus != pane {
			t.Fatalf("tab: %v", m.focus)
		}
	}
	m = dashboardKey(t, m, keyRune("/query"))
	m = dashboardKey(t, m, altKey("1"))
	if m.searching || m.focus != ui.Workspace || strings.Contains(m.searchInput.Value(), "1") {
		t.Fatal("pane shortcut leaked into search")
	}
}
func TestScopeShortcutsSearchAndSelection(t *testing.T) {
	m := renderFixture()
	loc := m.rawProjects[0].Location
	m = m.switchScope(&loc)
	m.selected = map[string]bool{m.rawProjects[0].Path: true}
	m.lastFilter = "billing"
	m.isFiltered = true
	m = m.applySearch()
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.workspaceScope == nil || m.workspaceScope.Domain != "backend" || len(m.workspaceScope.GroupPath) != 0 || m.focus != ui.Projects {
		t.Fatal("parent group")
	}
	if m.lastFilter != "billing" || len(m.selected) != 0 {
		t.Fatal("scope lost query or retained old selections")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.workspaceScope == nil || m.workspaceScope.Domain != "" || m.activeRoot() != loc.Root {
		t.Fatal("root shortcut")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlA})
	if m.workspaceScope != nil || m.activeRoot() != "" || m.lastFilter != "billing" {
		t.Fatal("all shortcut")
	}
}
func TestPaletteFuzzyScopeSelectionAndReturnFocus(t *testing.T) {
	m := renderFixture()
	m.focus = ui.Workspace
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlP})
	m = dashboardKey(t, m, keyRune("scope services"))
	if len(m.palette.visible) == 0 {
		t.Fatal("scope not discoverable")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != stateList || m.focus != ui.Projects || m.workspaceScope == nil || m.workspaceScope.Key() != m.rawProjects[0].Location.Key() || len(m.list.Items()) != 1 {
		t.Fatalf("scope selection: %+v", m.workspaceScope)
	}
}
func TestConciseScanWarningsKeepFullDetails(t *testing.T) {
	m := renderFixture()
	problem := errors.Join(errors.New("inspect /mounted/one: invalid TOML"), errors.New("inspect /mounted/two: unavailable"))
	next, _ := m.handleScanResult(scanResultMsg{id: m.scanID, snapshot: workspaceSnapshot{tree: m.workspaceTree, projects: m.rawProjects, domains: m.workspaceDomains, err: problem}})
	m = next.(model)
	if m.scanWarningCount != 2 || !strings.Contains(ansi.Strip(m.statusMsg), "2 warnings") || strings.Contains(m.statusMsg, "/mounted/") {
		t.Fatal("verbose main warning")
	}
	m = dashboardKey(t, m, keyRune("!"))
	if !strings.Contains(m.statusDetails, problem.Error()) {
		t.Fatal("details lost")
	}
}
func TestHumanTmuxIdentityKeepsExactBackend(t *testing.T) {
	m := renderFixture()
	p := m.rawProjects[2]
	session := tmux.SessionName(p)
	row := m.sessionItem(session)
	if row.displayName() != p.Name || row.confirmation != p.Name || row.name != session {
		t.Fatalf("identity %+v", row)
	}
	duplicate := m.sessionItem(tmux.SessionName(m.rawProjects[0]))
	if duplicate.confirmation == duplicate.name || duplicate.confirmation == m.rawProjects[0].Name {
		t.Fatal("duplicate confirmation ambiguous")
	}
	m.tmuxSessions = []string{session}
	m.activeTab = TabTmux
	m = m.refreshTabList()
	next, _ := m.startDeleteTmuxSession()
	m = next.(model)
	if m.confirmDelTmux.target() != p.Name || strings.Contains(ansi.Strip(m.View()), session) {
		t.Fatal("hashed primary confirmation")
	}
	m.confirmDelTmux.input.SetValue(session)
	next, cmd := m.updateDeleteTmuxSession(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd != nil || m.confirmDelTmux.err == "" {
		t.Fatal("accepted internal name instead of human target")
	}
	m.confirmDelTmux.input.SetValue(p.Name)
	next, cmd = m.updateDeleteTmuxSession(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || !next.(model).tmuxDeleting || next.(model).confirmDelTmux.sessionName != session {
		t.Fatal("kill lost exact backend target")
	}
}
func TestProjectMetadataPrioritizedAtResponsiveWidths(t *testing.T) {
	m := renderFixture()
	p := m.rawProjects[0]
	p.Languages = []string{"Express"}
	p.LocalGit = true
	m.rawProjects[0] = p
	m.cachedTmux = map[string]bool{tmux.SessionName(p): true}
	for _, width := range []int{120, 80, 40} {
		m.termW = width
		m = m.rebuildList(false)
		m.sizePresentation()
		text := ansi.Strip(m.View())
		if !strings.Contains(text, "billing-api") || !strings.Contains(text, "Express") {
			t.Fatalf("missing identifying stack at %d: %s", width, text)
		}
		if width == 120 && !strings.Contains(text, "GitHub") {
			t.Fatal("GitHub metadata hidden")
		}
	}
}
func TestDelayedShimmerAndStaticConfirmations(t *testing.T) {
	m := renderFixture()
	m.scanInFlight = true
	if m.scheduleAnimation() == nil || m.motion.frame != 0 {
		t.Fatal("activity delay not scheduled")
	}
	if !strings.HasPrefix(ansi.Strip(m.activityView()), "… ") {
		t.Fatal("immediate animation")
	}
	next, _ := m.handleAnimationTick(animationTickMsg{generation: m.motion.generation})
	m = next.(model)
	if strings.HasPrefix(ansi.Strip(m.activityView()), "… ") {
		t.Fatal("long activity remains static")
	}
	m.motion.reduced = true
	if !strings.HasPrefix(ansi.Strip(m.activityView()), "… ") {
		t.Fatal("reduced motion animates")
	}
	m.motion.reduced = false
	m.state = stateBulkConfirm
	if m.activityLabel() != "" || m.scheduleAnimation() != nil {
		t.Fatal("confirmation animates")
	}
}
