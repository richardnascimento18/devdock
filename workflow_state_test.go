package main

import (
	"testing"

	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/preset"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
)

func TestNewCreationOwnsAndResetsItsDraft(t *testing.T) {
	m := fixtureModel(t, t.TempDir())
	m.creation = creationState{name: "old", preset: preset.Preset{Name: "old"}, template: &tmpl.Template{Name: "old"}, repo: gh.Repo{FullName: "owner/old"}, createGitHub: true, private: true}
	m.query.value = "retained query"
	next, _ := m.startNewProject("")
	m = next.(model)
	if m.creation.name != "" || m.creation.preset.Name != "" || m.creation.template != nil || m.creation.repo.FullName != "" || m.creation.createGitHub || m.creation.private {
		t.Fatal("new creation inherited an abandoned draft")
	}
	if m.query.value != "retained query" || m.state != stateNewProjectName {
		t.Fatal("creation reset unrelated navigation/query")
	}
}
