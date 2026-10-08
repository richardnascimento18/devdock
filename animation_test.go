package main

import (
	"fmt"
	"github.com/richardnascimento18/devdock/internal/app"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/config"
	gh "github.com/richardnascimento18/devdock/internal/github"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/tmux"
	"github.com/richardnascimento18/devdock/internal/ui"
)

func TestSharedClockLifecycle(t *testing.T) {
	m := renderFixture()
	m.motion = animationClock{}
	if m.scheduleAnimation() != nil {
		t.Fatal("idle schedules ticks")
	}
	m.scan.inFlight = true
	if m.scheduleAnimation() == nil || !m.motion.pending {
		t.Fatal("activity has no tick")
	}
	generation := m.motion.generation
	if m.scheduleAnimation() != nil {
		t.Fatal("duplicate tick")
	}
	next, cmd := m.Update(animationTickMsg{generation - 1})
	m = next.(model)
	if m.motion.frame != 0 || cmd != nil {
		t.Fatal("stale tick changed clock")
	}
	next, cmd = m.Update(animationTickMsg{generation})
	m = next.(model)
	if m.motion.frame != 1 || !m.motion.pending || cmd == nil {
		t.Fatal("accepted tick not rescheduled")
	}
	next, _ = m.Update(scanResultMsg{id: m.scan.id, snapshot: app.Snapshot{Tree: m.navigation.tree, Projects: m.navigation.projects, Domains: m.navigation.domains}})
	m = next.(model)
	if m.motion.pending || m.scan.inFlight {
		t.Fatal("completed scan keeps ticking")
	}
	next, cmd = m.Update(animationTickMsg{m.motion.generation - 1})
	if next.(model).motion.frame != 1 || cmd != nil {
		t.Fatal("late completion tick mutated idle UI")
	}
	m.state = stateHelp
	m.scan.inFlight = true
	if m.scheduleAnimation() != nil {
		t.Fatal("hidden work animates help")
	}
}

func TestReducedMotionAndAsyncStartup(t *testing.T) {
	for _, environment := range []struct{ key, value string }{{"DEVDOCK_REDUCED_MOTION", "1"}, {"DEVDOCK_REDUCED_MOTION", "true"}, {"DEVDOCK_REDUCED_MOTION", "on"}, {"NO_COLOR", "1"}, {"TERM", "dumb"}} {
		t.Run(environment.key+environment.value, func(t *testing.T) {
			t.Setenv("DEVDOCK_REDUCED_MOTION", "")
			t.Setenv("NO_COLOR", "")
			t.Setenv("TERM", "xterm-256color")
			t.Setenv(environment.key, environment.value)
			if !reducedMotion() {
				t.Fatal("environment ignored")
			}
		})
	}
	root := t.TempDir()
	project := filepath.Join(root, "apps", "日本語-👩‍💻-é")
	if err := os.MkdirAll(project, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module demo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	m := renderFixture()
	m.cfg = config.Config{Roots: []string{root}}
	m.motion = animationClock{reduced: true}
	m.navigation.projects = nil
	m.scan.startup = m.scanCommand()
	if !m.scan.inFlight || m.scheduleAnimation() != nil {
		t.Fatal("startup/motion flags")
	}
	// A static UI still executes the service command, without waiting on a tick.
	message := operationMessage(m.Init()).(scanResultMsg)
	stale, _ := m.Update(scanResultMsg{id: message.id - 1})
	if !stale.(model).scan.inFlight {
		t.Fatal("stale scan cleared live activity")
	}
	next, cmd := m.Update(message)
	m = next.(model)
	if len(m.navigation.projects) != 1 || m.navigation.projects[0].Path != project || m.scan.inFlight || m.motion.pending || cmd != nil {
		t.Fatal("asynchronous scan did not complete independently")
	}
}

func TestAllScreensResizeUnicodeAndTransparency(t *testing.T) {
	for name, seed := range fixtureScreens() {
		t.Run(name, func(t *testing.T) {
			for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 15}, {24, 8}, {1, 1}, {0, 0}, {300, 80}, {40, 15}} {
				next, _ := seed.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				m := next.(model)
				output := m.View()
				if !utf8.ValidString(output) {
					t.Fatal("invalid UTF-8")
				}
				if backgroundSequence(output) != "" {
					t.Fatal("background emitted")
				}
				if output != "" && len(strings.Split(output, "\n")) > size[1] {
					t.Fatalf("height at %v", size)
				}
				for _, line := range strings.Split(output, "\n") {
					if ansi.StringWidth(line) > size[0] {
						t.Fatalf("width at %v", size)
					}
				}
				if m.scan.requested {
					t.Fatal("resize requested scan")
				}
			}
		})
	}
	deep := fixtureScreens()["deep-tree"]
	if len(deep.navigation.rows) != 203 {
		t.Fatalf("lost logical nodes: %d", len(deep.navigation.rows))
	}
	for _, row := range deep.navigation.rows {
		if ansi.StringWidth(row.guide) > 10 {
			t.Fatal("unbounded visual indentation")
		}
	}
	if deep.navigation.rows[len(deep.navigation.rows)-1].depth != 201 {
		t.Fatal("lost logical depth")
	}
	for _, label := range []string{"日本語", "👩‍💻", "é"} {
		if !strings.Contains(ansi.Strip(ui.Activity(label, 10, false)), label) {
			t.Fatalf("shimmer corrupted graphemes %q", label)
		}
	}
}

func TestNarrowEditorHintsAndLongDefinitionPaging(t *testing.T) {
	for _, width := range []int{24, 40, 60} {
		e := newEditorScreen(nil, nil, width, 15)
		e.layer = editorLayerConfig
		e.ce = newConfigurationEditor(strings.Repeat("long-value-", 30))
		output := ansi.Strip(e.View())
		if !strings.Contains(output, "ctrl+s") || !strings.Contains(output, "esc") {
			t.Fatalf("lost save/cancel at %d: %s", width, output)
		}
	}
	e := newEditorScreen(nil, nil, 40, 15)
	e.deleting = true
	e.deleteName = strings.Repeat("日本語-folder/", 80) + "ZZEND"
	e.tab = editorTabTemplates
	if e.editorScrollLimit() == 0 {
		t.Fatal("long definition not pageable")
	}
	for i := 0; i < 100; i++ {
		e, _ = e.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	}
	readable := strings.Join(strings.Fields(strings.ReplaceAll(ansi.Strip(e.View()), "│", "")), "")
	if !strings.Contains(readable, "ZZEND") {
		t.Fatalf("full target not recoverable at %d/%d:\n%s", e.scroll, e.editorScrollLimit(), ansi.Strip(e.View()))
	}
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if e.deleting {
		t.Fatal("confirmation did not cancel")
	}
}

func TestStatusDetailsRecoverFullError(t *testing.T) {
	m := renderFixture()
	m.termW, m.termH = 40, 15
	m.statusMsg = "! Failed: " + strings.Repeat("/日本語-é", 200) + "/END-OF-PATH"
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
	m = next.(model)
	if m.state != stateStatusDetails || m.statusDetails != m.statusMsg {
		t.Fatal("status details lost original")
	}
	for i := 0; i < 300; i++ {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
		m = next.(model)
	}
	if !strings.Contains(ansi.Strip(m.View()), "END-OF-PATH") {
		t.Fatal("long error cannot be read")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(model).state != stateList {
		t.Fatal("status escape")
	}
}

func TestCachedInspectorAndCompactPTYControls(t *testing.T) {
	m := renderFixture()
	name := tmux.SessionName(m.navigation.projects[0])
	next, _ := m.Update(tmuxSessionsMsg{id: m.tmux.id, sessions: []string{name}})
	m = next.(model)
	if !strings.Contains(strings.Join(m.inspectorLines(40), "\n"), m.navigation.projects[0].Name+" · active (cached)") {
		t.Fatal("known session missing")
	}
	for _, width := range []int{24, 40, 120} {
		m.termW, m.termH = width, 8
		staticPTYFixture(&m)
		if !strings.Contains(ansi.Strip(m.View()), "ctrl+c") {
			t.Fatalf("interrupt hint lost at %d", width)
		}
	}
}

func BenchmarkDeepTreeAndSelection(b *testing.B) {
	b.Run("deep-200", func(b *testing.B) {
		m := fixtureScreens()["deep-tree"]
		m.sizePresentation()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = m.View()
		}
	})
	b.Run("selection-5000", func(b *testing.B) {
		m := renderFixture()
		p := m.navigation.projects[0]
		m.navigation.projects = nil
		m.selected = map[string]bool{}
		for i := 0; i < 5000; i++ {
			q := p
			q.Name = fmt.Sprint("project-", i)
			q.Path = p.Path + fmt.Sprint(i)
			m.navigation.projects = append(m.navigation.projects, q)
			m.selected[q.Path] = true
		}
		m = m.rebuildList(false)
		m.sizePresentation()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = m.View()
		}
	})
	b.Run("resize-burst", func(b *testing.B) {
		m := renderFixture()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			next, _ := m.Update(tea.WindowSizeMsg{Width: 40 + i%100, Height: 15 + i%30})
			m = next.(model)
		}
	})
}

// Keep a PTY render fixture free of subprocesses and timestamps.
func staticPTYFixture(m *model) {
	m.state = statePTYExecution
	m.ptyScr = newPTYScreen(m.termW, m.termH, nil, "/workspace/work/backend/billing", tmpl.Vars{}, []tmpl.TemplateStep{{Type: "command", Run: "echo done"}}, "/workspace/work/backend/billing", gh.Repo{})
	m.ptyScr.scaffold.Execution.Next()
	m.ptyScr.viewport.SetContent("$ printf 日本語 é 👩‍💻\nPreparing workspace…")
}
