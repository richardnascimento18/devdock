package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/app"
)

type createPreflightMsg struct {
	id  uint64
	err error
}
type createDoneMsg struct {
	id     uint64
	result app.CreateResult
	err    error
}

func (m model) preflightCreation() (tea.Model, tea.Cmd) {
	m.operationID++
	id, service, ctx := m.operationID, m.workspaces, m.context()
	location, name := m.currentLocation(), m.creation.name
	m.creation.returnState = m.state
	m.creation.failure = app.Diagnostic{}
	m.state = statePreparingProject
	m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Preparing %q...", name))
	return m, func() tea.Msg { return createPreflightMsg{id: id, err: service.CheckCreate(ctx, location, name)} }
}
func (m model) handleCreatePreflight(msg createPreflightMsg) (tea.Model, tea.Cmd) {
	if m.state != statePreparingProject || msg.id != m.operationID {
		return m, nil
	}
	if msg.err != nil {
		m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: msg.err.Error(), Cause: msg.err, Operation: "create-preflight"}
		m.state = stateList
		return m, nil
	}
	if m.creation.createGitHub && m.cfg.IsGitHubConnected() {
		m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Creating GitHub repo %q...", m.creation.name))
		m.state = stateCreatingGitHub
		return m, cmdCreateRepo(m.context(), m.cfg.GitHubToken, m.creation.name, m.creation.private)
	}
	return m.beginLocalCreation()
}
func (m model) beginLocalCreation() (tea.Model, tea.Cmd) {
	m.operationID++
	id, service, ctx := m.operationID, m.workspaces, m.context()
	request := app.CreateRequest{Location: m.currentLocation(), Name: m.creation.name, Template: m.creation.template, Repo: m.creation.repo}
	m.state = statePreparingProject
	m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Creating %q...", request.Name))
	return m, func() tea.Msg {
		result, err := service.Create(ctx, request)
		return createDoneMsg{id: id, result: result, err: err}
	}
}
func (m model) handleCreateDone(msg createDoneMsg) (tea.Model, tea.Cmd) {
	if m.state != statePreparingProject || msg.id != m.operationID {
		return m, nil
	}
	if msg.err != nil {
		if m.creation.template == nil && m.creation.repo.FullName == "" {
			m.state = m.creation.returnState
			m.creation.failure = app.Diagnostic{Severity: app.Error, Summary: "create project: " + msg.err.Error(), Cause: msg.err, Operation: "create"}
		} else {
			m.state = stateList
			m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "create project: " + msg.err.Error(), Cause: msg.err, Operation: "create"}
		}
		return m.rescan(), nil
	}
	if msg.result.Scaffold != nil {
		m.ptyScr = newPTYScreenWithScaffold(m.context(), m.termW, m.termH, m.creation.template, *msg.result.Scaffold)
		m.operationID++
		m.ptyScr.operationID = m.operationID
		m.state = statePTYExecution
		m.launch.preset = m.creation.preset
		return m, m.ptyScr.startNextStep()
	}
	if msg.result.Warning != nil {
		m.diagnostic = app.Diagnostic{Severity: app.Info, Summary: fmt.Sprintf("note: could not write .devdock marker: %v", msg.result.Warning), Cause: msg.result.Warning, Operation: "create-marker"}
	}
	m.uiState.AddRecent(msg.result.Project)
	m.saveState()
	m.launch.project, m.launch.ready, m.launch.preset = msg.result.Project, true, m.creation.preset
	m.state = stateList
	return m.rescan(), tea.Quit
}
