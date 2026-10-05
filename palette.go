package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/ui"
	"github.com/sahilm/fuzzy"
	"strings"
)

type actionBinding struct {
	label, key, reason string
	project            bool
}
type paletteScreen struct {
	input     textinput.Model
	actions   []actionBinding
	visible   []int
	cursor    int
	target    string
	selection string
}

// Action metadata supplies palette labels/shortcuts and grouped help.
func (m model) availableActions() []actionBinding {
	_, local := projectFromItem(m.list.SelectedItem())
	_, remote := m.list.SelectedItem().(githubItem)
	_, session := m.list.SelectedItem().(tmuxSessionItem)
	projectReason := ""
	if !local {
		projectReason = "Select a local project"
	}
	mutableReason := ""
	if _, ok := m.actionProject(); !ok {
		mutableReason = "Select a current local project"
	}
	rootReason := ""
	if len(m.cfg.ActiveRoots()) == 0 {
		rootReason = "Add a workspace root first"
	}
	cloneReason := ""
	if !remote {
		cloneReason = "Select an uncloned GitHub repository"
	}
	if remote && rootReason != "" {
		cloneReason = rootReason
	}
	tmuxReason := ""
	if !session {
		tmuxReason = "Select a session in the tmux tab"
	}
	actions := []actionBinding{
		{"Open project", "enter", projectReason, true},
		{"Attach tmux session", "attach", tmuxReason, true},
		{"Toggle favorite", "f", mutableReason, true},
		{"Move project", "m", mutableReason, true},
		{"Delete project", "x", mutableReason, true},
		{"Create project", "n", rootReason, false},
		{"Clone repository", "clone", cloneReason, true},
		{"Create domain", "N", rootReason, false},
		{"Create nested group", "G", rootReason, false},
		{"Delete group", "ctrl+g", rootReason, false},
		{"Delete domain", "X", rootReason, false},
		{"Add root", "a", "", false},
		{"Remove root", "A", rootReason, false},
		{"Open Settings", "settings", "", false},
		{"Edit presets", "presets", "", false},
		{"Edit templates", "templates", "", false},
		{"Toggle workspace / flat view", "v", "", false},
		{"Switch root", "}", rootReason, false},
		{"Refresh workspace", "r", "", false},
		{"Connect / refresh GitHub", "g", "", false},
		{"Keyboard help", "?", "", false},
		{"Show status details", "!", "", false},
		{"Quit", "q", "", false},
	}
	if len(m.selected) > 0 {
		reason := ""
		if m.hiddenSelection() > 0 {
			reason = "Clear filter to review all selected projects"
		}
		for i := range actions {
			switch actions[i].key {
			case "m":
				actions[i] = actionBinding{"Move selected projects", "m", reason, false}
			case "f":
				actions[i] = actionBinding{"Favorite / unfavorite selected projects", "f", reason, false}
			case "x", "X", "ctrl+g", "A":
				actions[i].reason = "Clear selection before destructive actions"
			}
		}
		actions = append(actions, actionBinding{"Clear project selection", "clear-selection", "", false})
	}
	return actions
}

func (m model) openPalette() (tea.Model, tea.Cmd) {
	m.palette = paletteScreen{input: transparentInput(), actions: m.availableActions(), target: rowIdentity(m.list.SelectedItem()), selection: m.selectionSignature()}
	m.palette.input.Prompt = "/ "
	m.palette.input.Placeholder = "Find an action…"
	m.palette.actions = append(m.palette.actions, m.scopeActions()...)
	m.palette.filter()
	m.state = statePalette
	return m, m.palette.input.Focus()
}

func (p *paletteScreen) filter() {
	labels := make([]string, len(p.actions))
	for i, a := range p.actions {
		labels[i] = a.label
	}
	p.visible = nil
	if strings.TrimSpace(p.input.Value()) == "" {
		for i := range p.actions {
			p.visible = append(p.visible, i)
		}
	} else {
		for _, match := range fuzzy.Find(p.input.Value(), labels) {
			p.visible = append(p.visible, match.Index)
		}
	}
	p.cursor = min(max(p.cursor, 0), max(len(p.visible)-1, 0))
}

func (p paletteScreen) View(width, height int) string {
	p.input.Width = max(min(width-10, 66), 1)
	rows := max(height-10, 1)
	start := min(max(p.cursor-rows/2, 0), max(len(p.visible)-rows, 0))
	lines := []string{p.input.View(), ""}
	if len(p.visible) == 0 {
		lines = append(lines, dimStyle.Render("No matching actions. Try another query."))
	}
	for i := start; i < min(start+rows, len(p.visible)); i++ {
		a := p.actions[p.visible[i]]
		label := a.label
		if a.reason != "" {
			label += " · " + a.reason
		} else if len(a.key) < 8 {
			label += "  [" + a.key + "]"
		}
		prefix := "  "
		if i == p.cursor {
			prefix = "> "
			label = activeStyle.Render(label)
		}
		lines = append(lines, ansi.Truncate(prefix+label, max(min(width-8, 68), 1), "…"))
	}
	if len(p.visible) > 0 {
		a := p.actions[p.visible[p.cursor]]
		if a.reason != "" {
			lines = append(lines, "", warningStyle.Render("! "+a.reason))
		}
	}
	return ui.Modal("Commands", strings.Join(lines, "\n"), "↑/↓ choose · enter run · esc cancel", width, height)
}

func (m model) updatePalette(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "ctrl+p":
			m.state = stateList
			return m, nil
		case "ctrl+c":
			return m, tea.Quit
		case "up", "ctrl+k":
			m.palette.cursor = max(m.palette.cursor-1, 0)
			return m, nil
		case "down", "ctrl+j":
			m.palette.cursor = min(m.palette.cursor+1, max(len(m.palette.visible)-1, 0))
			return m, nil
		case "enter":
			if len(m.palette.visible) == 0 {
				return m, nil
			}
			a := m.palette.actions[m.palette.visible[m.palette.cursor]]
			if a.reason != "" {
				return m, nil
			}
			if m.palette.selection != m.selectionSignature() {
				m.state = stateList
				m.statusMsg = warningStyle.Render("! Selection changed. Open commands again.")
				return m, nil
			}
			if a.project && rowIdentity(m.list.SelectedItem()) != m.palette.target {
				m.statusMsg = warningStyle.Render("! Project selection changed. Choose the action again.")
				m.state = stateList
				return m, nil
			}
			m.state = stateList
			if strings.HasPrefix(a.key, "scope:") {
				return m.chooseScope(strings.TrimPrefix(a.key, "scope:")), nil
			}
			switch a.key {
			case "parent-scope", "root-scope", "all-scope":
				keys := map[string]string{"parent-scope": "backspace", "root-scope": "ctrl+u", "all-scope": "ctrl+a"}
				return m.navigateScope(keys[a.key]), nil
			case "clear-selection":
				m.selected = nil
				return m, nil
			case "settings", "presets", "templates":
				result, cmd := m.startEditor()
				next := result.(model)
				if a.key == "settings" {
					next.editorScr.tab = editorTabSettings
				}
				if a.key == "templates" {
					next.editorScr.tab = editorTabTemplates
				}
				return next, cmd
			case "clone":
				return m.startCloneFlow(m.list.SelectedItem().(githubItem).repo), nil
			case "attach":
				m.pendingTmuxAttach = m.list.SelectedItem().(tmuxSessionItem).name
				return m, tea.Quit
			}
			m.focus = ui.Projects
			if a.key == "ctrl+g" {
				return m.updateList(tea.KeyMsg{Type: tea.KeyCtrlG})
			}
			return m.updateList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(a.key)})
		}
	}
	var cmd tea.Cmd
	m.palette.input, cmd = m.palette.input.Update(msg)
	m.palette.cursor = 0
	m.palette.filter()
	return m, cmd
}

func paletteActionDescription(a actionBinding) string { return fmt.Sprintf("%-12s %s", a.key, a.label) }
