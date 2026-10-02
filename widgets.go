package main

import (
	"strings"

	"github.com/richardnascimento18/devdock/internal/preset"

	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/ui"
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
	{"Navigation", []helpEntry{{"j/k · ↑/↓", "navigate focused pane"}, {"tab / shift+tab", "cycle Workspace / Projects / Inspector"}, {"←/→ · h/l", "collapse / expand workspace node"}, {"enter", "show workspace scope / open project"}, {"{ / }", "cycle root scope"}, {"[ / ]", "Search / Recents / Favorites / tmux"}, {"/", "live in-memory fuzzy search"}, {"esc", "cancel search / return to Projects"}, {"ctrl+p", "filterable command palette"}, {"v", "workspace tree / flat projects"}}},
	{"Multi-select", []helpEntry{{"space", "toggle project selection"}, {"m", "preflight and move all visible selections"}, {"f", "favorite all / unfavorite if all favorite"}, {"esc", "clear filter, then clear selection"}}},
	{"Editors", []helpEntry{{"tab / h / l", "switch Presets / Templates / Settings"}, {"i / enter", "edit focused field or item"}, {"ctrl+s", "validate and save"}, {"esc", "stop typing / cancel / return"}}},
}

func RenderHelpOverlay(termW, termH int) string {
	return renderHelpOverlay(termW, termH, 0, model{}.availableActions())
}

func renderHelpOverlay(termW, termH, offset int, actions []actionBinding) string {
	var lines []string
	width := min(70, max(termW-8, 1))
	for _, section := range helpSections {
		lines = append(lines, promptStyle.Render(section.title))
		for _, entry := range section.entries {
			line := entry.key + "  " + entry.desc
			lines = append(lines, strings.Split(ansi.Hardwrap(line, width, true), "\n")...)
		}
		lines = append(lines, "")
	}
	lines = append(lines, promptStyle.Render("Actions"))
	for _, action := range actions {
		lines = append(lines, strings.Split(ansi.Hardwrap(paletteActionDescription(action), width, true), "\n")...)
	}
	lines = append(lines, "", hintStyle.Render("q / ctrl+c quit main view · ? / esc close help"))
	rows := max(termH-8, 1)
	offset = min(max(offset, 0), max(len(lines)-rows, 0))
	body := strings.Join(lines[offset:min(offset+rows, len(lines))], "\n")
	return ui.Modal("Keyboard Reference", body, "j/k scroll · ? / esc close", termW, termH)
}
