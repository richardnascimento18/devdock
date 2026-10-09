package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

// Activity applies a restrained moving foreground highlight to transient work.
// Reduced motion is static and all glyph/padding cells inherit default background.
func Activity(label string, frame int, reduced bool) string {
	label = ansi.Truncate(SafeText(label), 72, "…")
	if reduced || frame == 0 {
		return Foreground(Default.Info).Render("… " + label)
	}
	frames := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	var parts []string
	graphemes := uniseg.NewGraphemes(label)
	for graphemes.Next() {
		parts = append(parts, graphemes.Str())
	}
	peak := (frame/2)%(len(parts)+12) - 6
	var out strings.Builder
	out.WriteString(Foreground(Default.Accent).Bold(true).Render(frames[frame%len(frames)] + " "))
	for i, text := range parts {
		distance := i - peak
		if distance < 0 {
			distance = -distance
		}
		style := Foreground(Default.Muted)
		if distance < 2 {
			style = Foreground(Default.Info)
		} else if distance < 5 {
			style = Foreground(Default.Secondary)
		}
		out.WriteString(style.Render(text))
	}
	return out.String()
}
