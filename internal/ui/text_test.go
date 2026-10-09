package ui

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/x/ansi"
)

func TestSafeText(t *testing.T) {
	for _, input := range []string{
		"name\x1b]52;c;c2VjcmV0\a", "\x1b]8;;https://example.invalid\x1b\\link\x1b]8;;\x1b\\",
		"\x1b[48;5;42m\r\n\t\x00\x7f\u009b31m", "report\u202eexe\u2066name\u2069",
		string([]byte{'a', 0xff, 0x9b}), "界 café e\u0301 👩‍💻", strings.Repeat("a", 100000), "a" + strings.Repeat("\u0301", 100000),
	} {
		t.Run(input[:min(len(input), 30)], func(t *testing.T) {
			got := SafeText(input)
			if !utf8.ValidString(got) || utf8.RuneCountInString(got) > 4096*9+1 {
				t.Fatal("invalid or unbounded display text")
			}
			for _, r := range got {
				if unicode.IsControl(r) || r == '\u202e' || r == '\u2066' || r == '\u2069' {
					t.Fatalf("active control %U", r)
				}
			}
			if strings.Contains(got, "\x1b]52") || strings.Contains(got, "\x1b]8") {
				t.Fatal("live OSC")
			}
		})
	}
	if got := SafeText("界 café e\u0301 👩‍💻"); got != "界 café e\u0301 👩‍💻" {
		t.Fatalf("ordinary Unicode changed: %q", got)
	}
	if got := SafeBlock("one\ntwo\r"); got != "one\ntwo␍" {
		t.Fatal(got)
	}
	if got := Fit(SafeText("界界界"), 3, 1); ansi.StringWidth(got) > 3 || !utf8.ValidString(got) {
		t.Fatal("unsafe clipping")
	}
}

func TestInputViewDoesNotChangeIdentity(t *testing.T) {
	input := textinput.New()
	input.SetValue("name\u202e")
	before, position := input.Value(), input.Position()
	view := InputView(input)
	if strings.Contains(view, "\u202e") || input.Value() != before || input.Position() != position {
		t.Fatal("unsafe view or changed editable identity")
	}
}

func FuzzSafeText(f *testing.F) {
	f.Add("\x1b]52;c;abc\a界\u202e")
	f.Fuzz(func(t *testing.T, value string) {
		for _, r := range SafeText(value) {
			if unicode.IsControl(r) {
				t.Fatalf("control %U", r)
			}
		}
	})
}
