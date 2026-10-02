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
	"github.com/richardnascimento18/devdock/internal/preset"
	uistate "github.com/richardnascimento18/devdock/internal/state"
)

func openSettings(t *testing.T, m model) model {
	t.Helper()
	next, _ := m.startEditor()
	m = next.(model)
	for range 2 {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = next.(model)
	}
	if m.editorScr.tab != editorTabSettings {
		t.Fatal("settings tab unreachable")
	}
	return m
}

func TestConfigurationEditorTransaction(t *testing.T) {
	for _, scenario := range []string{"save", "blank", "unknown-preset", "invalid-root", "duplicate-root", "failed-write", "cancel", "no-op"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			m := fixtureModel(t, root)
			cfg := config.Config{Roots: []string{root}, DefaultPreset: "walker", GitHubToken: "private-token", GitHubUsername: "private-user"}
			m.cfg = cfg.Clone()
			if err := config.Save(cfg); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(config.Path())
			if err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(config.Path())
			if err != nil {
				t.Fatal(err)
			}
			m = openSettings(t, m)
			view := m.editorScr.View()
			if !strings.Contains(view, "Settings") || !strings.Contains(view, "Configured roots:") || strings.Contains(view, cfg.GitHubToken) || strings.Contains(view, cfg.GitHubUsername) {
				t.Fatal("settings navigation or credential exclusion")
			}
			next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = next.(model)
			value := "nvim"
			switch scenario {
			case "blank":
				value = "  "
			case "unknown-preset":
				value = "does-not-exist"
			case "invalid-root":
				m.editorScr.cfg.Roots = []string{"relative"}
			case "duplicate-root":
				m.editorScr.cfg.Roots = []string{root, root}
			case "no-op":
				value = "walker"
			}
			m.editorScr.ce.defaultPreset.SetValue(value)
			if scenario == "failed-write" || scenario == "no-op" {
				if os.Geteuid() == 0 {
					t.Skip("root bypasses directory permissions")
				}
				if err := os.Chmod(config.Dir(), 0o555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := os.Chmod(config.Dir(), 0o755); err != nil {
						t.Error(err)
					}
				})
			}
			key := tea.KeyCtrlS
			if scenario == "cancel" {
				key = tea.KeyCtrlC
			}
			next, _ = m.Update(tea.KeyMsg{Type: key})
			m = next.(model)
			loaded, err := config.Load()
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "save", "blank":
				expected := strings.TrimSpace(value)
				if loaded.DefaultPreset != expected || m.cfg.DefaultPreset != expected || m.editorScr.revision != 1 || m.editorScr.layer != editorLayerList {
					t.Fatal("saved configuration not committed")
				}
				if expected == "" {
					expected = preset.DefaultPresets[0].Name
				}
				if m.presetSel.SelectedName() != expected {
					t.Fatal("active default selector not updated")
				}
				if loaded.GitHubToken != cfg.GitHubToken || !reflect.DeepEqual(loaded.Roots, cfg.Roots) {
					t.Fatal("unrelated fields changed")
				}
			default:
				after, err := os.ReadFile(config.Path())
				if err != nil || string(after) != string(before) {
					t.Fatal("disk config changed")
				}
				if !reflect.DeepEqual(loaded, cfg) || !reflect.DeepEqual(m.cfg, cfg) || m.editorScr.revision != 0 {
					t.Fatal("live configuration changed on unsuccessful save")
				}
				if scenario == "no-op" {
					afterInfo, err := os.Stat(config.Path())
					if err != nil || !os.SameFile(info, afterInfo) || m.editorScr.layer != editorLayerList {
						t.Fatal("no-op rewrote file or failed")
					}
				} else if scenario != "cancel" {
					if m.editorScr.layer != editorLayerConfig || m.editorScr.ce.defaultPreset.Value() != value || m.editorScr.ce.statusMsg == "" {
						t.Fatal("failed save lost draft or error")
					}
				}
			}
		})
	}
}

func TestConfigurationEditorTypingAndCancellation(t *testing.T) {
	m := fixtureModel(t, t.TempDir())
	m = openSettings(t, m)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("nvim")})
	m = next.(model)
	if m.editorScr.ce.defaultPreset.Value() != "nvim" {
		t.Fatal("typing failed")
	}
	for i := range 2 {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = next.(model)
		if i == 0 && (m.editorScr.ce.typing || m.editorScr.layer != editorLayerConfig) {
			t.Fatal("escape should leave typing first")
		}
	}
	if m.state != stateEditor || m.editorScr.layer != editorLayerList || m.cfg.DefaultPreset != "" {
		t.Fatal("draft cancellation mutated live configuration")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if m.editorScr.ce.defaultPreset.Value() != "" {
		t.Fatal("cancelled draft leaked into new edit")
	}
	m.editorScr.layer = editorLayerList
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = next.(model)
	if m.editorScr.tab != editorTabTemplates {
		t.Fatal("left navigation did not select previous tab")
	}
}

func TestConfigurationEditorRootWorkflows(t *testing.T) {
	for _, scenario := range []string{"add", "remove", "cancel-add", "cancel-remove", "invalid-add", "duplicate-add", "failed-add", "failed-remove"} {
		t.Run(scenario, func(t *testing.T) {
			root, other := t.TempDir(), t.TempDir()
			p, err := core.CreateProject(core.Location{Root: root, Domain: "apps"}, "demo")
			if err != nil {
				t.Fatal(err)
			}
			m := fixtureModel(t, root)
			if err := config.Save(m.cfg); err != nil {
				t.Fatal(err)
			}
			m.uiState.ToggleFavorite(p.Path)
			m.uiState.AddRecent(p)
			m.uiState.CollapsedNodes[core.Location{Root: root, Domain: "apps"}.Key()] = true
			m.committedState = m.uiState.Clone()
			m = openSettings(t, m)
			remove := strings.Contains(scenario, "remove")
			m.editorScr.cursor = 1
			if remove {
				m.editorScr.cursor = 2
			}
			next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = next.(model)
			if (remove && m.state != stateRemoveRoot) || (!remove && m.state != stateAddRoot) {
				t.Fatal("root editor not reused")
			}
			if !remove && m.inputScr.input.CharLimit != 0 {
				t.Fatal("root path input has an application length cap")
			}
			m.inputScr.input.SetValue(other)
			if scenario == "invalid-add" {
				m.inputScr.input.SetValue(filepath.Join(other, "missing"))
			}
			if scenario == "duplicate-add" {
				m.inputScr.input.SetValue(root)
			}
			if strings.HasPrefix(scenario, "failed-") {
				if os.Geteuid() == 0 {
					t.Skip("root bypasses directory permissions")
				}
				if err := os.Chmod(config.Dir(), 0o555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := os.Chmod(config.Dir(), 0o755); err != nil {
						t.Error(err)
					}
				})
			}
			key := tea.KeyEnter
			if strings.HasPrefix(scenario, "cancel-") {
				key = tea.KeyEsc
			}
			next, _ = m.Update(tea.KeyMsg{Type: key})
			m = next.(model)
			loaded, err := config.Load()
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "add":
				if !reflect.DeepEqual(loaded.Roots, []string{root, other}) {
					t.Fatal("root not saved")
				}
			case "remove":
				if len(loaded.Roots) != 0 || len(m.uiState.Favorites) != 0 || len(m.uiState.Recents) != 0 || len(m.collapsedNodes) != 0 {
					t.Fatal("root removal did not reconcile state")
				}
				saved, err := uistate.Load(config.Dir())
				if err != nil || len(saved.Favorites) != 0 || len(saved.Recents) != 0 {
					t.Fatal("root state cleanup not persisted")
				}
			case "invalid-add", "duplicate-add", "failed-add", "failed-remove":
				if m.state == stateEditor || !m.editorRootReturn || !reflect.DeepEqual(m.cfg, loaded) || len(loaded.Roots) != 1 || !m.uiState.Favorites[p.Path] {
					t.Fatal("failed root save committed or exited")
				}
				next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
				m = next.(model)
			default:
				if !reflect.DeepEqual(loaded.Roots, []string{root}) {
					t.Fatal("cancel/no-op mutated roots")
				}
			}
			if m.state != stateEditor || m.editorRootReturn || m.editorScr.tab != editorTabSettings || !reflect.DeepEqual(m.cfg, m.editorScr.cfg) {
				t.Fatal("root flow did not return with synchronized settings")
			}
			// Saving a default after root management must retain the new roots.
			if scenario == "add" || scenario == "remove" {
				m.editorScr.cursor = 0
				next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m = next.(model)
				m.editorScr.ce.defaultPreset.SetValue("nvim")
				next, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m = next.(model)
				final, err := config.Load()
				if err != nil || !reflect.DeepEqual(final.Roots, loaded.Roots) || final.DefaultPreset != "nvim" {
					t.Fatal("later config edit reverted root change")
				}
			}
		})
	}
}
