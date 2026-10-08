package main

import (
	"fmt"
	"github.com/richardnascimento18/devdock/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
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
	return m, func() tea.Msg { return deleteDoneMsg{id: id, path: path, err: m.workspaces.Delete(root, path)} }
}
func (m model) handleDeleteDone(msg deleteDoneMsg) (tea.Model, tea.Cmd) {
	if m.state != stateDeletingWorkspace || msg.id != m.operationID {
		return m, nil
	}
	if msg.err != nil {
		m.statusMsg = errorStyle.Render(ui.SafeBlock("delete incomplete: " + msg.err.Error()))
	} else {
		m.uiState.RemovePath(msg.path)
		if m.saveFilesystemState() {
			m.statusMsg = successStyle.Render(ui.SafeBlock("Deleted " + msg.path))
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

func (m model) refreshTmuxSessions() model { m.tmux.requested = true; return m }
func (m *model) tmuxRefreshCommand() tea.Cmd {
	m.tmux.requested = false
	m.tmux.refreshing = true
	m.tmux.id++
	id := m.tmux.id
	return func() tea.Msg {
		sessions, err := m.tmuxClient.ListSessions()
		return tmuxSessionsMsg{id: id, sessions: sessions, err: err}
	}
}
func (m model) handleTmuxSessions(msg tmuxSessionsMsg) (tea.Model, tea.Cmd) {
	if msg.id != m.tmux.id {
		return m, nil
	}
	m.tmux.refreshing = false
	if msg.err != nil {
		m.statusMsg = errorStyle.Render(ui.SafeBlock(msg.err.Error()))
	} else {
		m.tmux.sessions = msg.sessions
		m.tmux.cached = map[string]bool{}
		for _, name := range msg.sessions {
			m.tmux.cached[name] = true
		}
		m = m.refreshTabList()
	}
	return m, nil
}
func (m model) handleTmuxKilled(msg tmuxKilledMsg) (tea.Model, tea.Cmd) {
	if !m.tmux.deleting || m.confirmDelTmux.sessionName != msg.name {
		return m, nil
	}
	m.tmux.deleting = false
	if msg.err != nil {
		m.confirmDelTmux.err = msg.err.Error()
		return m, nil
	}
	m.state = stateList
	m.statusMsg = successStyle.Render(ui.SafeBlock(fmt.Sprintf("Session %q killed", m.confirmDelTmux.target())))
	return m.refreshTmuxSessions(), nil
}
