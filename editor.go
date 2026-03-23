package main

// editor.go — interactive preset/template editor
//
// Navigation is entirely keyboard-driven (vim-style: j/k, h/l, tab, enter).
// The editor is split into three layers:
//
//  1. editorScreen  — top-level: two tabs (Presets / Templates) + item list
//  2. presetEditor  — edit a single Preset (windows, pane splits)
//  3. templateEditor — edit a single Template (steps list)
//
// All mutations happen in-memory; hitting ctrl+s (or 's') commits the file.
// ESC / ctrl+c returns to the previous layer without saving.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

type editorSavedMsg struct{ kind string } // "preset" or "template"

// ---------------------------------------------------------------------------
// Editor tab constants
// ---------------------------------------------------------------------------

const (
	editorTabPresets   = 0
	editorTabTemplates = 1
)

var editorTabNames = []string{"Presets", "Templates"}

// ---------------------------------------------------------------------------
// editorLayer controls which layer is visible
// ---------------------------------------------------------------------------

type editorLayer int

const (
	editorLayerList     editorLayer = iota // browsing list of presets/templates
	editorLayerPreset                      // editing a single preset
	editorLayerTemplate                    // editing a single template
)

// ---------------------------------------------------------------------------
// editorScreen — top-level editor model
// ---------------------------------------------------------------------------

type editorScreen struct {
	tab     int
	cursor  int
	layer   editorLayer
	presets []preset.Preset
	tmpls   []tmpl.Template

	pe presetEditor
	te templateEditor

	statusMsg string
	termW     int
	termH     int
}

func newEditorScreen(presets []preset.Preset, templates []tmpl.Template, w, h int) editorScreen {
	return editorScreen{
		tab:     editorTabPresets,
		presets: deepCopyPresets(presets),
		tmpls:   deepCopyTemplates(templates),
		termW:   w,
		termH:   h,
	}
}

// ---------------------------------------------------------------------------
// Deep-copy helpers (editor works on its own copy, never the live slice)
// ---------------------------------------------------------------------------

func deepCopyPresets(src []preset.Preset) []preset.Preset {
	b, _ := json.Marshal(src)
	var dst []preset.Preset
	_ = json.Unmarshal(b, &dst)
	return dst
}

func deepCopyTemplates(src []tmpl.Template) []tmpl.Template {
	b, _ := json.Marshal(src)
	var dst []tmpl.Template
	_ = json.Unmarshal(b, &dst)
	return dst
}

// ---------------------------------------------------------------------------
// editorScreen.listItems returns display strings for the current tab
// ---------------------------------------------------------------------------

func (e *editorScreen) listLen() int {
	switch e.tab {
	case editorTabPresets:
		return len(e.presets) + 1 // +1 for "create new"
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

func (e editorScreen) Update(msg tea.Msg) (editorScreen, tea.Cmd) {
	switch e.layer {
	case editorLayerPreset:
		return e.updatePresetEditor(msg)
	case editorLayerTemplate:
		return e.updateTemplateEditor(msg)
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
		e.tab = (e.tab + 1) % 2 // only 2 tabs
		e.cursor = 0
		e.statusMsg = ""
	case "l", "right", "tab":
		e.tab = (e.tab + 1) % 2
		e.cursor = 0
		e.statusMsg = ""
	case "enter":
		e.statusMsg = ""
		switch e.tab {
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
		p := e.pe.toPreset()
		if errMsg := preset.ValidatePreset(p); errMsg != "" {
			e.pe.result = editorResultNone
			e.pe.statusMsg = errorStyle.Render("✗  " + errMsg)
			return e, nil
		}
		// upsert
		found := false
		for i, existing := range e.presets {
			if existing.Name == e.pe.originalName {
				e.presets[i] = p
				found = true
				break
			}
		}
		if !found {
			e.presets = append(e.presets, p)
		}
		if err := savePresetsFile(e.presets); err != nil {
			e.statusMsg = errorStyle.Render("✗  save failed: " + err.Error())
		} else {
			e.statusMsg = successStyle.Render("✓  preset \"" + p.Name + "\" saved")
		}
		e.pe.result = editorResultNone
		e.layer = editorLayerList
		e.cursor = 0
		e.clampCursor()
		return e, func() tea.Msg { return editorSavedMsg{kind: "preset"} }
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
		t := e.te.toTemplate()
		errs := tmpl.ValidateFile(tmpl.TemplateFile{Templates: []tmpl.Template{t}})
		if len(errs) > 0 {
			e.te.result = editorResultNone
			e.te.statusMsg = errorStyle.Render("✗  " + errs[0])
			return e, nil
		}
		found := false
		for i, existing := range e.tmpls {
			if existing.Name == e.te.originalName {
				e.tmpls[i] = t
				found = true
				break
			}
		}
		if !found {
			e.tmpls = append(e.tmpls, t)
		}
		if err := saveTemplatesFile(e.tmpls); err != nil {
			e.statusMsg = errorStyle.Render("✗  save failed: " + err.Error())
		} else {
			e.statusMsg = successStyle.Render("✓  template \"" + t.Name + "\" saved")
		}
		e.te.result = editorResultNone
		e.layer = editorLayerList
		e.cursor = 0
		e.clampCursor()
		return e, func() tea.Msg { return editorSavedMsg{kind: "template"} }
	case editorResultCancel:
		e.te.result = editorResultNone
		e.layer = editorLayerList
	}
	return e, cmd
}

// ---------------------------------------------------------------------------
// View — editorScreen
// ---------------------------------------------------------------------------

func (e editorScreen) View() string {
	switch e.layer {
	case editorLayerPreset:
		return e.pe.View(e.termW, e.termH)
	case editorLayerTemplate:
		return e.te.View(e.termW, e.termH)
	default:
		return e.viewList()
	}
}

func (e editorScreen) viewList() string {
	// Tab bar
	var tabBar strings.Builder
	for i, name := range editorTabNames {
		if i > 0 {
			tabBar.WriteString(dimStyle.Render("  │  "))
		}
		if i == e.tab {
			tabBar.WriteString(activeStyle.Render(" " + name + " "))
		} else {
			tabBar.WriteString(dimStyle.Render(name))
		}
	}
	tabBar.WriteString(dimStyle.Render("  (tab / h / l to switch)"))

	// List
	var listLines []string
	switch e.tab {
	case editorTabPresets:
		for i, p := range e.presets {
			var windows []string
			for _, w := range p.Windows {
				if w.Layout != nil {
					windows = append(windows, w.Name+dimStyle.Render("[split]"))
				} else {
					windows = append(windows, w.Name)
				}
			}
			summary := dimStyle.Render("  [" + strings.Join(windows, " · ") + "]")
			line := renderEditorListItem(i == e.cursor, p.Name, summary)
			listLines = append(listLines, line)
		}
		// "create new" entry
		createLabel := lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("✦  new preset")
		idx := len(e.presets)
		if e.cursor == idx {
			listLines = append(listLines, activeStyle.Render("▶ ")+createLabel)
		} else {
			listLines = append(listLines, "  "+createLabel)
		}
	case editorTabTemplates:
		for i, t := range e.tmpls {
			desc := dimStyle.Render("  " + t.Description)
			line := renderEditorListItem(i == e.cursor, t.Name, desc)
			listLines = append(listLines, line)
		}
		createLabel := lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("✦  new template")
		idx := len(e.tmpls)
		if e.cursor == idx {
			listLines = append(listLines, activeStyle.Render("▶ ")+createLabel)
		} else {
			listLines = append(listLines, "  "+createLabel)
		}
	}

	listContent := strings.Join(listLines, "\n")
	boxed := editorListBoxStyle.Render(listContent)

	statusLine := ""
	if e.statusMsg != "" {
		statusLine = "\n" + e.statusMsg
	}

	hint := hintStyle.Render("j/k navigate  •  enter select/edit  •  tab switch  •  esc back")

	content := lipgloss.JoinVertical(lipgloss.Left,
		RenderTitle(),
		promptStyle.Render("  Config Editor"),
		"",
		tabBar.String(),
		"",
		boxed,
		statusLine,
		"",
		hint,
	)
	return centerInTerminal(e.termW, e.termH, content)
}

func renderEditorListItem(selected bool, name, extra string) string {
	nameStr := lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(name)
	if selected {
		return activeStyle.Render("▶ "+name) + extra
	}
	return "  " + nameStr + extra
}

var editorListBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorPurple).
	Padding(0, 2).
	Width(70)

// ---------------------------------------------------------------------------
// editorResult — signals from sub-editors back to editorScreen
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

// presetEditorLayer controls which sub-screen is shown inside presetEditor
type presetEditorLayer int

const (
	pelWindowList    presetEditorLayer = iota // browsing windows
	pelWindowEdit                             // editing window name / command
	pelWindowSplit                            // editing split layout
	pelConfirmDelete                          // confirm delete window
)

type windowDraft struct {
	name    string
	command string
	layout  *preset.PaneLayout
}

type presetEditor struct {
	originalName string
	isNew        bool

	nameInput textinput.Model
	windows   []windowDraft
	cursor    int // selected window index
	layer     presetEditorLayer

	// for window edit
	editIdx      int
	editNameInp  textinput.Model
	editCmdInp   textinput.Model
	editFocus    int // 0=name, 1=command, 2=split-toggle
	editHasSplit bool

	// for split editor
	splitEditor splitPaneEditor

	// typing=true means keystrokes go to the focused text input;
	// j/k/navigation only work when typing=false.
	typing bool

	statusMsg string
	result    editorResult
}

func newPresetEditor(p preset.Preset, isNew bool) presetEditor {
	nameInp := textinput.New()
	nameInp.SetValue(p.Name)
	nameInp.CharLimit = 60
	nameInp.Width = 40
	nameInp.Cursor.Style = cursorStyle
	nameInp.PromptStyle = promptStyle
	nameInp.TextStyle = lipgloss.NewStyle().Foreground(colorWhite)

	var windows []windowDraft
	for _, w := range p.Windows {
		var layoutCopy *preset.PaneLayout
		if w.Layout != nil {
			b, _ := json.Marshal(w.Layout)
			var lc preset.PaneLayout
			_ = json.Unmarshal(b, &lc)
			layoutCopy = &lc
		}
		windows = append(windows, windowDraft{name: w.Name, command: w.Command, layout: layoutCopy})
	}
	if len(windows) == 0 {
		windows = []windowDraft{{name: "terminal"}}
	}

	return presetEditor{
		originalName: p.Name,
		isNew:        isNew,
		nameInput:    nameInp,
		windows:      windows,
	}
}

func (pe *presetEditor) toPreset() preset.Preset {
	p := preset.Preset{Name: strings.TrimSpace(pe.nameInput.Value())}
	for _, wd := range pe.windows {
		w := preset.Window{Name: wd.name, Command: wd.command}
		if wd.layout != nil {
			lc := *wd.layout
			w.Layout = &lc
		}
		p.Windows = append(p.Windows, w)
	}
	return p
}

func (pe presetEditor) Update(msg tea.Msg) (presetEditor, tea.Cmd) {
	switch pe.layer {
	case pelWindowList:
		return pe.updateWindowList(msg)
	case pelWindowEdit:
		return pe.updateWindowEdit(msg)
	case pelWindowSplit:
		return pe.updateWindowSplit(msg)
	case pelConfirmDelete:
		return pe.updateConfirmDelete(msg)
	}
	return pe, nil
}

func (pe presetEditor) updateWindowList(msg tea.Msg) (presetEditor, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		if pe.typing {
			var cmd tea.Cmd
			pe.nameInput, cmd = pe.nameInput.Update(msg)
			return pe, cmd
		}
		return pe, nil
	}

	// When in typing mode all keys except esc/ctrl+s go to the name input.
	if pe.typing {
		switch k.String() {
		case "esc":
			pe.typing = false
			pe.nameInput.Blur()
			return pe, nil
		case "ctrl+s":
			pe.result = editorResultSave
			return pe, nil
		case "ctrl+c":
			pe.result = editorResultCancel
			return pe, nil
		}
		var cmd tea.Cmd
		pe.nameInput, cmd = pe.nameInput.Update(msg)
		return pe, cmd
	}

	// Navigation mode.
	switch k.String() {
	case "ctrl+c", "esc":
		pe.result = editorResultCancel
	case "ctrl+s":
		pe.result = editorResultSave
	case "i":
		// Enter typing mode for the preset name input.
		pe.typing = true
		pe.nameInput.Focus()
	case "j", "down":
		if pe.cursor < len(pe.windows) {
			pe.cursor++
		}
	case "k", "up":
		if pe.cursor > 0 {
			pe.cursor--
		}
	case "enter":
		if pe.cursor == len(pe.windows) {
			// add new window
			pe.windows = append(pe.windows, windowDraft{name: fmt.Sprintf("window-%d", len(pe.windows)+1)})
			pe.cursor = len(pe.windows) - 1
		}
		// open edit for this window
		pe.layer = pelWindowEdit
		pe.typing = false
		pe.editIdx = pe.cursor
		pe.editNameInp = newSmallInput(pe.windows[pe.cursor].name, "window name", 40)
		pe.editCmdInp = newSmallInput(pe.windows[pe.cursor].command, "command (blank = shell)", 40)
		pe.editFocus = 0
		pe.editHasSplit = pe.windows[pe.cursor].layout != nil
	case "d", "x":
		if pe.cursor < len(pe.windows) {
			if len(pe.windows) <= 1 {
				pe.statusMsg = errorStyle.Render("✗  a preset must have at least 1 window")
			} else {
				pe.layer = pelConfirmDelete
			}
		}
	}
	return pe, nil
}

func (pe presetEditor) updateWindowEdit(msg tea.Msg) (presetEditor, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		if pe.typing {
			return pe.forwardWindowEditInput(msg)
		}
		return pe, nil
	}

	// Typing mode: all keys go to the focused input except esc/ctrl+s/ctrl+c.
	if pe.typing {
		switch k.String() {
		case "esc":
			pe.typing = false
			pe.editNameInp.Blur()
			pe.editCmdInp.Blur()
			return pe, nil
		case "ctrl+s":
			pe.result = editorResultSave
			return pe, nil
		case "ctrl+c":
			pe.result = editorResultCancel
			return pe, nil
		}
		return pe.forwardWindowEditInput(msg)
	}

	// Navigation mode.
	switch k.String() {
	case "ctrl+c":
		pe.result = editorResultCancel
		return pe, nil
	case "esc":
		pe.layer = pelWindowList
		pe.typing = false
		return pe, nil
	case "ctrl+s":
		pe.result = editorResultSave
		return pe, nil
	case "tab", "down", "j":
		pe.editFocus = (pe.editFocus + 1) % 3
		pe.syncWindowEditFocus()
		return pe, nil
	case "shift+tab", "up", "k":
		pe.editFocus = (pe.editFocus + 2) % 3
		pe.syncWindowEditFocus()
		return pe, nil
	case "i":
		// Enter typing mode for the currently focused text field (0 or 1).
		if pe.editFocus == 0 || pe.editFocus == 1 {
			pe.typing = true
			pe.syncWindowEditFocus()
		}
		return pe, nil
	case "enter":
		if pe.editFocus == 2 {
			// toggle / open split editor
			if pe.editHasSplit {
				pe.editHasSplit = false
				pe.windows[pe.editIdx].layout = nil
			} else {
				pe.editHasSplit = true
				existing := pe.windows[pe.editIdx].layout
				if existing == nil {
					def := &preset.PaneLayout{
						Direction: "horizontal",
						Panes:     []preset.PaneLayout{{Command: ""}, {Command: ""}},
					}
					existing = def
				}
				pe.splitEditor = newSplitPaneEditor(*existing)
				pe.layer = pelWindowSplit
				return pe, nil
			}
		} else {
			// Save name/command and go back.
			name := strings.TrimSpace(pe.editNameInp.Value())
			if name == "" {
				pe.statusMsg = errorStyle.Render("✗  window name cannot be empty")
				return pe, nil
			}
			pe.windows[pe.editIdx].name = name
			pe.windows[pe.editIdx].command = strings.TrimSpace(pe.editCmdInp.Value())
			if !pe.editHasSplit {
				pe.windows[pe.editIdx].layout = nil
			}
			pe.layer = pelWindowList
			pe.statusMsg = ""
		}
	case "e":
		if pe.editFocus == 2 && pe.editHasSplit && pe.windows[pe.editIdx].layout != nil {
			pe.splitEditor = newSplitPaneEditor(*pe.windows[pe.editIdx].layout)
			pe.layer = pelWindowSplit
		}
	}
	return pe, nil
}

func (pe presetEditor) forwardWindowEditInput(msg tea.Msg) (presetEditor, tea.Cmd) {
	var cmd tea.Cmd
	switch pe.editFocus {
	case 0:
		pe.editNameInp, cmd = pe.editNameInp.Update(msg)
	case 1:
		pe.editCmdInp, cmd = pe.editCmdInp.Update(msg)
	}
	return pe, cmd
}

func (pe *presetEditor) syncWindowEditFocus() {
	pe.editNameInp.Blur()
	pe.editCmdInp.Blur()
	if !pe.typing {
		return
	}
	switch pe.editFocus {
	case 0:
		pe.editNameInp.Focus()
	case 1:
		pe.editCmdInp.Focus()
	}
}

func (pe presetEditor) updateWindowSplit(msg tea.Msg) (presetEditor, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if ok {
		switch k.String() {
		case "ctrl+c":
			pe.result = editorResultCancel
			return pe, nil
		case "esc":
			pe.layer = pelWindowEdit
			return pe, nil
		case "ctrl+s":
			pe.result = editorResultSave
			return pe, nil
		case "enter":
			// save layout back
			l := pe.splitEditor.toLayout()
			pe.windows[pe.editIdx].layout = &l
			pe.editHasSplit = true
			pe.layer = pelWindowEdit
			return pe, nil
		}
	}
	var cmd tea.Cmd
	pe.splitEditor, cmd = pe.splitEditor.Update(msg)
	return pe, cmd
}

func (pe presetEditor) updateConfirmDelete(msg tea.Msg) (presetEditor, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return pe, nil
	}
	switch k.String() {
	case "y", "enter":
		pe.windows = append(pe.windows[:pe.cursor], pe.windows[pe.cursor+1:]...)
		if pe.cursor >= len(pe.windows) {
			pe.cursor = len(pe.windows) - 1
		}
		if pe.cursor < 0 {
			pe.cursor = 0
		}
		pe.layer = pelWindowList
	case "n", "esc":
		pe.layer = pelWindowList
	}
	return pe, nil
}

// ---------------------------------------------------------------------------
// presetEditor.View
// ---------------------------------------------------------------------------

func (pe presetEditor) View(w, h int) string {
	switch pe.layer {
	case pelWindowEdit:
		return pe.viewWindowEdit(w, h)
	case pelWindowSplit:
		return pe.viewWindowSplit(w, h)
	case pelConfirmDelete:
		return pe.viewConfirmDelete(w, h)
	default:
		return pe.viewWindowList(w, h)
	}
}

func (pe presetEditor) viewWindowList(w, h int) string {
	title := "Edit Preset"
	if pe.isNew {
		title = "New Preset"
	}
	var inner strings.Builder
	inner.WriteString(promptStyle.Render("Preset name:") + "\n")
	inner.WriteString(pe.nameInput.View() + "\n\n")
	inner.WriteString(promptStyle.Render("Windows:") + "\n")
	for i, wd := range pe.windows {
		var suffix string
		if wd.layout != nil {
			suffix = dimStyle.Render(" [split]")
		} else if wd.command != "" {
			suffix = dimStyle.Render("  $ " + wd.command)
		}
		name := lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(wd.name)
		var line string
		if i == pe.cursor {
			line = activeStyle.Render("▶ "+wd.name) + suffix
		} else {
			line = "  " + name + suffix
		}
		inner.WriteString(line + "\n")
	}
	// "add window" entry
	addLabel := lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("✦  add window")
	if pe.cursor == len(pe.windows) {
		inner.WriteString(activeStyle.Render("▶ ") + addLabel + "\n")
	} else {
		inner.WriteString("  " + addLabel + "\n")
	}
	if pe.statusMsg != "" {
		inner.WriteString("\n" + pe.statusMsg)
	}
	inner.WriteString("\n\n" + hintStyle.Render("j/k navigate  •  i to edit name  •  enter open  •  d/x delete  •  ctrl+s save  •  esc cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		RenderTitle(),
		promptStyle.Render("  "+title),
		"",
		wrapInBox(title, inner.String()),
	)
	return centerInTerminal(w, h, content)
}

func (pe presetEditor) viewWindowEdit(w, h int) string {
	wd := pe.windows[pe.editIdx]
	_ = wd

	focusIndicator := func(idx int, label string) string {
		if pe.editFocus == idx {
			return promptStyle.Render("▶ " + label)
		}
		return dimStyle.Render("  " + label)
	}

	splitStatus := "disabled"
	if pe.editHasSplit {
		splitStatus = "enabled"
	}
	splitLine := focusIndicator(2, "split layout: "+splitStatus)
	if pe.editFocus == 2 {
		if pe.editHasSplit {
			splitLine += dimStyle.Render("  (enter to edit, or press enter again to disable)")
		} else {
			splitLine += dimStyle.Render("  (enter to enable)")
		}
	}

	var inner strings.Builder
	inner.WriteString(focusIndicator(0, "Window name:") + "\n")
	inner.WriteString(pe.editNameInp.View() + "\n\n")
	inner.WriteString(focusIndicator(1, "Command (blank = shell):") + "\n")
	inner.WriteString(pe.editCmdInp.View() + "\n\n")
	inner.WriteString(splitLine + "\n")
	if pe.statusMsg != "" {
		inner.WriteString("\n" + pe.statusMsg)
	}
	if pe.typing {
		inner.WriteString("\n\n" + hintStyle.Render("typing mode  •  esc to stop typing  •  ctrl+s save all"))
	} else {
		inner.WriteString("\n\n" + hintStyle.Render("j/k navigate  •  i to type  •  enter confirm  •  esc back  •  ctrl+s save all"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		RenderTitle(),
		promptStyle.Render("  Edit Window"),
		"",
		wrapInBox("Edit Window", inner.String()),
	)
	return centerInTerminal(w, h, content)
}

func (pe presetEditor) viewWindowSplit(w, h int) string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		RenderTitle(),
		promptStyle.Render("  Pane Layout Editor"),
		"",
		pe.splitEditor.View(),
		"",
		hintStyle.Render("enter confirm  •  esc back"),
	)
	return centerInTerminal(w, h, content)
}

func (pe presetEditor) viewConfirmDelete(w, h int) string {
	name := ""
	if pe.cursor < len(pe.windows) {
		name = pe.windows[pe.cursor].name
	}
	inner := warningStyle.Render(fmt.Sprintf("Delete window \"%s\"?", name)) + "\n\n" +
		hintStyle.Render("y/enter — yes  •  n/esc — no")
	content := lipgloss.JoinVertical(lipgloss.Left,
		RenderTitle(),
		wrapInWarningBox("Confirm Delete", inner),
	)
	return centerInTerminal(w, h, content)
}

// ===========================================================================
// SPLIT PANE EDITOR — simplified flat representation
// ===========================================================================
// For UX clarity we represent the split as a flat list of leaf panes, each
// with a command, plus a top-level direction (horizontal / vertical).
// This covers the vast majority of use-cases without needing a tree UI.

type paneLeaf struct {
	command string
	size    int
}

type splitPaneEditor struct {
	direction string // "horizontal" or "vertical"
	panes     []paneLeaf
	cursor    int
	editIdx   int
	editing   bool
	typing    bool // true = keystrokes go to text input; i to enter, esc to leave
	cmdInput  textinput.Model
	sizeInput textinput.Model
	editFocus int // 0=cmd 1=size
}

func newSplitPaneEditor(pl preset.PaneLayout) splitPaneEditor {
	var panes []paneLeaf
	collectLeaves(pl, &panes)
	if len(panes) < 2 {
		panes = []paneLeaf{{}, {}}
	}
	dir := pl.Direction
	if dir != "horizontal" && dir != "vertical" {
		dir = "horizontal"
	}
	return splitPaneEditor{direction: dir, panes: panes}
}

func collectLeaves(pl preset.PaneLayout, out *[]paneLeaf) {
	if pl.IsLeaf() {
		*out = append(*out, paneLeaf{command: pl.Command, size: pl.Size})
		return
	}
	for _, child := range pl.Panes {
		collectLeaves(child, out)
	}
}

func (sp splitPaneEditor) toLayout() preset.PaneLayout {
	if len(sp.panes) == 0 {
		return preset.PaneLayout{}
	}
	root := preset.PaneLayout{Direction: sp.direction}
	for _, p := range sp.panes {
		root.Panes = append(root.Panes, preset.PaneLayout{Command: p.command, Size: p.size})
	}
	return root
}

func (sp splitPaneEditor) Update(msg tea.Msg) (splitPaneEditor, tea.Cmd) {
	if sp.editing {
		return sp.updateEdit(msg)
	}
	return sp.updateList(msg)
}

func (sp splitPaneEditor) updateList(msg tea.Msg) (splitPaneEditor, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return sp, nil
	}
	switch k.String() {
	case "j", "down":
		if sp.cursor < len(sp.panes) {
			sp.cursor++
		}
	case "k", "up":
		if sp.cursor > 0 {
			sp.cursor--
		}
	case "r":
		if sp.direction == "horizontal" {
			sp.direction = "vertical"
		} else {
			sp.direction = "horizontal"
		}
	case "a":
		sp.panes = append(sp.panes, paneLeaf{})
	case "d", "x":
		if len(sp.panes) > 2 && sp.cursor < len(sp.panes) {
			sp.panes = append(sp.panes[:sp.cursor], sp.panes[sp.cursor+1:]...)
			if sp.cursor >= len(sp.panes) {
				sp.cursor = len(sp.panes) - 1
			}
		}
	case "enter":
		if sp.cursor < len(sp.panes) {
			sp.editIdx = sp.cursor
			sp.cmdInput = newSmallInput(sp.panes[sp.cursor].command, "command", 40)
			sp.sizeInput = newSmallInput("", "size % (0=auto)", 6)
			if sp.panes[sp.cursor].size > 0 {
				sp.sizeInput.SetValue(fmt.Sprintf("%d", sp.panes[sp.cursor].size))
			}
			sp.editFocus = 0
			sp.typing = false // start in navigation mode
			sp.editing = true
		}
	}
	return sp, nil
}

func (sp splitPaneEditor) updateEdit(msg tea.Msg) (splitPaneEditor, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		if sp.typing {
			return sp.forwardPaneInput(msg)
		}
		return sp, nil
	}

	// --- Typing mode. ---
	if sp.typing {
		switch k.String() {
		case "esc":
			sp.typing = false
			sp.cmdInput.Blur()
			sp.sizeInput.Blur()
			return sp, nil
		case "enter":
			// commit and exit typing mode, but stay in pane edit
			sp.typing = false
			sp.syncEditFocus()
			return sp, nil
		}
		return sp.forwardPaneInput(msg)
	}

	// --- Navigation mode. ---
	switch k.String() {
	case "esc":
		sp.editing = false
		return sp, nil
	case "i":
		sp.typing = true
		sp.syncEditFocus()
		return sp, nil
	case "tab", "down", "j":
		sp.editFocus = (sp.editFocus + 1) % 2
		sp.syncEditFocus()
		return sp, nil
	case "shift+tab", "up", "k":
		sp.editFocus = (sp.editFocus + 1) % 2
		sp.syncEditFocus()
		return sp, nil
	case "enter":
		sp.panes[sp.editIdx].command = strings.TrimSpace(sp.cmdInput.Value())
		var sz int
		fmt.Sscanf(strings.TrimSpace(sp.sizeInput.Value()), "%d", &sz)
		if sz < 0 || sz >= 100 {
			sz = 0
		}
		sp.panes[sp.editIdx].size = sz
		sp.editing = false
		return sp, nil
	}
	return sp, nil
}

func (sp splitPaneEditor) forwardPaneInput(msg tea.Msg) (splitPaneEditor, tea.Cmd) {
	var cmd tea.Cmd
	switch sp.editFocus {
	case 0:
		sp.cmdInput, cmd = sp.cmdInput.Update(msg)
	case 1:
		sp.sizeInput, cmd = sp.sizeInput.Update(msg)
	}
	return sp, cmd
}

func (sp *splitPaneEditor) syncEditFocus() {
	sp.cmdInput.Blur()
	sp.sizeInput.Blur()
	if !sp.typing {
		return
	}
	switch sp.editFocus {
	case 0:
		sp.cmdInput.Focus()
	case 1:
		sp.sizeInput.Focus()
	}
}

func (sp splitPaneEditor) View() string {
	if sp.editing {
		var inner strings.Builder
		focusIndicator := func(idx int, label string) string {
			if sp.editFocus == idx {
				return promptStyle.Render("▶ " + label)
			}
			return dimStyle.Render("  " + label)
		}
		inner.WriteString(focusIndicator(0, "Command:") + "\n")
		inner.WriteString(sp.cmdInput.View() + "\n\n")
		inner.WriteString(focusIndicator(1, "Size % (0 = auto):") + "\n")
		inner.WriteString(sp.sizeInput.View() + "\n\n")
		if sp.typing {
			inner.WriteString(hintStyle.Render("typing mode  •  esc to stop typing  •  enter confirm"))
		} else {
			inner.WriteString(hintStyle.Render("j/k navigate  •  i to type  •  enter save pane  •  esc back"))
		}
		return wrapInBox("Edit Pane", inner.String())
	}

	var inner strings.Builder
	dirLabel := lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render(sp.direction)
	inner.WriteString(promptStyle.Render("Direction: ") + dirLabel + dimStyle.Render("  (r to toggle)") + "\n\n")
	inner.WriteString(promptStyle.Render("Panes:") + "\n")
	for i, p := range sp.panes {
		cmdStr := p.command
		if cmdStr == "" {
			cmdStr = "(shell)"
		}
		sizeStr := ""
		if p.size > 0 {
			sizeStr = dimStyle.Render(fmt.Sprintf(" [%d%%]", p.size))
		}
		label := fmt.Sprintf("pane %d: %s", i+1, cmdStr) + sizeStr
		if i == sp.cursor {
			inner.WriteString(activeStyle.Render("▶ "+fmt.Sprintf("pane %d: %s", i+1, cmdStr)) + sizeStr + "\n")
		} else {
			inner.WriteString(dimStyle.Render("  ") + lipgloss.NewStyle().Foreground(colorGray).Render(label) + "\n")
		}
	}
	inner.WriteString("\n" + hintStyle.Render("j/k navigate  •  enter edit  •  a add  •  d delete  •  r toggle dir  •  esc back"))
	return wrapInBox("Pane Layout", inner.String())
}

// ===========================================================================
// TEMPLATE EDITOR
// ===========================================================================

type templateEditorLayer int

const (
	telStepList   templateEditorLayer = iota
	telStepEdit                       // editing a single step
	telMetaEdit                       // editing name/description/flags
	telConfirmDel                     // confirm delete step
)

type stepDraft struct {
	stepType string // "command" or "builtin"
	run      string
	action   string
	path     string
	shell    bool
	isPost   bool // belongs to post_steps
}

type templateEditor struct {
	originalName string
	isNew        bool

	nameInput            textinput.Model
	descInput            textinput.Model
	interactive          bool
	createsProjectFolder bool

	steps  []stepDraft
	cursor int
	layer  templateEditorLayer

	// step edit fields
	editIdx        int
	editTypeCursor int // 0=command, 1=builtin
	editRunInput   textinput.Model
	editActCursor  int // 0=touch,1=mkdir,2=rm
	editPathInput  textinput.Model
	editIsPost     bool
	editFocus      int // which field is focused

	// meta edit
	metaFocus int // 0=name, 1=desc, 2=interactive, 3=createsFolder

	// typing=true means keystrokes go to the focused text input;
	// j/k/navigation only work when typing=false.
	typing bool

	statusMsg string
	result    editorResult
}

var builtinActions = []string{"touch", "mkdir", "rm"}

func newTemplateEditor(t tmpl.Template, isNew bool) templateEditor {
	nameInp := newSmallInput(t.Name, "template name", 40)
	descInp := newSmallInput(t.Description, "short description", 60)

	var steps []stepDraft
	for _, s := range t.Steps {
		steps = append(steps, stepDraftFromTemplate(s, false))
	}
	for _, s := range t.PostSteps {
		steps = append(steps, stepDraftFromTemplate(s, true))
	}

	return templateEditor{
		originalName:         t.Name,
		isNew:                isNew,
		nameInput:            nameInp,
		descInput:            descInp,
		interactive:          t.Interactive,
		createsProjectFolder: t.CreatesProjectFolder,
		steps:                steps,
	}
}

func stepDraftFromTemplate(s tmpl.TemplateStep, isPost bool) stepDraft {
	return stepDraft{
		stepType: s.Type,
		run:      s.Run,
		action:   s.Action,
		path:     s.Path,
		shell:    s.Shell,
		isPost:   isPost,
	}
}

func (te *templateEditor) toTemplate() tmpl.Template {
	t := tmpl.Template{
		Name:                 strings.TrimSpace(te.nameInput.Value()),
		Description:          strings.TrimSpace(te.descInput.Value()),
		Interactive:          te.interactive,
		CreatesProjectFolder: te.createsProjectFolder,
	}
	for _, sd := range te.steps {
		step := tmpl.TemplateStep{
			Type:   sd.stepType,
			Run:    sd.run,
			Action: sd.action,
			Path:   sd.path,
			Shell:  sd.shell,
		}
		if sd.isPost {
			t.PostSteps = append(t.PostSteps, step)
		} else {
			t.Steps = append(t.Steps, step)
		}
	}
	return t
}

func (te templateEditor) Update(msg tea.Msg) (templateEditor, tea.Cmd) {
	switch te.layer {
	case telStepEdit:
		return te.updateStepEdit(msg)
	case telMetaEdit:
		return te.updateMetaEdit(msg)
	case telConfirmDel:
		return te.updateConfirmDel(msg)
	default:
		return te.updateStepList(msg)
	}
}

func (te templateEditor) updateStepList(msg tea.Msg) (templateEditor, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return te, nil
	}
	listLen := len(te.steps) + 2 // +2: "add step", "add post-step"
	switch k.String() {
	case "ctrl+c", "esc":
		te.result = editorResultCancel
	case "ctrl+s":
		te.result = editorResultSave
	case "j", "down":
		if te.cursor < listLen-1 {
			te.cursor++
		}
	case "k", "up":
		if te.cursor > 0 {
			te.cursor--
		}
	case "m":
		// edit meta — start in navigation mode, no input focused
		te.metaFocus = 0
		te.typing = false
		te.nameInput.Blur()
		te.descInput.Blur()
		te.layer = telMetaEdit
	case "enter":
		te.statusMsg = ""
		if te.cursor < len(te.steps) {
			te.editIdx = te.cursor
			te.openStepEdit(te.steps[te.cursor])
		} else if te.cursor == len(te.steps) {
			// add normal step
			te.steps = append(te.steps, stepDraft{stepType: "command"})
			te.editIdx = len(te.steps) - 1
			te.openStepEdit(te.steps[te.editIdx])
		} else {
			// add post step
			te.steps = append(te.steps, stepDraft{stepType: "command", isPost: true})
			te.editIdx = len(te.steps) - 1
			te.openStepEdit(te.steps[te.editIdx])
		}
	case "d", "x":
		if te.cursor < len(te.steps) {
			te.layer = telConfirmDel
		}
	}
	return te, nil
}

func (te *templateEditor) openStepEdit(sd stepDraft) {
	te.editTypeCursor = 0
	if sd.stepType == "builtin" {
		te.editTypeCursor = 1
	}
	te.editRunInput = newSmallInput(sd.run, "command string e.g. go mod init {{project_name}}", 60)
	te.editActCursor = 0
	for i, a := range builtinActions {
		if a == sd.action {
			te.editActCursor = i
			break
		}
	}
	te.editPathInput = newSmallInput(sd.path, "relative path e.g. cmd/main.go", 50)
	te.editIsPost = sd.isPost
	te.editFocus = 0
	te.typing = false // always start in navigation mode
	te.layer = telStepEdit
}

func (te templateEditor) updateStepEdit(msg tea.Msg) (templateEditor, tea.Cmd) {
	// isTextFocus returns true when the cursor is on a field that has a text
	// input — those are the only fields that benefit from typing mode.
	isTextFocus := func() bool {
		return (te.editTypeCursor == 0 && te.editFocus == 1) ||
			(te.editTypeCursor == 1 && te.editFocus == 2)
	}

	k, ok := msg.(tea.KeyMsg)
	if !ok {
		if te.typing {
			return te.forwardStepInput(msg)
		}
		return te, nil
	}

	// --- Typing mode: all keys go straight to the focused input. ---
	if te.typing {
		switch k.String() {
		case "esc":
			te.typing = false
			te.editRunInput.Blur()
			te.editPathInput.Blur()
			return te, nil
		case "ctrl+s":
			te.result = editorResultSave
			return te, nil
		case "ctrl+c":
			te.result = editorResultCancel
			return te, nil
		}
		return te.forwardStepInput(msg)
	}

	// --- Navigation mode. ---
	switch k.String() {
	case "ctrl+c":
		te.result = editorResultCancel
		return te, nil
	case "esc":
		// discard this edit, revert steps if it was a new append
		if te.editIdx == len(te.steps)-1 && te.steps[te.editIdx].run == "" && te.steps[te.editIdx].path == "" {
			te.steps = te.steps[:te.editIdx]
		}
		te.layer = telStepList
		return te, nil
	case "ctrl+s":
		te.result = editorResultSave
		return te, nil
	case "i":
		if isTextFocus() {
			te.typing = true
			te.syncStepEditFocus()
		}
		return te, nil
	case "tab", "down", "j":
		te.editFocus = (te.editFocus + 1) % te.editFieldCount()
		te.syncStepEditFocus()
		return te, nil
	case "shift+tab", "up", "k":
		te.editFocus = (te.editFocus - 1 + te.editFieldCount()) % te.editFieldCount()
		te.syncStepEditFocus()
		return te, nil
	case "left", "h":
		if te.editFocus == 0 {
			te.editTypeCursor = (te.editTypeCursor + 1) % 2
			te.syncStepEditFocus()
			return te, nil
		}
		if te.editFocus == 1 && te.editTypeCursor == 1 {
			te.editActCursor = (te.editActCursor + len(builtinActions) - 1) % len(builtinActions)
			return te, nil
		}
	case "right", "l":
		if te.editFocus == 0 {
			te.editTypeCursor = (te.editTypeCursor + 1) % 2
			te.syncStepEditFocus()
			return te, nil
		}
		if te.editFocus == 1 && te.editTypeCursor == 1 {
			te.editActCursor = (te.editActCursor + 1) % len(builtinActions)
			return te, nil
		}
	case "enter", " ":
		lastField := te.editFieldCount() - 1
		if te.editFocus == lastField {
			te.editIsPost = !te.editIsPost
			return te, nil
		}
		return te.commitStepEdit()
	}
	return te, nil
}

// forwardStepInput routes a message to whichever text input is active.
func (te templateEditor) forwardStepInput(msg tea.Msg) (templateEditor, tea.Cmd) {
	var cmd tea.Cmd
	switch te.editTypeCursor {
	case 0:
		if te.editFocus == 1 {
			te.editRunInput, cmd = te.editRunInput.Update(msg)
		}
	case 1:
		if te.editFocus == 2 {
			te.editPathInput, cmd = te.editPathInput.Update(msg)
		}
	}
	return te, cmd
}

func (te templateEditor) editFieldCount() int {
	if te.editTypeCursor == 0 {
		return 3 // type, run, isPost
	}
	return 4 // type, action, path, isPost
}

func (te *templateEditor) syncStepEditFocus() {
	te.editRunInput.Blur()
	te.editPathInput.Blur()
	if !te.typing {
		return
	}
	if te.editTypeCursor == 0 && te.editFocus == 1 {
		te.editRunInput.Focus()
	}
	if te.editTypeCursor == 1 && te.editFocus == 2 {
		te.editPathInput.Focus()
	}
}

func (te templateEditor) commitStepEdit() (templateEditor, tea.Cmd) {
	sd := &te.steps[te.editIdx]
	if te.editTypeCursor == 0 {
		run := strings.TrimSpace(te.editRunInput.Value())
		if run == "" {
			te.statusMsg = errorStyle.Render("✗  command cannot be empty")
			return te, nil
		}
		sd.stepType = "command"
		sd.run = run
		sd.shell = false
		sd.action = ""
		sd.path = ""
	} else {
		path := strings.TrimSpace(te.editPathInput.Value())
		if path == "" {
			te.statusMsg = errorStyle.Render("✗  path cannot be empty")
			return te, nil
		}
		if filepath.IsAbs(filepath.FromSlash(path)) || strings.HasPrefix(path, "..") {
			te.statusMsg = errorStyle.Render("✗  path must be relative and cannot traverse upward")
			return te, nil
		}
		sd.stepType = "builtin"
		sd.action = builtinActions[te.editActCursor]
		sd.path = path
		sd.run = ""
	}
	sd.isPost = te.editIsPost
	te.statusMsg = ""
	te.layer = telStepList
	return te, nil
}

func (te templateEditor) updateMetaEdit(msg tea.Msg) (templateEditor, tea.Cmd) {
	isTextFocus := te.metaFocus == 0 || te.metaFocus == 1

	k, ok := msg.(tea.KeyMsg)
	if !ok {
		if te.typing && isTextFocus {
			return te.forwardMetaInput(msg)
		}
		return te, nil
	}

	// --- Typing mode. ---
	if te.typing {
		switch k.String() {
		case "esc":
			te.typing = false
			te.nameInput.Blur()
			te.descInput.Blur()
			return te, nil
		case "ctrl+s":
			te.result = editorResultSave
			return te, nil
		case "ctrl+c":
			te.result = editorResultCancel
			return te, nil
		}
		return te.forwardMetaInput(msg)
	}

	// --- Navigation mode. ---
	switch k.String() {
	case "ctrl+c":
		te.result = editorResultCancel
		return te, nil
	case "esc":
		te.layer = telStepList
		return te, nil
	case "ctrl+s":
		te.result = editorResultSave
		return te, nil
	case "i":
		if isTextFocus {
			te.typing = true
			te.syncMetaFocus()
		}
		return te, nil
	case "tab", "j", "down":
		te.metaFocus = (te.metaFocus + 1) % 4
		te.syncMetaFocus()
		return te, nil
	case "shift+tab", "k", "up":
		te.metaFocus = (te.metaFocus - 1 + 4) % 4
		te.syncMetaFocus()
		return te, nil
	case "enter", " ":
		switch te.metaFocus {
		case 2:
			te.interactive = !te.interactive
		case 3:
			te.createsProjectFolder = !te.createsProjectFolder
		}
		return te, nil
	}
	return te, nil
}

// forwardMetaInput routes a message to whichever meta text input is active.
func (te templateEditor) forwardMetaInput(msg tea.Msg) (templateEditor, tea.Cmd) {
	var cmd tea.Cmd
	switch te.metaFocus {
	case 0:
		te.nameInput, cmd = te.nameInput.Update(msg)
	case 1:
		te.descInput, cmd = te.descInput.Update(msg)
	}
	return te, cmd
}

func (te *templateEditor) syncMetaFocus() {
	te.nameInput.Blur()
	te.descInput.Blur()
	if !te.typing {
		return
	}
	switch te.metaFocus {
	case 0:
		te.nameInput.Focus()
	case 1:
		te.descInput.Focus()
	}
}

func (te templateEditor) updateConfirmDel(msg tea.Msg) (templateEditor, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return te, nil
	}
	switch k.String() {
	case "y", "enter":
		te.steps = append(te.steps[:te.cursor], te.steps[te.cursor+1:]...)
		if te.cursor >= len(te.steps) {
			te.cursor = len(te.steps) - 1
		}
		if te.cursor < 0 {
			te.cursor = 0
		}
		te.layer = telStepList
	case "n", "esc":
		te.layer = telStepList
	}
	return te, nil
}

// ---------------------------------------------------------------------------
// templateEditor.View
// ---------------------------------------------------------------------------

func (te templateEditor) View(w, h int) string {
	switch te.layer {
	case telStepEdit:
		return te.viewStepEdit(w, h)
	case telMetaEdit:
		return te.viewMetaEdit(w, h)
	case telConfirmDel:
		return te.viewConfirmDel(w, h)
	default:
		return te.viewStepList(w, h)
	}
}

func (te templateEditor) viewStepList(w, h int) string {
	title := "Edit Template"
	if te.isNew {
		title = "New Template"
	}

	var inner strings.Builder
	name := strings.TrimSpace(te.nameInput.Value())
	if name == "" {
		name = "(unnamed)"
	}
	inner.WriteString(promptStyle.Render("Template: ") +
		lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(name) +
		dimStyle.Render("  (m to edit meta)") + "\n\n")

	inner.WriteString(promptStyle.Render("Steps:") + "\n")
	for i, sd := range te.steps {
		label := stepLabel(sd)
		postTag := ""
		if sd.isPost {
			postTag = dimStyle.Render(" [post]")
		}
		if i == te.cursor {
			inner.WriteString(activeStyle.Render("▶ "+label) + postTag + "\n")
		} else {
			inner.WriteString("  " + lipgloss.NewStyle().Foreground(colorGray).Render(label) + postTag + "\n")
		}
	}

	addIdx := len(te.steps)
	addPostIdx := len(te.steps) + 1
	addLabel := lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("✦  add step")
	addPostLabel := lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("✦  add post-step")

	if te.cursor == addIdx {
		inner.WriteString(activeStyle.Render("▶ ") + addLabel + "\n")
	} else {
		inner.WriteString("  " + addLabel + "\n")
	}
	if te.cursor == addPostIdx {
		inner.WriteString(activeStyle.Render("▶ ") + addPostLabel + "\n")
	} else {
		inner.WriteString("  " + addPostLabel + "\n")
	}

	if te.statusMsg != "" {
		inner.WriteString("\n" + te.statusMsg)
	}
	inner.WriteString("\n\n" + hintStyle.Render("j/k navigate  •  enter edit  •  d delete  •  m meta  •  ctrl+s save  •  esc cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		RenderTitle(),
		promptStyle.Render("  "+title),
		"",
		wrapInBox(title, inner.String()),
	)
	return centerInTerminal(w, h, content)
}

func stepLabel(sd stepDraft) string {
	if sd.stepType == "builtin" {
		return fmt.Sprintf("[builtin] %s %s", sd.action, sd.path)
	}
	run := sd.run
	if len(run) > 55 {
		run = run[:52] + "..."
	}
	return fmt.Sprintf("[cmd] %s", run)
}

func (te templateEditor) viewStepEdit(w, h int) string {
	focusIndicator := func(idx int, label string) string {
		if te.editFocus == idx {
			return promptStyle.Render("▶ " + label)
		}
		return dimStyle.Render("  " + label)
	}

	// Step type selector
	typeLabels := []string{"command", "builtin"}
	var typeParts []string
	for i, tl := range typeLabels {
		if i == te.editTypeCursor {
			typeParts = append(typeParts, activeStyle.Render(" "+tl+" "))
		} else {
			typeParts = append(typeParts, dimStyle.Render(tl))
		}
	}
	typeLine := focusIndicator(0, "Type: ") + strings.Join(typeParts, dimStyle.Render("  ·  ")) +
		dimStyle.Render("  (h/l or ←/→ to switch)")

	postLabel := "add to post-steps: "
	if te.editIsPost {
		postLabel += lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("yes")
	} else {
		postLabel += dimStyle.Render("no")
	}

	var inner strings.Builder
	inner.WriteString(typeLine + "\n\n")

	lastFieldIdx := 0
	if te.editTypeCursor == 0 {
		inner.WriteString(focusIndicator(1, "Command:") + "\n")
		inner.WriteString(te.editRunInput.View() + "\n\n")
		inner.WriteString(dimStyle.Render("  Tip: use {{project_name}}, {{project_path}}, {{domain}}, {{root}}") + "\n\n")
		lastFieldIdx = 2 // post-step is field index 2 (editFieldCount=3, last=2)
	} else {
		// action selector — field index 1
		var actParts []string
		for i, a := range builtinActions {
			if i == te.editActCursor {
				actParts = append(actParts, activeStyle.Render(" "+a+" "))
			} else {
				actParts = append(actParts, dimStyle.Render(a))
			}
		}
		actLine := focusIndicator(1, "Action: ") + strings.Join(actParts, dimStyle.Render("  ·  ")) +
			dimStyle.Render("  (h/l to cycle)")
		inner.WriteString(actLine + "\n\n")
		// path input — field index 2
		inner.WriteString(focusIndicator(2, "Path (relative):") + "\n")
		inner.WriteString(te.editPathInput.View() + "\n\n")
		lastFieldIdx = 3 // post-step is field index 3 (editFieldCount=4, last=3)
	}

	postFocusIdx := lastFieldIdx
	postLineStr := focusIndicator(postFocusIdx, postLabel)
	if te.editFocus == postFocusIdx {
		postLineStr += dimStyle.Render("  (enter/space to toggle)")
	}
	inner.WriteString(postLineStr + "\n")

	if te.statusMsg != "" {
		inner.WriteString("\n" + te.statusMsg)
	}
	if te.typing {
		inner.WriteString("\n\n" + hintStyle.Render("typing mode  •  esc to stop typing  •  ctrl+s save all"))
	} else {
		inner.WriteString("\n\n" + hintStyle.Render("j/k navigate  •  i to type  •  h/l cycle type/action  •  enter confirm  •  esc back  •  ctrl+s save all"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		RenderTitle(),
		promptStyle.Render("  Edit Step"),
		"",
		wrapInBox("Edit Step", inner.String()),
	)
	return centerInTerminal(w, h, content)
}

func (te templateEditor) viewMetaEdit(w, h int) string {
	focusIndicator := func(idx int, label string) string {
		if te.metaFocus == idx {
			return promptStyle.Render("▶ " + label)
		}
		return dimStyle.Render("  " + label)
	}
	boolStr := func(b bool) string {
		if b {
			return lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("yes")
		}
		return dimStyle.Render("no")
	}

	var inner strings.Builder
	inner.WriteString(focusIndicator(0, "Name:") + "\n")
	inner.WriteString(te.nameInput.View() + "\n\n")
	inner.WriteString(focusIndicator(1, "Description:") + "\n")
	inner.WriteString(te.descInput.View() + "\n\n")

	interactiveLine := focusIndicator(2, "Interactive (uses PTY): ") + boolStr(te.interactive)
	if te.metaFocus == 2 {
		interactiveLine += dimStyle.Render("  (enter/space to toggle)")
	}
	inner.WriteString(interactiveLine + "\n\n")

	folderLine := focusIndicator(3, "Creates project folder: ") + boolStr(te.createsProjectFolder)
	if te.metaFocus == 3 {
		folderLine += dimStyle.Render("  (enter/space to toggle)")
	}
	inner.WriteString(folderLine + "\n")

	if te.typing {
		inner.WriteString("\n\n" + hintStyle.Render("typing mode  •  esc to stop typing  •  ctrl+s save all"))
	} else {
		inner.WriteString("\n\n" + hintStyle.Render("j/k navigate  •  i to type  •  enter/space toggle bools  •  esc back  •  ctrl+s save all"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		RenderTitle(),
		promptStyle.Render("  Template Meta"),
		"",
		wrapInBox("Template Meta", inner.String()),
	)
	return centerInTerminal(w, h, content)
}

func (te templateEditor) viewConfirmDel(w, h int) string {
	label := ""
	if te.cursor < len(te.steps) {
		label = stepLabel(te.steps[te.cursor])
	}
	inner := warningStyle.Render(fmt.Sprintf("Delete step \"%s\"?", label)) + "\n\n" +
		hintStyle.Render("y/enter — yes  •  n/esc — no")
	content := lipgloss.JoinVertical(lipgloss.Left,
		RenderTitle(),
		wrapInWarningBox("Confirm Delete", inner),
	)
	return centerInTerminal(w, h, content)
}

// ===========================================================================
// Persistence helpers
// ===========================================================================

func savePresetsFile(presets []preset.Preset) error {
	pf := preset.PresetFile{Presets: presets}
	data, err := json.MarshalIndent(pf, "", "    ")
	if err != nil {
		return err
	}
	path := filepath.Join(config.Dir(), "presets.json")
	return atomicWrite(path, data, 0o644)
}

func saveTemplatesFile(templates []tmpl.Template) error {
	tf := tmpl.TemplateFile{Templates: templates}
	data, err := json.MarshalIndent(tf, "", "    ")
	if err != nil {
		return err
	}
	path := filepath.Join(config.Dir(), "templates.json")
	return atomicWrite(path, data, 0o644)
}

func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-editor-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpName)
	}()
	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// ===========================================================================
// Shared input helper
// ===========================================================================

func newSmallInput(value, placeholder string, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetValue(value)
	ti.CharLimit = 300
	ti.Width = width
	ti.Cursor.Style = cursorStyle
	ti.PromptStyle = promptStyle
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorWhite)
	return ti
}
