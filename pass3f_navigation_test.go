package main

import (
	"errors"
	"github.com/richardnascimento18/devdock/internal/app"
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
		if m.navigation.focus != test.pane {
			t.Fatalf("alt+%s: %v", test.key, m.navigation.focus)
		}
	}
	for _, pane := range []ui.Pane{ui.Projects, ui.Inspector, ui.Workspace} {
		m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyTab})
		if m.navigation.focus != pane {
			t.Fatalf("tab: %v", m.navigation.focus)
		}
	}
	m = dashboardKey(t, m, keyRune("/query"))
	m = dashboardKey(t, m, altKey("1"))
	if m.query.editing || m.navigation.focus != ui.Workspace || strings.Contains(m.query.input.Value(), "1") {
		t.Fatal("pane shortcut leaked into search")
	}
}
func TestScopeShortcutsSearchAndSelection(t *testing.T) {
	m := renderFixture()
	loc := m.navigation.projects[0].Location
	m = m.switchScope(&loc)
	m.selected = map[string]bool{m.navigation.projects[0].Path: true}
	m.query.value = "billing"
	m.isFiltered = true
	m = m.applySearch()
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.navigation.scope == nil || m.navigation.scope.Domain != "backend" || len(m.navigation.scope.GroupPath) != 0 || m.navigation.focus != ui.Projects {
		t.Fatal("parent group")
	}
	if m.query.value != "billing" || len(m.selected) != 0 {
		t.Fatal("scope lost query or retained old selections")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.navigation.scope == nil || m.navigation.scope.Domain != "" || m.activeRoot() != loc.Root {
		t.Fatal("root shortcut")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlA})
	if m.navigation.scope != nil || m.activeRoot() != "" || m.query.value != "billing" {
		t.Fatal("all shortcut")
	}
}
func TestPaletteFuzzyScopeSelectionAndReturnFocus(t *testing.T) {
	m := renderFixture()
	m.navigation.focus = ui.Workspace
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlP})
	m = dashboardKey(t, m, keyRune("scope services"))
	if len(m.palette.visible) == 0 {
		t.Fatal("scope not discoverable")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != stateList || m.navigation.focus != ui.Projects || m.navigation.scope == nil || m.navigation.scope.Key() != m.navigation.projects[0].Location.Key() || len(m.list.Items()) != 1 {
		t.Fatalf("scope selection: %+v", m.navigation.scope)
	}
}
func TestConciseScanWarningsKeepFullDetails(t *testing.T) {
	m := renderFixture()
	problem := errors.Join(errors.New("inspect /mounted/one: invalid TOML"), errors.New("inspect /mounted/two: unavailable"))
	next, _ := m.handleScanResult(scanResultMsg{id: m.scan.id, snapshot: app.Snapshot{Tree: m.navigation.tree, Projects: m.navigation.projects, Domains: m.navigation.domains, Err: problem}})
	m = next.(model)
	if m.scan.warningCount != 2 || !strings.Contains(m.diagnostic.Summary, "2 warnings") || strings.Contains(m.diagnostic.Summary, "/mounted/") {
		t.Fatal("verbose main warning")
	}
	m = dashboardKey(t, m, keyRune("!"))
	if !strings.Contains(m.statusDetails, problem.Error()) {
		t.Fatal("details lost")
	}
}
func TestHumanTmuxIdentityKeepsExactBackend(t *testing.T) {
	m := renderFixture()
	p := m.navigation.projects[2]
	session := tmux.SessionName(p)
	row := m.sessionItem(session)
	if row.displayName() != p.Name || row.confirmation != p.Name || row.name != session {
		t.Fatalf("identity %+v", row)
	}
	duplicate := m.sessionItem(tmux.SessionName(m.navigation.projects[0]))
	if duplicate.confirmation == duplicate.name || duplicate.confirmation == m.navigation.projects[0].Name {
		t.Fatal("duplicate confirmation ambiguous")
	}
	m.tmux.sessions = []string{session}
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
	if cmd == nil || !next.(model).tmux.deleting || next.(model).confirmDelTmux.sessionName != session {
		t.Fatal("kill lost exact backend target")
	}
}
func TestProjectMetadataPrioritizedAtResponsiveWidths(t *testing.T) {
	m := renderFixture()
	p := m.navigation.projects[0]
	p.Languages = []string{"Express"}
	p.LocalGit = true
	m.navigation.projects[0] = p
	m.tmux.cached = map[string]bool{tmux.SessionName(p): true}
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
	m.scan.inFlight = true
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
