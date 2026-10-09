package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/app"
)

type configurationEditor struct {
	defaultPreset textinput.Model
	typing        bool
	diagnostic    app.Diagnostic
}

func newConfigurationEditor(value string) configurationEditor {
	input := newSmallInput(value, "blank selects the first preset", 45)
	input.CharLimit = 0
	return configurationEditor{defaultPreset: input}
}

func (e editorScreen) updateConfigurationEditor(msg tea.Msg) (editorScreen, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		e.termW, e.termH = size.Width, size.Height
		return e, nil
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			e.layer = editorLayerList
			return e, nil
		case "esc":
			if e.ce.typing {
				e.ce.typing = false
				e.ce.defaultPreset.Blur()
			} else {
				e.layer = editorLayerList
			}
			return e, nil
		case "ctrl+s":
			proposed := e.cfg.Clone()
			proposed.DefaultPreset = strings.TrimSpace(e.ce.defaultPreset.Value())
			if proposed.DefaultPreset != "" {
				found := false
				for _, p := range e.presets {
					found = found || p.Name == proposed.DefaultPreset
				}
				if !found {
					e.ce.diagnostic = app.Diagnostic{Severity: app.Error, Summary: fmt.Sprintf("unknown preset %q; choose an existing preset or leave blank", proposed.DefaultPreset)}
					return e, nil
				}
			}
			committed, changed, err := e.preferences.CommitConfig(e.cfg, proposed)
			if err != nil {
				e.ce.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "save failed: " + err.Error()}
				return e, nil
			}
			e.cfg = committed
			if changed {
				e.revision++
				e.diagnostic = app.Diagnostic{Severity: app.Success, Summary: "✓  configuration saved"}
			} else {
				e.diagnostic = app.Diagnostic{Severity: app.Info, Summary: "Configuration unchanged"}
			}
			e.layer = editorLayerList
			return e, nil
		case "i", "enter":
			if !e.ce.typing {
				e.ce.typing = true
				return e, e.ce.defaultPreset.Focus()
			}
		}
	}
	if e.ce.typing {
		var cmd tea.Cmd
		e.ce.defaultPreset, cmd = e.ce.defaultPreset.Update(msg)
		return e, cmd
	}
	return e, nil
}
