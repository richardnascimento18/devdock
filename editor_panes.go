package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/richardnascimento18/devdock/internal/preset"
	"strings"
)

// For UX clarity we represent the split as a flat list of leaf panes, each
// with a command, plus a top-level direction (horizontal / vertical).
// This covers the vast majority of use-cases without needing a tree UI.

type paneLeaf struct {
	command string
	size    int
}

type splitPaneEditor struct {
	direction string // "horizontal" or "vertical"
	panes     []paneLeaf
	cursor    int
	editIdx   int
	editing   bool
	typing    bool // true = keystrokes go to text input; i to enter, esc to leave
	cmdInput  textinput.Model
	sizeInput textinput.Model
	editFocus int // 0=cmd 1=size
}

func newSplitPaneEditor(pl preset.PaneLayout) splitPaneEditor {
	var panes []paneLeaf
	collectLeaves(pl, &panes)
	if len(panes) < 2 {
		panes = []paneLeaf{{}, {}}
	}
	dir := pl.Direction
	if dir != "horizontal" && dir != "vertical" {
		dir = "horizontal"
	}
	return splitPaneEditor{direction: dir, panes: panes}
}

func collectLeaves(pl preset.PaneLayout, out *[]paneLeaf) {
	if pl.IsLeaf() {
		*out = append(*out, paneLeaf{command: pl.Command, size: pl.Size})
		return
	}
	for _, child := range pl.Panes {
		collectLeaves(child, out)
	}
}

func (sp splitPaneEditor) toLayout() preset.PaneLayout {
	if len(sp.panes) == 0 {
		return preset.PaneLayout{}
	}
	root := preset.PaneLayout{Direction: sp.direction}
	for _, p := range sp.panes {
		root.Panes = append(root.Panes, preset.PaneLayout{Command: p.command, Size: p.size})
	}
	return root
}

func (sp splitPaneEditor) Update(msg tea.Msg) (splitPaneEditor, tea.Cmd) {
	if sp.editing {
		return sp.updateEdit(msg)
	}
	return sp.updateList(msg)
}

func (sp splitPaneEditor) updateList(msg tea.Msg) (splitPaneEditor, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return sp, nil
	}
	switch k.String() {
	case "j", "down":
		if sp.cursor < len(sp.panes) {
			sp.cursor++
		}
	case "k", "up":
		if sp.cursor > 0 {
			sp.cursor--
		}
	case "r":
		if sp.direction == "horizontal" {
			sp.direction = "vertical"
		} else {
			sp.direction = "horizontal"
		}
	case "a":
		sp.panes = append(sp.panes, paneLeaf{})
	case "d", "x":
		if len(sp.panes) > 2 && sp.cursor < len(sp.panes) {
			sp.panes = append(sp.panes[:sp.cursor], sp.panes[sp.cursor+1:]...)
			if sp.cursor >= len(sp.panes) {
				sp.cursor = len(sp.panes) - 1
			}
		}
	case "enter":
		if sp.cursor < len(sp.panes) {
			sp.editIdx = sp.cursor
			sp.cmdInput = newSmallInput(sp.panes[sp.cursor].command, "command", 40)
			sp.sizeInput = newSmallInput("", "size % (0=auto)", 6)
			if sp.panes[sp.cursor].size > 0 {
				sp.sizeInput.SetValue(fmt.Sprintf("%d", sp.panes[sp.cursor].size))
			}
			sp.editFocus = 0
			sp.typing = false // start in navigation mode
			sp.editing = true
		}
	}
	return sp, nil
}

func (sp splitPaneEditor) updateEdit(msg tea.Msg) (splitPaneEditor, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		if sp.typing {
			return sp.forwardPaneInput(msg)
		}
		return sp, nil
	}

	// --- Typing mode. ---
	if sp.typing {
		switch k.String() {
		case "esc":
			sp.typing = false
			sp.cmdInput.Blur()
			sp.sizeInput.Blur()
			return sp, nil
		case "enter":
			// commit and exit typing mode, but stay in pane edit
			sp.typing = false
			sp.syncEditFocus()
			return sp, nil
		}
		return sp.forwardPaneInput(msg)
	}

	// --- Navigation mode. ---
	switch k.String() {
	case "esc":
		sp.editing = false
		return sp, nil
	case "i":
		sp.typing = true
		sp.syncEditFocus()
		return sp, nil
	case "tab", "down", "j":
		sp.editFocus = (sp.editFocus + 1) % 2
		sp.syncEditFocus()
		return sp, nil
	case "shift+tab", "up", "k":
		sp.editFocus = (sp.editFocus + 1) % 2
		sp.syncEditFocus()
		return sp, nil
	case "enter":
		sp.panes[sp.editIdx].command = strings.TrimSpace(sp.cmdInput.Value())
		var sz int
		fmt.Sscanf(strings.TrimSpace(sp.sizeInput.Value()), "%d", &sz)
		if sz < 0 || sz >= 100 {
			sz = 0
		}
		sp.panes[sp.editIdx].size = sz
		sp.editing = false
		return sp, nil
	}
	return sp, nil
}

func (sp splitPaneEditor) forwardPaneInput(msg tea.Msg) (splitPaneEditor, tea.Cmd) {
	var cmd tea.Cmd
	switch sp.editFocus {
	case 0:
		sp.cmdInput, cmd = sp.cmdInput.Update(msg)
	case 1:
		sp.sizeInput, cmd = sp.sizeInput.Update(msg)
	}
	return sp, cmd
}

func (sp *splitPaneEditor) syncEditFocus() {
	sp.cmdInput.Blur()
	sp.sizeInput.Blur()
	if !sp.typing {
		return
	}
	switch sp.editFocus {
	case 0:
		sp.cmdInput.Focus()
	case 1:
		sp.sizeInput.Focus()
	}
}

func (sp splitPaneEditor) View() string {
	if sp.editing {
		var inner strings.Builder
		focusIndicator := func(idx int, label string) string {
			if sp.editFocus == idx {
				return promptStyle.Render("▶ " + label)
			}
			return dimStyle.Render("  " + label)
		}
		inner.WriteString(focusIndicator(0, "Command:") + "\n")
		inner.WriteString(sp.cmdInput.View() + "\n\n")
		inner.WriteString(focusIndicator(1, "Size % (0 = auto):") + "\n")
		inner.WriteString(sp.sizeInput.View() + "\n\n")
		if sp.typing {
			inner.WriteString(hintStyle.Render("typing mode  •  esc to stop typing  •  enter confirm"))
		} else {
			inner.WriteString(hintStyle.Render("j/k navigate  •  i to type  •  enter save pane  •  esc back"))
		}
		return wrapInBox("Edit Pane", inner.String())
	}

	var inner strings.Builder
	dirLabel := lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render(sp.direction)
	inner.WriteString(promptStyle.Render("Direction: ") + dirLabel + dimStyle.Render("  (r to toggle)") + "\n\n")
	inner.WriteString(promptStyle.Render("Panes:") + "\n")
	for i, p := range sp.panes {
		cmdStr := p.command
		if cmdStr == "" {
			cmdStr = "(shell)"
		}
		sizeStr := ""
		if p.size > 0 {
			sizeStr = dimStyle.Render(fmt.Sprintf(" [%d%%]", p.size))
		}
		label := fmt.Sprintf("pane %d: %s", i+1, cmdStr) + sizeStr
		if i == sp.cursor {
			inner.WriteString(activeStyle.Render("▶ "+fmt.Sprintf("pane %d: %s", i+1, cmdStr)) + sizeStr + "\n")
		} else {
			inner.WriteString(dimStyle.Render("  ") + lipgloss.NewStyle().Foreground(colorGray).Render(label) + "\n")
		}
	}
	inner.WriteString("\n" + hintStyle.Render("j/k navigate  •  enter edit  •  a add  •  d delete  •  r toggle dir  •  esc back"))
	return wrapInBox("Pane Layout", inner.String())
}

// ===========================================================================
// TEMPLATE EDITOR
// ===========================================================================
