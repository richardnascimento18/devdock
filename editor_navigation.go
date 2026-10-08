package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/preset"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/ui"
	"strings"
)

func (e *editorScreen) sizeInputs() {
	width := max(ui.ModalInnerWidth(e.termW)-2, 1)
	if e.termW >= 74 {
		_, _, detail := e.editorColumnWidths()
		width = max(detail-4, 1)
	}
	inputs := []*textinput.Model{&e.ce.defaultPreset, &e.pe.nameInput, &e.pe.editNameInp, &e.pe.editCmdInp, &e.pe.splitEditor.cmdInput, &e.pe.splitEditor.sizeInput, &e.te.nameInput, &e.te.descInput, &e.te.editRunInput, &e.te.editPathInput, &e.te.editOutputInput}
	for _, input := range inputs {
		input.Width = width
	}
}
func (e editorScreen) editorScrollLimit() int {
	if e.deleting {
		return ui.ModalScrollLimit(e.definitionBody(), "enter remove · esc cancel", e.termW, e.termH)
	}
	if e.termW >= 74 {
		return e.contextScrollLimit()
	}
	if e.layer == editorLayerList {
		labels, preview := e.collectionDetails()
		width := e.termW
		rows := max(e.termH-6, 1)
		if width >= 74 {
			width = max(min(width-min(32, width/3)-2, 70), 1)
		} else {
			preview = editorChoices(labels, e.cursor, max(min(rows/2, 5), 1), width) + "\n\n" + preview
		}
		return max(len(strings.Split(ansi.Hardwrap(preview, max(width, 1), true), "\n"))-rows, 0)
	}
	f := e.form()
	body := strings.Join(f.blocks, "\n\n")
	if f.dirty {
		body = "* Unsaved draft\n\n" + body
	}
	if f.message != "" {
		body = f.message + "\n\n" + body
	}
	return ui.ModalScrollLimit(body, f.hint, e.termW, e.termH)
}
func (e editorScreen) Update(msg tea.Msg) (editorScreen, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		e.termW, e.termH = size.Width, size.Height
		e.scroll = min(e.scroll, e.editorScrollLimit())
		e.sizeInputs()
		return e, nil
	}
	e.sizeInputs()
	if key, ok := msg.(tea.KeyMsg); ok {
		if e.discarding {
			switch key.String() {
			case "y":
				e.discarding = false
				e.contextMaster = false
				e.layer = editorLayerList
				e.scroll = 0
			case "esc", "n":
				e.discarding = false
			}
			return e, nil
		}
		if e.deleting {
			switch key.String() {
			case "pgdown":
				e.scroll = min(e.scroll+max(e.termH-7, 1), e.editorScrollLimit())
			case "pgup":
				e.scroll = max(e.scroll-max(e.termH-7, 1), 0)
			case "esc", "ctrl+c":
				e.deleting = false
				e.statusMsg = ""
			case "enter":
				return e.commitDelete(), nil
			}
			return e, nil
		}
		if e.termW >= 74 && !e.discarding && !e.deleting {
			if next, handled := e.contextNavigation(key); handled {
				return next, nil
			}
		}
		if key.String() == "pgdown" || key.String() == "pgup" {
			e.manualScroll = true
			step := max(e.termH-7, 1)
			if key.String() == "pgdown" {
				e.scroll = min(e.scroll+step, e.editorScrollLimit())
			} else {
				e.scroll = max(e.scroll-step, 0)
			}
			return e, nil
		}
		e.scroll = 0
		e.manualScroll = false
		leaving := key.String() == "ctrl+c" && e.layer != editorLayerList
		if key.String() == "esc" {
			switch e.layer {
			case editorLayerPreset:
				leaving = e.pe.layer == pelWindowList && !e.pe.typing
			case editorLayerTemplate:
				leaving = e.te.layer == telStepList
			case editorLayerConfig:
				leaving = !e.ce.typing
			}
		}
		if leaving && e.dirty() {
			e.discarding = true
			return e, nil
		}
		if e.layer == editorLayerList && (key.String() == "d" || key.String() == "x") {
			return e.beginDelete(), nil
		}
	}
	result, cmd := e.updateDraft(msg)
	result.sizeInputs()
	result.scroll = min(result.scroll, result.editorScrollLimit())
	return result, cmd
}
func (e editorScreen) beginDelete() editorScreen {
	switch e.tab {
	case editorTabPresets:
		if e.cursor >= len(e.presets) {
			return e
		}
		name := e.presets[e.cursor].Name
		if len(e.presets) <= 1 || e.cfg.DefaultPreset == name || e.cfg.DefaultPreset == "" && e.cursor == 0 {
			e.statusMsg = warningStyle.Render("! Choose another default preset in Settings before removing this preset.")
			return e
		}
		e.deleteName = name
	case editorTabTemplates:
		if e.cursor >= len(e.tmpls) {
			return e
		}
		e.deleteName = e.tmpls[e.cursor].Name
	default:
		return e
	}
	e.deleting = true
	e.statusMsg = ""
	return e
}
func (e editorScreen) commitDelete() editorScreen {
	if e.tab == editorTabPresets {
		if e.cursor >= len(e.presets) || e.presets[e.cursor].Name != e.deleteName {
			e.deleting = false
			return e
		}
		values := deepCopyPresets(e.presets)
		values = append(values[:e.cursor], values[e.cursor+1:]...)
		if err := preset.Save(config.Dir(), values); err != nil {
			e.statusMsg = errorStyle.Render(ui.SafeBlock("! Save failed: " + err.Error()))
			return e
		}
		e.presets = values
	} else {
		if e.cursor >= len(e.tmpls) || e.tmpls[e.cursor].Name != e.deleteName {
			e.deleting = false
			return e
		}
		values := deepCopyTemplates(e.tmpls)
		values = append(values[:e.cursor], values[e.cursor+1:]...)
		if err := tmpl.Save(config.Dir(), values); err != nil {
			e.statusMsg = errorStyle.Render(ui.SafeBlock("! Save failed: " + err.Error()))
			return e
		}
		e.tmpls = values
	}
	e.deleting = false
	e.revision++
	e.clampCursor()
	e.statusMsg = successStyle.Render(ui.SafeBlock("✓ Definition removed: " + e.deleteName))
	return e
}
