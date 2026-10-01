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

func TestPaneEditEnterAndCurrentDraftSave(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := newEditorScreen(preset.DefaultPresets, tmpl.DefaultTemplates, 100, 40)
	e.layer = editorLayerPreset
	e.pe = newPresetEditor(e.presets[2], false)
	e.pe, _ = e.pe.Update(tea.KeyMsg{Type: tea.KeyEnter})
	e.pe.layer = pelWindowSplit
	e.pe.editIdx = 0
	e.pe.splitEditor = newSplitPaneEditor(*e.presets[2].Windows[0].Layout)
	e.pe.splitEditor, _ = e.pe.splitEditor.Update(tea.KeyMsg{Type: tea.KeyEnter})
	e.pe.splitEditor.cmdInput.SetValue("changed")
	e.pe.splitEditor.sizeInput.SetValue("30")
	next, _ := e.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if next.pe.layer != pelWindowSplit || next.pe.splitEditor.editing || next.pe.splitEditor.panes[0].command != "changed" {
		t.Fatal("pane enter was intercepted by parent")
	}
	next.pe.splitEditor, _ = next.pe.splitEditor.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next.pe.splitEditor.cmdInput.SetValue("saved")
	next, _ = next.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if next.presets[2].Windows[0].Layout.Panes[0].Command != "saved" {
		t.Fatal("current pane draft lost")
	}
}

func TestPaneEditorPreservesNestedLayoutWithoutTopologyChange(t *testing.T) {
	original := preset.DefaultPresets[2].Windows[0].Layout
	editor := newSplitPaneEditor(*original)
	round := editor.toLayout()
	if !reflect.DeepEqual(*original, round) {
		t.Fatal("opening editor flattened nested layout")
	}
	editor.panes[2].command = "changed"
	round = editor.toLayout()
	if round.Panes[1].Direction != "vertical" || round.Panes[1].Panes[1].Command != "changed" {
		t.Fatal("editing leaf lost nested direction")
	}
}
