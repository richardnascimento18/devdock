package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/richardnascimento18/devdock/internal/preset"
)

var leakedSGR = regexp.MustCompile(`\[38;(?:2|5);`)

func ansiIntegrityProblem(output string, literalUserText ...string) string {
	// Views only emit SGR, so any remaining ESC is an incomplete/invalid escape.
	if strings.Contains(sgrPattern.ReplaceAllString(output, ""), "\x1b") {
		return "malformed escape"
	}
	plain := ansi.Strip(output)
	// Literal, explicitly supplied user text is not renderer metadata.
	for _, literal := range literalUserText {
		plain = strings.ReplaceAll(plain, literal, "")
	}
	if leakedSGR.MatchString(plain) {
		return "visible SGR parameters"
	}
	return ""
}
func TestANSIIntegrityScreensAndPresetFragments(t *testing.T) {
	oldProfile, oldDark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
	defer lipgloss.SetColorProfile(oldProfile)
	defer lipgloss.SetHasDarkBackground(oldDark)
	for _, profile := range []termenv.Profile{termenv.TrueColor, termenv.ANSI256, termenv.ANSI, termenv.Ascii} {
		lipgloss.SetColorProfile(profile)
		lipgloss.SetHasDarkBackground(true)
		for name, m := range fixtureScreens() {
			for _, width := range []int{120, 80, 40, 24} {
				m.termW = width
				m.sizePresentation()
				if m.state == stateEditor {
					m.editorScr.termW = width
				}
				if problem := ansiIntegrityProblem(m.View()); problem != "" {
					t.Fatalf("%s width %d profile %v: %s", name, width, profile, problem)
				}
			}
		}
		ps := preset.Clone(preset.DefaultPresets)
		ps[0].Name = strings.Repeat("long-preset-", 15)
		picker := newPresetPicker(ps, ps[0].Name)
		for _, width := range []int{120, 80, 40, 24} {
			output := picker.View(width, 24)
			if problem := ansiIntegrityProblem(output); problem != "" {
				t.Fatalf("selected/default preset width %d: %s", width, problem)
			}
			if !strings.Contains(ansi.Strip(picker.content(120, 24).body), "(default)") {
				t.Fatal("default marker lost")
			}
		}
	}
}
func TestANSIIntegrityDetectorDistinguishesUserText(t *testing.T) {
	for _, broken := range []string{"[38;2;1;2;3mhello", "[38;5;81mhello", "\x1b[38;2;1;2", "\x1b[broken"} {
		if ansiIntegrityProblem(broken) == "" {
			t.Fatalf("missed %q", broken)
		}
	}
	literal := "example [38;2;100;150;200m"
	if ansiIntegrityProblem(literal, literal) != "" {
		t.Fatal("literal user text misclassified")
	}
	if ansiIntegrityProblem("\x1b[38;2;100;150;200mhello\x1b[0m") != "" {
		t.Fatal("valid SGR rejected")
	}
}
