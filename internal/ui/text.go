package ui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
)

// SafeText encodes terminal controls as visible text before trusted styling.
// Identities and stored values must never be replaced with the display value.
func SafeText(value string) string { return safeText(value, false) }

// SafeBlock allows LF for diagnostic paragraphs and trusted layout composition.
// It otherwise applies exactly the single-line policy. Never pass styled text.
func SafeBlock(value string) string { return safeText(value, true) }

func safeText(value string, multiline bool) string {
	var out strings.Builder
	marks, count := 0, 0
	for _, r := range value { // range replaces malformed UTF-8 with U+FFFD
		if count >= 4096 {
			out.WriteRune('…')
			break
		}
		count++
		if unicode.Is(unicode.M, r) {
			marks++
			if marks > 16 {
				continue
			}
		} else {
			marks = 0
		}
		switch {
		case r == '\n' && multiline:
			out.WriteRune(r)
		case r < 0x20:
			out.WriteRune(0x2400 + r)
		case r == 0x7f:
			out.WriteRune('␡')
		case r >= 0x80 && r <= 0x9f, r == 0x061c, r == 0x200e, r == 0x200f,
			r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x206f,
			r == 0x2028, r == 0x2029, r == 0xfeff:
			fmt.Fprintf(&out, "[U+%04X]", r)
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

// InputView sanitizes a detached presentation copy, retaining the editable value
// and cursor in the model. Bubbles' copy shares rune storage until SetValue.
func InputView(input textinput.Model) string {
	position := input.Position()
	value := input.Value()
	safe := SafeText(value)
	if value != safe {
		input.SetValue(safe)
		input.SetCursor(len([]rune(SafeText(string([]rune(value)[:position])))))
	}
	input.Placeholder = SafeText(input.Placeholder)
	input.Prompt = SafeText(input.Prompt)
	return input.View()
}
