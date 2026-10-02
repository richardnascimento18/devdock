package ui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"strings"
)

// Fit bounds output using terminal cell widths, preserving ANSI and graphemes.
// Components should allocate space before rendering; this is the final guard.
func Fit(content string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "…")
	}
	return strings.Join(lines, "\n")
}

func Header(context string, width int) string {
	return Fit(Foreground(Default.Accent).Bold(true).Render("DevDock")+"  "+Foreground(Default.Muted).Render(context), width, 1)
}

func Footer(hint, status string, width int) string {
	if status == "" {
		status = " "
	}
	return Fit(status+"\n"+Foreground(Default.Muted).Render(hint), width, 2)
}

func PaneTitle(name string, focused bool, width int) string {
	style := Foreground(Default.Muted)
	prefix := "  "
	if focused {
		style = Foreground(Default.BorderFocused).Bold(true)
		prefix = "> "
	}
	return Fit(style.Render(prefix+name), width, 1)
}

func TooSmall(width, height int) string {
	return Fit(fmt.Sprintf("Terminal too small\nNeed at least %d×%d", MinWidth, MinHeight), width, height)
}

// Modal wraps text and preserves its full content. Screen owners handle vertical
// scrolling; titles and footer remain separately visible. It never paints cells.
func Modal(title, body, hint string, width, height int) string {
	w := min(76, max(width-2, 1))
	inner := max(w-4, 1)
	body = ansi.Hardwrap(body, inner, true)
	panel := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(Default.BorderFocused).Padding(0, 1).Width(inner).
		Render(PaneTitle(title, true, inner) + "\n" + body + "\n" + Foreground(Default.Muted).Render(ansi.Hardwrap(hint, inner, true)))
	return Fit(lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, panel), width, height)
}
