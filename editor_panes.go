package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/preset"
	"github.com/richardnascimento18/devdock/internal/ui"
	"strconv"
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
	original  *preset.PaneLayout
	direction string // "horizontal" or "vertical"
	statusMsg string
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
	return splitPaneEditor{direction: dir, panes: panes, original: preset.CloneLayout(&pl)}
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
	if sp.original != nil && !sp.original.IsLeaf() {
		var originalLeaves []paneLeaf
		collectLeaves(*sp.original, &originalLeaves)
		if len(originalLeaves) == len(sp.panes) {
			layout := preset.CloneLayout(sp.original)
			layout.Direction = sp.direction
			index := 0
			var updateLeaves func(*preset.PaneLayout)
			updateLeaves = func(node *preset.PaneLayout) {
				if node.IsLeaf() {
					node.Command = sp.panes[index].command
					node.Size = sp.panes[index].size
					index++
					return
				}
				for i := range node.Panes {
					updateLeaves(&node.Panes[i])
				}
			}
			updateLeaves(layout)
			return *layout
		}
	}

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
		sz, err := parsePaneSize(sp.sizeInput.Value())
		if err != nil {
			sp.statusMsg = errorStyle.Render(ui.SafeBlock(err.Error()))
			return sp, nil
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

func parsePaneSize(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	size, err := strconv.Atoi(value)
	if err != nil || size < 0 || size >= 100 {
		return 0, fmt.Errorf("pane size must be an integer from 0 to 99")
	}
	return size, nil
}
