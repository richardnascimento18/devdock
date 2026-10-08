package main

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestScopeMutationRunsInCommandAndRetainsInputOnFailure(t *testing.T) {
	m := fixtureModel(t, t.TempDir())
	m.state = stateCreateDomainOnly
	m.inputScr = newInputScreen("Domain", "example", "Enter to create")
	m.inputScr.input.SetValue("example")
	ran := false
	m.workspaces.CreateDomain = func(string, string) error {
		ran = true
		return errors.New("injected domain failure")
	}
	next, cmd := m.beginScopeMutation(mutationDomain, "example")
	m = next.(model)
	if ran || cmd == nil || m.state != stateChangingScope {
		t.Fatal("filesystem work ran in update")
	}
	next, _ = m.routeInput(tea.KeyMsg{Type: tea.KeyEnter})
	if next.(model).state != stateChangingScope || ran {
		t.Fatal("operation accepted unrelated dashboard input")
	}
	result := cmd().(scopeDoneMsg)
	stale := result
	stale.id++
	next, _ = m.handleScopeDone(stale)
	if next.(model).state != stateChangingScope {
		t.Fatal("stale completion changed route")
	}
	next, _ = m.handleScopeDone(result)
	m = next.(model)
	if !ran || m.state != stateCreateDomainOnly || m.inputScr.input.Value() != "example" || m.inputScr.err == "" {
		t.Fatal("failure lost input or did not restore route")
	}
}
