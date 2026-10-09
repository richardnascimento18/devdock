package terminal

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestStreamingUnicodeRedrawAndControls(t *testing.T) {
	var screen Screen
	input := []byte("old\r\x1b[K界 café\nnext\x1b[1A\r\x1b[Kupdated\x1b]52;c;secret\a\x1b]8;;url\x1b\\link\x1b]8;;\x1b\\")
	for _, b := range input {
		screen.Write([]byte{b})
	}
	got := strings.Join(screen.Lines(), "\n")
	if got != "updatedlink\nnext" || !utf8.ValidString(got) {
		t.Fatalf("%q", got)
	}
}
func TestBoundsAndSplitUTF8(t *testing.T) {
	var screen Screen
	for _, b := range []byte("界é") {
		screen.Write([]byte{b})
	}
	if screen.Lines()[0] != "界é" {
		t.Fatal(screen.Lines())
	}
	screen.Write([]byte("\x1b[99999999B" + strings.Repeat("x", 20000) + strings.Repeat("\n", 3000)))
	if len(screen.Lines()) > maxRows {
		t.Fatal("unbounded rows")
	}
	for _, line := range screen.Lines() {
		if utf8.RuneCountInString(line) > maxColumns {
			t.Fatal("unbounded line")
		}
	}
}

func TestRedrawProtocolAcrossReadBoundaries(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"abc\rd", "d"},
		{"ab\bc", "abc"}, // Unsupported C0 controls remain consumed, not forwarded.
		{"first\nsecond\x1b[1Aupdated", "updated\nsecond"},
		{"abc\x1b[2Dx", "x"},
		{"abc\x1b[1Gx", "x"},
		{"first\nsecond\x1b[Jnew", "first\nnew"},
		{"abc\r\x1b[K界e\u0301", "界e\u0301"},
		{"\xff\nnext", "�\nnext"},
		{"\xe2\nnext", "�\nnext"},
		{"before\x1bPdiscard\x1b\\after", "beforeafter"},
	} {
		for chunk := 1; chunk <= len(tc.input); chunk++ {
			var screen Screen
			for start := 0; start < len(tc.input); start += chunk {
				screen.Write([]byte(tc.input[start:min(start+chunk, len(tc.input))]))
			}
			if got := strings.Join(screen.Lines(), "\n"); got != tc.want {
				t.Fatalf("input %q, chunk %d: %q, want %q", tc.input, chunk, got, tc.want)
			}
		}
	}
}

func TestMaximumLineRedrawAndReset(t *testing.T) {
	var screen Screen
	screen.Write([]byte(strings.Repeat("界", maxColumns+100)))
	if utf8.RuneCountInString(screen.Lines()[0]) != maxColumns {
		t.Fatal("line bound changed")
	}
	screen.Write([]byte("\rnew"))
	if screen.Lines()[0] != "new" {
		t.Fatal("bounded line cannot redraw")
	}
	screen.Reset()
	if len(screen.Lines()) != 0 {
		t.Fatal("reset retained output")
	}
	screen.Write([]byte("fresh"))
	if screen.Lines()[0] != "fresh" {
		t.Fatal("reset retained parser state")
	}
}
