package main

import (
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/state"
)

func deepFixture(t *testing.T) (string, core.Location) {
	t.Helper()
	root := t.TempDir()
	loc := core.Location{Root: root, Domain: "apps"}
	names := []string{}
	for i := 0; i < 35; i++ {
		names = append(names, fmt.Sprintf("g%d", i))
	}
	if err := core.CreateGroupPath(loc, names); err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		loc = loc.Child(n)
	}
	return root, loc
}
func TestArbitraryDepthDestinationPickerAndCreation(t *testing.T) {
	root, loc := deepFixture(t)
	m := fixtureModel(t, root)
	m = m.openPlacement(root, "apps", placeProject)
	if len(m.movePlacementOpts) != 36 {
		t.Fatalf("picker locations %d", len(m.movePlacementOpts))
	}
	m.creation.name = "api"
	m.genericPicker.cursor = 35
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.state != statePickPreset || m.currentLocation().Key() != loc.Key() {
		t.Fatal("lost deep selection")
	}
	next, cmd := m.finishCreateProject(root, "apps", m.presets[0])
	m = next.(model)
	next, cmd = m.handleCreatePreflight(cmd().(createPreflightMsg))
	m = next.(model)
	next, _ = m.handleCreateDone(cmd().(createDoneMsg))
	m = next.(model)
	if m.launch.project.Path != filepath.Join(loc.Path(), "api") || !m.launch.ready {
		t.Fatalf("create %+v", m.launch.project)
	}
	if _, err := os.Stat(m.launch.project.Path); err != nil {
		t.Fatal(err)
	}
	// Nested clone uses the same selected location; don't execute network work.
	m = fixtureModel(t, root)
	m.creation.repo = gh.Repo{Name: "repo", CloneURL: "unused"}
	m = m.openPlacement(root, "apps", placeClone)
	m.genericPicker.cursor = 35
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.state != stateCloningRepo || cmd == nil || m.currentLocation().Key() != loc.Key() {
		t.Fatal("clone lost ancestry")
	}
}
func TestDeepGroupDeleteSelectsExactLocation(t *testing.T) {
	root, loc := deepFixture(t)
	sibling := core.Location{Root: root, Domain: "apps", GroupPath: []string{"other"}}
	if err := core.CreateGroupPath(sibling, []string{"g34"}); err != nil {
		t.Fatal(err)
	}
	m := fixtureModel(t, root)
	next, _ := m.startDeleteGroup()
	m = next.(model)
	idx := -1
	for i, l := range m.groupFlow.deleteLocations {
		if l.Key() == loc.Key() {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatal("deep group unavailable")
	}
	m.genericPicker.cursor = idx
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.groupFlow.deletePath != loc.Path() || m.state != stateConfirmDeleteGroup {
		t.Fatal("wrong group")
	}
	m.inputScr.input.SetValue(loc.Path())
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	next, _ = m.Update(operationMessage(cmd))
	m = next.(model)
	if _, err := os.Stat(loc.Path()); !os.IsNotExist(err) {
		t.Fatal("group remains")
	}
	if _, err := os.Stat(sibling.Child("g34").Path()); err != nil {
		t.Fatal("same name sibling deleted")
	}
}
func TestTreeStateRenderingAndNoFilesystemReads(t *testing.T) {
	root, loc := deepFixture(t)
	p, err := core.CreateProject(loc, "selected-api")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.Path, "go.mod"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	m := fixtureModel(t, root)
	// Rename root out of the way: all view/collapse/picker work must use snapshots.
	movedRoot := root + "-away"
	if err := os.Rename(root, movedRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Rename(movedRoot, root); err != nil {
			t.Error(err)
		}
	})
	m = m.rebuildList(false)
	m = m.openMovePlacementPicker(root, "apps")
	if len(m.movePlacementOpts) != 36 || m.scan.requested {
		t.Fatal("picker triggered scan")
	}
	m.state = stateList
	m = m.rebuildList(false)
	// Hierarchy now lives in Workspace. Collapsing it retains the project action
	// list and selected project; the complete loaded hierarchy is still available.
	deepest := loc.Key()
	foundNode := false
	for _, row := range m.navigation.rows {
		if row.node != nil && row.node.Key() == deepest {
			foundNode = true
		}
	}
	if !foundNode {
		t.Fatal("deep workspace node missing")
	}
	m.collapsedNodes[deepest] = true
	m = m.rebuildList(false)
	projectVisible := false
	for _, it := range m.list.Items() {
		if rowIdentity(it) == p.Path {
			projectVisible = true
		}
	}
	if !projectVisible {
		t.Fatal("workspace collapse hid project action list")
	}
	m.collapsedNodes[deepest] = false
	m = m.rebuildList(false)
	found := false
	for i, it := range m.list.Items() {
		if rowIdentity(it) == p.Path {
			found = true
			m.list.Select(i)
			title := it.(item).Title()
			if !strings.Contains(title, "selected-api") || lipgloss.Width(visualIndent(35)) > 12 {
				t.Fatal("deep project unreadable")
			}
		}
	}
	if !found {
		t.Fatal("missing deep row")
	}
	m = m.rebuildList(false)
	if rowIdentity(m.list.SelectedItem()) != p.Path {
		t.Fatal("selection lost")
	}
}
func TestMoveStateWriteFailureRetainsNewLocationForRetry(t *testing.T) {
	root, loc := deepFixture(t)
	p, err := core.CreateProject(core.Location{Root: root, Domain: "apps"}, "api")
	if err != nil {
		t.Fatal(err)
	}
	m := fixtureModel(t, root)
	m.uiState.ToggleFavorite(p.Path)
	m.uiState.AddRecent(p)
	if !m.saveState() {
		t.Fatal("save")
	}
	moved, err := core.MoveProject(p, loc, "api")
	if err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(config.Dir(), "state.json")
	if err := os.Remove(statePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(statePath, 0o755); err != nil {
		t.Fatal(err)
	}
	m.state = stateMovingProject
	m.moveTarget = p
	next, _ := m.handleMoveProjectDone(moveProjectDoneMsg{newProject: moved})
	m = next.(model)
	if !m.uiState.Favorites[moved.Path] || m.uiState.Favorites[p.Path] || !m.filesystemStatePending || m.persistenceErr == nil {
		t.Fatal("filesystem state rolled back")
	}
	if err := os.Remove(statePath); err != nil {
		t.Fatal(err)
	}
	if !m.saveState() {
		t.Fatal("retry")
	}
	saved, err := state.Load(config.Dir())
	if err != nil || !saved.Favorites[moved.Path] || saved.Recents[0].Path != moved.Path {
		t.Fatal("retry lost state")
	}
}

func TestLongConfirmationAndScrollingPicker(t *testing.T) {
	path := "/root/apps/" + strings.Repeat("deep/", 60) + "api"
	s := newInputScreen("Delete project", path, "confirm")
	s.input.SetValue(path)
	if s.input.Value() != path {
		t.Fatal("deep confirmation truncated")
	}
	options := make([]string, 100)
	for i := range options {
		options[i] = fmt.Sprintf("ancestor/ancestor/ancestor/ancestor/destination-%d", i)
	}
	picker := newGenericPicker("Destination", options, "navigate")
	picker.cursor = 99
	rendered := picker.View(60, 24)
	if !strings.Contains(rendered, "destination-99") || strings.Contains(rendered, "destination-0") {
		t.Fatal("selected destination offscreen")
	}
}

func TestDeepLabelsRemainDistinctWhenBreadcrumbsTruncate(t *testing.T) {
	a := core.Location{Root: "/workspace", Domain: "apps", GroupPath: []string{"backend"}}
	b := core.Location{Root: "/workspace", Domain: "apps", GroupPath: []string{"frontend"}}
	for i := 0; i < 40; i++ {
		a = a.Child("shared")
		b = b.Child("shared")
	}
	labels := []string{a.Breadcrumb(), b.Breadcrumb()}
	picker := newLocationPicker("Location", []core.Location{a, b}, labels, "select")
	rendered := picker.View(60, 30)
	if core.PathID(a.Path()) == core.PathID(b.Path()) || !strings.Contains(rendered, core.PathID(a.Path())) || !strings.Contains(rendered, core.PathID(b.Path())) {
		t.Fatal("truncation aliased locations")
	}
	rootPicker := newRootPicker("Root", []string{"/work/projects", "/personal/projects"}, "select")
	rendered = rootPicker.View(80, 30)
	if !strings.Contains(rendered, "/work/projects") || !strings.Contains(rendered, "/personal/projects") {
		t.Fatal("root basename alias")
	}
}
func TestLongConfirmationDisplaysWholeTarget(t *testing.T) {
	path := "/workspace/apps/" + strings.Repeat("shared/", 50) + "distinct-group/api"
	screen := newInputScreen("Delete", path, "type full path")
	rendered := screen.View(80, 40)
	flattened := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '│' {
			return -1
		}
		return r
	}, ansi.Strip(rendered))
	if !strings.Contains(rendered, core.PathID(path)) || !strings.Contains(flattened, path) {
		t.Fatal("target text lost during wrapping")
	}
	screen.input.SetValue(path)
	if screen.input.Value() != path {
		t.Fatal("confirmation truncated")
	}
}
