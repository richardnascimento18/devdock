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
	return ModalAt(title, body, hint, width, height, 0)
}

func ModalInnerWidth(width int) int { return max(min(76, max(width-2, 1))-4, 1) }

// ModalAt reserves title/footer space and pages the entire wrapped body. The
// caller owns offset and captures PageUp/PageDown before routing form keys.
func modalLayout(body, hint string, width, height int) ([]string, string, int, int) {
	inner := ModalInnerWidth(width)
	bodyLines := strings.Split(ansi.Hardwrap(body, inner, true), "\n")
	originalHint := hint
	hint = ansi.Wrap(hint, inner, "")
	hintLines := strings.Split(hint, "\n")
	if len(hintLines) > 2 {
		hint = compactModalHint(originalHint)
		hintLines = strings.Split(ansi.Wrap(hint, inner, ""), "\n")
	}

	rows := max(height-3-len(hintLines), 1)
	if len(bodyLines) > rows {
		if inner < 25 {
			switch {
			case strings.Contains(originalHint, "ctrl+s"):
				hint = "ctrl+s save\nesc · pgup/pgdn"
			case strings.Contains(originalHint, "y discard"):
				hint = "y discard · esc\npgup/pgdn"
			case strings.Contains(originalHint, "enter"):
				hint = "enter · esc\npgup/pgdn"
			default:
				hint = "esc back\npgup/pgdn"
			}
		} else if inner < 35 {
			hint = compactModalHint(originalHint) + "\npgup/pgdn details"
		} else {
			hint = originalHint + " · pgup/pgdn details"
		}

		hintLines = strings.Split(ansi.Wrap(hint, inner, ""), "\n")
		if len(hintLines) > 2 {
			hintLines = hintLines[:2]
		}
		rows = max(height-3-len(hintLines), 1)
	}
	return bodyLines, strings.Join(hintLines, "\n"), rows, inner
}

func ModalScrollLimit(body, hint string, width, height int) int {
	lines, _, rows, _ := modalLayout(body, hint, width, height)
	return max(len(lines)-rows, 0)
}

func ModalAt(title, body, hint string, width, height, offset int) string {
	lines, hint, rows, inner := modalLayout(body, hint, width, height)
	offset = min(max(offset, 0), max(len(lines)-rows, 0))
	lines = lines[offset:min(offset+rows, len(lines))]
	title = ansi.Truncate(title, max(inner-2, 1), "…")
	panel := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(Default.BorderFocused).Padding(0, 1).Width(inner + 2).
		Render(PaneTitle(title, true, inner) + "\n" + strings.Join(lines, "\n") + "\n" + Foreground(Default.Muted).Render(hint))
	return Fit(lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, panel), width, height)
}

func compactModalHint(hint string) string {
	switch {
	case strings.Contains(hint, "ctrl+s"):
		return "ctrl+s save · esc back"
	case strings.Contains(hint, "y discard"):
		return "y discard · esc"
	case strings.Contains(hint, "enter"):
		return "enter · esc cancel"
	default:
		return "esc back"
	}
}
