package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"io"
)

// Explicit glyph selection remains readable with no color and no filled rows.
type projectDelegate struct{}

func (projectDelegate) Height() int                         { return 1 }
func (projectDelegate) Spacing() int                        { return 0 }
func (projectDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (projectDelegate) Render(w io.Writer, m list.Model, index int, it list.Item) {
	entry, ok := it.(interface{ Title() string })
	if !ok {
		return
	}
	prefix := "  "
	label := entry.Title()
	if index == m.Index() {
		prefix = "> "
		label = selectedStyle.Render(ansi.Strip(label))
	}
	fmt.Fprint(w, ansi.Truncate(prefix+label, m.Width(), "…"))
}
