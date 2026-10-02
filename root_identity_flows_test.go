package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	uistate "github.com/richardnascimento18/devdock/internal/state"
)

func TestRootAdditionConflictsLeaveConfigAndStateUnchanged(t *testing.T) {
	for _, scenario := range []string{"duplicate", "inside", "contains", "alias", "physical-inside", "physical-contains"} {
		for _, editor := range []bool{false, true} {
			name := scenario + "/normal"
			if editor {
				name = scenario + "/settings"
			}
			t.Run(name, func(t *testing.T) {
				base := t.TempDir()
				real := filepath.Join(base, "real")
				child := filepath.Join(real, "client")
				alias := filepath.Join(base, "alias")
				if err := os.MkdirAll(child, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(real, alias); err != nil {
					t.Fatal(err)
				}
				existing, candidate := real, real
				switch scenario {
				case "inside":
					candidate = child
				case "contains":
					existing, candidate = child, real
				case "alias":
					candidate = alias
				case "physical-inside":
					existing, candidate = alias, child
				case "physical-contains":
					existing, candidate = child, alias
				}
				p, err := core.CreateProject(core.Location{Root: existing, Domain: "apps"}, "demo")
				if err != nil {
					t.Fatal(err)
				}
				m := fixtureModel(t, existing)
				if err := config.Save(m.cfg); err != nil {
					t.Fatal(err)
				}
				m.uiState.ToggleFavorite(p.Path)
				m.uiState.AddRecent(p)
				if err := uistate.Save(config.Dir(), m.uiState); err != nil {
					t.Fatal(err)
				}
				beforeCfg := m.cfg.Clone()
				beforeState := m.uiState.Clone()
				configBytes, err := os.ReadFile(config.Path())
				if err != nil {
					t.Fatal(err)
				}
				statePath := filepath.Join(config.Dir(), "state.json")
				stateBytes, err := os.ReadFile(statePath)
				if err != nil {
					t.Fatal(err)
				}
				var next tea.Model
				if editor {
					m = openSettings(t, m)
					m.editorScr.cursor = 1
					next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				} else {
					next, _ = m.startAddRoot()
				}
				m = next.(model)
				m.inputScr.input.SetValue(candidate)
				next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m = next.(model)
				if m.state != stateAddRoot || !strings.Contains(m.inputScr.err, existing) || !reflect.DeepEqual(m.cfg, beforeCfg) || !reflect.DeepEqual(m.uiState, beforeState) {
					t.Fatalf("conflict mutated/exited: %s", m.inputScr.err)
				}
				afterCfg, err := os.ReadFile(config.Path())
				if err != nil || string(afterCfg) != string(configBytes) {
					t.Fatal("failed addition changed persisted config")
				}
				afterState, err := os.ReadFile(statePath)
				if err != nil || string(afterState) != string(stateBytes) {
					t.Fatal("failed addition changed persisted state")
				}
			})
		}
	}
}

func TestAddRootNormalizesRelativeInputWithoutPrefixFalsePositive(t *testing.T) {
	base := t.TempDir()
	code := filepath.Join(base, "code")
	code2 := filepath.Join(base, "code2")
	for _, dir := range []string{code, code2} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	m := fixtureModel(t, code)
	if err := config.Save(m.cfg); err != nil {
		t.Fatal(err)
	}
	t.Chdir(base)
	next, _ := m.startAddRoot()
	m = next.(model)
	m.inputScr.input.SetValue("./code2/../code2/")
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	loaded, err := config.Load()
	if err != nil || m.state != stateList || !reflect.DeepEqual(loaded.Roots, []string{code, code2}) {
		t.Fatalf("normalized sibling rejected: %+v %v %s", loaded, err, m.inputScr.err)
	}
}

func TestMissingRootSettingsRetainFavoritesAndRecents(t *testing.T) {
	available := t.TempDir()
	missing := filepath.Join(t.TempDir(), "unavailable")
	m := fixtureModel(t, available)
	m.cfg.Roots = append(m.cfg.Roots, missing)
	p := core.Project{Location: core.Location{Root: missing, Domain: "apps"}, Name: "demo", Path: filepath.Join(missing, "apps", "demo")}
	m.uiState.ToggleFavorite(p.Path)
	m.uiState.AddRecent(p)
	before := m.uiState.Clone()
	if err := config.Save(m.cfg); err != nil {
		t.Fatal(err)
	}
	m = openSettings(t, m)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	m.editorScr.ce.defaultPreset.SetValue("nvim")
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = next.(model)
	loaded, err := config.Load()
	if err != nil || !reflect.DeepEqual(loaded.Roots, []string{available, missing}) || !reflect.DeepEqual(m.uiState, before) {
		t.Fatal("missing root or associated state lost")
	}
}
