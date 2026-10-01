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
	if m.authCancel != nil {
		m.authCancel()
		m.authCancel = nil
	}
	m.authID++
}
func (m model) startGitHubAuth() (tea.Model, tea.Cmd) {
	if m.cfg.IsGitHubConnected() {
		return m, m.fetchRepos()
	}
	m.cancelAuth()
	ctx, cancel := context.WithCancel(context.Background())
	m.authCancel = cancel
	m.authContext = ctx
	if m.authClient == nil {
		m.authClient = gh.NewClient()
	}
	client, id := m.authClient, m.authID
	m.githubAuthScr = githubAuthScreen{}
	m.state = stateGitHubAuth
	return m, func() tea.Msg {
		dc, err := client.StartDeviceFlow(ctx)
		return gh.DeviceStartedMsg{Code: dc, Err: err, ID: id}
	}
}
func (m model) handleDeviceStarted(msg gh.DeviceStartedMsg) (tea.Model, tea.Cmd) {
	if m.state != stateGitHubAuth || msg.ID != m.authID {
		return m, nil
	}
	if msg.Err != nil {
		m.githubAuthScr.err = msg.Err.Error()
		return m, nil
	}
	m.deviceCode = msg.Code
	m.githubAuthScr.userCode = msg.Code.UserCode
	m.githubAuthScr.verificationURI = msg.Code.VerificationURI
	client, ctx, id := m.authClient, m.authContext, m.authID
	return m, func() tea.Msg {
		token, err := client.PollForToken(ctx, msg.Code)
		if err != nil {
			return gh.AuthDoneMsg{Err: err, ID: id}
		}
		username, err := client.FetchUsername(ctx, token)
		return gh.AuthDoneMsg{Token: token, Username: username, Err: err, ID: id}
	}
}
func (m *model) fetchRepos() tea.Cmd {
	m.repoLoadID++
	return gh.CmdFetchRepos(m.cfg.GitHubToken, m.repoLoadID)
}
