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
