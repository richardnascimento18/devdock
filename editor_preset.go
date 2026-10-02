package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/richardnascimento18/devdock/internal/preset"
	"strings"
)

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
	original     preset.Preset
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
	nameInp := transparentInput()
	nameInp.Prompt = "  "
	nameInp.SetValue(p.Name)
	nameInp.CharLimit = 60
	nameInp.Width = 40
	nameInp.Cursor.Style = cursorStyle
	nameInp.PromptStyle = promptStyle
	nameInp.TextStyle = lipgloss.NewStyle().Foreground(theme.Primary)

	var windows []windowDraft
	for _, w := range p.Windows {
		var layoutCopy *preset.PaneLayout
		if w.Layout != nil {
			layoutCopy = preset.CloneLayout(w.Layout)
		}
		windows = append(windows, windowDraft{name: w.Name, command: w.Command, layout: layoutCopy})
	}
	if len(windows) == 0 {
		windows = []windowDraft{{name: "terminal"}}
	}

	return presetEditor{
		original:     preset.Clone([]preset.Preset{p})[0],
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
			if pe.splitEditor.editing {
				var cmd tea.Cmd
				pe.splitEditor, cmd = pe.splitEditor.Update(msg)
				return pe, cmd
			}
			pe.layer = pelWindowEdit
			return pe, nil
		case "ctrl+s":
			pe.result = editorResultSave
			return pe, nil
		case "enter":
			if pe.splitEditor.editing {
				break
			}
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
