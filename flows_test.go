package main

import (
	"errors"
	"fmt"
	"github.com/richardnascimento18/devdock/internal/app"
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
	if snapshot.Err != nil {
		t.Fatal(snapshot.Err)
	}
	m := newModel(snapshot.Projects, config.Config{Roots: roots}, preset.DefaultPresets, tmpl.DefaultTemplates, ui)
	m.workspaceDomains = snapshot.Domains
	m.workspaceTree = snapshot.Tree
	return m.rebuildList(false)
}
func TestMultiRootGroupDeleteRequiresFullPath(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	if err := core.CreateGroup(core.Location{Root: b, Domain: "apps"}, "team"); err != nil {
		t.Fatal(err)
	}
	project, err := core.CreateProject(core.Location{Root: b, Domain: "apps"}, "other")
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
	next, _ = m.Update(operationMessage(cmd))
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
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.state != stateMovePickPlacement {
		t.Fatal("missing clone placement")
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.state != stateCloningRepo || cmd == nil {
		t.Fatal("clone routed into creation")
	}
	// Do not execute the network clone command.
}
func TestSpinnerLifecycleAndEmptyRoots(t *testing.T) {
	m := model{state: stateList, motion: animationClock{generation: 1, pending: true}, spinnerScr: newSpinnerScreen("loading")}
	_, cmd := m.Update(animationTickMsg{generation: 1})
	if cmd != nil {
		t.Fatal("spinner continued after loading")
	}
	m.state = stateMovingProject
	_, cmd = m.Update(animationTickMsg{generation: 0})
	if cmd != nil {
		t.Fatal("stale spinner tick continued")
	}
	_, cmd = m.Update(animationTickMsg{generation: 1})
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
	p, err := core.CreateProject(core.Location{Root: root, Domain: "apps"}, "demo")
	if err != nil {
		t.Fatal(err)
	}
	m := fixtureModel(t, root)
	m.rawProjects = []core.Project{p}
	m.allItems = nil
	m.treeMode = false
	next, _ := m.openDomainPicker(root, "demo")
	if next.domainPicker.disabled["apps"] {
		t.Fatal("domain incorrectly disabled for all locations")
	}
	next = next.openPlacement(root, "apps", placeProject)
	returned, _ := next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next = returned.(model)
	if next.genericPicker.err == "" || next.state != stateMovePickPlacement {
		t.Fatal("placement missed collision")
	}
}
func TestFavoritesAndRecentsRespectRootWithoutPruning(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	p, err := core.CreateProject(core.Location{Root: a, Domain: "apps"}, "a")
	if err != nil {
		t.Fatal(err)
	}
	q, err := core.CreateProject(core.Location{Root: b, Domain: "apps"}, "b")
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
	next, _ = m.Update(scanResultMsg{id: 1, snapshot: app.Snapshot{Projects: []core.Project{{Name: "stale"}}}})
	if len(next.(model).rawProjects) != before {
		t.Fatal("stale scan applied")
	}
}

func TestDomainCreationRefreshesSnapshot(t *testing.T) {
	for _, reduced := range []bool{false, true} {
		t.Run(fmt.Sprintf("reduced-%v", reduced), func(t *testing.T) {
			root := t.TempDir()
			m := fixtureModel(t, root)
			m.motion.reduced = reduced
			next, _ := m.startNewDomainOnly()
			m = next.(model)
			m.inputScr.input.SetValue("new")
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = next.(model)
			if cmd == nil {
				t.Fatal("domain creation did not schedule rescan")
			}
			if m.motion.pending == reduced {
				t.Fatal("clock does not respect motion mode")
			}
			next, _ = m.Update(operationMessage(cmd))
			m = next.(model)
			if len(m.workspaceDomains[root]) != 1 || m.workspaceDomains[root][0] != "new" {
				t.Fatal("new domain absent from picker snapshot")
			}
		})
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
	if m.persistenceErr == nil {
		t.Fatal("save failure must remain visible after TUI shutdown")
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
	m.uiState.AddRecent(core.Project{Location: core.Location{Root: missing, Domain: "apps"}, Path: filepath.Join(missing, "apps", "demo"), Name: "demo"})
	snapshot := scanWorkspace(m.cfg.Roots, nil)
	if snapshot.Err == nil {
		t.Fatal("scan failure suppressed")
	}
	next, _ := m.handleScanResult(scanResultMsg{id: m.scanID, snapshot: snapshot})
	m = next.(model)
	if !m.uiState.Favorites[filepath.Join(missing, "apps", "demo")] || len(m.uiState.Recents) != 1 {
		t.Fatal("unavailable root state pruned")
	}
}
func TestMainNavigationAndScreensRender(t *testing.T) {
	root := t.TempDir()
	if _, err := core.CreateProject(core.Location{Root: root, Domain: "apps"}, "demo"); err != nil {
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
	p, err := core.CreateProject(core.Location{Root: root, Domain: "apps"}, "demo")
	if err != nil {
		t.Fatal(err)
	}
	m := fixtureModel(t, root)
	m.uiState.ToggleFavorite(p.Path)
	m.uiState.AddRecent(p)
	if !m.saveState() {
		t.Fatal("save")
	}
	moved, err := core.MoveProject(p, core.Location{Root: root, Domain: "tools"}, "demo")
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

func TestProjectAndDomainDeletionConfirmation(t *testing.T) {
	for _, domainDeletion := range []bool{false, true} {
		t.Run(map[bool]string{false: "project", true: "domain"}[domainDeletion], func(t *testing.T) {
			root := t.TempDir()
			project, err := core.CreateProject(core.Location{Root: root, Domain: "apps"}, "demo")
			if err != nil {
				t.Fatal(err)
			}
			m := fixtureModel(t, root)
			m.uiState.ToggleFavorite(project.Path)
			m.uiState.AddRecent(project)
			if !m.saveState() {
				t.Fatal("initial save failed")
			}
			target := project.Path
			m.state = stateDeleteProject
			m.deleteTarget = project
			m.inputScr = newInputScreen("Delete", "path", "")
			if domainDeletion {
				m.state = stateDeleteDomain
				m.inputScr.input.SetValue("apps")
				next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m = next.(model)
				target = filepath.Join(root, "apps")
				m.confirmDelDomain.input.SetValue("apps")
			} else {
				m.inputScr.input.SetValue(project.Name)
			}
			next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = next.(model)
			if _, err := os.Stat(project.Path); err != nil {
				t.Fatal("name-only confirmation deleted project")
			}
			if domainDeletion {
				m.confirmDelDomain.input.SetValue(target)
			} else {
				m.inputScr.input.SetValue(target)
			}
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = next.(model)
			if m.state != stateDeletingWorkspace || cmd == nil {
				t.Fatal("full path did not start deletion")
			}
			next, _ = m.Update(operationMessage(cmd))
			m = next.(model)
			if _, err := os.Stat(target); !os.IsNotExist(err) {
				t.Fatalf("confirmed target remains: %v", err)
			}
			if len(m.uiState.Favorites) != 0 || len(m.uiState.Recents) != 0 {
				t.Fatal("deleted project metadata remains")
			}
		})
	}
}

func TestExplicitRootRemovalPrunesFavoritesAndRecents(t *testing.T) {
	for _, removedCount := range []int{5, 4} {
		t.Run(fmt.Sprintf("%d-removed-favorites", removedCount), func(t *testing.T) {
			base := t.TempDir()
			removed, other := filepath.Join(base, "code"), filepath.Join(base, "code2")
			for _, root := range []string{removed, other} {
				if err := os.Mkdir(root, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			m := fixtureModel(t, removed, other)
			add := func(root, name string) core.Project {
				p, err := core.CreateProject(core.Location{Root: root, Domain: "apps"}, name)
				if err != nil {
					t.Fatal(err)
				}
				if !m.uiState.ToggleFavorite(p.Path) {
					t.Fatal("setup reached favorites limit early")
				}
				m.uiState.AddRecent(p)
				return p
			}
			for i := 0; i < removedCount; i++ {
				add(removed, fmt.Sprintf("demo%d", i))
			}
			var keep core.Project
			if removedCount == 4 {
				keep = add(other, "keep")
			}
			if len(m.uiState.Favorites) != 5 {
				t.Fatal("did not reproduce full favorites map")
			}
			if err := config.Save(m.cfg); err != nil {
				t.Fatal(err)
			}
			if !m.saveState() {
				t.Fatal("setup save failed")
			}
			m.state = stateRemoveRoot
			m.genericPicker.cursor = 0
			next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = next.(model)
			if len(m.cfg.Roots) != 1 || m.cfg.Roots[0] != other {
				t.Fatal("removed wrong root")
			}
			expected := 5 - removedCount
			if len(m.uiState.Favorites) != expected || len(m.uiState.Recents) != expected {
				t.Fatal("removed root state still counts")
			}
			if removedCount == 4 && (!m.uiState.Favorites[keep.Path] || m.uiState.Recents[0].Path != keep.Path) {
				t.Fatal("neighbor prefix root state was removed")
			}
			saved, err := state.Load(config.Dir())
			if err != nil || len(saved.Favorites) != expected || len(saved.Recents) != expected {
				t.Fatal("cleanup was not persisted", err)
			}
			fresh := add(other, "fresh")
			if !m.uiState.Favorites[fresh.Path] {
				t.Fatal("new favorite not accepted")
			}
			if _, err := os.Stat(removed); err != nil {
				t.Fatal("root removal deleted workspace")
			}
		})
	}
}

func TestFailedRootRemovalKeepsStateAndConfiguration(t *testing.T) {
	for _, failedFile := range []string{"config.toml", "state.json"} {
		t.Run(failedFile, func(t *testing.T) {
			root, other := t.TempDir(), t.TempDir()
			m := fixtureModel(t, root, other)
			p, err := core.CreateProject(core.Location{Root: root, Domain: "apps"}, "demo")
			if err != nil {
				t.Fatal(err)
			}
			m.uiState.ToggleFavorite(p.Path)
			m.uiState.AddRecent(p)
			if err := config.Save(m.cfg); err != nil {
				t.Fatal(err)
			}
			if !m.saveState() {
				t.Fatal("setup save failed")
			}
			blocked := filepath.Join(config.Dir(), failedFile)
			if err := os.Remove(blocked); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(blocked, 0o755); err != nil {
				t.Fatal(err)
			}
			m.state = stateRemoveRoot
			m.genericPicker.cursor = 0
			next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = next.(model)
			if len(m.cfg.Roots) != 2 || !m.uiState.Favorites[p.Path] || len(m.uiState.Recents) != 1 || m.genericPicker.err == "" {
				t.Fatal("failed removal committed live state")
			}
			if failedFile == "state.json" {
				saved, err := config.Load()
				if err != nil || len(saved.Roots) != 2 {
					t.Fatal("configuration was not rolled back", err)
				}
			}
		})
	}
}

// Operation commands can be alone (reduced motion) or batched with the UI clock.
func operationMessage(cmd tea.Cmd) tea.Msg {
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		return operationMessage(batch[0])
	}
	return msg
}
