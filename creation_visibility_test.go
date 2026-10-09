package main

import (
	"context"
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/state"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/testutil"
)

func TestCreationFailuresVisibleInRestoredViews(t *testing.T) {
	for _, screen := range []appState{statePickTemplate, stateAskCreateGitHub, stateAskRepoPrivacy} {
		for _, reduced := range []bool{false, true} {
			m := fixtureModel(t, t.TempDir())
			m.termW, m.termH, m.motion.reduced = 120, 40, reduced
			m.pendingRoot, m.pendingDomain = m.cfg.Roots[0], "apps"
			m.creation.name, m.state = "example", screen
			m.templatePicker = newTemplatePicker(m.templates)
			m.yesNoScr = newYesNoScreen("Create a GitHub repository?")
			failure := errors.New("injected creation failure")
			calls := 0
			m.workspaces.CreateProject = func(core.Location, string) (core.Project, error) { calls++; return core.Project{}, failure }
			next, cmd := m.finishCreateProject(m.pendingRoot, "apps", m.presets[0])
			m = next.(model)
			next, cmd = m.Update(operationMessage(cmd))
			m = next.(model)
			completion := operationMessage(cmd)
			next, _ = m.Update(completion)
			m = next.(model)
			if m.state != screen || m.creation.name != "example" || m.launch.ready || calls != 1 {
				t.Fatal("failed creation changed draft, launched, or executed twice")
			}
			if !strings.Contains(m.View(), failure.Error()) {
				t.Fatalf("creation error invisible in restored screen %d", screen)
			}
			assertSafeDisplay(t, m.View())
			// A duplicate completion is rejected after returning to the picker.
			next, cmd = m.Update(completion)
			m = next.(model)
			if cmd != nil || m.state != screen || m.launch.ready {
				t.Fatal("duplicate failure changed workflow")
			}
			// Starting a retry clears the old diagnostic and rejects the previous ID.
			next, cmd = m.finishCreateProject(m.pendingRoot, "apps", m.presets[0])
			m = next.(model)
			next, _ = m.Update(completion)
			m = next.(model)
			if m.state != statePreparingProject || strings.Contains(m.View(), failure.Error()) {
				t.Fatal("stale completion corrupted retry")
			}
			next, cmd = m.Update(operationMessage(cmd))
			m = next.(model)
			next, _ = m.Update(operationMessage(cmd))
			m = next.(model)
			if calls != 2 || !strings.Contains(m.View(), failure.Error()) {
				t.Fatal("retry outcome invisible")
			}
		}
	}
}

func TestCreationPartialOutcomesRemainVisible(t *testing.T) {
	for _, mode := range []string{"local-marker", "remote-marker", "template-prepare", "persistence"} {
		t.Run(mode, func(t *testing.T) {
			m := fixtureModel(t, t.TempDir())
			m.termW, m.termH = 120, 40
			m.motion.reduced = true
			m.pendingRoot, m.pendingDomain = m.cfg.Roots[0], "apps"
			m.creation.name, m.creation.returnState = "example", statePickTemplate
			failure := errors.New("controlled partial outcome")
			m.workspaces.WriteMarker = func(string) error { return failure }
			if mode == "remote-marker" {
				m.creation.repo = gh.Repo{FullName: "owner/example"}
				m.workspaces.Git.Run = func(context.Context, string, ...string) error { t.Fatal("Git after marker failure"); return nil }
			}
			if mode == "template-prepare" {
				m.creation.template = &tmpl.Template{Name: "example"}
				m.workspaces.Prepare = func(core.Location, string, bool) (string, string, error) { return "", "", failure }
			}
			if mode == "persistence" {
				m.workspaces.WriteMarker = tmpl.WriteDevDockMarkerFile
				m.preferences.State = testutil.StateSave(func(state.UIState) error { return failure })
			}
			next, cmd := m.beginLocalCreation()
			m = next.(model)
			completion := operationMessage(cmd)
			next, cmd = m.Update(completion)
			m = next.(model)
			// Completion schedules the asynchronous filesystem reconciliation.
			m = deliverCreationScan(m, cmd)
			if mode == "remote-marker" || mode == "template-prepare" {
				if m.launch.ready || !strings.Contains(m.View(), failure.Error()) {
					t.Fatal("fatal outcome invisible or launched")
				}
			} else if !m.launch.ready {
				t.Fatal("nonfatal warning blocked existing launch policy")
			}
			if mode == "local-marker" && !strings.Contains(m.View(), failure.Error()) {
				t.Fatal("marker warning invisible")
			}
			if mode == "persistence" && !errors.Is(m.persistenceErr, failure) {
				t.Fatal("post-creation persistence error lost")
			}
			if mode != "template-prepare" {
				if _, err := os.Stat(filepath.Join(m.pendingRoot, "apps", "example")); err != nil {
					t.Fatal("partial directory lost", err)
				}
			}
			next, cmd = m.Update(completion)
			if cmd != nil || next.(model).state != m.state {
				t.Fatal("duplicate completion finalized twice")
			}
		})
	}
}

func TestCancelledAndSupersededCreationCannotLaunch(t *testing.T) {
	for _, reduced := range []bool{false, true} {
		m := fixtureModel(t, t.TempDir())
		m.termW, m.termH, m.motion.reduced = 120, 40, reduced
		m.creation.name, m.creation.returnState = "example", statePickTemplate
		m.pendingRoot, m.pendingDomain = m.cfg.Roots[0], "apps"
		ctx, cancel := context.WithCancel(context.Background())
		m.appContext = ctx
		next, cmd := m.beginLocalCreation()
		m = next.(model)
		cancel()
		completion := operationMessage(cmd)
		next, _ = m.Update(completion)
		m = next.(model)
		if m.launch.ready || !strings.Contains(m.View(), context.Canceled.Error()) {
			t.Fatal("cancellation was silent or launched")
		}
		m.state, m.operationID = statePreparingProject, m.operationID+1
		next, cmd = m.Update(createDoneMsg{id: m.operationID - 1})
		after := next.(model)
		if after.launch.ready || after.state != statePreparingProject || after.operationID != m.operationID || after.creation.name != m.creation.name {
			t.Fatal("late success changed newer operation")
		}
		assertOnlyAnimationCommand(t, cmd)
	}
}

// Reject operation effects while permitting Update's independent motion clock.
func assertOnlyAnimationCommand(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		for _, command := range msg {
			assertOnlyAnimationCommand(t, command)
		}
	case animationTickMsg:
	default:
		t.Fatalf("stale completion scheduled an operation effect: %T", msg)
	}
}

// Creation success batches Quit with reconciliation; consume the scan rather
// than mistaking the first Quit message for the filesystem result.
func deliverCreationScan(m model, cmd tea.Cmd) model {
	if cmd == nil {
		return m
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		for _, command := range msg {
			m = deliverCreationScan(m, command)
		}
	case scanResultMsg:
		next, _ := m.Update(msg)
		m = next.(model)
	}
	return m
}
