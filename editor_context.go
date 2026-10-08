package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/ui"
)

func (e editorScreen) editorColumnWidths() (int, int, int) {
	left := min(28, e.termW/4)
	if e.termW >= 110 && e.tab != editorTabSettings {
		middle := min(32, e.termW/4)
		return left, middle, max(e.termW-left-middle-4, 1)
	}
	return left, 0, max(e.termW-left-2, 1)
}
func (e editorScreen) editorBreadcrumb() string {
	switch e.layer {
	case editorLayerPreset:
		title := "Presets › " + e.pe.nameInput.Value()
		if e.pe.layer != pelWindowList && e.pe.editIdx < len(e.pe.windows) {
			title += " › " + e.pe.windows[e.pe.editIdx].name
		}
		return title
	case editorLayerTemplate:
		title := "Templates › " + e.te.nameInput.Value()
		if e.te.layer == telStepEdit {
			title += fmt.Sprintf(" › step %d", e.te.editIdx+1)
		}
		return title
	case editorLayerConfig:
		return "Settings"
	}
	return editorTabNames[e.tab]
}

// Preview uses independent draft values. Merely browsing never changes live
// configuration or the identity of the active editor object.
func (e editorScreen) contextDraft() editorScreen {
	if e.layer != editorLayerList {
		return e
	}
	if e.tab == editorTabPresets && e.cursor < len(e.presets) {
		e.layer = editorLayerPreset
		e.pe = newPresetEditor(e.presets[e.cursor], false)
	} else if e.tab == editorTabTemplates && e.cursor < len(e.tmpls) {
		e.layer = editorLayerTemplate
		e.te = newTemplateEditor(e.tmpls[e.cursor], false)
	}
	return e
}
func (e editorScreen) componentDetails(width, rows int) (string, string) {
	title := "Windows"
	labels := []string{}
	cursor := 0
	switch e.layer {
	case editorLayerPreset:
		for _, win := range e.pe.windows {
			label := win.name
			if win.layout != nil {
				label += " · split"
			}
			labels = append(labels, label)
		}
		labels = append(labels, "+ Add window")
		cursor = e.pe.cursor
	case editorLayerTemplate:
		title = "Steps"
		for i, step := range e.te.steps {
			label := fmt.Sprintf("%d %s", i+1, stepLabel(step))
			if step.isPost {
				label += " · post"
			}
			labels = append(labels, label)
		}
		labels = append(labels, "+ Add step", "+ Add post-step")
		cursor = e.te.cursor
	}
	return title, editorChoices(labels, cursor, rows, width)
}
func (e editorScreen) contextForm(detailWidth int) editorForm {
	draft := e.contextDraft()
	if draft.layer == editorLayerList {
		_, preview := draft.collectionDetails()
		if draft.tab == editorTabSettings {
			preview = promptStyle.Render("Configured roots:") + "\n" + safeRoots(draft.cfg.ActiveRoots())
		}
		return editorForm{title: "Details", blocks: []string{preview}}
	}
	draft.termW = detailWidth + 6
	form := draft.form()
	if draft.layer == editorLayerPreset && draft.pe.layer == pelWindowList {
		form.blocks = []string{fieldLabel("Preset name", draft.pe.typing) + "\n" + inputView(draft.pe.nameInput, detailWidth)}
		if draft.pe.cursor < len(draft.pe.windows) {
			win := draft.pe.windows[draft.pe.cursor]
			command := win.command
			if command == "" {
				command = "shell"
			}
			form.blocks = append(form.blocks, promptStyle.Render("Selected window")+"\n"+ui.SafeText(win.name), promptStyle.Render("Command")+"\n"+ui.SafeBlock(command))
		}
	} else if draft.layer == editorLayerTemplate && draft.te.layer == telStepList {
		form.blocks = []string{selectedStyle.Render(ui.SafeBlock(draft.te.nameInput.Value())), dimStyle.Render(ui.SafeBlock(draft.te.descInput.Value()))}
		if draft.te.cursor < len(draft.te.steps) {
			step := draft.te.steps[draft.te.cursor]
			form.blocks = append(form.blocks, promptStyle.Render("Selected step")+"\n"+stepLabel(step))
			if step.output != "" {
				form.blocks = append(form.blocks, "Output · "+ui.SafeText(step.output))
			}
		}
	}
	if e.layer == editorLayerList {
		form.dirty = false
		form.hint = "enter edit selected definition"
	}
	return form
}
func contextFormLines(form editorForm, width int) []string {
	return strings.Split(ansi.Hardwrap(strings.Join(form.blocks, "\n\n"), max(width, 1), true), "\n")
}
func (e editorScreen) contextScrollLimit() int {
	_, middle, detail := e.editorColumnWidths()
	form := e.contextForm(detail)
	lines := contextFormLines(form, detail)
	if middle == 0 && e.tab != editorTabSettings && (e.layer == editorLayerList || e.layer == editorLayerPreset && e.pe.layer == pelWindowList || e.layer == editorLayerTemplate && e.te.layer == telStepList) {
		draft := e.contextDraft()
		_, components := draft.componentDetails(detail, max(e.termH/3, 1))
		lines = append(strings.Split(components, "\n"), lines...)
	}
	return max(len(lines)-max(e.termH-9, 1), 0)
}
func (e editorScreen) viewContextEditor() string {
	left, middle, detail := e.editorColumnWidths()
	rows := max(e.termH-6, 1)
	labels, _ := e.collectionDetails()
	if e.layer != editorLayerList {
		name := ""
		if e.layer == editorLayerPreset {
			name = e.pe.originalName
		}
		if e.layer == editorLayerTemplate {
			name = e.te.originalName
		}
		for i, label := range labels {
			if label == name || strings.HasPrefix(label, name+" · ") && name != "" {
				labels[i] += " · draft"
				break
			}
		}
	}
	collectionFocus := e.layer == editorLayerList || e.contextMaster
	collection := ui.PaneTitle(editorTabNames[e.tab], collectionFocus, left) + "\n" + editorChoices(labels, e.cursor, max(rows-2, 1), left)
	draft := e.contextDraft()
	form := e.contextForm(detail)
	title := form.title
	if e.layer == editorLayerList {
		title = "Preview"
	}
	lines := contextFormLines(form, detail)
	componentTitle, components := draft.componentDetails(func() int {
		if middle > 0 {
			return middle
		}
		return detail
	}(), max(rows-2, 1))
	if middle == 0 && e.tab != editorTabSettings && (e.layer == editorLayerList || e.layer == editorLayerPreset && e.pe.layer == pelWindowList || e.layer == editorLayerTemplate && e.te.layer == telStepList) {
		_, components = draft.componentDetails(detail, max(min(rows/3, 5), 1))
		lines = append(strings.Split(promptStyle.Render(ui.SafeBlock(componentTitle))+"\n"+components+"\n", "\n"), lines...)
	}
	bodyRows := max(rows-1, 1)
	status := ""
	if form.dirty {
		status = warningStyle.Render("* Unsaved draft")
	}
	if form.message != "" {
		status = form.message
	}
	if status != "" {
		bodyRows = max(bodyRows-2, 1)
	}
	offset := min(e.scroll, max(len(lines)-bodyRows, 0))
	if !e.manualScroll && form.focus > 0 && form.focus < len(form.blocks) {
		anchor := len(contextFormLines(editorForm{blocks: form.blocks[:form.focus]}, detail))
		offset = min(max(anchor-bodyRows/2, 0), max(len(lines)-bodyRows, 0))
	}
	properties := ui.PaneTitle(title, !collectionFocus && e.layer != editorLayerList, detail) + "\n"
	if status != "" {
		properties += ui.Fit(status, detail, 2) + "\n"
	}
	properties += strings.Join(lines[offset:min(offset+bodyRows, len(lines))], "\n")
	panes := []string{paneContent(collection, left, rows), "  "}
	if middle > 0 {
		panes = append(panes, paneContent(ui.PaneTitle(componentTitle, !collectionFocus && (draft.layer == editorLayerPreset && draft.pe.layer == pelWindowList || draft.layer == editorLayerTemplate && draft.te.layer == telStepList), middle)+"\n"+components, middle, rows), "  ")
	}
	panes = append(panes, paneContent(properties, detail, rows))
	hint := form.hint
	if e.layer == editorLayerList {
		hint = "j/k choose · enter edit · d remove · tab section · esc back"
	}
	hint += " · alt+1 collection · alt+2 items · alt+3 fields"
	if e.termW < 110 {
		hint = "enter / i edit · ctrl+s save · esc back · alt+1 collection · alt+2 items · pgup/pgdn"
	}
	tabs := []string{}
	for i, name := range editorTabNames {
		tabs = append(tabs, fieldLabel(name, e.tab == i))
	}
	header := ui.Header("Configuration · "+e.editorBreadcrumb(), e.termW)
	return strings.Join([]string{header, ui.Fit(strings.Join(tabs, " · "), e.termW, 1), "", lipgloss.JoinHorizontal(lipgloss.Top, panes...), ui.Footer(hint, e.statusMsg, e.termW)}, "\n")
}

// Direct navigation reuses existing validation and draft guards. Property edits
// stay in the detail column; collection context remains visible throughout.
func (e editorScreen) contextNavigation(key tea.KeyMsg) (editorScreen, bool) {
	switch key.String() {
	case "alt+1":
		if e.layer == editorLayerList {
			return e, true
		}
		e.contextMaster = true
		// Keep the entire draft (including unfinished field values) until the user
		// explicitly leaves it. Enter on the collection invokes the discard guard.
		return e, true
	case "alt+3":
		e.contextMaster = false
		return e, true
	case "alt+2":
		e.contextMaster = false
		if e.layer == editorLayerPreset && e.pe.layer == pelWindowEdit {
			e.pe.typing = false
			e.pe.editFocus = 0
			e.pe, _ = e.pe.Update(tea.KeyMsg{Type: tea.KeyEnter})
		} else if e.layer == editorLayerTemplate && e.te.layer == telStepEdit {
			e.te.typing = false
			e.te, _ = e.te.commitStepEdit()
		}
		return e, true
	}
	if !e.contextMaster || e.layer == editorLayerList {
		return e, false
	}
	switch key.String() {
	case "enter", "esc":
		if e.dirty() {
			e.discarding = true
		} else {
			e.layer = editorLayerList
			e.contextMaster = false
		}
		return e, true
	case "j", "down", "k", "up":
		// Browsing an alternative does not replace the active object's identity.
		if key.String() == "j" || key.String() == "down" {
			e.cursor++
		} else {
			e.cursor--
		}
		e.clampCursor()
		return e, true
	}
	return e, false
}

func safeRoots(roots []string) string {
	display := make([]string, len(roots))
	for i, root := range roots {
		display[i] = ui.SafeText(root)
	}
	return strings.Join(display, "\n")
}
