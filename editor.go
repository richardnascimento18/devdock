package main

// editor.go — interactive preset/template/configuration editor
//
// Navigation is entirely keyboard-driven (vim-style: j/k, h/l, tab, enter).
// The editor is split into three layers:
//
//  1. editorScreen  — top-level: Presets / Templates / Settings + item list
//  2. presetEditor  — edit a single Preset (windows, pane splits)
//  3. templateEditor — edit a single Template (steps list)
//
// All mutations happen in-memory; hitting ctrl+s (or 's') commits the file.
// ESC / ctrl+c returns to the previous layer without saving.

import (
	"github.com/richardnascimento18/devdock/internal/app"
	"github.com/richardnascimento18/devdock/internal/ui"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/preset"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
)

// ---------------------------------------------------------------------------
// Saved-to-disk message
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Editor tab constants
// ---------------------------------------------------------------------------

const (
	editorTabPresets   = 0
	editorTabTemplates = 1
	editorTabSettings  = 2
)

var editorTabNames = []string{"Presets", "Templates", "Settings"}

// ---------------------------------------------------------------------------
// editorLayer controls which layer is visible
// ---------------------------------------------------------------------------

type editorLayer int

const (
	editorLayerList     editorLayer = iota // browsing list of presets/templates
	editorLayerPreset                      // editing a single preset
	editorLayerTemplate                    // editing a single template
	editorLayerConfig                      // editing application configuration
)

// ---------------------------------------------------------------------------
// editorScreen — top-level editor model
// ---------------------------------------------------------------------------

type editorScreen struct {
	preferences app.Preferences
	cfg         config.Config
	revision    uint64
	tab         int
	cursor      int
	layer       editorLayer
	presets     []preset.Preset
	tmpls       []tmpl.Template

	pe presetEditor
	te templateEditor
	ce configurationEditor

	discarding    bool
	contextMaster bool
	deleting      bool
	deleteName    string
	scroll        int
	manualScroll  bool
	statusMsg     string
	termW         int
	termH         int
}

func newEditorScreenWithPreferences(presets []preset.Preset, templates []tmpl.Template, w, h int, preferences app.Preferences) editorScreen {
	return editorScreen{
		preferences: preferences,
		tab:         editorTabPresets,
		presets:     deepCopyPresets(presets),
		tmpls:       deepCopyTemplates(templates),
		termW:       w,
		termH:       h,
	}
}

// ---------------------------------------------------------------------------
// Deep-copy helpers (editor works on its own copy, never the live slice)
// ---------------------------------------------------------------------------

func deepCopyPresets(src []preset.Preset) []preset.Preset   { return preset.Clone(src) }
func deepCopyTemplates(src []tmpl.Template) []tmpl.Template { return tmpl.Clone(src) }

// ---------------------------------------------------------------------------
// editorScreen.listItems returns display strings for the current tab
// ---------------------------------------------------------------------------

func (e *editorScreen) listLen() int {
	switch e.tab {
	case editorTabPresets:
		return len(e.presets) + 1 // +1 for "create new"
	case editorTabSettings:
		return 3
	default:
		return len(e.tmpls) + 1
	}
}

func (e *editorScreen) clampCursor() {
	max := e.listLen() - 1
	if e.cursor > max {
		e.cursor = max
	}
	if e.cursor < 0 {
		e.cursor = 0
	}
}

// ---------------------------------------------------------------------------
// Update — editorScreen
// ---------------------------------------------------------------------------

func (e editorScreen) updateDraft(msg tea.Msg) (editorScreen, tea.Cmd) {
	switch e.layer {
	case editorLayerPreset:
		return e.updatePresetEditor(msg)
	case editorLayerTemplate:
		return e.updateTemplateEditor(msg)
	case editorLayerConfig:
		return e.updateConfigurationEditor(msg)
	default:
		return e.updateList(msg)
	}
}

func (e editorScreen) updateList(msg tea.Msg) (editorScreen, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		e.termW = sz.Width
		e.termH = sz.Height
		return e, nil
	}
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return e, nil
	}
	switch k.String() {
	case "j", "down":
		e.cursor++
		e.clampCursor()
	case "k", "up":
		e.cursor--
		e.clampCursor()
	case "h", "left", "shift+tab":
		e.tab = (e.tab + len(editorTabNames) - 1) % len(editorTabNames)
		e.cursor = 0
		e.statusMsg = ""
	case "l", "right", "tab":
		e.tab = (e.tab + 1) % len(editorTabNames)
		e.cursor = 0
		e.statusMsg = ""
	case "enter":
		e.statusMsg = ""
		switch e.tab {
		case editorTabSettings:
			if e.cursor == 0 {
				e.ce = newConfigurationEditor(e.cfg.DefaultPreset)
				e.layer = editorLayerConfig
			}
		case editorTabPresets:
			if e.cursor == len(e.presets) {
				// Create new preset
				e.pe = newPresetEditor(preset.Preset{Name: "new-preset", Windows: []preset.Window{{Name: "terminal"}}}, true)
				e.layer = editorLayerPreset
			} else {
				e.pe = newPresetEditor(e.presets[e.cursor], false)
				e.layer = editorLayerPreset
			}
		case editorTabTemplates:
			if e.cursor == len(e.tmpls) {
				e.te = newTemplateEditor(tmpl.Template{Name: "new-template", Description: ""}, true)
				e.layer = editorLayerTemplate
			} else {
				e.te = newTemplateEditor(e.tmpls[e.cursor], false)
				e.layer = editorLayerTemplate
			}
		}
	}
	return e, nil
}

func (e editorScreen) updatePresetEditor(msg tea.Msg) (editorScreen, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		e.termW = sz.Width
		e.termH = sz.Height
		return e, nil
	}
	var cmd tea.Cmd
	e.pe, cmd = e.pe.Update(msg)
	switch e.pe.result {
	case editorResultSave:
		if e.pe.layer == pelWindowEdit || e.pe.layer == pelWindowSplit {
			e.pe.windows = append([]windowDraft(nil), e.pe.windows...)
			e.pe.windows[e.pe.editIdx].name = strings.TrimSpace(e.pe.editNameInp.Value())
			e.pe.windows[e.pe.editIdx].command = strings.TrimSpace(e.pe.editCmdInp.Value())
			if e.pe.layer == pelWindowSplit {
				if e.pe.splitEditor.editing {
					size, err := parsePaneSize(e.pe.splitEditor.sizeInput.Value())
					if err != nil {
						e.pe.statusMsg = errorStyle.Render(ui.SafeBlock(err.Error()))
						e.pe.result = editorResultNone
						return e, nil
					}
					e.pe.splitEditor.panes = append([]paneLeaf(nil), e.pe.splitEditor.panes...)
					e.pe.splitEditor.panes[e.pe.splitEditor.editIdx] = paneLeaf{command: strings.TrimSpace(e.pe.splitEditor.cmdInput.Value()), size: size}
				}
				layout := e.pe.splitEditor.toLayout()
				e.pe.windows[e.pe.editIdx].layout = &layout
			}
		}
		p := e.pe.toPreset()
		proposed := deepCopyPresets(e.presets)
		if errMsg := preset.ValidatePreset(p); errMsg != "" {
			e.pe.result = editorResultNone
			e.pe.statusMsg = errorStyle.Render(ui.SafeBlock("✗  " + errMsg))
			return e, nil
		}
		// upsert
		found := false
		for i, existing := range proposed {
			if !e.pe.isNew && existing.Name == e.pe.originalName {
				proposed[i] = p
				found = true
				break
			}
		}
		if !found {
			proposed = append(proposed, p)
		}
		oldName := e.pe.originalName
		if e.pe.isNew {
			oldName = ""
		}
		proposedConfig, err := e.preferences.SavePresetProposal(e.cfg, proposed, oldName, p.Name)
		if err != nil {
			e.pe.statusMsg = errorStyle.Render(ui.SafeBlock("✗  save failed: " + err.Error()))
			e.pe.result = editorResultNone
			return e, nil
		} else {
			e.presets = proposed
			e.cfg = proposedConfig
			e.revision++
			e.statusMsg = successStyle.Render(ui.SafeBlock("✓  preset \"" + p.Name + "\" saved"))
		}
		e.pe.result = editorResultNone
		e.layer = editorLayerList
		e.cursor = 0
		e.clampCursor()
		return e, nil
	case editorResultCancel:
		e.pe.result = editorResultNone
		e.layer = editorLayerList
	}
	return e, cmd
}

func (e editorScreen) updateTemplateEditor(msg tea.Msg) (editorScreen, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		e.termW = sz.Width
		e.termH = sz.Height
		return e, nil
	}
	var cmd tea.Cmd
	e.te, cmd = e.te.Update(msg)
	switch e.te.result {
	case editorResultSave:
		if e.te.layer == telStepEdit {
			e.te.steps = append([]stepDraft(nil), e.te.steps...)
			e.te, _ = e.te.commitStepEdit() // commitStepEdit only updates the draft; it returns no command
			if e.te.layer == telStepEdit {
				e.te.result = editorResultNone
				return e, nil
			}
		}
		t := e.te.toTemplate()
		proposed := deepCopyTemplates(e.tmpls)
		errs := tmpl.ValidateFile(tmpl.TemplateFile{Templates: []tmpl.Template{t}})
		if len(errs) > 0 {
			e.te.result = editorResultNone
			e.te.statusMsg = errorStyle.Render(ui.SafeBlock("✗  " + errs[0]))
			return e, nil
		}
		found := false
		for i, existing := range proposed {
			if !e.te.isNew && existing.Name == e.te.originalName {
				proposed[i] = t
				found = true
				break
			}
		}
		if !found {
			proposed = append(proposed, t)
		}
		if err := e.preferences.Templates.Save(proposed); err != nil {
			e.te.statusMsg = errorStyle.Render(ui.SafeBlock("✗  save failed: " + err.Error()))
			e.te.result = editorResultNone
			return e, nil
		} else {
			e.tmpls = proposed
			e.revision++
			e.statusMsg = successStyle.Render(ui.SafeBlock("✓  template \"" + t.Name + "\" saved"))
		}
		e.te.result = editorResultNone
		e.layer = editorLayerList
		e.cursor = 0
		e.clampCursor()
		return e, nil
	case editorResultCancel:
		e.te.result = editorResultNone
		e.layer = editorLayerList
	}
	return e, cmd
}

// ---------------------------------------------------------------------------
// View — editorScreen
// ---------------------------------------------------------------------------

type editorResult int

const (
	editorResultNone   editorResult = iota
	editorResultSave                // user confirmed save (ctrl+s)
	editorResultCancel              // user cancelled (esc / ctrl+c)
)

// ===========================================================================
// PRESET EDITOR
// ===========================================================================

// ===========================================================================
// Shared input helper
// ===========================================================================

func newSmallInput(value, placeholder string, width int) textinput.Model {
	ti := transparentInput()
	ti.Prompt = "  "
	ti.Placeholder = placeholder
	ti.SetValue(value)
	ti.CharLimit = 0
	ti.Width = width
	ti.Cursor.Style = cursorStyle
	ti.PromptStyle = promptStyle
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Primary)
	return ti
}
