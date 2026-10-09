// Package terminal interprets the small redraw protocol used by scaffolders.
// Controls are never forwarded. This is a bounded output model, not a host terminal.
package terminal

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxRows = 2000
const maxColumns = 4096

type line struct {
	runes []rune
	text  string
	dirty bool
}

type Screen struct {
	lines    []line
	row, col int
	mode     uint8
	sequence []byte
	utf8     []byte
}

// Lines snapshots the output. Only changed rows require rune-to-string copies.
// Cursor columns retain the existing rune-based redraw protocol; this is not
// complete terminal display-cell emulation for wide/combining characters.
func (s *Screen) Lines() []string {
	result := make([]string, len(s.lines))
	for i := range s.lines {
		row := &s.lines[i]
		if row.dirty {
			row.text = string(row.runes)
			row.dirty = false
		}
		result[i] = row.text
	}
	return result
}
func (s *Screen) Reset() { *s = Screen{} }

func (s *Screen) ensureLine() {
	if s.row >= maxRows {
		shift := min(s.row-maxRows+1, len(s.lines))
		clear(s.lines[:shift])
		s.lines = s.lines[shift:]
		s.row = maxRows - 1
	}
	for len(s.lines) <= s.row {
		s.lines = append(s.lines, line{})
	}
}
func (s *Screen) writeRune(r rune) {
	if unicode.IsControl(r) {
		return
	}
	s.ensureLine()
	if s.col >= maxColumns {
		return
	}
	row := &s.lines[s.row]
	if s.col <= len(row.runes) {
		row.runes = row.runes[:s.col]
	}
	row.runes = append(row.runes, r)
	row.dirty = true
	s.col++
}

// Write keeps parser state across chunks, including UTF-8, CSI, and OSC/ST.
// OSC-8, OSC-52, DCS and other string controls are consumed locally in full.
func (s *Screen) Write(data []byte) {
	for _, b := range data {
		switch s.mode {
		case 1: // ESC
			s.mode = 0
			switch b {
			case '[':
				s.mode = 2
				s.sequence = s.sequence[:0]
			case ']', 'P', '^', '_':
				s.mode = 3
			}
		case 2: // CSI
			if b >= 0x40 && b <= 0x7e {
				s.control(b)
				s.mode = 0
				s.sequence = s.sequence[:0]
			} else if len(s.sequence) < 64 {
				s.sequence = append(s.sequence, b)
			} else {
				s.mode = 0
				s.sequence = s.sequence[:0]
			}
		case 3: // control string
			if b == 7 {
				s.mode = 0
			} else if b == 27 {
				s.mode = 4
			}
		case 4: // ESC within control string
			if b == '\\' {
				s.mode = 0
			} else {
				s.mode = 3
			}
		default:
			if len(s.utf8) > 0 {
				if b < 0x80 {
					s.writeRune(utf8.RuneError)
					s.utf8 = s.utf8[:0]
				} else {
					s.utf8 = append(s.utf8, b)
					if utf8.FullRune(s.utf8) {
						r, _ := utf8.DecodeRune(s.utf8)
						s.writeRune(r)
						s.utf8 = s.utf8[:0]
					}
					continue
				}
			}
			switch {
			case b == 27:
				s.mode = 1
			case b == '\r':
				s.col = 0
			case b == '\n':
				s.row++
				s.col = 0
				s.ensureLine()
			case b >= utf8.RuneSelf:
				s.utf8 = append(s.utf8[:0], b)
				if utf8.FullRune(s.utf8) {
					r, _ := utf8.DecodeRune(s.utf8)
					s.writeRune(r)
					s.utf8 = s.utf8[:0]
				}
			case b >= 0x20 && b != 0x7f:
				s.writeRune(rune(b))
			}
		}
	}
}
func (s *Screen) control(final byte) {
	first := strings.SplitN(string(s.sequence), ";", 2)[0]
	n, _ := strconv.Atoi(strings.TrimPrefix(first, "?"))
	n = min(max(n, 1), maxRows)
	switch final {
	case 'A':
		s.row = max(s.row-n, 0)
		s.col = 0
	case 'B':
		s.row += n
		s.col = 0
		s.ensureLine()
	case 'D', 'G':
		s.col = 0
	case 'K':
		s.ensureLine()
		row := &s.lines[s.row]
		row.runes = row.runes[:0]
		row.text, row.dirty = "", false
		s.col = 0
	case 'J':
		if s.row < len(s.lines) {
			clear(s.lines[s.row:])
			s.lines = s.lines[:s.row]
		}
	}
}
