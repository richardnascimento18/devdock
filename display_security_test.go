package main

import (
	"strings"
	"testing"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/preset"
	"github.com/richardnascimento18/devdock/internal/state"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/ui"
)

func assertSafeDisplay(t *testing.T, output string) {
	t.Helper()
	plain := sgrPattern.ReplaceAllString(output, "")
	if strings.ContainsAny(plain, "\x1b\a\r\u009b\u009d\u202e\u2066\u2069") {
		t.Fatalf("active external terminal control in display: %q", plain)
	}
}

func TestHostileDisplayDoesNotChangeWorkspaceIdentity(t *testing.T) {
	name := "project\x1b]52;c;payload\a\x1b]8;;https://example.invalid\x1b\\link\u202e"
	path := "/workspace/" + name
	p := core.Project{Name: name, Path: path, Location: core.Location{Root: "/workspace", Domain: "domain\u009d\r"}, Languages: []string{name}, GitHubRepo: name}
	m := newModel([]core.Project{p}, config.Config{Roots: []string{"/workspace"}}, preset.DefaultPresets, tmpl.DefaultTemplates, state.UIState{})
	for _, width := range []int{120, 80, 40, 24} {
		m.termW, m.termH = width, 40
		m.sizePresentation()
		assertSafeDisplay(t, m.View())
		assertSafeDisplay(t, strings.Join(m.inspectorLines(width), "\n"))
	}
	if m.navigation.projects[0].Name != name || m.navigation.projects[0].Path != path {
		t.Fatal("display sanitization changed identity")
	}
	assertSafeDisplay(t, githubItem{repo: gh.Repo{Name: name}}.Title())
	assertSafeDisplay(t, tmuxSessionItem{name: name}.Title())
}

func TestHostileAuxiliaryDisplays(t *testing.T) {
	payload := "value\x1b]52;c;clipboard\a\x1b]8;;https://example.invalid\x1b\\\u202e"
	for _, reduced := range []bool{false, true} {
		for _, frame := range []int{0, 1, 16} {
			assertSafeDisplay(t, ui.Activity(payload, frame, reduced))
		}
	}
	e := newEditorScreen([]preset.Preset{{Name: payload, Windows: []preset.Window{{Name: payload, Command: payload}}}}, []tmpl.Template{{Name: payload, Description: payload}}, 120, 40)
	for _, width := range []int{120, 80, 40} {
		e.termW = width
		assertSafeDisplay(t, e.View())
		e.tab = editorTabTemplates
		assertSafeDisplay(t, e.View())
		e.tab = editorTabSettings
		e.cfg.Roots = []string{"/workspace/" + payload}
		assertSafeDisplay(t, e.View())
		e.tab = editorTabPresets
	}
	input := newInputScreen(payload, payload, "enter confirm")
	input.input.SetValue(payload)
	input.err = payload
	assertSafeDisplay(t, input.View(120, 40))
	picker := genericPickerScreen{title: payload, options: []string{payload, payload}}
	assertSafeDisplay(t, picker.View(120, 40))
	palette := paletteScreen{actions: []actionBinding{{label: payload}, {label: payload}}, visible: []int{0, 1}, input: transparentInput()}
	assertSafeDisplay(t, palette.View(120, 40))
	pty := newPTYScreen(120, 40, nil, "", tmpl.Vars{}, nil, "", gh.Repo{})
	defer pty.cancel()
	pty.ingestPTYData([]byte(payload))
	assertSafeDisplay(t, pty.View())
}

func TestExternalSingleLineLabelsEncodeNewlines(t *testing.T) {
	name := "example\nspoofed-row\r\t"
	project := core.Project{Name: name, Location: core.Location{Root: "/workspace", Domain: name}}
	for _, value := range []string{
		(item{project: project}).Title(),
		(favoriteItem{project: project}).Title(),
		(githubItem{repo: gh.Repo{Name: name}}).Title(),
		(tmuxSessionItem{name: name}).Title(),
		(groupItem{name: name, location: project.Location}).Title(),
		ui.Activity(name, 1, false),
	} {
		assertSafeDisplay(t, value)
		if strings.Contains(value, "\n") {
			t.Fatal("external label injected a row")
		}
	}
}
