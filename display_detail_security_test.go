package main

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/richardnascimento18/devdock/internal/preset"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/ui"
)

func TestSelectedMetadataSafeDisplay(t *testing.T) {
	payloads := []string{
		"example\x1b]52;c;clipboard\a",
		"example\x1b]8;;https://example.invalid\x1b\\link\x1b]8;;\x1b\\",
		"example\x1b[31mred\x1b[0m\r\n\t\x00\u009b\u009d\u202e\u2066\u2069",
		"example\xff\xfe",
		"界é" + strings.Repeat("\u0301", 8000),
	}
	for i, value := range payloads {
		for _, reduced := range []bool{false, true} {
			m := fixtureModel(t, t.TempDir())
			m.termW, m.termH, m.motion.reduced = 120, 40, reduced
			m.presetPicker = newPresetPicker([]preset.Preset{{Name: value}}, "")
			m.templatePicker = newTemplatePicker([]tmpl.Template{{Name: value, Description: value}})
			m.templatePicker.cursor = 1
			for _, screen := range []appState{statePickPreset, statePickTemplate} {
				t.Run(fmt.Sprintf("payload-%d/reduced-%t/screen-%d", i, reduced, screen), func(t *testing.T) {
					m.state = screen
					body := m.currentModal().body
					assertSafeDisplay(t, body)
					assertSafeDisplay(t, m.View())
					if !utf8.ValidString(body) {
						t.Fatalf("payload %d: invalid display UTF-8", i)
					}
					if !strings.Contains(body, "Selected: "+ui.SafeText(value)) {
						t.Fatalf("payload %d: missing safe selected detail", i)
					}
					if strings.Count(body, "\u0301") > 64 {
						t.Fatal("unbounded combining metadata")
					}
				})
			}
			if m.presetPicker.Selected().Name != value || m.templatePicker.Selected().Description != value {
				t.Fatal("display changed stored metadata")
			}
		}
	}
}

func TestEmptySearchSafeDisplay(t *testing.T) {
	for _, value := range []string{"example\nrow\r\x1b]52;c;clipboard\a\u202e\xff", "界" + strings.Repeat("\u0301", 8000)} {
		for _, reduced := range []bool{false, true} {
			m := fixtureModel(t, t.TempDir())
			m.termW, m.termH, m.motion.reduced = 120, 40, reduced
			m.query.value, m.isFiltered = value, true
			output := m.viewProjects(110, 30)
			assertSafeDisplay(t, output)
			assertSafeDisplay(t, m.View())
			if !utf8.ValidString(output) || strings.Count(output, "\u0301") > 32 {
				t.Fatal("search display is not bounded valid Unicode")
			}
			if m.query.value != value {
				t.Fatal("display changed query")
			}
		}
	}
}
