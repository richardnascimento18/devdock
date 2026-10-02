// Package ui provides presentation-only primitives. It never accesses workspace
// services. Every style preserves the terminal's default background.
package ui

import (
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// Theme roles are foregrounds and borders only. Adaptive foregrounds support
// light and dark terminal defaults; Lip Gloss degrades them to the color profile.
// Primary inherits the terminal foreground, including NO_COLOR terminals.
type Theme struct {
	Primary, Secondary, Accent, Muted, Faint              lipgloss.TerminalColor
	Success, Warning, Error, Info                         lipgloss.TerminalColor
	Border, BorderFocused, Selection, Favorite, Git, Tmux lipgloss.TerminalColor
}

var Default = Theme{
	Primary:       lipgloss.NoColor{},
	Secondary:     lipgloss.AdaptiveColor{Light: "#57576A", Dark: "#C0C0D0"},
	Accent:        lipgloss.AdaptiveColor{Light: "#6941B6", Dark: "#C5A5FF"},
	Muted:         lipgloss.AdaptiveColor{Light: "#666675", Dark: "#ADADBD"},
	Faint:         lipgloss.AdaptiveColor{Light: "#727280", Dark: "#9696A5"},
	Success:       lipgloss.AdaptiveColor{Light: "#257344", Dark: "#80DCA2"},
	Warning:       lipgloss.AdaptiveColor{Light: "#916000", Dark: "#E8C070"},
	Error:         lipgloss.AdaptiveColor{Light: "#B23345", Dark: "#FF939E"},
	Info:          lipgloss.AdaptiveColor{Light: "#186B88", Dark: "#83D5EC"},
	Border:        lipgloss.AdaptiveColor{Light: "#727280", Dark: "#9696A5"},
	BorderFocused: lipgloss.AdaptiveColor{Light: "#6941B6", Dark: "#C5A5FF"},
	Selection:     lipgloss.AdaptiveColor{Light: "#6941B6", Dark: "#C5A5FF"},
	Favorite:      lipgloss.AdaptiveColor{Light: "#916000", Dark: "#E8C070"},
	Git:           lipgloss.AdaptiveColor{Light: "#186B88", Dark: "#83D5EC"},
	Tmux:          lipgloss.AdaptiveColor{Light: "#257344", Dark: "#80DCA2"},
}

func Foreground(color lipgloss.TerminalColor) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(color)
}

// ConfigureInput replaces Bubbles' inverse-video cursor (an implicit fill).
// An underlined insertion character and a visible prompt provide focus without
// painting a cell. Hidden cursors also require no independent blink timer.
func ConfigureInput(input *textinput.Model) {
	input.Cursor.SetMode(cursor.CursorHide)
	input.Cursor.TextStyle = Foreground(Default.Accent).Underline(true)
	input.TextStyle = Foreground(Default.Primary)
	input.PlaceholderStyle = Foreground(Default.Muted)
	input.PromptStyle = Foreground(Default.Accent).Bold(true)
}
