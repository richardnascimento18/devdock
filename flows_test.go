package main

import (
	"errors"
	"github.com/richardnascimento18/devdock/internal/pty"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/preset"
	"github.com/richardnascimento18/devdock/internal/state"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
)

func fixtureModel(t *testing.T, roots ...string) model {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	ui, _ := state.Load(config.Dir())
	snapshot := scanWorkspace(roots, nil)
	if snapshot.err != nil {
		t.Fatal(snapshot.err)
	}
	m := newModel(snapshot.projects, config.Config{Roots: roots}, preset.DefaultPresets, tmpl.DefaultTemplates, ui)
	m.workspaceDomains = snapshot.domains
	m.workspaceGroups = snapshot.groups
	return m.rebuildList(false)
}
func TestMultiRootGroupDeleteRequiresFullPath(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	if err := core.CreateGroup(b, "apps", "team"); err != nil {
		t.Fatal(err)
	}
	project, err := core.CreateProject(b, "apps", "other")
	if err != nil {
		t.Fatal(err)
	}
	m := fixtureModel(t, a, b)
	next, _ := m.startDeleteGroup()
	m = next.(model)
	m.genericPicker.cursor = 1
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.state != stateDeleteGroup {
		t.Fatalf("wrong root intent state %v", m.state)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	path := filepath.Join(b, "apps", "team")
	if m.state != stateConfirmDeleteGroup {
		t.Fatal("missing confirmation")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("single Enter deleted group")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if _, err := os.Stat(path); err != nil {
		t.Fatal("empty confirmation deleted group")
	}
	m.inputScr.input.SetValue(path)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.state != stateDeletingWorkspace {
		t.Fatal("delete did not enter async state")
	}
	commands := cmd().(tea.BatchMsg)
	next, _ = m.Update(commands[0]())
	m = next.(model)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("confirmed group remains")
	}
	if _, err := os.Stat(project.Path); err != nil {
		t.Fatal("deleted unrelated project")
	}
}
func TestGroupCreateResetsPreviousDomainAndSentinelNamesAreNames(t *testing.T) {
	root := t.TempDir()
	if err := core.CreateDomain(root, "apps"); err != nil {
		t.Fatal(err)
	}
	m := fixtureModel(t, root)
	m.pendingDomain = "stale"
	next, _ := m.startCreateGroup()
	m = next.(model)
	if m.groupFlow.phase != groupPickDomain {
		t.Fatal("reused previous domain")
	}
	for _, name := range []string{"__creategroup__", "__deletegroup__", "__move__"} {
		m = fixtureModel(t, root, t.TempDir())
		next, _ = m.startNewProject(name)
		m = next.(model)
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = next.(model)
		if m.state != statePickDomain || m.pendingProjectName != name {
			t.Fatalf("name %s treated as sentinel", name)
		}
	}
}
func TestCloneNewDomainRetainsIntent(t *testing.T) {
	root := t.TempDir()
	m := fixtureModel(t, root)
	m = m.startCloneFlow(gh.Repo{Name: "repo", CloneURL: "unused"})
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.state != stateNewDomainName {
		t.Fatal("expected new domain input")
	}
	m.inputScr.input.SetValue("apps")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.state != stateCloningRepo || cmd == nil {
		t.Fatal("clone routed into creation")
	}
	// Do not execute the network clone command.
}
func TestSpinnerLifecycleAndEmptyRoots(t *testing.T) {
	m := model{state: stateList, spinnerID: 1, spinnerScr: newSpinnerScreen("loading")}
	_, cmd := m.Update(spinnerTickMsg{id: 1})
	if cmd != nil {
		t.Fatal("spinner continued after loading")
	}
	m.state = stateMovingProject
	_, cmd = m.Update(spinnerTickMsg{id: 0})
	if cmd != nil {
		t.Fatal("stale spinner tick continued")
	}
	_, cmd = m.Update(spinnerTickMsg{id: 1})
	if cmd == nil {
		t.Fatal("active spinner stopped")
	}
	m = fixtureModel(t)
	m.state = statePickRoot
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if next.(model).state != stateList {
		t.Fatal("empty root picker remains active")
	}
}
func TestPickerCollisionsIndependentOfPresentation(t *testing.T) {
	root := t.TempDir()
	p, err := core.CreateProject(root, "apps", "demo")
	if err != nil {
		t.Fatal(err)
	}
	m := fixtureModel(t, root)
	m.rawProjects = []core.Project{p}
	m.allItems = nil
	m.treeMode = false
	next, _ := m.openDomainPicker(root, "demo")
	if !next.domainPicker.disabled["apps"] {
		t.Fatal("flat/collapsed picker missed collision")
	}
}
func TestFavoritesAndRecentsRespectRootWithoutPruning(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	p, err := core.CreateProject(a, "apps", "a")
	if err != nil {
		t.Fatal(err)
	}
	q, err := core.CreateProject(b, "apps", "b")
	if err != nil {
		t.Fatal(err)
	}
	m := fixtureModel(t, a, b)
	m.rawProjects = []core.Project{p, q}
	m.uiState.ToggleFavorite(p.Path)
	m.uiState.ToggleFavorite(q.Path)
	m.uiState.AddRecent(p)
	m.uiState.AddRecent(q)
	m.rootSel.cursor = 1
	m.activeTab = TabFavorites
	m = m.refreshTabList()
	if len(m.list.Items()) != 1 || len(m.uiState.Favorites) != 2 {
		t.Fatal("favorite root filtering")
	}
	m.activeTab = TabRecents
	m = m.refreshTabList()
	if len(m.list.Items()) != 1 || len(m.uiState.Recents) != 2 {
		t.Fatal("recent root filtering")
	}
}
func TestResizeAndStaleScanResults(t *testing.T) {
	m := fixtureModel(t, t.TempDir())
	m.state = stateEditor
	m.editorScr = newEditorScreen(m.presets, m.templates, 100, 40)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 70, Height: 20})
	m = next.(model)
	if m.editorScr.termW != 70 || m.editorScr.termH != 20 {
		t.Fatal("resize swallowed")
	}
	m.scanID = 2
	before := len(m.rawProjects)
	next, _ = m.Update(scanResultMsg{id: 1, snapshot: workspaceSnapshot{projects: []core.Project{{Name: "stale"}}}})
	if len(next.(model).rawProjects) != before {
		t.Fatal("stale scan applied")
	}
}

func TestDomainCreationRefreshesSnapshot(t *testing.T) {
	root := t.TempDir()
	m := fixtureModel(t, root)
	next, _ := m.startNewDomainOnly()
	m = next.(model)
	m.inputScr.input.SetValue("new")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd == nil {
		t.Fatal("domain creation did not schedule rescan")
	}
	next, _ = m.Update(cmd())
	m = next.(model)
	if len(m.workspaceDomains[root]) != 1 || m.workspaceDomains[root][0] != "new" {
		t.Fatal("new domain absent from picker snapshot")
	}
}

func TestFailedStateAndRootSavesKeepLiveState(t *testing.T) {
	root := t.TempDir()
	m := fixtureModel(t, root)
	if !m.saveState() {
		t.Fatal("initial state save")
	}
	path := filepath.Join(config.Dir(), "state.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	m.uiState.ToggleFavorite(filepath.Join(root, "candidate"))
	if m.saveState() || len(m.uiState.Favorites) != 0 {
		t.Fatal("failed state save committed favorite")
	}
	if err := config.Save(m.cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(config.Path()); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(config.Path(), 0o755); err != nil {
		t.Fatal(err)
	}
	m.state = stateAddRoot
	m.inputScr = newInputScreen("", "", "")
	m.inputScr.input.SetValue(t.TempDir())
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if len(m.cfg.Roots) != 1 || m.inputScr.err == "" {
		t.Fatal("failed root save committed root")
	}
}
func TestSnapshotKeepsStateForUnavailableRoots(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(t.TempDir(), "missing")
	m := fixtureModel(t, root)
	m.cfg.Roots = append(m.cfg.Roots, missing)
	m.uiState.ToggleFavorite(filepath.Join(missing, "apps", "demo"))
	snapshot := scanWorkspace(m.cfg.Roots, nil)
	if snapshot.err == nil {
		t.Fatal("scan failure suppressed")
	}
	next, _ := m.handleScanResult(scanResultMsg{id: m.scanID, snapshot: snapshot})
	m = next.(model)
	if !m.uiState.Favorites[filepath.Join(missing, "apps", "demo")] {
		t.Fatal("unavailable root state pruned")
	}
}
func TestMainNavigationAndScreensRender(t *testing.T) {
	root := t.TempDir()
	if _, err := core.CreateProject(root, "apps", "demo"); err != nil {
		t.Fatal(err)
	}
	m := fixtureModel(t, root)
	for _, key := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune("v")}, {Type: tea.KeyTab}, {Type: tea.KeyRunes, Runes: []rune("f")}, {Type: tea.KeyRunes, Runes: []rune("e")}, {Type: tea.KeyEsc}} {
		next, _ := m.Update(key)
		m = next.(model)
		if m.View() == "" {
			t.Fatal("empty screen")
		}
	}
	if m.state != stateList {
		t.Fatal("editor did not return to list")
	}
}

func TestTemplateFailureNeverQueuesSuccessfulLaunch(t *testing.T) {
	root := t.TempDir()
	m := fixtureModel(t, root)
	m.state = statePTYExecution
	m.ptyScr = newPTYScreen(100, 40, nil, root, tmpl.Vars{ProjectPath: root}, nil, root, gh.Repo{})
	next, cmd := m.Update(pty.ExitMsg{Err: errors.New("exit status 7")})
	m = next.(model)
	if m.pendingLaunchReady || m.state != stateList || cmd != nil || !strings.Contains(m.statusMsg, "failed") {
		t.Fatal("template failure shown as success")
	}
}
func TestSuccessfulMoveReconcilesFavoritesAndRecents(t *testing.T) {
	root := t.TempDir()
	p, err := core.CreateProject(root, "apps", "demo")
	if err != nil {
		t.Fatal(err)
	}
	m := fixtureModel(t, root)
	m.uiState.ToggleFavorite(p.Path)
	m.uiState.AddRecent(p)
	if !m.saveState() {
		t.Fatal("save")
	}
	moved, err := core.MoveProject(p, filepath.Join(root, "tools", "demo"), root, "tools")
	if err != nil {
		t.Fatal(err)
	}
	m.moveTarget = p
	m.state = stateMovingProject
	next, _ := m.Update(moveProjectDoneMsg{newProject: moved})
	m = next.(model)
	if m.uiState.Favorites[p.Path] || !m.uiState.Favorites[moved.Path] || m.uiState.Recents[0].Path != moved.Path {
		t.Fatal("move state stale")
	}
	loaded, err := state.Load(config.Dir())
	if err != nil || !loaded.Favorites[moved.Path] {
		t.Fatal("move state not persisted")
	}
}
