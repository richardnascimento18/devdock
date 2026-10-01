package main

import (
	"os"
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/preset"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
)

func TestEditorSaveTransactions(t *testing.T) {
	for _, kind := range []string{"preset", "template"} {
		for _, scenario := range []string{"success", "failure", "duplicate", "new-duplicate"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				t.Setenv("HOME", t.TempDir())
				ps, err := preset.Load(config.Dir())
				if err != nil {
					t.Fatal(err)
				}
				ts, err := tmpl.Load(config.Dir())
				if err != nil {
					t.Fatal(err)
				}
				e := newEditorScreen(ps, ts, 100, 40)
				if scenario == "failure" {
					path := preset.Path(config.Dir())
					if kind == "template" {
						path = tmpl.Path(config.Dir())
					}
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(path, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				name := "renamed"
				if scenario == "duplicate" || scenario == "new-duplicate" {
					if kind == "preset" {
						name = ps[1].Name
					} else {
						name = ts[1].Name
					}
				}
				if kind == "preset" {
					e.layer = editorLayerPreset
					e.pe = newPresetEditor(ps[0], scenario == "new-duplicate")
					e.pe.nameInput.SetValue(name)
				} else {
					e.layer = editorLayerTemplate
					e.te = newTemplateEditor(ts[0], scenario == "new-duplicate")
					e.te.nameInput.SetValue(name)
				}
				next, cmd := e.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				if cmd != nil {
					t.Fatal("save result should be synchronous")
				}
				if scenario == "success" {
					if next.revision != 1 || next.layer != editorLayerList {
						t.Fatal("save did not commit")
					}
					if kind == "preset" {
						loaded, err := preset.Load(config.Dir())
						if err != nil || loaded[0].Name != name {
							t.Fatal(err)
						}
					} else {
						loaded, err := tmpl.Load(config.Dir())
						if err != nil || loaded[0].Name != name {
							t.Fatal(err)
						}
					}
				} else {
					if next.revision != 0 || next.layer == editorLayerList || !reflect.DeepEqual(next.presets, ps) || !reflect.DeepEqual(next.tmpls, ts) {
						t.Fatal("failed save mutated active collection")
					}
					if scenario != "failure" {
						loaded, _ := preset.Load(config.Dir())
						loadedT, _ := tmpl.Load(config.Dir())
						if !reflect.DeepEqual(loaded, ps) || !reflect.DeepEqual(loadedT, ts) {
							t.Fatal("validation failure modified files")
						}
					}
				}
			})
		}
	}
}
func TestEditorPreservesMetadataAndSavesCurrentWindowDraft(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := newEditorScreen(preset.DefaultPresets, tmpl.DefaultTemplates, 100, 40)
	e.layer = editorLayerPreset
	e.pe = newPresetEditor(e.presets[0], false)
	e.pe, _ = e.pe.Update(tea.KeyMsg{Type: tea.KeyEnter})
	e.pe.editNameInp.SetValue("edited")
	next, _ := e.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if next.presets[0].Windows[0].Name != "edited" {
		t.Fatal("active window draft lost")
	}
	te := newTemplateEditor(tmpl.DefaultTemplates[4], false)
	round := te.toTemplate()
	if round.Layout != tmpl.DefaultTemplates[4].Layout || round.Steps[len(round.Steps)-1].Output != "requirements.txt" {
		t.Fatal("template metadata lost")
	}
}
