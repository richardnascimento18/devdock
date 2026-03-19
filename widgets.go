package main

import (
	"strings"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/preset"

	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// rootSelectorWidget
// ---------------------------------------------------------------------------

type rootSelectorWidget struct {
	roots  []string
	cursor int
}

func newRootSelector(roots []string) rootSelectorWidget { return rootSelectorWidget{roots: roots} }

func (r rootSelectorWidget) Selected() string {
	if r.cursor == 0 {
		return ""
	}
	return r.roots[r.cursor-1]
}

func (r *rootSelectorWidget) SetRoots(roots []string) {
	r.roots = roots
	if r.cursor > len(roots) {
		r.cursor = 0
	}
}
func (r *rootSelectorWidget) Next() { r.cursor = (r.cursor + 1) % (len(r.roots) + 1) }
func (r *rootSelectorWidget) Prev() {
	total := len(r.roots) + 1
	r.cursor = (r.cursor - 1 + total) % total
}

func (r rootSelectorWidget) View() string {
	var b strings.Builder
	b.WriteString(dimStyle.Render("root │ "))
	if r.cursor == 0 {
		b.WriteString(activeStyle.Render(" all "))
	} else {
		b.WriteString(dimStyle.Render("all"))
	}
	for i, root := range r.roots {
		b.WriteString(dimStyle.Render("  ·  "))
		label := config.RootName(root)
		if r.cursor == i+1 {
			b.WriteString(activeStyle.Render(" " + label + " "))
		} else {
			b.WriteString(dimStyle.Render(label))
		}
	}
	b.WriteString(dimStyle.Render("  (tab / shift+tab)"))
	return b.String()
}

// ---------------------------------------------------------------------------
// presetSelectorWidget
// ---------------------------------------------------------------------------

type presetSelectorWidget struct {
	presets []preset.Preset
	cursor  int
}

func newPresetSelector(presets []preset.Preset, defaultName string) presetSelectorWidget {
	cursor := 0
	for i, p := range presets {
		if p.Name == defaultName {
			cursor = i
			break
		}
	}
	return presetSelectorWidget{presets: presets, cursor: cursor}
}

func (s presetSelectorWidget) Selected() preset.Preset {
	if s.cursor < len(s.presets) {
		return s.presets[s.cursor]
	}
	return preset.DefaultPresets[0]
}

func (s presetSelectorWidget) SelectedName() string { return s.Selected().Name }

func (s *presetSelectorWidget) SetPresets(presets []preset.Preset) {
	s.presets = presets
	if s.cursor >= len(presets) {
		s.cursor = 0
	}
}

func (s *presetSelectorWidget) Next() {
	if len(s.presets) == 0 {
		return
	}
	s.cursor = (s.cursor + 1) % len(s.presets)
}

func (s *presetSelectorWidget) Prev() {
	if len(s.presets) == 0 {
		return
	}
	s.cursor = (s.cursor - 1 + len(s.presets)) % len(s.presets)
}

func (s presetSelectorWidget) View() string {
	var b strings.Builder
	b.WriteString(dimStyle.Render("preset │ "))
	for i, p := range s.presets {
		if i > 0 {
			b.WriteString(dimStyle.Render("  ·  "))
		}
		if i == s.cursor {
			b.WriteString(activeStyle.Render(" " + p.Name + " "))
		} else {
			b.WriteString(dimStyle.Render(p.Name))
		}
	}
	b.WriteString(dimStyle.Render("  (p / P)"))
	return b.String()
}

// ---------------------------------------------------------------------------
// helpScreen
// ---------------------------------------------------------------------------

type helpEntry struct {
	key  string
	desc string
}

var helpSections = []struct {
	title   string
	entries []helpEntry
}{
	{
		title: "Navigation",
		entries: []helpEntry{
			{"↑ / k", "move up"},
			{"↓ / j", "move down"},
			{"enter", "open project / toggle group"},
			{"space", "collapse / expand group"},
			{"v", "toggle tree / flat view"},
			{"/", "filter list"},
			{"c  /  esc", "clear active filter"},
			{"[  /  ]", "switch tab (Search / Recents / Favorites)"},
			{"tab", "cycle root filter →"},
			{"shift+tab", "cycle root filter ←"},
			{"p / P", "cycle default preset →/←"},
		},
	},
	{
		title: "Projects",
		entries: []helpEntry{
			{"n", "new project"},
			{"N", "new domain"},
			{"m", "move selected project"},
			{"f", "toggle favorite on selected project"},
			{"x", "delete selected project"},
			{"X", "delete a domain"},
			{"r", "rescan projects"},
			{"G", "new group"},
			{"ctrl+g", "delete group"},
		},
	},
	{
		title: "Roots",
		entries: []helpEntry{
			{"a", "add root directory"},
			{"A", "remove root directory"},
		},
	},
	{
		title: "GitHub",
		entries: []helpEntry{
			{"g", "connect / refresh GitHub"},
		},
	},
	{
		title: "General",
		entries: []helpEntry{
			{"?", "toggle this help screen"},
			{"esc", "close help / cancel"},
			{"ctrl+c", "quit"},
		},
	},
}

var helpOverlayStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorPurple).
	Background(lipgloss.Color("232")).
	Padding(1, 4).
	Width(76)

func RenderHelpOverlay(termW, termH int) string {
	var body strings.Builder
	body.WriteString(promptStyle.Render("Keyboard Reference") + "\n")
	body.WriteString(dimStyle.Render("────────────────────────────────────────────────────────────") + "\n\n")
	keyW := 16
	for _, section := range helpSections {
		body.WriteString(lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurpleLight).
			Render(section.title) + "\n")
		for _, e := range section.entries {
			keyStr := lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Width(keyW).Render(e.key)
			descStr := lipgloss.NewStyle().Foreground(colorGray).Render(e.desc)
			body.WriteString("  " + keyStr + "  " + descStr + "\n")
		}
		body.WriteString("\n")
	}
	body.WriteString(hintStyle.Render("press ? or esc to close"))
	panel := helpOverlayStyle.Render(body.String())
	logoBlock := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render(asciiTitle),
		subtitleStyle.Render("your terminal workspace manager"),
		"",
	)
	full := lipgloss.JoinVertical(lipgloss.Center, logoBlock, panel)
	return lipgloss.Place(termW, termH, lipgloss.Center, lipgloss.Center, full)
}
