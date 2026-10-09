package main

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	gh "github.com/richardnascimento18/devdock/internal/github"
)

type authClient interface {
	StartDeviceFlow(context.Context) (gh.DeviceCodeResponse, error)
	PollForToken(context.Context, gh.DeviceCodeResponse) (string, error)
	FetchUsername(context.Context, string) (string, error)
}

func (m *model) cancelAuth() {
	if m.auth.cancel != nil {
		m.auth.cancel()
		m.auth.cancel = nil
	}
	m.auth.id++
}
func (m model) startGitHubAuth() (tea.Model, tea.Cmd) {
	if m.cfg.IsGitHubConnected() {
		return m, m.fetchRepos()
	}
	m.cancelAuth()
	ctx, cancel := context.WithCancel(m.context())
	m.auth.cancel = cancel
	m.auth.context = ctx
	if m.auth.client == nil {
		m.auth.client = gh.NewClient()
	}
	client, id := m.auth.client, m.auth.id
	m.auth.screen = githubAuthScreen{}
	m.state = stateGitHubAuth
	return m, func() tea.Msg {
		dc, err := client.StartDeviceFlow(ctx)
		return DeviceStartedMsg{Code: dc, Err: err, ID: id}
	}
}
func (m model) handleDeviceStarted(msg DeviceStartedMsg) (tea.Model, tea.Cmd) {
	if m.state != stateGitHubAuth || msg.ID != m.auth.id {
		return m, nil
	}
	if msg.Err != nil {
		m.auth.screen.err = msg.Err.Error()
		return m, nil
	}
	m.auth.code = msg.Code
	m.auth.screen.userCode = msg.Code.UserCode
	m.auth.screen.verificationURI = msg.Code.VerificationURI
	client, ctx, id := m.auth.client, m.auth.context, m.auth.id
	return m, func() tea.Msg {
		token, err := client.PollForToken(ctx, msg.Code)
		if err != nil {
			return AuthDoneMsg{Err: err, ID: id}
		}
		username, err := client.FetchUsername(ctx, token)
		return AuthDoneMsg{Token: token, Username: username, Err: err, ID: id}
	}
}
func (m *model) fetchRepos() tea.Cmd {
	m.repositories.loading = true
	m.repositories.id++
	return cmdFetchRepos(m.context(), m.cfg.GitHubToken, m.repositories.id)
}
