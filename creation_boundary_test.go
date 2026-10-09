package main

import (
	"errors"
	"testing"

	"github.com/richardnascimento18/devdock/internal/core"
)

func TestCreationEffectsRunOutsideUpdateAndFailuresRetainDraft(t *testing.T) {
	root := t.TempDir()
	m := fixtureModel(t, root)
	m.pendingRoot, m.pendingDomain = root, "apps"
	m.creation.name = "example"
	m.state = statePickTemplate
	ran := false
	m.workspaces.CreateProject = func(core.Location, string) (core.Project, error) {
		ran = true
		return core.Project{}, errors.New("injected create failure")
	}
	next, cmd := m.finishCreateProject(root, "apps", m.presets[0])
	m = next.(model)
	if ran || cmd == nil || m.state != statePreparingProject {
		t.Fatal("creation blocked update")
	}
	next, cmd = m.handleCreatePreflight(cmd().(createPreflightMsg))
	m = next.(model)
	if ran || cmd == nil {
		t.Fatal("creation ran while handling preflight")
	}
	result := cmd().(createDoneMsg)
	if !ran {
		t.Fatal("command did not execute")
	}
	next, _ = m.handleCreateDone(result)
	m = next.(model)
	if m.state != statePickTemplate || m.creation.name != "example" || m.creation.failure.Summary == "" || m.launch.ready {
		t.Fatal("failed creation lost draft or launched")
	}
}
