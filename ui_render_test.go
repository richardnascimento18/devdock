package main

import (
	"github.com/richardnascimento18/devdock/internal/app"

	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/preset"
	"github.com/richardnascimento18/devdock/internal/state"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/tmux"
	"github.com/richardnascimento18/devdock/internal/ui"
)

func renderFixture() model {
	projects := []core.Project{
		{Location: core.Location{Root: "/workspace/work", Domain: "backend", GroupPath: []string{"services"}}, Name: "billing-api", Path: "/workspace/work/backend/services/billing-api", Languages: []string{"Go"}, GitHubRepo: "acme/billing-api"},
		{Location: core.Location{Root: "/workspace/work", Domain: "frontend"}, Name: "billing-api", Path: "/workspace/work/frontend/billing-api", Languages: []string{"TypeScript"}},
		{Location: core.Location{Root: "/workspace/work", Domain: "backend"}, Name: "auth-日本語-e\u0301", Path: "/workspace/work/backend/auth-日本語-e\u0301"},
	}
	templates := []tmpl.Template{{Name: "service", Description: "A service scaffold"}}
	m := newModel(projects, config.Config{Roots: []string{"/workspace/work"}}, preset.DefaultPresets, templates, state.UIState{TreeMode: true, Favorites: map[string]bool{projects[0].Path: true}})
	// Fixtures choose motion explicitly, independent of the test runner's env.
	m.motion.reduced = false
	root := &core.Node{Location: core.Location{Root: "/workspace/work"}, Kind: core.NodeRoot, Name: "work", ProjectCount: 3}
	backend := &core.Node{Location: core.Location{Root: "/workspace/work", Domain: "backend"}, Kind: core.NodeDomain, Name: "backend", Projects: projects[2:], ProjectCount: 2}
	services := &core.Node{Location: projects[0].Location, Kind: core.NodeGroup, Name: "services", Projects: projects[:1], ProjectCount: 1}
	frontend := &core.Node{Location: projects[1].Location, Kind: core.NodeDomain, Name: "frontend", Projects: projects[1:2], ProjectCount: 1}
	backend.Children = []*core.Node{services}
	root.Children = []*core.Node{backend, frontend}
	m.navigation.tree = core.Workspace{Roots: []*core.Node{root}, Projects: projects}
	m.navigation.domains = map[string][]string{"/workspace/work": {"backend", "frontend"}}
	return m.rebuildList(false)
}

func fixtureScreens() map[string]model {
	base := renderFixture()
	values := map[string]model{}
	add := func(name string, edit func(*model)) { m := base; edit(&m); values[name] = m }
	add("main-wide", func(m *model) {})
	add("main-medium", func(m *model) { m.termW, m.termH = 80, 24 })
	add("main-narrow", func(m *model) { m.termW, m.termH = 40, 15 })
	add("empty", func(m *model) { m.navigation.projects = nil; m.allItems = nil; m.list.SetItems(nil) })
	add("favorites", func(m *model) { m.activeTab = TabFavorites; *m = m.refreshTabList() })
	add("recents", func(m *model) {
		m.uiState.AddRecent(m.navigation.projects[0])
		m.uiState.Recents[0].OpenedAt = time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC)
		m.activeTab = TabRecents
		*m = m.refreshTabList()
	})
	add("input", func(m *model) {
		m.state = stateNewProjectName
		m.inputScr = newInputScreen("New project", "project-name", "enter confirm · esc cancel")
		m.inputScr.input.SetValue("billing")
	})
	add("long-path", func(m *model) {
		m.state = stateDeleteProject
		m.inputScr = newInputScreen("Delete project", "/workspace/"+strings.Repeat("very-long-group/", 10)+"billing-api", "esc cancel")
	})
	add("error", func(m *model) {
		m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "! Refresh failed: root unavailable. Press r to retry."}
	})
	add("loading", func(m *model) { m.state = stateCloningRepo; m.spinnerScr = newSpinnerScreen("Cloning repository…") })
	add("oauth", func(m *model) {
		m.state = stateGitHubAuth
		m.auth.screen = githubAuthScreen{verificationURI: "https://github.com/login/device", userCode: "ABCD-EFGH"}
	})
	add("help", func(m *model) { m.state = stateHelp })
	add("picker", func(m *model) {
		m.state = stateMovePickPlacement
		m.genericPicker = newRootPicker("Move project", []string{"/workspace/work/backend/services", "/workspace/work/frontend"}, "enter choose · esc cancel")
	})
	add("settings", func(m *model) {
		m.state = stateEditor
		m.editorScr = newEditorScreen(m.presets, m.templates, 120, 40)
		m.editorScr.cfg = m.cfg.Clone()
		m.editorScr.tab = editorTabSettings
	})
	add("preset", func(m *model) {
		m.state = stateEditor
		m.editorScr = newEditorScreen(m.presets, m.templates, 120, 40)
		m.editorScr.layer = editorLayerPreset
		m.editorScr.pe = newPresetEditor(m.presets[0], false)
	})
	add("template", func(m *model) {
		m.state = stateEditor
		m.editorScr = newEditorScreen(m.presets, m.templates, 120, 40)
		m.editorScr.layer = editorLayerTemplate
		m.editorScr.te = newTemplateEditor(m.templates[0], false)
	})
	add("deep-tree", func(m *model) {
		root := &core.Node{Location: core.Location{Root: "/workspace/work"}, Kind: core.NodeRoot, Name: "work", ProjectCount: 1}
		parent := &core.Node{Location: core.Location{Root: "/workspace/work", Domain: "backend"}, Kind: core.NodeDomain, Name: "backend", ProjectCount: 1}
		root.Children = []*core.Node{parent}
		for i := 0; i < 200; i++ {
			child := &core.Node{Location: parent.Location.Child(fmt.Sprintf("level-%d", i)), Kind: core.NodeGroup, Name: fmt.Sprintf("level-%d", i), ProjectCount: 1}
			parent.Children = []*core.Node{child}
			parent = child
		}
		m.navigation.tree = core.Workspace{Roots: []*core.Node{root}}
		m.navigation.selection = parent.Key()
		m.navigation.focus = ui.Workspace
		m.refreshWorkspaceRows()
	})
	add("palette", func(m *model) { next, _ := m.openPalette(); *m = next.(model) })
	add("search-active", func(m *model) {
		m.query.editing = true
		m.query.input.SetValue("billing")
		m.query.input.Focus()
		m.query.value = "billing"
		*m = m.applySearch()
	})
	add("inspector-narrow", func(m *model) { m.termW, m.termH = 40, 15; m.navigation.focus = ui.Inspector })
	add("workspace-narrow", func(m *model) { m.termW, m.termH = 40, 15; m.navigation.focus = ui.Workspace })
	add("settings-narrow", func(m *model) {
		m.termW, m.termH = 40, 15
		m.state = stateEditor
		m.editorScr = newEditorScreen(m.presets, m.templates, 40, 15)
		m.editorScr.cfg = m.cfg.Clone()
		m.editorScr.tab = editorTabSettings
	})
	add("preset-window", func(m *model) {
		m.state = stateEditor
		m.editorScr = newEditorScreen(m.presets, m.templates, 120, 40)
		m.editorScr.layer = editorLayerPreset
		m.editorScr.pe = newPresetEditor(m.presets[0], false)
		m.editorScr.pe, _ = m.editorScr.pe.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m.editorScr.pe.editFocus = 1
	})
	add("preset-pane", func(m *model) {
		m.state = stateEditor
		m.editorScr = newEditorScreen(m.presets, m.templates, 120, 40)
		m.editorScr.layer = editorLayerPreset
		m.editorScr.pe = newPresetEditor(preset.DefaultPresets[2], false)
		m.editorScr.pe.layer = pelWindowSplit
		m.editorScr.pe.splitEditor = newSplitPaneEditor(*m.editorScr.pe.windows[0].layout)
	})
	add("template-step", func(m *model) {
		m.termW, m.termH = 40, 15
		m.state = stateEditor
		m.editorScr = newEditorScreen(m.presets, m.templates, 40, 15)
		m.editorScr.layer = editorLayerTemplate
		m.editorScr.te = newTemplateEditor(tmpl.Template{Name: "service", Steps: []tmpl.TemplateStep{{Type: "command", Run: `printf "日本語 é"`, Output: "result.txt"}}}, false)
		m.editorScr.te.openStepEdit(m.editorScr.te.steps[0])
		m.editorScr.te.editFocus = 2
	})
	add("template-metadata", func(m *model) {
		m.state = stateEditor
		m.editorScr = newEditorScreen(m.presets, m.templates, 120, 40)
		m.editorScr.layer = editorLayerTemplate
		m.editorScr.te = newTemplateEditor(m.templates[0], false)
		m.editorScr.te.layer = telMetaEdit
		m.editorScr.te.metaFocus = 1
	})
	add("editor-unsaved", func(m *model) {
		m.state = stateEditor
		m.editorScr = newEditorScreen(m.presets, m.templates, 120, 40)
		m.editorScr.layer = editorLayerPreset
		m.editorScr.pe = newPresetEditor(m.presets[0], false)
		m.editorScr.pe.nameInput.SetValue("my-preset")
		m.editorScr.discarding = true
	})
	add("editor-validation", func(m *model) {
		m.state = stateEditor
		m.editorScr = newEditorScreen(m.presets, m.templates, 120, 40)
		m.editorScr.layer = editorLayerConfig
		m.editorScr.ce = newConfigurationEditor("unknown")
		m.editorScr.ce.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "! Choose an existing preset or leave blank."}
	})
	add("definition-confirmation", func(m *model) {
		m.state = stateEditor
		m.editorScr = newEditorScreen(m.presets, m.templates, 120, 40)
		m.editorScr.tab = editorTabTemplates
		m.editorScr.deleting = true
		m.editorScr.deleteName = "service"
	})
	add("multi-selection", func(m *model) {
		m.selected = map[string]bool{m.navigation.projects[0].Path: true, m.navigation.projects[1].Path: true}
	})
	add("bulk-move-confirmation", func(m *model) {
		m.state = stateBulkConfirm
		m.bulk = bulkWorkflow{projects: []core.Project{m.navigation.projects[0], m.navigation.projects[2]}, destination: core.Location{Root: "/workspace/work", Domain: "destination"}}
		for _, p := range m.bulk.projects {
			m.bulk.rows = append(m.bulk.rows, bulkMoveRow{Plan: core.MovePlan{Source: p, Destination: m.bulk.destination, Path: m.bulk.destination.Path() + "/" + p.Name}})
		}
	})
	add("confirmation", func(m *model) {
		m.state = stateDeleteDomain
		m.pendingDomainName = "backend"
		m.confirmDelDomain = newConfirmDeleteDomainScreen("backend")
	})
	add("reduced-motion", func(m *model) {
		m.state = stateCloningRepo
		m.spinnerScr = newSpinnerScreen("Cloning repository…")
		m.motion.reduced = true
	})
	add("scan-loading", func(m *model) { m.scan.inFlight = true; m.motion.frame = 10 })
	add("oauth-connecting", func(m *model) { m.state = stateGitHubAuth; m.auth.screen = githubAuthScreen{} })
	add("oauth-error", func(m *model) {
		m.state = stateGitHubAuth
		m.auth.screen = githubAuthScreen{err: "Connection unavailable. Escape and retry with g."}
	})
	add("pty-narrow", func(m *model) { m.termW, m.termH = 40, 15; staticPTYFixture(m) })
	add("status-details", func(m *model) {
		m.termW, m.termH = 40, 15
		m.state = stateStatusDetails
		m.statusDetails = "! Refresh failed: " + strings.Repeat("/日本語-👩‍💻-é", 20) + ". Check this root, then press r to retry."
	})
	add("selected-narrow", func(m *model) {
		m.termW, m.termH = 40, 15
		m.selected = map[string]bool{m.navigation.projects[0].Path: true}
	})
	add("tiny", func(m *model) { m.termW, m.termH = 20, 6 })
	pass3fFixtures(add)
	return values
}

var sgrPattern = regexp.MustCompile(`\x1b\[([0-9;:]*)m`)

// Skip complete foreground color payloads: an RGB component of 48 is legal.
func backgroundSequence(output string) string {
	for _, match := range sgrPattern.FindAllStringSubmatch(output, -1) {
		params := strings.Split(match[1], ";")
		for i := 0; i < len(params); i++ {
			compound := strings.Split(params[i], ":")
			n, _ := strconv.Atoi(compound[0])
			if n == 7 || n >= 40 && n <= 47 || n >= 100 && n <= 107 || n == 48 {
				return match[0]
			}
			if n == 38 && len(compound) == 1 && i+1 < len(params) {
				mode, _ := strconv.Atoi(params[i+1])
				if mode == 5 {
					i += 2
				} else if mode == 2 {
					i += 4
				}
			}
		}
	}
	return ""
}

func TestBackgroundDetector(t *testing.T) {
	for _, s := range []string{"\x1b[41m", "\x1b[104m", "\x1b[48;5;2m", "\x1b[48;2;1;2;3m", "\x1b[48:2:0:1:2:3m", "\x1b[7m"} {
		if backgroundSequence(s) == "" {
			t.Fatalf("missed %q", s)
		}
	}
	for _, s := range []string{"\x1b[0m", "\x1b[49m", "\x1b[38;2;48;100;40m", "\x1b[38;5;104m", "\x1b[38:2:0:48:100:40m", "\x1b[38:2::48:100:40m", "\x1b[38:5:104;49m"} {
		if backgroundSequence(s) != "" {
			t.Fatalf("rejected %q", s)
		}
	}
}

func TestPresentationStyleInvariant(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (strings.HasPrefix(entry.Name(), ".") && path != "." || strings.HasPrefix(path, "docs/")) {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "context" {
				return true
			}
			switch sel.Sel.Name {
			case "Background", "BorderBackground", "BorderTopBackground", "BorderBottomBackground", "BorderLeftBackground", "BorderRightBackground", "Reverse":
				t.Errorf("%s uses forbidden presentation style %s", path, sel.Sel.Name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRenderTransparencyProfiles(t *testing.T) {
	oldProfile, oldDark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
	defer lipgloss.SetColorProfile(oldProfile)
	defer lipgloss.SetHasDarkBackground(oldDark)
	for _, profile := range []termenv.Profile{termenv.TrueColor, termenv.ANSI256, termenv.ANSI, termenv.Ascii} {
		for _, dark := range []bool{true, false} {
			lipgloss.SetColorProfile(profile)
			lipgloss.SetHasDarkBackground(dark)
			for name, m := range fixtureScreens() {
				m.sizePresentation()
				rendered := m.View()
				if seq := backgroundSequence(rendered); seq != "" {
					t.Errorf("%s profile %v dark %v background %q", name, profile, dark, seq)
				}
				if profile == termenv.Ascii && strings.Contains(rendered, "\x1b[") {
					t.Errorf("%s styles in NO_COLOR output", name)
				}
			}
		}
	}
}

func TestRenderGoldens(t *testing.T) {
	oldProfile, oldDark := lipgloss.ColorProfile(), lipgloss.HasDarkBackground()
	defer lipgloss.SetColorProfile(oldProfile)
	defer lipgloss.SetHasDarkBackground(oldDark)
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)
	for name, m := range fixtureScreens() {
		t.Run(name, func(t *testing.T) {
			m.sizePresentation()
			rendered := m.View()
			for _, ext := range []string{"ansi", "txt"} {
				value := rendered
				if ext == "txt" {
					value = ansi.Strip(rendered)
				}
				// Padding is meaningful inside the render, but trailing blank cells
				// need not be stored. Keep glyph positions and ANSI sequences intact.
				lines := strings.Split(value, "\n")
				for i := range lines {
					lines[i] = strings.TrimRight(lines[i], " \t\r")
				}
				value = strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
				path := filepath.Join("testdata", "ui", name+"."+ext)
				if os.Getenv("DEVDOCK_UPDATE_GOLDEN") == "1" {
					if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte(value), 0644); err != nil {
						t.Fatal(err)
					}
				}
				want, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if string(want) != value {
					t.Fatalf("render differs: %s; inspect before DEVDOCK_UPDATE_GOLDEN=1", path)
				}
			}
		})
	}
}

func TestFocusHelpResizeAndEscape(t *testing.T) {
	m := renderFixture()
	for _, k := range []tea.KeyMsg{{Type: tea.KeyTab}, {Type: tea.KeyShiftTab}} {
		next, _ := m.Update(k)
		m = next.(model)
	}
	if m.navigation.focus != ui.Projects {
		t.Fatal("focus cycle")
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if cmd != nil && cmd() == tea.Quit() {
		t.Fatal("escape quit")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = next.(model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(model)
	if m.state != stateHelp || m.helpScroll != 1 {
		t.Fatal("help scrolling")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if m.state != stateList {
		t.Fatal("help cancel")
	}
	for _, sz := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 15}, {1, 1}, {0, 0}, {300, 80}, {40, 15}} {
		next, _ = m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		m = next.(model)
		v := m.View()
		if v != "" && len(strings.Split(v, "\n")) > sz[1] {
			t.Fatalf("height %v", sz)
		}
		for _, line := range strings.Split(v, "\n") {
			if ansi.StringWidth(line) > sz[0] {
				t.Fatalf("width %v", sz)
			}
		}
		if m.scan.requested {
			t.Fatal("resize requested scan")
		}
	}
}

func BenchmarkFoundationRender(b *testing.B) {
	for _, count := range []int{1000, 5000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			m := renderFixture()
			seed := m.navigation.projects[0]
			m.navigation.projects = nil
			for i := 0; i < count; i++ {
				p := seed
				p.Name = fmt.Sprintf("service-%d", i)
				p.Path = seed.Path + strconv.Itoa(i)
				m.navigation.projects = append(m.navigation.projects, p)
			}
			m = m.rebuildList(false)
			m.sizePresentation()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = m.View()
			}
		})
	}
}

func pass3fFixtures(add func(string, func(*model))) {
	for _, focus := range []ui.Pane{ui.Workspace, ui.Projects, ui.Inspector} {
		pane := focus
		add("focus-"+strings.ToLower(pane.String()), func(m *model) { m.navigation.focus = pane })
	}
	add("metadata-tmux-github-local", func(m *model) {
		m.navigation.projects = append([]core.Project(nil), m.navigation.projects...)
		m.navigation.projects[0].Languages = []string{"Express"}
		m.navigation.projects[0].LocalGit = true
		m.navigation.projects[1].LocalGit = true
		m.tmux.cached = map[string]bool{tmux.SessionName(m.navigation.projects[0]): true}
		*m = m.rebuildList(false)
	})
	add("scope-picker", func(m *model) {
		next, _ := m.openPalette()
		*m = next.(model)
		m.palette.input.SetValue("scope services")
		m.palette.filter()
	})
	add("scan-warning-concise", func(m *model) {
		m.scan.warningCount = 2
		m.scan.warnings = "inspect /workspace/work/one: invalid .devdock marker\ninspect /workspace/work/two: unknown .devdock type"
		m.diagnostic = app.Diagnostic{Severity: app.Warning, Summary: "⚠ Workspace scan completed with 2 warnings · ! details"}
	})
	add("preset-picker", func(m *model) {
		m.state = statePickPreset
		m.presetPicker = newPresetPicker(m.presets, m.presets[0].Name)
	})
	add("preset-picker-long-narrow", func(m *model) {
		m.termW, m.termH = 40, 15
		m.state = statePickPreset
		values := preset.Clone(m.presets)
		values[0].Name = strings.Repeat("long-preset-", 10)
		m.presetPicker = newPresetPicker(values, values[0].Name)
	})
	add("tmux-human-confirmation", func(m *model) {
		m.state = stateDeleteTmuxSession
		m.confirmDelTmux = newConfirmDeleteTmuxScreen(tmux.SessionName(m.navigation.projects[0]))
		m.confirmDelTmux.targetName = "backend/services/billing-api"
		m.confirmDelTmux.input.Placeholder = m.confirmDelTmux.targetName
	})
	add("tmux-human-list", func(m *model) {
		m.tmux.sessions = []string{tmux.SessionName(m.navigation.projects[0]), tmux.SessionName(m.navigation.projects[2]), "legacy-session"}
		m.activeTab = TabTmux
		*m = m.refreshTabList()
	})
	add("shimmer-delayed", func(m *model) { m.scan.inFlight = true; m.motion.frame = 0 })
	add("shimmer-running", func(m *model) { m.scan.inFlight = true; m.motion.frame = 24 })
	for _, width := range []int{140, 80, 40} {
		width := width
		for _, kind := range []string{"preset", "template"} {
			kind := kind
			add(fmt.Sprintf("editor-%s-%d", kind, width), func(m *model) {
				m.termW, m.termH = width, 30
				m.state = stateEditor
				m.editorScr = newEditorScreen(m.presets, m.templates, width, 30)
				if kind == "preset" {
					m.editorScr.layer = editorLayerPreset
					m.editorScr.pe = newPresetEditor(m.presets[0], false)
					m.editorScr.pe, _ = m.editorScr.pe.Update(tea.KeyMsg{Type: tea.KeyEnter})
				} else {
					m.editorScr.tab = editorTabTemplates
					m.editorScr.layer = editorLayerTemplate
					m.editorScr.te = newTemplateEditor(tmpl.Template{Name: "service", Steps: []tmpl.TemplateStep{{Type: "command", Run: "pwd", Output: "result.txt"}}}, false)
					m.editorScr.te.openStepEdit(m.editorScr.te.steps[0])
				}
			})
		}
	}
	add("settings-add-root-inline", func(m *model) {
		m.state = stateAddRoot
		m.editorRootReturn = true
		m.editorScr = newEditorScreen(m.presets, m.templates, 120, 40)
		m.editorScr.tab = editorTabSettings
		m.editorScr.cursor = 1
		m.inputScr = newInputScreen("Add root directory", "/mounted/workspace", "enter save · esc cancel")
	})
}
