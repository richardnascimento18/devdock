package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/app"
)

type scopeMutation uint8

const (
	mutationGroup scopeMutation = iota
	mutationDomain
	mutationMoveDomain
	mutationCloneDomain
)

type scopeDoneMsg struct {
	id       uint64
	kind     scopeMutation
	name     string
	previous appState
	err      error
}

func (m model) beginScopeMutation(kind scopeMutation, name string) (tea.Model, tea.Cmd) {
	m.operationID++
	id, previous, service, ctx := m.operationID, m.state, m.workspaces, m.context()
	root, parent, tree := m.pendingRoot, m.groupFlow.parent, m.navigation.tree
	m.state = stateChangingScope
	m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Creating %q...", name))
	return m, func() tea.Msg {
		var err error
		if err = ctx.Err(); err == nil {
			if kind == mutationGroup {
				err = service.CreateGroups(tree, parent, strings.Split(name, "/"))
			} else {
				err = service.CreateDomain(root, name)
			}
		}
		return scopeDoneMsg{id: id, kind: kind, name: name, previous: previous, err: err}
	}
}
func (m model) handleScopeDone(msg scopeDoneMsg) (tea.Model, tea.Cmd) {
	if m.state != stateChangingScope || msg.id != m.operationID {
		return m, nil
	}
	if msg.err != nil {
		m.state = msg.previous
		m.inputScr.err = msg.err.Error()
		return m, nil
	}
	switch msg.kind {
	case mutationCloneDomain:
		return m.openPlacement(m.pendingRoot, msg.name, placeClone), nil
	case mutationMoveDomain:
		return m.openMovePlacementPicker(m.pendingRoot, msg.name), nil
	case mutationGroup:
		m.pendingDomain = ""
		m.diagnostic = app.Diagnostic{Severity: app.Success, Summary: fmt.Sprintf("✓  group %q created", msg.name)}
	}
	m.state = stateList
	return m.rescan(), nil
}
