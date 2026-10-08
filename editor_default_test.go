package main

import (
	"os"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/preset"
)

func TestDefaultPresetRenameTransaction(t *testing.T) {
	for _, scenario := range []string{"success", "duplicate", "preset-write-failure", "config-write-failure"} {
		t.Run(scenario, func(t *testing.T) {
			cfg := config.Config{Roots: []string{t.TempDir()}, DefaultPreset: preset.DefaultPresets[1].Name}
			m := fixtureModel(t, cfg.Roots...)
			m.cfg = cfg
			if err := config.Save(cfg); err != nil {
				t.Fatal(err)
			}
			if err := preset.Save(config.Dir(), preset.DefaultPresets); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(config.Path())
			if err != nil {
				t.Fatal(err)
			}
			next, _ := m.startEditor()
			m = next.(model)
			m.editorScr.layer = editorLayerPreset
			m.editorScr.pe = newPresetEditor(m.presets[1], false)
			name := "renamed-default"
			if scenario == "duplicate" {
				name = m.presets[0].Name
			}
			m.editorScr.pe.nameInput.SetValue(name)
			if scenario == "preset-write-failure" {
				path := preset.Path(config.Dir())
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "config-write-failure" {
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
			next, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
			m = next.(model)
			loaded, err := config.Load()
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "success" {
				if m.cfg.DefaultPreset != name || loaded.DefaultPreset != name || m.presets[1].Name != name {
					t.Fatal("default reference lost")
				}
				restarted, err := preset.Load(config.Dir())
				if err != nil || preset.ByName(restarted, loaded.DefaultPreset).Name != name {
					t.Fatal("restart selected fallback")
				}
			} else {
				if !reflect.DeepEqual(m.cfg, cfg) || !reflect.DeepEqual(loaded, cfg) || m.editorScr.revision != 0 || m.editorScr.pe.nameInput.Value() != name || m.editorScr.pe.diagnostic.Summary == "" {
					t.Fatal("failure committed or discarded draft")
				}
				after, err := os.ReadFile(config.Path())
				if err != nil || string(before) != string(after) {
					t.Fatal("configuration not restored")
				}
			}
		})
	}
}
