package main

import (
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

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/preset"
	"github.com/richardnascimento18/devdock/internal/state"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/ui"
)

func renderFixture() model {
	projects := []core.Project{
		{Location: core.Location{Root: "/workspace/work", Domain: "backend", GroupPath: []string{"services"}}, Name: "billing-api", Path: "/workspace/work/backend/services/billing-api", Languages: []string{"Go"}, GitHubRepo: "acme/billing-api"},
		{Location: core.Location{Root: "/workspace/work", Domain: "frontend"}, Name: "billing-api", Path: "/workspace/work/frontend/billing-api", Languages: []string{"TypeScript"}},
		{Location: core.Location{Root: "/workspace/work", Domain: "backend"}, Name: "auth-日本語-e\u0301", Path: "/workspace/work/backend/auth-日本語-e\u0301"},
	}
	templates := []tmpl.Template{{Name: "service", Description: "A service scaffold"}}
	return newModel(projects, config.Config{Roots: []string{"/workspace/work"}}, preset.DefaultPresets, templates, state.UIState{Favorites: map[string]bool{projects[0].Path: true}})
}

func fixtureScreens() map[string]model {
	base := renderFixture()
	values := map[string]model{}
	add := func(name string, edit func(*model)) { m := base; edit(&m); values[name] = m }
	add("main-wide", func(m *model) {})
	add("main-medium", func(m *model) { m.termW, m.termH = 80, 24 })
	add("main-narrow", func(m *model) { m.termW, m.termH = 40, 15 })
	add("empty", func(m *model) { m.rawProjects = nil; m.allItems = nil; m.list.SetItems(nil) })
	add("favorites", func(m *model) { m.activeTab = TabFavorites; *m = m.refreshTabList() })
	add("recents", func(m *model) {
		m.uiState.AddRecent(m.rawProjects[0])
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
		m.statusMsg = errorStyle.Render("! Refresh failed: root unavailable. Press r to retry.")
	})
	add("loading", func(m *model) { m.state = stateCloningRepo; m.spinnerScr = newSpinnerScreen("Cloning repository…") })
	add("oauth", func(m *model) {
		m.state = stateGitHubAuth
		m.githubAuthScr = githubAuthScreen{verificationURI: "https://github.com/login/device", userCode: "ABCD-EFGH"}
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
		m.list.SetItems([]list.Item{groupItem{name: "services", depth: 200, location: core.Location{Root: "/workspace/work", Domain: "backend", GroupPath: strings.Split(strings.Repeat("g/", 199)+"leaf", "/")}, totalCount: 3}})
	})
	add("confirmation", func(m *model) {
		m.state = stateDeleteDomain
		m.pendingDomainName = "backend"
		m.confirmDelDomain = newConfirmDeleteDomainScreen("backend")
	})
	return values
}

var sgrPattern = regexp.MustCompile(`\x1b\[([0-9;:]*)m`)

// Skip complete foreground color payloads: an RGB component of 48 is legal.
func backgroundSequence(output string) string {
	for _, match := range sgrPattern.FindAllStringSubmatch(output, -1) {
		params := strings.FieldsFunc(match[1], func(r rune) bool { return r == ';' || r == ':' })
		for i := 0; i < len(params); i++ {
			n, _ := strconv.Atoi(params[i])
			if n == 7 || n >= 40 && n <= 47 || n >= 100 && n <= 107 || n == 48 {
				return match[0]
			}
			if n == 38 && i+1 < len(params) {
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
	for _, s := range []string{"\x1b[0m", "\x1b[49m", "\x1b[38;2;48;100;40m", "\x1b[38;5;104m"} {
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
				value = strings.TrimRight(value, "\n") + "\n"
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
	if m.focus != ui.Projects {
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
		if m.scanRequested {
			t.Fatal("resize requested scan")
		}
	}
}

func BenchmarkFoundationRender(b *testing.B) {
	for _, count := range []int{1000, 5000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			m := renderFixture()
			seed := m.rawProjects[0]
			m.rawProjects = nil
			for i := 0; i < count; i++ {
				p := seed
				p.Name = fmt.Sprintf("service-%d", i)
				p.Path = seed.Path + strconv.Itoa(i)
				m.rawProjects = append(m.rawProjects, p)
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
