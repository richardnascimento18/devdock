package main

import (
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/core"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/richardnascimento18/devdock/internal/preset"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/ui"
)

// ---------------------------------------------------------------------------
// inputScreen
// ---------------------------------------------------------------------------

type inputScreen struct {
	modalScreen
	title      string
	targetPath string
	input      textinput.Model
	err        string
	hint       string
}

func newInputScreen(title, placeholder, hint string) inputScreen {
	ti := transparentInput()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 52
	ti.Cursor.Style = cursorStyle
	ti.PromptStyle = promptStyle
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Primary)
	s := inputScreen{title: title, input: ti, hint: hint}
	if filepath.IsAbs(placeholder) {
		s.targetPath = placeholder
	}
	return s
}

func (s inputScreen) Update(msg tea.Msg) (inputScreen, tea.Cmd) {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s inputScreen) content(termW, termH int) modalContent {
	s.input.Width = max(ui.ModalInnerWidth(termW)-2, 1)
	var inner strings.Builder
	inner.WriteString(ui.InputView(s.input) + "\n")
	if s.err != "" {
		inner.WriteString(errorStyle.Render(ui.SafeBlock("! "+s.err)) + "\n")
	}
	if s.targetPath != "" {
		inner.WriteString("\n" + warningStyle.Render(ui.SafeBlock("Filesystem target ["+core.PathID(s.targetPath)+"]:")) + "\n" + ui.SafeText(s.targetPath))
	}
	return modalContent{title: s.title, body: inner.String(), hint: s.hint}
}

// ---------------------------------------------------------------------------
// yesNoScreen
// ---------------------------------------------------------------------------

type yesNoScreen struct {
	modalScreen
	title  string
	cursor int
	err    string
}

func newYesNoScreen(title string) yesNoScreen { return yesNoScreen{title: title} }
func (y yesNoScreen) IsYes() bool             { return y.cursor == 0 }

func (y yesNoScreen) content(termW, termH int) modalContent {
	var inner strings.Builder
	for i, opt := range []string{"  Yes", "  No"} {
		if i == y.cursor {
			inner.WriteString(activeStyle.Render(ui.SafeBlock(opt)) + "\n")
		} else {
			inner.WriteString(lipgloss.NewStyle().Foreground(theme.Secondary).Render(ui.SafeBlock(opt)) + "\n")
		}
	}
	if y.err != "" {
		inner.WriteString("\n" + errorStyle.Render(ui.SafeBlock("✗  "+y.err)))
	}
	return modalContent{title: y.title, body: inner.String(), hint: "↑/↓ choose · enter confirm · esc cancel"}
}

// ---------------------------------------------------------------------------
// spinnerScreen
// ---------------------------------------------------------------------------

type spinnerScreen struct{ message string }

func newSpinnerScreen(message string) spinnerScreen { return spinnerScreen{message: message} }
func (s spinnerScreen) View(w, h, frame int, reduced, cancel bool) string {
	hint := "Operation in progress"
	if cancel {
		hint = "esc cancel"
	}
	return ui.Modal("Working", ui.Activity(s.message, frame, reduced), hint, w, h)
}

// // githubAuthScreen
// ---------------------------------------------------------------------------

type githubAuthScreen struct {
	modalScreen
	userCode        string
	verificationURI string
	err             string
	done            bool
}

func (g githubAuthScreen) content(termW, termH int) modalContent {
	var inner strings.Builder
	inner.WriteString(lipgloss.NewStyle().Foreground(theme.Secondary).Render("1. Open this URL in your browser:") + "\n")
	inner.WriteString("   " + lipgloss.NewStyle().Foreground(theme.Info).Underline(true).Render(ui.SafeBlock(g.verificationURI)) + "\n\n")
	inner.WriteString(lipgloss.NewStyle().Foreground(theme.Secondary).Render("2. Enter this one-time code:") + "\n")
	inner.WriteString("   " + lipgloss.NewStyle().
		Bold(true).Foreground(theme.Favorite).Padding(0, 2).
		Render(ui.SafeBlock(ui.SafeText(g.userCode))) + "\n\n")
	if g.done {
		inner.WriteString(successStyle.Render("✓  Authorized! Loading your repositories..."))
	} else {
		inner.WriteString(lipgloss.NewStyle().Foreground(theme.Muted).Render("Waiting for authorization..."))
	}
	if g.err != "" {
		inner.WriteString("\n\n" + errorStyle.Render(ui.SafeBlock("✗  "+g.err)))
	}
	return modalContent{title: "Connect GitHub Account", body: inner.String(), hint: "esc cancel"}
}

// ---------------------------------------------------------------------------
// genericPickerScreen
// ---------------------------------------------------------------------------

type genericPickerScreen struct {
	modalScreen
	title      string
	options    []string
	identities []string
	cursor     int
	disabled   map[string]bool
	err        string
	hint       string
}

func newGenericPicker(title string, options []string, hint string) genericPickerScreen {
	return genericPickerScreen{title: title, options: options, disabled: map[string]bool{}, hint: hint}
}

func (g genericPickerScreen) content(termW, termH int) modalContent {
	var inner strings.Builder
	rows := max(termH-10, 1)
	start := max(0, g.cursor-rows/2)
	if start+rows > len(g.options) {
		start = max(0, len(g.options)-rows)
	}
	end := min(start+rows, len(g.options))
	for i := start; i < end; i++ {
		o := g.options[i]
		identity := ""
		if i < len(g.identities) {
			identity = g.identities[i] + " "
		}
		available := max(ui.ModalInnerWidth(termW)-2-ansi.StringWidth(identity), 1)
		if ansi.StringWidth(o) > available {
			o = ansi.TruncateLeft(o, ansi.StringWidth(o)-available+1, "…")
		}

		var line string
		o = identity + o
		switch {
		case g.disabled[g.options[i]]:
			line = dimStyle.Render(ui.SafeBlock("  "+o)) + dimStyle.Render(" (already exists)")
		case i == g.cursor:
			line = activeStyle.Render(ui.SafeBlock("▶ " + o))
		default:
			line = lipgloss.NewStyle().Foreground(theme.Secondary).Render(ui.SafeBlock("  " + o))
		}
		inner.WriteString(line + "\n")
	}
	if g.cursor >= 0 && g.cursor < len(g.options) {
		inner.WriteString("\n" + dimStyle.Render("Selected location:") + "\n" + ui.SafeText(g.options[g.cursor]) + "\n")
	}
	if len(g.options) > rows {
		inner.WriteString(dimStyle.Render(ui.SafeBlock(fmt.Sprintf("%d/%d", g.cursor+1, len(g.options)))) + "\n")
	}
	if g.err != "" {
		inner.WriteString("\n" + errorStyle.Render(ui.SafeBlock("✗  "+g.err)))
	}
	return modalContent{title: g.title, body: inner.String(), hint: g.hint}
}

// ---------------------------------------------------------------------------
// presetPickerScreen
// ---------------------------------------------------------------------------

type presetPickerScreen struct {
	modalScreen
	presets     []preset.Preset
	cursor      int
	defaultName string
}

func newPresetPicker(presets []preset.Preset, defaultName string) presetPickerScreen {
	cursor := 0
	for i, p := range presets {
		if p.Name == defaultName {
			cursor = i
			break
		}
	}
	return presetPickerScreen{presets: presets, cursor: cursor, defaultName: defaultName}
}

func (p presetPickerScreen) Selected() preset.Preset {
	if p.cursor < len(p.presets) {
		return p.presets[p.cursor]
	}
	return preset.DefaultPresets[0]
}

func (p presetPickerScreen) content(termW, termH int) modalContent {
	var inner strings.Builder
	rows := max(termH-9, 1)
	start := min(max(p.cursor-rows/2, 0), max(len(p.presets)-rows, 0))
	for i := start; i < min(start+rows, len(p.presets)); i++ {
		ps := p.presets[i]
		// Style semantic fragments once; styling an ANSI-bearing label can
		// corrupt its escapes when Lip Gloss applies underline/width handling.
		nameStyle := ui.Foreground(theme.Secondary)
		arrow := "  "
		if i == p.cursor {
			nameStyle = activeStyle
			arrow = "▶ "
		}
		marker := ""
		if ps.Name == p.defaultName {
			marker = " (default)"
		}
		budget := max(ui.ModalInnerWidth(termW)-ansi.StringWidth(arrow+marker), 1)
		if budget > 40 {
			budget -= 12
		}
		line := nameStyle.Render(ui.SafeBlock(arrow)) + nameStyle.Render(ui.SafeBlock(ansi.Truncate(ui.SafeText(ps.Name), budget, "…"))) + dimStyle.Render(ui.SafeBlock(marker))
		var windows []string
		for _, win := range ps.Windows {
			label := win.Name
			if win.Layout != nil {
				label += "[split]"
			}
			windows = append(windows, label)
		}
		line += dimStyle.Render(ui.SafeBlock("  [" + strings.Join(windows, " · ") + "]"))
		inner.WriteString(ansi.Truncate(line, ui.ModalInnerWidth(termW), "…") + "\n")
	}
	if len(p.presets) > 0 {
		inner.WriteString("\nSelected: " + p.Selected().Name)
	}
	return modalContent{title: "Choose tmux preset", body: inner.String(), hint: "↑/↓ choose · enter confirm · esc cancel"}
}

// ---------------------------------------------------------------------------
// templatePickerScreen
// ---------------------------------------------------------------------------

type templatePickerScreen struct {
	modalScreen
	templates []tmpl.Template
	cursor    int
}

func newTemplatePicker(templates []tmpl.Template) templatePickerScreen {
	return templatePickerScreen{templates: templates}
}

func (t templatePickerScreen) Selected() *tmpl.Template {
	if t.cursor == 0 {
		return nil
	}
	if t.cursor-1 < len(t.templates) {
		retm := t.templates[t.cursor-1]
		return &retm
	}
	return nil
}

func (t templatePickerScreen) content(termW, termH int) modalContent {
	var inner strings.Builder
	rows := max(termH-10, 1)
	start := min(max(t.cursor-1-rows/2, 0), max(len(t.templates)-rows, 0))
	if t.cursor == 0 {
		inner.WriteString(activeStyle.Render("▶  No template  ") + dimStyle.Render("  create an empty project") + "\n")
	} else {
		inner.WriteString(lipgloss.NewStyle().Foreground(theme.Secondary).Render("   No template") + dimStyle.Render("  create an empty project") + "\n")
	}
	for i := start; i < min(start+rows, len(t.templates)); i++ {
		tmpl := t.templates[i]
		idx := i + 1
		desc := dimStyle.Render(ui.SafeBlock("  " + tmpl.Description))
		var line string
		if idx == t.cursor {
			line = activeStyle.Render(ui.SafeBlock("▶  "+tmpl.Name)) + desc
		} else {
			line = lipgloss.NewStyle().Foreground(theme.Secondary).Render(ui.SafeBlock("   "+tmpl.Name)) + desc
		}
		inner.WriteString(ansi.Truncate(line, ui.ModalInnerWidth(termW), "…") + "\n")
	}
	if selected := t.Selected(); selected != nil {
		inner.WriteString("\nSelected: " + selected.Name + "\n" + selected.Description)
	}
	return modalContent{title: "Choose a template", body: inner.String(), hint: "↑/↓ choose · enter confirm · esc cancel"}
}

// ---------------------------------------------------------------------------
// domainPickerScreen
// ---------------------------------------------------------------------------

const createNewDomainOption = "+ create new domain"

type domainPickerScreen struct {
	genericPickerScreen
	projectName            string
	existingProjectDomains map[string]bool
}

func newDomainPickerScreen(projectName string, domains []string, existing map[string]bool) domainPickerScreen {
	opts := append(append([]string{}, domains...), createNewDomainOption)
	g := newGenericPicker(
		fmt.Sprintf("Select domain for \"%s\":", projectName),
		opts,
		"↑/↓ navigate  •  enter select  •  esc cancel",
	)
	g.disabled = existing
	return domainPickerScreen{genericPickerScreen: g, projectName: projectName, existingProjectDomains: existing}
}

// ---------------------------------------------------------------------------
// confirmDeleteDomainScreen
// ---------------------------------------------------------------------------

type confirmDeleteDomainScreen struct {
	modalScreen
	domainName string
	input      textinput.Model
	err        string
}

func newConfirmDeleteDomainScreen(domainName string) confirmDeleteDomainScreen {
	ti := transparentInput()
	ti.Placeholder = domainName
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 52
	ti.Cursor.Style = cursorStyle
	ti.PromptStyle = promptStyle
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Primary)
	return confirmDeleteDomainScreen{domainName: domainName, input: ti}
}

func (s confirmDeleteDomainScreen) Update(msg tea.Msg) (confirmDeleteDomainScreen, tea.Cmd) {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s confirmDeleteDomainScreen) content(termW, termH int) modalContent {
	s.input.Width = max(ui.ModalInnerWidth(termW)-2, 1)
	var inner strings.Builder
	inner.WriteString(errorStyle.Render(ui.SafeBlock(fmt.Sprintf(
		"ALL projects inside \"%s\" will be permanently deleted.", s.domainName,
	))) + "\n\n")
	inner.WriteString(lipgloss.NewStyle().Foreground(theme.Secondary).Render(
		ui.SafeBlock(fmt.Sprintf("Type \"%s\" to confirm:", s.domainName)),
	) + "\n\n")
	inner.WriteString(ui.InputView(s.input))
	if s.err != "" {
		inner.WriteString("\n\n" + errorStyle.Render(ui.SafeBlock("✗  "+s.err)))
	}
	return modalContent{title: "! Delete domain", body: inner.String(), hint: "enter confirm · esc cancel"}
}

// ---------------------------------------------------------------------------
// confirmDeleteTmuxScreen
// ---------------------------------------------------------------------------

type confirmDeleteTmuxScreen struct {
	modalScreen
	sessionName string
	targetName  string
	input       textinput.Model
	err         string
}

func newConfirmDeleteTmuxScreen(sessionName string) confirmDeleteTmuxScreen {
	ti := transparentInput()
	ti.Placeholder = sessionName
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 52
	ti.Cursor.Style = cursorStyle
	ti.PromptStyle = promptStyle
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Primary)
	return confirmDeleteTmuxScreen{sessionName: sessionName, input: ti}
}

func (s confirmDeleteTmuxScreen) Update(msg tea.Msg) (confirmDeleteTmuxScreen, tea.Cmd) {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s confirmDeleteTmuxScreen) target() string {
	if s.targetName != "" {
		return s.targetName
	}
	return s.sessionName
}

func (s confirmDeleteTmuxScreen) content(termW, termH int) modalContent {
	s.input.Width = max(ui.ModalInnerWidth(termW)-2, 1)
	var inner strings.Builder
	inner.WriteString(warningStyle.Render(ui.SafeBlock(fmt.Sprintf(
		"Kill tmux session \"%s\"? All windows and panes will be lost.", s.target(),
	))) + "\n\n")
	inner.WriteString(lipgloss.NewStyle().Foreground(theme.Secondary).Render(
		ui.SafeBlock(fmt.Sprintf("Type \"%s\" to confirm:", s.target())),
	) + "\n\n")
	inner.WriteString(ui.InputView(s.input))
	if s.err != "" {
		inner.WriteString("\n\n" + errorStyle.Render(ui.SafeBlock("✗  "+s.err)))
	}
	return modalContent{title: "! Kill tmux session", body: inner.String(), hint: "enter confirm · esc cancel"}
}

func newLocationPicker(title string, locations []core.Location, labels []string, hint string) genericPickerScreen {
	picker := newGenericPicker(title, labels, hint)
	for _, loc := range locations {
		picker.identities = append(picker.identities, core.PathID(loc.Path()))
	}
	return picker
}
func newRootPicker(title string, roots []string, hint string) genericPickerScreen {
	picker := newGenericPicker(title, roots, hint)
	for _, root := range roots {
		picker.identities = append(picker.identities, core.PathID(root))
	}
	return picker
}

func (s inputScreen) View(termW, termH int) string {
	return s.content(termW, termH).View(termW, termH, s.scroll)
}

func (y yesNoScreen) View(termW, termH int) string {
	return y.content(termW, termH).View(termW, termH, y.scroll)
}

func (g githubAuthScreen) View(termW, termH int) string {
	return g.content(termW, termH).View(termW, termH, g.scroll)
}

func (g genericPickerScreen) View(termW, termH int) string {
	return g.content(termW, termH).View(termW, termH, g.scroll)
}

func (p presetPickerScreen) View(termW, termH int) string {
	return p.content(termW, termH).View(termW, termH, p.scroll)
}

func (t templatePickerScreen) View(termW, termH int) string {
	return t.content(termW, termH).View(termW, termH, t.scroll)
}

func (s confirmDeleteDomainScreen) View(termW, termH int) string {
	return s.content(termW, termH).View(termW, termH, s.scroll)
}

func (s confirmDeleteTmuxScreen) View(termW, termH int) string {
	return s.content(termW, termH).View(termW, termH, s.scroll)
}
