package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/tmux"
)

type deleteDoneMsg struct {
	id   uint64
	path string
	err  error
}

func (m model) beginDelete(root, path string) (tea.Model, tea.Cmd) {
	m.operationID++
	id := m.operationID
	m.spinnerScr = newSpinnerScreen("Deleting " + path + "...")
	m.state = stateDeletingWorkspace
	return m, func() tea.Msg { return deleteDoneMsg{id: id, path: path, err: core.DeletePath(root, path)} }
}
func (m model) handleDeleteDone(msg deleteDoneMsg) (tea.Model, tea.Cmd) {
	if m.state != stateDeletingWorkspace || msg.id != m.operationID {
		return m, nil
	}
	if msg.err != nil {
		m.statusMsg = errorStyle.Render("delete incomplete: " + msg.err.Error())
	} else {
		m.uiState.RemovePath(msg.path)
		if m.saveFilesystemState() {
			m.statusMsg = successStyle.Render("Deleted " + msg.path)
		}
	}
	m.state = stateList
	m.pendingDomainName = ""
	m.groupFlow = groupWorkflow{}
	return m.rescan(), nil
}

type tmuxSessionsMsg struct {
	id       uint64
	sessions []string
	err      error
}
type tmuxKilledMsg struct {
	name string
	err  error
}

func (m model) refreshTmuxSessions() model { m.tmuxRefreshRequested = true; return m }
func (m *model) tmuxRefreshCommand() tea.Cmd {
	m.tmuxRefreshRequested = false
	m.tmuxRefreshing = true
	m.tmuxRefreshID++
	id := m.tmuxRefreshID
	return func() tea.Msg {
		sessions, err := tmux.ListSessions()
		return tmuxSessionsMsg{id: id, sessions: sessions, err: err}
	}
}
func (m model) handleTmuxSessions(msg tmuxSessionsMsg) (tea.Model, tea.Cmd) {
	if msg.id != m.tmuxRefreshID {
		return m, nil
	}
	m.tmuxRefreshing = false
	if msg.err != nil {
		m.statusMsg = errorStyle.Render(msg.err.Error())
	} else {
		m.tmuxSessions = msg.sessions
		m.cachedTmux = map[string]bool{}
		for _, name := range msg.sessions {
			m.cachedTmux[name] = true
		}
		m = m.refreshTabList()
	}
	return m, nil
}
func (m model) handleTmuxKilled(msg tmuxKilledMsg) (tea.Model, tea.Cmd) {
	if !m.tmuxDeleting || m.confirmDelTmux.sessionName != msg.name {
		return m, nil
	}
	m.tmuxDeleting = false
	if msg.err != nil {
		m.confirmDelTmux.err = msg.err.Error()
		return m, nil
	}
	m.state = stateList
	m.statusMsg = successStyle.Render(fmt.Sprintf("Session %q killed", m.confirmDelTmux.target()))
	return m.refreshTmuxSessions(), nil
}
