package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/preset"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
)

func TestEditorUnsavedDraftDiscardContinueAndSave(t *testing.T) {
	for _, kind := range []string{"preset", "template", "settings"} {
		t.Run(kind, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			e := newEditorScreen(preset.DefaultPresets, tmpl.DefaultTemplates, 40, 15)
			switch kind {
			case "preset":
				e.layer = editorLayerPreset
				e.pe = newPresetEditor(e.presets[0], false)
				e.pe.nameInput.SetValue("renamed")
			case "template":
				e.layer = editorLayerTemplate
				e.te = newTemplateEditor(e.tmpls[0], false)
				e.te.descInput.SetValue("new description")
			case "settings":
				e.layer = editorLayerConfig
				e.ce = newConfigurationEditor("nvim")
			}
			if !e.dirty() || !strings.Contains(ansi.Strip(e.View()), "Unsaved") {
				t.Fatal("dirty indicator missing")
			}
			e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEsc})
			if !e.discarding {
				t.Fatal("discard confirmation missing")
			}
			e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEsc})
			if e.discarding || !e.dirty() {
				t.Fatal("continue lost draft")
			}
			e, _ = e.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
			if e.layer != editorLayerList || e.revision != 1 {
				t.Fatal("save did not commit")
			}
		})
	}
}
func TestEditorCollectionRemovalAndDefaultGuard(t *testing.T) {
	for _, kind := range []string{"preset", "template"} {
		t.Run(kind, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			e := newEditorScreen(preset.DefaultPresets, tmpl.DefaultTemplates, 80, 24)
			if kind == "preset" {
				e.cfg.DefaultPreset = e.presets[0].Name
				e, _ = e.Update(keyRune("d"))
				if e.deleting || !strings.Contains(e.statusMsg, "default") {
					t.Fatal("default deletion allowed")
				}
				e.cursor = 1
			} else {
				e.tab = editorTabTemplates
			}
			before := e.listLen()
			e, _ = e.Update(keyRune("d"))
			if !e.deleting {
				t.Fatal("remove unreachable")
			}
			e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEsc})
			if e.listLen() != before {
				t.Fatal("cancel removed definition")
			}
			e, _ = e.Update(keyRune("d"))
			e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if e.deleting || e.listLen() != before-1 || e.revision != 1 {
				t.Fatal("remove not committed")
			}
		})
	}
}
func TestEditorCollectionRemovalWriteFailureRetainsDefinition(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ps, err := preset.Load(config.Dir())
	if err != nil {
		t.Fatal(err)
	}
	path := preset.Path(config.Dir())
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	e := newEditorScreen(ps, tmpl.DefaultTemplates, 80, 24)
	e.cursor = 1
	e = e.beginDelete()
	e = e.commitDelete()
	if !e.deleting || len(e.presets) != len(ps) || e.revision != 0 || e.statusMsg == "" {
		t.Fatal("failed write removed live definition")
	}
}
func TestTemplateOutputAndQuoteValidationNearField(t *testing.T) {
	te := newTemplateEditor(tmpl.Template{Name: "test", Steps: []tmpl.TemplateStep{{Type: "command", Run: `printf "hello world"`, Output: "result.txt"}}}, false)
	te.openStepEdit(te.steps[0])
	if te.dirty() {
		t.Fatal("unchanged step dirty")
	}
	te.editOutputInput.SetValue("../escape")
	te, _ = te.commitStepEdit()
	if te.layer != telStepEdit || te.statusMsg == "" || te.steps[0].output != "result.txt" {
		t.Fatal("invalid output accepted")
	}
	te.editOutputInput.SetValue("nested/result.txt")
	te.editRunInput.SetValue(`printf "broken`)
	te, _ = te.commitStepEdit()
	if te.layer != telStepEdit {
		t.Fatal("bad quoting accepted")
	}
	te.editRunInput.SetValue(`printf "日本語 é"`)
	te, _ = te.commitStepEdit()
	if te.layer != telStepList || te.toTemplate().Steps[0].Output != "nested/result.txt" || te.toTemplate().Steps[0].Run != `printf "日本語 é"` {
		t.Fatal("output or quoting lost")
	}
	if !utf8.ValidString(stepLabel(stepDraft{run: strings.Repeat("日本語é", 60)})) {
		t.Fatal("label split Unicode bytes")
	}
}
func TestSplitEditorEscapeKeepsNestedTypingSemantics(t *testing.T) {
	pe := newPresetEditor(preset.DefaultPresets[2], false)
	pe.layer = pelWindowSplit
	pe.splitEditor = newSplitPaneEditor(*pe.windows[0].layout)
	pe.splitEditor, _ = pe.splitEditor.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pe.splitEditor, _ = pe.splitEditor.Update(keyRune("i"))
	pe, _ = pe.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if pe.layer != pelWindowSplit || !pe.splitEditor.editing || pe.splitEditor.typing {
		t.Fatal("Escape bypassed typing state")
	}
	pe, _ = pe.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if pe.layer != pelWindowSplit || pe.splitEditor.editing {
		t.Fatal("Escape bypassed pane state")
	}
}
func TestEditorResponsiveLongCollectionsFocusAndScroll(t *testing.T) {
	e := newEditorScreen(preset.DefaultPresets, tmpl.DefaultTemplates, 40, 15)
	e.tab = editorTabSettings
	e.cfg.Roots = []string{"/workspace/" + strings.Repeat("日本語/group/", 200) + "last-root"}
	seen := ""
	for i := 0; i < 500; i++ {
		seen += strings.Join(strings.Fields(ansi.Strip(e.View())), "")
		e, _ = e.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	}
	if !strings.Contains(seen, "last-root") {
		t.Fatal("root cannot be recovered")
	}
	end := e.scroll
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if e.scroll >= end {
		t.Fatal("scroll unbounded")
	}
	e.layer = editorLayerTemplate
	e.te = newTemplateEditor(e.tmpls[0], false)
	e.te.openStepEdit(e.te.steps[0])
	e.te.editFocus = 3
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 15}, {24, 8}} {
		e, _ = e.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		v := e.View()
		if backgroundSequence(v) != "" {
			t.Fatal("opaque editor")
		}
		for _, line := range strings.Split(v, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatal("wide form", size)
			}
		}
		if !strings.Contains(ansi.Strip(v), "esc") {
			t.Fatal("cancel hint hidden", fmt.Sprint(size))
		}
	}
}

func TestRemovingEarlierPresetPreservesConfiguredDefaultSelection(t *testing.T) {
	m := fixtureModel(t, t.TempDir())
	m.cfg.DefaultPreset = m.presets[2].Name
	m.presetSel = newPresetSelector(m.presets, m.cfg.DefaultPreset)
	next, _ := m.startEditor()
	m = next.(model)
	m = dashboardKey(t, m, keyRune("d"))
	if !m.editorScr.deleting {
		t.Fatal("non-default removal blocked")
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.presetSel.SelectedName() != m.cfg.DefaultPreset || m.cfg.DefaultPreset != preset.DefaultPresets[2].Name {
		t.Fatal("removal shifted configured default selection")
	}
}
