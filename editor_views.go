package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/ui"
)

type editorForm struct {
	title         string
	blocks        []string
	focus         int
	hint, message string
	dirty         bool
}

func fieldLabel(label string, focused bool) string {
	if focused {
		return activeStyle.Render(ui.SafeText("> " + label))
	}
	return dimStyle.Render(ui.SafeText("  " + label))
}
func inputView(input textinput.Model, width int) string {
	input.Width = max(width-2, 1)
	return ui.InputView(input)
}
func editorFormView(form editorForm, width, height, scroll int, manual bool) string {
	title := form.title
	if form.dirty {
		title += " · Unsaved"
	}
	body := strings.Join(form.blocks, "\n\n")
	if form.dirty {
		body = warningStyle.Render("* Unsaved draft") + "\n\n" + body
	}
	if form.message != "" {
		body = form.message + "\n\n" + body
	}
	offset := scroll
	if !manual && form.focus > 0 && form.focus < len(form.blocks) {
		before := strings.Join(form.blocks[:form.focus], "\n\n") + "\n\n"
		if form.dirty {
			before = "* Unsaved draft\n\n" + before
		}
		if form.message != "" {
			before = form.message + "\n\n" + before
		}
		anchor := len(strings.Split(ansi.Hardwrap(before, ui.ModalInnerWidth(width), true), "\n")) - 1
		// Keep the active field visible while preserving nearby context.
		offset = max(anchor-max(height-8, 1)/2, 0)
	}
	return ui.ModalAt(title, body, form.hint, width, height, offset)
}
func (e editorScreen) dirty() bool {
	switch e.layer {
	case editorLayerPreset:
		return e.pe.dirty()
	case editorLayerTemplate:
		return e.te.dirty()
	case editorLayerConfig:
		return strings.TrimSpace(e.ce.defaultPreset.Value()) != e.cfg.DefaultPreset
	}
	return false
}
func (pe presetEditor) dirty() bool {
	p := pe.toPreset()
	if pe.isNew || !reflect.DeepEqual(p, pe.original) {
		return true
	}
	if pe.layer == pelWindowEdit || pe.layer == pelWindowSplit {
		w := pe.windows[pe.editIdx]
		if strings.TrimSpace(pe.editNameInp.Value()) != w.name || strings.TrimSpace(pe.editCmdInp.Value()) != w.command {
			return true
		}
		if pe.layer == pelWindowSplit {
			return pe.splitEditor.dirty()
		}
	}
	return false
}
func (sp splitPaneEditor) dirty() bool {
	if sp.original != nil && !reflect.DeepEqual(sp.toLayout(), *sp.original) {
		return true
	}
	if sp.editing && sp.editIdx < len(sp.panes) {
		p := sp.panes[sp.editIdx]
		size, err := parsePaneSize(sp.sizeInput.Value())
		return err != nil || strings.TrimSpace(sp.cmdInput.Value()) != p.command || size != p.size
	}
	return false
}
func (te templateEditor) dirty() bool {
	t := te.toTemplate()
	if te.isNew || !reflect.DeepEqual(t, te.original) {
		return true
	}
	if te.layer == telStepEdit && te.editIdx < len(te.steps) {
		s := te.steps[te.editIdx]
		kind := "command"
		if te.editTypeCursor == 1 {
			kind = "builtin"
		}
		return kind != s.stepType || strings.TrimSpace(te.editRunInput.Value()) != s.run || strings.TrimSpace(te.editPathInput.Value()) != s.path || strings.TrimSpace(te.editOutputInput.Value()) != s.output || te.editIsPost != s.isPost || builtinActions[te.editActCursor] != s.action && kind == "builtin"
	}
	return false
}
func editorChoices(labels []string, cursor, rows, width int) string {
	rows = max(rows, 1)
	start := min(max(cursor-rows/2, 0), max(len(labels)-rows, 0))
	var lines []string
	if start > 0 {
		lines = append(lines, dimStyle.Render(ui.SafeBlock(fmt.Sprintf("↑ %d earlier", start))))
	}
	for i := start; i < min(start+rows, len(labels)); i++ {
		lines = append(lines, ui.Fit(fieldLabel(labels[i], i == cursor), width, 1))
	}
	if start+rows < len(labels) {
		lines = append(lines, dimStyle.Render("↓ more"))
	}
	return strings.Join(lines, "\n")
}
func (e editorScreen) View() string {
	switch e.layer {
	case editorLayerConfig:
		e.tab = editorTabSettings
	case editorLayerPreset:
		e.tab = editorTabPresets
	case editorLayerTemplate:
		e.tab = editorTabTemplates
	}
	if e.discarding {
		return ui.Modal("Discard unsaved changes?", "The draft has changes.\n\nDiscard the draft and return to the collection?", "y discard · esc continue editing", e.termW, e.termH)
	}
	if e.deleting {
		return ui.ModalAt("Remove definition", e.definitionBody(), "enter remove · esc cancel", e.termW, e.termH, e.scroll)
	}
	if e.termW >= 74 {
		return e.viewContextEditor()
	}
	if e.layer == editorLayerList {
		return e.viewList()
	}
	form := e.form()
	form.title = e.editorBreadcrumb() + " › " + form.title
	return editorFormView(form, e.termW, e.termH, e.scroll, e.manualScroll)
}
func (e editorScreen) form() editorForm {
	switch e.layer {
	case editorLayerPreset:
		f := e.pe.form(e.termW, e.termH)
		if e.pe.originalName == e.cfg.DefaultPreset || e.cfg.DefaultPreset == "" && len(e.presets) > 0 && e.pe.originalName == e.presets[0].Name {
			f.title += " · Default"
		}
		return f
	case editorLayerTemplate:
		return e.te.form(e.termW, e.termH)
	}
	var names []string
	for _, p := range e.presets {
		names = append(names, ui.SafeText(p.Name))
	}
	return editorForm{title: "Default preset", blocks: []string{fieldLabel("Default preset", true) + "\n" + inputView(e.ce.defaultPreset, ui.ModalInnerWidth(e.termW)), "Available: " + strings.Join(names, ", ") + "\nBlank uses the first preset."}, hint: "i / enter edit · ctrl+s save · esc back", message: diagnosticView(e.ce.diagnostic), dirty: e.dirty()}
}
func (e editorScreen) viewList() string {
	w, h := e.termW, e.termH
	var tabs []string
	for i, name := range editorTabNames {
		tabs = append(tabs, fieldLabel(name, i == e.tab))
	}
	labels, preview := e.collectionDetails()
	selected := e.cursor
	rows := max(h-6, 1)
	var body string
	if w >= 74 {
		left := min(32, w/3)
		choices := editorChoices(labels, selected, max(rows-2, 1), left)
		wrapped := strings.Split(ansi.Hardwrap(preview, max(min(w-left-2, 70), 1), true), "\n")
		offset := min(e.scroll, max(len(wrapped)-rows, 0))
		body = lipgloss.JoinHorizontal(lipgloss.Top, paneContent(choices, left, rows), "  ", paneContent(strings.Join(wrapped[offset:], "\n"), w-left-2, rows))
	} else {
		choices := editorChoices(labels, selected, max(min(rows/2, 5), 1), w)
		combined := choices + "\n\n" + preview
		lines := strings.Split(ansi.Hardwrap(combined, w, true), "\n")
		offset := min(e.scroll, max(len(lines)-rows, 0))
		body = paneContent(strings.Join(lines[offset:], "\n"), w, rows)
	}
	hint := "j/k choose · enter edit · d remove · tab switch · pgup/pgdn details · esc back"
	if e.tab == editorTabSettings {
		hint = "j/k choose · enter edit · tab switch · pgup/pgdn roots · esc back"
	}
	if w < 74 {
		hint = "enter edit · tab section · esc back"
	}
	return strings.Join([]string{ui.Header("Configuration", w), ui.Fit(strings.Join(tabs, " · "), w, 1), "", body, ui.Footer(hint, diagnosticView(e.diagnostic), w)}, "\n")
}
func (pe presetEditor) form(w, h int) editorForm {
	width := ui.ModalInnerWidth(w)
	f := editorForm{title: "Edit Preset", dirty: pe.dirty(), message: diagnosticView(pe.diagnostic), hint: "j/k choose · i name · enter edit · d remove · ctrl+s save · esc back"}
	switch pe.layer {
	case pelWindowEdit:
		split := "disabled"
		if pe.editHasSplit {
			split = "enabled"
		}
		f.title = "Window"
		f.focus = pe.editFocus
		f.blocks = []string{fieldLabel("Window name", pe.editFocus == 0) + "\n" + inputView(pe.editNameInp, width), fieldLabel("Command (blank = shell)", pe.editFocus == 1) + "\n" + inputView(pe.editCmdInp, width), fieldLabel("Split layout: "+split, pe.editFocus == 2) + "\nenter toggles · e edits enabled layout"}
		f.hint = "tab field · i type · enter accept · ctrl+s save all · esc discard fields"
	case pelWindowSplit:
		f = pe.splitEditor.form(w, h)
		f.dirty = pe.dirty()
		f.hint += " · ctrl+s save all"
	case pelConfirmDelete:
		f.title = "Remove window"
		f.blocks = []string{ui.SafeText(pe.windows[pe.cursor].name)}
		f.hint = "enter remove · esc cancel"
	default:
		if pe.isNew {
			f.title = "New Preset"
		}
		labels := []string{}
		for _, win := range pe.windows {
			label := win.name
			if win.layout != nil {
				label += " · split"
			}
			labels = append(labels, label)
		}
		labels = append(labels, "+ Add window")
		f.blocks = []string{fieldLabel("Preset name", pe.typing) + "\n" + inputView(pe.nameInput, width), "Windows\n" + editorChoices(labels, pe.cursor, max(h-13, 1), width)}
		if pe.cursor < len(pe.windows) {
			win := pe.windows[pe.cursor]
			f.blocks = append(f.blocks, "Selected: "+ui.SafeText(win.name)+"\n"+ui.SafeBlock(win.command))
		}
	}
	if pe.typing || pe.layer == pelWindowSplit && pe.splitEditor.typing {
		f.hint = "Typing · esc stop · ctrl+s save"
	}
	return f
}
func (sp splitPaneEditor) form(w, h int) editorForm {
	width := ui.ModalInnerWidth(w)
	f := editorForm{title: "Pane layout", dirty: sp.dirty(), message: diagnosticView(sp.diagnostic), hint: "j/k choose · r direction · a add · d remove · enter accept · esc back"}
	if sp.editing {
		f.title = "Pane"
		f.focus = sp.editFocus
		f.blocks = []string{fieldLabel("Command", sp.editFocus == 0) + "\n" + inputView(sp.cmdInput, width), fieldLabel("Size % (0 = auto)", sp.editFocus == 1) + "\n" + inputView(sp.sizeInput, width)}
		f.hint = "tab field · i type · enter accept · esc discard fields"
		if sp.typing {
			f.hint = "Typing · esc stop · enter stop"
		}
		return f
	}
	labels := []string{}
	for i, p := range sp.panes {
		cmd := p.command
		if cmd == "" {
			cmd = "shell"
		}
		labels = append(labels, fmt.Sprintf("%d · %s · %d%%", i+1, cmd, p.size))
	}
	f.blocks = []string{"Direction: " + sp.direction, editorChoices(labels, sp.cursor, max(h-11, 1), width)}
	return f
}
func stepLabel(sd stepDraft) string {
	if sd.stepType == "builtin" {
		return "[builtin] " + ui.SafeText(sd.action) + " " + ui.SafeText(sd.path)
	}
	return "[command] " + ansi.Truncate(ui.SafeText(sd.run), 55, "…")
}
func (te templateEditor) form(w, h int) editorForm {
	width := ui.ModalInnerWidth(w)
	f := editorForm{title: "Edit Template", dirty: te.dirty(), message: diagnosticView(te.diagnostic), hint: "j/k step · enter edit · m metadata · d remove · ctrl+s save · esc back"}
	switch te.layer {
	case telMetaEdit:
		f.title = "Template metadata"
		f.focus = te.metaFocus
		f.blocks = []string{fieldLabel("Name", te.metaFocus == 0) + "\n" + inputView(te.nameInput, width), fieldLabel("Description", te.metaFocus == 1) + "\n" + inputView(te.descInput, width), fieldLabel(fmt.Sprintf("Interactive: %t", te.interactive), te.metaFocus == 2), fieldLabel(fmt.Sprintf("Creates project folder: %t", te.createsProjectFolder), te.metaFocus == 3)}
		f.hint = "tab field · i type · enter toggle · ctrl+s save · esc back"
	case telStepEdit:
		kind := "command"
		if te.editTypeCursor == 1 {
			kind = "builtin"
		}
		f.title = "Step"
		f.focus = te.editFocus
		f.blocks = []string{fieldLabel("Type: "+kind, te.editFocus == 0) + "\nh/l changes type"}
		if kind == "command" {
			f.blocks = append(f.blocks, fieldLabel("Command", te.editFocus == 1)+"\n"+inputView(te.editRunInput, width), fieldLabel("Output path (optional)", te.editFocus == 2)+"\n"+inputView(te.editOutputInput, width))
		} else {
			f.blocks = append(f.blocks, fieldLabel("Action: "+builtinActions[te.editActCursor], te.editFocus == 1)+"\nh/l changes action", fieldLabel("Relative path", te.editFocus == 2)+"\n"+inputView(te.editPathInput, width))
		}
		f.blocks = append(f.blocks, fieldLabel(fmt.Sprintf("Post-step: %t", te.editIsPost), te.editFocus == 3)+"\nenter toggles", "Variables: {{project_name}}, {{project_path}}, {{domain}}, {{root}}")
		f.hint = "tab field · i type · enter accept · ctrl+s save · esc discard fields"
	case telConfirmDel:
		f.title = "Remove step"
		f.blocks = []string{stepLabel(te.steps[te.cursor])}
		f.hint = "enter remove · esc cancel"
	default:
		if te.isNew {
			f.title = "New Template"
		}
		labels := []string{}
		for i, s := range te.steps {
			label := fmt.Sprintf("%d %s", i+1, stepLabel(s))
			if s.isPost {
				label += " · post"
			}
			labels = append(labels, label)
		}
		labels = append(labels, "+ Add step", "+ Add post-step")
		f.blocks = []string{te.nameInput.Value() + "\n" + te.descInput.Value(), "Steps\n" + editorChoices(labels, te.cursor, max(h-13, 1), width)}
		if te.cursor < len(te.steps) {
			s := te.steps[te.cursor]
			preview := s.run
			if s.stepType == "builtin" {
				preview = s.action + " " + s.path
			}
			f.blocks = append(f.blocks, "Selected step\n"+preview)
			if s.output != "" {
				f.blocks = append(f.blocks, "Output: "+s.output)
			}
		}
	}
	if te.typing {
		f.hint = "Typing · esc stop · ctrl+s save"
	}
	return f
}

func (e editorScreen) collectionDetails() ([]string, string) {
	labels := []string{}
	preview := ""
	selected := e.cursor
	switch e.tab {
	case editorTabSettings:
		value := e.cfg.DefaultPreset
		if value == "" {
			value = "first preset"
		}
		labels = []string{"Default preset · " + value, "Add root", "Remove root"}
		preview = "Configured roots:\n" + safeRoots(e.cfg.ActiveRoots())
		if len(e.cfg.ActiveRoots()) == 0 {
			preview = "No roots configured.\nChoose Add root to begin."
		}
	case editorTabPresets:
		for i, p := range e.presets {
			label := p.Name
			if p.Name == e.cfg.DefaultPreset || e.cfg.DefaultPreset == "" && i == 0 {
				label += " · default"
			}
			labels = append(labels, label)
		}
		labels = append(labels, "+ New preset")
		preview = "Create a preset to arrange tmux windows and pane commands."
		if selected < len(e.presets) {
			p := e.presets[selected]
			preview = ui.SafeText(p.Name) + "\n" + fmt.Sprintf("%d windows", len(p.Windows))
			for _, win := range p.Windows {
				preview += "\n\n" + ui.SafeText(win.Name)
				if win.Layout != nil {
					preview += " · split layout"
				}
				if win.Command != "" {
					preview += "\n" + ui.SafeBlock(win.Command)
				}
			}
		}
	case editorTabTemplates:
		for _, t := range e.tmpls {
			labels = append(labels, t.Name)
		}
		labels = append(labels, "+ New template")
		preview = "Create a template with command or filesystem steps."
		if selected < len(e.tmpls) {
			t := e.tmpls[selected]
			preview = ui.SafeText(t.Name) + "\n" + ui.SafeBlock(t.Description) + fmt.Sprintf("\n\n%d steps · %d post-steps", len(t.Steps), len(t.PostSteps))
			for _, s := range t.Steps {
				preview += "\n" + stepLabel(stepDraftFromTemplate(s, false))
			}
		}
	}
	return labels, preview
}

func (e editorScreen) definitionBody() string {
	return ui.SafeText(e.deleteName) + "\n\nRemove this definition from DevDock configuration.\n" + diagnosticView(e.diagnostic)
}
