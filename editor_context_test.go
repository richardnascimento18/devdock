package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/preset"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
)

func TestEditorContextPropertyResizeIdentityAndSave(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := newEditorScreen(preset.DefaultPresets, tmpl.DefaultTemplates, 140, 40)
	e.cursor = 1
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEnter})
	original := e.pe.originalName
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEnter})
	e.pe.editCmdInp.SetValue("nvim README.md")
	e.pe.editFocus = 1
	for _, size := range [][2]int{{140, 40}, {80, 24}, {40, 15}, {140, 40}} {
		e, _ = e.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		text := ansi.Strip(e.View())
		if !strings.Contains(text, "ctrl+s") || !strings.Contains(text, "esc") {
			t.Fatal("save/cancel missing", size)
		}
		if e.pe.originalName != original || e.pe.editCmdInp.Value() != "nvim README.md" {
			t.Fatal("resize changed identity/draft")
		}
		if size[0] >= 110 && (!strings.Contains(text, "Presets") || !strings.Contains(text, "Windows") || !strings.Contains(text, "Command")) {
			t.Fatal("wide context lost")
		}
		if size[0] == 40 && !strings.Contains(text, original) {
			t.Fatal("narrow breadcrumb lost identity")
		}
	}
	e, _ = e.Update(altKey("1"))
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyDown})
	if e.pe.originalName != original || e.pe.editCmdInp.Value() != "nvim README.md" {
		t.Fatal("collection browse replaced draft")
	}
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !e.discarding {
		t.Fatal("unsaved guard missing")
	}
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEsc})
	e, _ = e.Update(altKey("3"))
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if e.layer != editorLayerList || e.revision != 1 {
		t.Fatalf("save failed: %s", e.pe.statusMsg)
	}
	found := false
	for _, p := range e.presets {
		if p.Name == original && p.Windows[0].Command == "nvim README.md" {
			found = true
		}
	}
	if !found {
		t.Fatal("save committed wrong collection object")
	}
}
func TestContextTemplateStepValidationAndReturn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	templates := []tmpl.Template{{Name: "steps", Steps: []tmpl.TemplateStep{{Type: "command", Run: "pwd"}}}}
	e := newEditorScreen(preset.DefaultPresets, templates, 140, 40)
	e.tab = editorTabTemplates
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEnter})
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEnter})
	e.te.editOutputInput.SetValue("../escape")
	e, _ = e.Update(altKey("2"))
	if e.te.layer != telStepEdit || e.te.statusMsg == "" {
		t.Fatal("invalid field accepted")
	}
	text := ansi.Strip(e.View())
	if !strings.Contains(text, "Steps") || !strings.Contains(text, "Output path") || !strings.Contains(text, "escape") {
		t.Fatalf("validation/context missing: %s", text)
	}
	e.te.editOutputInput.SetValue("result.txt")
	e, _ = e.Update(altKey("2"))
	if e.te.layer != telStepList || e.te.steps[0].output != "result.txt" {
		t.Fatal("return to steps did not accept fields")
	}
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if e.layer != editorLayerList || e.revision != 1 {
		t.Fatal("save failed")
	}
}
func TestContextEditorCancelRetainsOriginalCollection(t *testing.T) {
	e := newEditorScreen(preset.DefaultPresets, tmpl.DefaultTemplates, 140, 40)
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEnter})
	e.pe.nameInput.SetValue("discarded")
	e, _ = e.Update(altKey("1"))
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEnter})
	e, _ = e.Update(keyRune("y"))
	if e.layer != editorLayerList || e.presets[0].Name != preset.DefaultPresets[0].Name || e.revision != 0 {
		t.Fatal("cancel committed draft")
	}
}

func TestSettingsRootFlowResizePreservesReturnLayout(t *testing.T) {
	m := renderFixture()
	m.state = stateAddRoot
	m.editorRootReturn = true
	m.editorScr = newEditorScreen(m.presets, m.templates, 120, 40)
	m.editorScr.tab = editorTabSettings
	m.inputScr = newInputScreen("Add root", "/workspace/new", "enter confirm · esc cancel")
	m = dashboardKey(t, m, keyRune("/pending/root"))
	for _, size := range [][2]int{{140, 40}, {80, 24}, {40, 15}} {
		next, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = next.(model)
		if m.editorScr.termW != size[0] || m.editorScr.termH != size[1] || m.inputScr.input.Value() != "/pending/root" {
			t.Fatal("paused Settings context/draft lost on resize")
		}
	}
	m = dashboardKey(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != stateEditor || m.editorScr.termW != 40 || !strings.Contains(ansi.Strip(m.View()), "Settings") || !strings.Contains(ansi.Strip(m.View()), "esc") {
		t.Fatal("return used stale editor dimensions")
	}
}
