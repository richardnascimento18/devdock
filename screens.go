package main

import (
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/richardnascimento18/devdock/internal/preset"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
)

// ---------------------------------------------------------------------------
// renderTabBar
// ---------------------------------------------------------------------------

func renderTabBar(active int) string {
	var b strings.Builder
	for i, name := range tabNames {
		if i > 0 {
			b.WriteString(dimStyle.Render("  │  "))
		}
		if i == active {
			b.WriteString(activeStyle.Render(" " + name + " "))
		} else {
			b.WriteString(dimStyle.Render(name))
		}
	}
	b.WriteString(dimStyle.Render("  ([ / ] to switch)"))
	return b.String()
}

// ---------------------------------------------------------------------------
// inputScreen
// ---------------------------------------------------------------------------

type inputScreen struct {
	title string
	input textinput.Model
	err   string
	hint  string
}

func newInputScreen(title, placeholder, hint string) inputScreen {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 52
	ti.Cursor.Style = cursorStyle
	ti.PromptStyle = promptStyle
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorWhite)
	return inputScreen{title: title, input: ti, hint: hint}
}

func (s inputScreen) Update(msg tea.Msg) (inputScreen, tea.Cmd) {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s inputScreen) View(termW, termH int) string {
	var inner strings.Builder
	inner.WriteString(s.input.View() + "\n")
	if s.err != "" {
		inner.WriteString("\n" + errorStyle.Render("✗  "+s.err))
	}
	inner.WriteString("\n\n" + hintStyle.Render(s.hint))
	content := lipgloss.JoinVertical(lipgloss.Left, RenderTitle(), wrapInBox(s.title, inner.String()))
	return centerInTerminal(termW, termH, content)
}

// ---------------------------------------------------------------------------
// yesNoScreen
// ---------------------------------------------------------------------------

type yesNoScreen struct {
	title  string
	cursor int
	err    string
}

func newYesNoScreen(title string) yesNoScreen { return yesNoScreen{title: title} }
func (y yesNoScreen) IsYes() bool             { return y.cursor == 0 }

func (y yesNoScreen) View(termW, termH int) string {
	var inner strings.Builder
	for i, opt := range []string{"  Yes", "  No"} {
		if i == y.cursor {
			inner.WriteString(activeStyle.Render(opt) + "\n")
		} else {
			inner.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(opt) + "\n")
		}
	}
	if y.err != "" {
		inner.WriteString("\n" + errorStyle.Render("✗  "+y.err))
	}
	inner.WriteString("\n" + hintStyle.Render("↑/↓  •  enter confirm  •  esc cancel"))
	content := lipgloss.JoinVertical(lipgloss.Left, RenderTitle(), wrapInBox(y.title, inner.String()))
	return centerInTerminal(termW, termH, content)
}

// ---------------------------------------------------------------------------
// spinnerScreen
// ---------------------------------------------------------------------------

type spinnerScreen struct {
	message string
	frames  []string
	frame   int
}

func newSpinnerScreen(message string) spinnerScreen {
	return spinnerScreen{
		message: message,
		frames:  []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
	}
}

type spinnerTickMsg struct{ id uint64 }

func spinnerTick(id uint64) tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(_ time.Time) tea.Msg { return spinnerTickMsg{id: id} })
}

func (s spinnerScreen) View(termW, termH int) string {
	spinner := lipgloss.NewStyle().Foreground(colorPink).Bold(true).Render(s.frames[s.frame])
	msg := lipgloss.NewStyle().Foreground(colorGray).Render(s.message)
	inner := spinner + "  " + msg + "\n\n" + hintStyle.Render("please wait...")
	content := lipgloss.JoinVertical(lipgloss.Center, RenderTitle(), boxStyle.Render(inner))
	return centerInTerminal(termW, termH, content)
}

// ---------------------------------------------------------------------------
// githubAuthScreen
// ---------------------------------------------------------------------------

type githubAuthScreen struct {
	userCode        string
	verificationURI string
	err             string
	done            bool
}

func (g githubAuthScreen) View(termW, termH int) string {
	var inner strings.Builder
	inner.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render("1. Open this URL in your browser:") + "\n")
	inner.WriteString("   " + lipgloss.NewStyle().Foreground(colorCyan).Underline(true).Render(g.verificationURI) + "\n\n")
	inner.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render("2. Enter this one-time code:") + "\n")
	inner.WriteString("   " + lipgloss.NewStyle().
		Bold(true).Foreground(colorYellow).Background(colorDark).Padding(0, 2).
		Render(g.userCode) + "\n\n")
	if g.done {
		inner.WriteString(successStyle.Render("✓  Authorized! Loading your repositories..."))
	} else {
		inner.WriteString(lipgloss.NewStyle().Foreground(colorPurpleDim).Render("Waiting for authorization..."))
	}
	if g.err != "" {
		inner.WriteString("\n\n" + errorStyle.Render("✗  "+g.err))
	}
	inner.WriteString("\n\n" + hintStyle.Render("esc to cancel"))
	content := lipgloss.JoinVertical(lipgloss.Left, RenderTitle(), wrapInBox("Connect GitHub Account", inner.String()))
	return centerInTerminal(termW, termH, content)
}

// ---------------------------------------------------------------------------
// genericPickerScreen
// ---------------------------------------------------------------------------

type genericPickerScreen struct {
	title    string
	options  []string
	cursor   int
	disabled map[string]bool
	err      string
	hint     string
}

func newGenericPicker(title string, options []string, hint string) genericPickerScreen {
	return genericPickerScreen{title: title, options: options, disabled: map[string]bool{}, hint: hint}
}

func (g genericPickerScreen) View(termW, termH int) string {
	var inner strings.Builder
	rows := max(termH-16, 3)
	start := max(0, g.cursor-rows/2)
	if start+rows > len(g.options) {
		start = max(0, len(g.options)-rows)
	}
	end := min(start+rows, len(g.options))
	for i := start; i < end; i++ {
		o := g.options[i]
		available := max(termW-14, 12)
		if ansi.StringWidth(o) > available {
			o = ansi.TruncateLeft(o, ansi.StringWidth(o)-available+1, "…")
		}

		var line string
		switch {
		case g.disabled[g.options[i]]:
			line = dimStyle.Render("  "+o) + dimStyle.Render(" (already exists)")
		case i == g.cursor:
			line = activeStyle.Render("▶ " + o)
		default:
			line = lipgloss.NewStyle().Foreground(colorGray).Render("  " + o)
		}
		inner.WriteString(line + "\n")
	}
	if len(g.options) > rows {
		inner.WriteString(dimStyle.Render(fmt.Sprintf("%d/%d", g.cursor+1, len(g.options))) + "\n")
	}
	if g.err != "" {
		inner.WriteString("\n" + errorStyle.Render("✗  "+g.err))
	}
	inner.WriteString("\n" + hintStyle.Render(g.hint))
	content := lipgloss.JoinVertical(lipgloss.Left, RenderTitle(), wrapInBox(g.title, inner.String()))
	return centerInTerminal(termW, termH, content)
}

// ---------------------------------------------------------------------------
// presetPickerScreen
// ---------------------------------------------------------------------------

type presetPickerScreen struct {
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

func (p presetPickerScreen) View(termW, termH int) string {
	var inner strings.Builder
	for i, ps := range p.presets {
		label := ps.Name
		if ps.Name == p.defaultName {
			label += dimStyle.Render(" (default)")
		}
		var winParts []string
		for _, w := range ps.Windows {
			if w.Layout != nil {
				winParts = append(winParts, w.Name+dimStyle.Render("[split]"))
			} else {
				winParts = append(winParts, w.Name)
			}
		}
		summary := dimStyle.Render("  [" + strings.Join(winParts, " · ") + "]")
		var line string
		if i == p.cursor {
			line = activeStyle.Render("▶ "+label) + summary
		} else {
			line = lipgloss.NewStyle().Foreground(colorGray).Render("  "+label) + summary
		}
		inner.WriteString(line + "\n")
	}
	inner.WriteString("\n" + hintStyle.Render("↑/↓ navigate  •  enter confirm  •  esc cancel"))
	content := lipgloss.JoinVertical(lipgloss.Left, RenderTitle(), wrapInBox("Choose tmux preset:", inner.String()))
	return centerInTerminal(termW, termH, content)
}

// ---------------------------------------------------------------------------
// templatePickerScreen
// ---------------------------------------------------------------------------

type templatePickerScreen struct {
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

func (t templatePickerScreen) View(termW, termH int) string {
	var inner strings.Builder
	if t.cursor == 0 {
		inner.WriteString(activeStyle.Render("▶  No template  ") + dimStyle.Render("  create an empty project") + "\n")
	} else {
		inner.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render("   No template") + dimStyle.Render("  create an empty project") + "\n")
	}
	for i, tmpl := range t.templates {
		idx := i + 1
		desc := dimStyle.Render("  " + tmpl.Description)
		var line string
		if idx == t.cursor {
			line = activeStyle.Render("▶  "+tmpl.Name) + desc
		} else {
			line = lipgloss.NewStyle().Foreground(colorGray).Render("   "+tmpl.Name) + desc
		}
		inner.WriteString(line + "\n")
	}
	inner.WriteString("\n" + hintStyle.Render("↑/↓ navigate  •  enter confirm  •  esc cancel"))
	content := lipgloss.JoinVertical(lipgloss.Left, RenderTitle(), wrapInBox("Choose a template:", inner.String()))
	return centerInTerminal(termW, termH, content)
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
	domainName string
	input      textinput.Model
	err        string
}

func newConfirmDeleteDomainScreen(domainName string) confirmDeleteDomainScreen {
	ti := textinput.New()
	ti.Placeholder = domainName
	ti.Focus()
	ti.CharLimit = 0
	ti.Width = 52
	ti.Cursor.Style = cursorStyle
	ti.PromptStyle = promptStyle
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorWhite)
	return confirmDeleteDomainScreen{domainName: domainName, input: ti}
}

func (s confirmDeleteDomainScreen) Update(msg tea.Msg) (confirmDeleteDomainScreen, tea.Cmd) {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s confirmDeleteDomainScreen) View(termW, termH int) string {
	var inner strings.Builder
	inner.WriteString(errorStyle.Render(fmt.Sprintf(
		"ALL projects inside \"%s\" will be permanently deleted.", s.domainName,
	)) + "\n\n")
	inner.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(
		fmt.Sprintf("Type \"%s\" to confirm:", s.domainName),
	) + "\n\n")
	inner.WriteString(s.input.View())
	if s.err != "" {
		inner.WriteString("\n\n" + errorStyle.Render("✗  "+s.err))
	}
	inner.WriteString("\n\n" + hintStyle.Render("enter confirm  •  esc cancel"))
	content := lipgloss.JoinVertical(lipgloss.Left, RenderTitle(), wrapInWarningBox("Danger Zone — Delete Domain", inner.String()))
	return centerInTerminal(termW, termH, content)
}

// ---------------------------------------------------------------------------
// confirmDeleteTmuxScreen
// ---------------------------------------------------------------------------

type confirmDeleteTmuxScreen struct {
	sessionName string
	input       textinput.Model
	err         string
}

func newConfirmDeleteTmuxScreen(sessionName string) confirmDeleteTmuxScreen {
	ti := textinput.New()
	ti.Placeholder = sessionName
	ti.Focus()
	ti.CharLimit = 80
	ti.Width = 52
	ti.Cursor.Style = cursorStyle
	ti.PromptStyle = promptStyle
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorWhite)
	return confirmDeleteTmuxScreen{sessionName: sessionName, input: ti}
}

func (s confirmDeleteTmuxScreen) Update(msg tea.Msg) (confirmDeleteTmuxScreen, tea.Cmd) {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s confirmDeleteTmuxScreen) View(termW, termH int) string {
	var inner strings.Builder
	inner.WriteString(warningStyle.Render(fmt.Sprintf(
		"Kill tmux session \"%s\"? All windows and panes will be lost.", s.sessionName,
	)) + "\n\n")
	inner.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(
		fmt.Sprintf("Type \"%s\" to confirm:", s.sessionName),
	) + "\n\n")
	inner.WriteString(s.input.View())
	if s.err != "" {
		inner.WriteString("\n\n" + errorStyle.Render("✗  "+s.err))
	}
	inner.WriteString("\n\n" + hintStyle.Render("enter confirm  •  esc cancel"))
	content := lipgloss.JoinVertical(lipgloss.Left, RenderTitle(), wrapInWarningBox("Kill tmux Session", inner.String()))
	return centerInTerminal(termW, termH, content)
}
