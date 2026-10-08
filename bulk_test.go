package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/ui"
)

func TestMultiSelectionNavigationFilterScopeAndReconciliation(t *testing.T) {
	m := renderFixture()
	m = dashboardKey(t, m, keyRune(" "))
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = dashboardKey(t, m, keyRune(" "))
	if len(m.selected) != 2 {
		t.Fatal("selection did not persist across navigation")
	}
	m = dashboardKey(t, m, keyRune("/frontend"))
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.hiddenSelection() != 1 || !strings.Contains(ansi.Strip(m.View()), "1 hidden") {
		t.Fatal("hidden selections not explicit")
	}
	m = dashboardKey(t, m, keyRune("m"))
	if m.state != stateList {
		t.Fatal("hidden move allowed")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if len(m.selected) != 2 || m.isFiltered {
		t.Fatal("Escape search priority")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if len(m.selected) != 0 {
		t.Fatal("clear selection")
	}
	m = dashboardKey(t, m, keyRune(" "))
	m = dashboardKey(t, m, keyRune("}"))
	if len(m.selected) != 0 {
		t.Fatal("root retained selection")
	}
	m = renderFixture()
	m = dashboardKey(t, m, keyRune(" "))
	m.navigation.focus = ui.Workspace
	m.navigation.cursor = 1
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.selected) != 0 {
		t.Fatal("scope retained selection")
	}
	m = renderFixture()
	m = dashboardKey(t, m, keyRune(" "))
	m.navigation.projects = nil
	m = m.rebuildList(false)
	if len(m.selected) != 0 {
		t.Fatal("stale selection")
	}
}
func TestBulkFavoritesAndDestructiveGuard(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m := renderFixture()
	m.selected = map[string]bool{m.navigation.projects[0].Path: true, m.navigation.projects[1].Path: true}
	m = dashboardKey(t, m, keyRune("x"))
	if m.state != stateList {
		t.Fatal("delete accepted while selected")
	}
	m = dashboardKey(t, m, keyRune("f"))
	for _, p := range m.selectedProjects() {
		if !m.uiState.Favorites[p.Path] {
			t.Fatal("favorite all")
		}
	}
	m = dashboardKey(t, m, keyRune("f"))
	for _, p := range m.selectedProjects() {
		if m.uiState.Favorites[p.Path] {
			t.Fatal("unfavorite all")
		}
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyCtrlP})
	m = dashboardKey(t, m, keyRune("move selected"))
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != stateMovePickDomain {
		t.Fatal("bulk palette intent")
	}
}
func bulkFixture(t *testing.T) ([]core.Project, core.Location) {
	t.Helper()
	root := t.TempDir()
	dest := core.Location{Root: root, Domain: "destination"}
	if err := os.MkdirAll(dest.Path(), 0755); err != nil {
		t.Fatal(err)
	}
	var projects []core.Project
	for i := 0; i < 2; i++ {
		loc := core.Location{Root: root, Domain: "source"}
		p, err := core.CreateProject(loc, fmt.Sprintf("api-%d", i))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p.Path, "go.mod"), []byte("module demo\n"), 0644); err != nil {
			t.Fatal(err)
		}
		projects = append(projects, p)
	}
	return projects, dest
}
func TestBulkPreflightAllAndChangedDestinationBlocksEveryMove(t *testing.T) {
	projects, dest := bulkFixture(t)
	rows := preflightBulk(projects, dest, core.PlanMove)
	if !bulkValid(rows) {
		t.Fatal(rows)
	}
	if err := os.Mkdir(rows[1].plan.Path, 0755); err != nil {
		t.Fatal(err)
	}
	calls := 0
	checked, executed := executeBulk(rows, core.PlanMove, func(plan core.MovePlan) (core.Project, error) { calls++; return core.ExecuteMove(plan) })
	if executed || calls != 0 || checked[1].err == nil {
		t.Fatal("obvious later collision discovered after starting move")
	}
	for _, p := range projects {
		if _, err := os.Stat(p.Path); err != nil {
			t.Fatal("source changed during blocked preflight")
		}
	}
}
func TestBulkInternalCollisionAndPartialResults(t *testing.T) {
	projects, dest := bulkFixture(t)
	// Two different source locations with the same basename must both be flagged.
	other, err := core.CreateProject(core.Location{Root: dest.Root, Domain: "other"}, filepath.Base(projects[0].Path))
	if err != nil {
		t.Fatal(err)
	}
	blocked := preflightBulk([]core.Project{projects[0], other}, dest, core.PlanMove)
	if bulkValid(blocked) || blocked[0].err == nil || blocked[1].err == nil {
		t.Fatal("internal collision allowed")
	}
	rows := preflightBulk(projects, dest, core.PlanMove)
	calls := 0
	results, executed := executeBulk(rows, core.PlanMove, func(plan core.MovePlan) (core.Project, error) {
		calls++
		if calls == 1 {
			return core.Project{Location: plan.Destination, Name: plan.Source.Name, Path: plan.Path}, &core.PartialMoveError{Source: plan.Source.Path, Destination: plan.Path, Err: errors.New("cleanup blocked")}
		}
		return core.Project{}, errors.New("destination changed")
	})
	if !executed || calls != 2 {
		t.Fatal("partial batch semantics")
	}
	m := renderFixture()
	m.bulk = bulkWorkflow{projects: projects, rows: results, executed: true}
	m.state = stateBulkResult
	v := ansi.Strip(m.View())
	if !strings.Contains(v, "1 cleanup warnings") || !strings.Contains(v, "1 failed") || !strings.Contains(v, "cleanup blocked") {
		t.Fatal("partial results concealed")
	}
}
func TestBulkMoveReconcilesStateAndStaleMessages(t *testing.T) {
	projects, dest := bulkFixture(t)
	m := fixtureModel(t, dest.Root)
	m.selected = map[string]bool{projects[0].Path: true, projects[1].Path: true}
	m.uiState.Favorites[projects[0].Path] = true
	m.uiState.AddRecent(projects[0])
	m.bulk = bulkWorkflow{id: 4, projects: projects, destination: dest}
	m.state = stateBulkMoving
	rows, executed := executeBulk(preflightBulk(projects, dest, core.PlanMove), core.PlanMove, core.ExecuteMove)
	next, _ := m.Update(bulkDoneMsg{3, rows, executed})
	if next.(model).state != stateBulkMoving {
		t.Fatal("stale completion accepted")
	}
	next, _ = m.Update(bulkDoneMsg{4, rows, executed})
	m = next.(model)
	if m.state != stateBulkResult || len(m.selected) != 0 || !m.uiState.Favorites[rows[0].project.Path] || m.uiState.Favorites[projects[0].Path] || m.uiState.Recents[0].Path != rows[0].project.Path {
		t.Fatal("bulk state reconciliation")
	}
	for _, row := range rows {
		if row.err != nil {
			t.Fatal(row.err)
		}
		if _, err := os.Stat(filepath.Join(row.project.Path, "go.mod")); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(row.plan.Source.Path); !os.IsNotExist(err) {
			t.Fatal("source retained")
		}
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != stateList {
		t.Fatal("result back")
	}
}
func TestBulkConfirmationCancelBlockedAndResize(t *testing.T) {
	projects, dest := bulkFixture(t)
	m := renderFixture()
	m.bulk = bulkWorkflow{id: 3, projects: projects, destination: dest, rows: preflightBulk(projects, dest, core.PlanMove)}
	m.bulk.rows[1].err = errors.New("collision")
	m.state = stateBulkConfirm
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || next.(model).state != stateBulkConfirm {
		t.Fatal("conflicted batch executable")
	}
	for _, size := range [][2]int{{120, 40}, {80, 24}, {40, 15}, {24, 8}} {
		next, _ = m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = next.(model)
		if backgroundSequence(m.View()) != "" {
			t.Fatal("opaque bulk modal")
		}
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != stateList || len(m.bulk.projects) != 0 {
		t.Fatal("cancel retained operation")
	}
	next, _ = m.Update(bulkPreflightMsg{3, nil})
	if next.(model).state != stateList {
		t.Fatal("stale preflight reopened modal")
	}
}

func TestBulkPreflightConfirmationExecutionFlow(t *testing.T) {
	projects, dest := bulkFixture(t)
	m := fixtureModel(t, dest.Root)
	m.bulk = bulkWorkflow{projects: projects}
	next, cmd := m.beginBulkPreflight(dest)
	m = next.(model)
	msg := cmd().(bulkPreflightMsg)
	for _, p := range projects {
		if _, err := os.Stat(p.Path); err != nil {
			t.Fatal("preflight mutated source")
		}
	}
	next, _ = m.Update(msg)
	m = next.(model)
	if m.state != stateBulkConfirm || !bulkValid(m.bulk.rows) {
		t.Fatal("confirmation missing")
	}
	next, cmd = m.updateBulk(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.state != stateBulkMoving {
		t.Fatal("confirmation did not execute")
	}
	done := cmd().(bulkDoneMsg)
	next, _ = m.Update(done)
	m = next.(model)
	if m.state != stateBulkResult || !m.bulk.executed {
		t.Fatal("completion did not summarize")
	}
}
