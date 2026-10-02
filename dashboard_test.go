package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/ui"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func keyRune(value string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)} }
func dashboardKey(t *testing.T, m model, key tea.KeyMsg) model {
	t.Helper()
	next, _ := m.Update(key)
	return next.(model)
}

func TestDashboardFocusAndResponsiveComposition(t *testing.T) {
	m := renderFixture()
	for _, expected := range []ui.Pane{ui.Inspector, ui.Workspace, ui.Projects} {
		m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyTab})
		if m.focus != expected {
			t.Fatal("focus cycle")
		}
	}
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 15}} {
		next, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = next.(model)
		v := ansi.Strip(m.View())
		if len(strings.Split(v, "\n")) > size[1] {
			t.Fatalf("overflow height %v", size)
		}
		if size[0] >= 98 && (!strings.Contains(v, "Workspace") || !strings.Contains(v, "Projects") || !strings.Contains(v, "Inspector")) {
			t.Fatalf("three panes %v", size)
		}
		m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyTab})
		if !strings.Contains(ansi.Strip(m.View()), "Inspector") {
			t.Fatalf("inspector inaccessible %v", size)
		}
		m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.focus != ui.Projects {
			t.Fatal("dedicated pane Escape")
		}
	}
}

func TestWorkspaceScopeCollapseAndSnapshotOnlyNavigation(t *testing.T) {
	root, loc := deepFixture(t)
	p, err := core.CreateProject(loc, "deep-api")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.Path, "go.mod"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	m := fixtureModel(t, root)
	m.workspaceSelection = loc.Key()
	m.refreshWorkspaceRows()
	m.focus = ui.Workspace
	away := root + "-away"
	if err := os.Rename(root, away); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Rename(away, root); err != nil {
			t.Error(err)
		}
	})
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.workspaceScope == nil || m.workspaceScope.Key() != loc.Key() || len(m.list.Items()) != 1 {
		t.Fatal("deep scoped projects")
	}
	m.focus = ui.Workspace
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyLeft})
	if !m.collapsedNodes[loc.Key()] || rowIdentity(m.list.SelectedItem()) != p.Path {
		t.Fatal("collapse lost selected project")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyRight})
	if m.collapsedNodes[loc.Key()] {
		t.Fatal("expand")
	}
	if !strings.Contains(ansi.Strip(m.View()), "deep-api") {
		t.Fatal("project not rendered")
	}
	m.focus = ui.Inspector
	m.inspectorScroll = 100
	_ = m.View()
	m.focus = ui.Projects
	m = dashboardKey(t, m, keyRune("/"))
	m = dashboardKey(t, m, keyRune("deep"))
	if !m.searching || len(m.list.Items()) != 1 || m.scanRequested {
		t.Fatal("search requested filesystem refresh")
	}
}

func TestLiveSearchSelectionCancelAndFuzzyLocation(t *testing.T) {
	m := renderFixture()
	m.list.Select(1)
	selected := rowIdentity(m.list.SelectedItem())
	m = dashboardKey(t, m, keyRune("/"))
	m = dashboardKey(t, m, keyRune("billing"))
	if len(m.list.Items()) != 2 || rowIdentity(m.list.SelectedItem()) != selected {
		t.Fatal("search selection not retained")
	}
	m = dashboardKey(t, m, keyRune(" frontend"))
	if len(m.list.Items()) != 1 || rowIdentity(m.list.SelectedItem()) != selected {
		t.Fatal("location disambiguation")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.searching || !m.isFiltered {
		t.Fatal("apply search")
	}
	m = dashboardKey(t, m, keyRune("/"))
	m = dashboardKey(t, m, keyRune("zzz"))
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.lastFilter != "billing frontend" || len(m.list.Items()) != 1 {
		t.Fatal("cancel did not restore committed query")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.lastFilter != "" || len(m.list.Items()) != 3 {
		t.Fatal("clear search")
	}
	m = dashboardKey(t, m, keyRune("/"))
	m = dashboardKey(t, m, keyRune("not-found"))
	if !strings.Contains(ansi.Strip(m.View()), "No projects match") {
		t.Fatal("search empty state")
	}
}

func TestSearchTriggerAndQueryInOneTerminalMessage(t *testing.T) {
	m := renderFixture()
	m = dashboardKey(t, m, keyRune("/billing"))
	if !m.searching || m.lastFilter != "billing" || len(m.list.Items()) != 2 {
		t.Fatal("coalesced search trigger ignored")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.pendingLaunchReady {
		t.Fatal("search apply launched a project")
	}
}

func TestPaletteFilteringExecutionCancellationAndDisabledAction(t *testing.T) {
	m := renderFixture()
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlP})
	m = dashboardKey(t, m, keyRune("settings"))
	if len(m.palette.visible) != 1 {
		t.Fatal("palette filtering")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != stateEditor || m.editorScr.tab != editorTabSettings {
		t.Fatal("palette Settings intent")
	}
	m = renderFixture()
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlP})
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != stateList || m.pendingLaunchReady {
		t.Fatal("palette cancel executed action")
	}
	m.list.SetItems(nil)
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlP})
	m = dashboardKey(t, m, keyRune("delete project"))
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != statePalette || m.palette.actions[m.palette.visible[0]].reason == "" {
		t.Fatal("disabled destructive action")
	}
	if m.scanRequested {
		t.Fatal("palette navigation scanned")
	}
}

func TestInspectorLongUnicodePathIsRecoverable(t *testing.T) {
	m := renderFixture()
	m.termW, m.termH = 40, 15
	m.focus = ui.Inspector
	p := m.rawProjects[0]
	p.Path = "/workspace/" + strings.Repeat("日本語/e\u0301/", 200) + "last-component"
	m.list.SetItems(m.buildListItems([]core.Project{p}, false))
	m.sizePresentation()
	seen := ""
	for i := 0; i < len(m.inspectorLines(40)); i += 5 {
		m.inspectorScroll = i
		seen += strings.Join(strings.Fields(ansi.Strip(m.View())), "")
	}
	if !strings.Contains(seen, "last-component") {
		t.Fatal("full path cannot be recovered by inspector scroll")
	}
}

func TestLongConfirmationPagesAndFooterRemainAccessible(t *testing.T) {
	m := renderFixture()
	m.termW, m.termH = 40, 15
	m.state = stateDeleteProject
	path := "/workspace/" + strings.Repeat("日本語/group/", 100) + "exact-target"
	m.inputScr = newInputScreen("Delete project — type full path", path, "enter confirm · esc cancel")
	first := ansi.Strip(m.View())
	if !strings.Contains(first, "esc cancel") || !strings.Contains(first, "Filesystem target") {
		t.Fatal("confirmation input/footer not visible")
	}
	seen := ""
	for i := 0; i < 70; i++ {
		content := strings.Map(func(r rune) rune {
			if r >= '─' && r <= '╿' {
				return -1
			}
			return r
		}, ansi.Strip(m.View()))
		seen += strings.Join(strings.Fields(content), "")
		m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyPgDown})
	}
	if !strings.Contains(seen, "exact-target") {
		t.Fatalf("confirmation path cannot be recovered: offset %d, final frame:\n%s", m.modalScroll, ansi.Strip(m.View()))
	}
	endOffset := m.modalScroll
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyPgUp})
	if m.modalScroll >= endOffset {
		t.Fatal("PageUp did not immediately leave the final page")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != stateList || m.modalScroll != 0 {
		t.Fatal("confirmation cancel")
	}
}

func BenchmarkDashboardSearch(b *testing.B) {
	for _, count := range []int{1000, 5000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			m := renderFixture()
			seed := m.rawProjects[0]
			m.rawProjects = nil
			for i := 0; i < count; i++ {
				p := seed
				p.Name = fmt.Sprintf("service-%d", i)
				p.Path = seed.Path + fmt.Sprint(i)
				m.rawProjects = append(m.rawProjects, p)
			}
			m = m.rebuildList(false)
			m.searching = true
			m.lastFilter = "srvc10"
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m = m.applySearch()
			}
		})
	}
}
