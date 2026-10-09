package main

import (
	"os"
	"syscall"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/app"
	"github.com/richardnascimento18/devdock/internal/core"
)

func TestBulkMovesThroughPortableFilesystem(t *testing.T) {
	root := t.TempDir()
	var projects []core.Project
	for _, name := range []string{"one", "two"} {
		p, err := core.CreateProject(core.Location{Root: root, Domain: "from"}, name)
		if err != nil {
			t.Fatal(err)
		}
		projects = append(projects, p)
	}
	destination := core.Location{Root: root, Domain: "to"}
	rows := app.PreflightMoves(projects, destination, core.PlanMove)
	mover := core.Mover{NoReplace: func(string, string) error { return syscall.EINVAL }}
	results, executed := app.ExecuteMoves(rows, core.PlanMove, mover.Execute)
	if !executed {
		t.Fatal("bulk not executed")
	}
	for i, row := range results {
		if row.Err != nil || row.Project.Path == "" {
			t.Fatalf("row: %+v", row)
		}
		if _, err := os.Stat(row.Project.Path); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(projects[i].Path); !os.IsNotExist(err) {
			t.Fatal("old source still exists")
		}
	}
}

func TestStandaloneMoveCommandThroughPortableFilesystem(t *testing.T) {
	root := t.TempDir()
	m := fixtureModel(t, root)
	p, err := core.CreateProject(core.Location{Root: root, Domain: "from"}, "demo")
	if err != nil {
		t.Fatal(err)
	}
	destination := core.Location{Root: root, Domain: "to"}
	if err := os.MkdirAll(destination.Path(), 0755); err != nil {
		t.Fatal(err)
	}
	m.navigation.projects = []core.Project{p}
	m.moveTarget = p
	m.uiState.ToggleFavorite(p.Path)
	m.uiState.AddRecent(p)
	m.moveFilesystem = core.Mover{NoReplace: func(string, string) error { return syscall.EINVAL }}
	m = m.openMovePlacementPicker(root, "to")
	next, cmd := m.updatePlacement(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.state != stateMovingProject || cmd == nil {
		t.Fatal("move command not created")
	}
	message := cmd().(moveProjectDoneMsg)
	if message.err != nil || message.newProject.Path == "" {
		t.Fatalf("move failed: %v", message.err)
	}
	next, _ = m.handleMoveProjectDone(message)
	m = next.(model)
	if !m.uiState.Favorites[message.newProject.Path] || m.uiState.Favorites[p.Path] || len(m.uiState.Recents) != 1 || m.uiState.Recents[0].Path != message.newProject.Path {
		t.Fatal("move failed to reconcile favorite/recent")
	}
}
