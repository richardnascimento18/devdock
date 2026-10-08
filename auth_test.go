package main

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/app"
	"github.com/richardnascimento18/devdock/internal/config"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"testing"
)

type fakeAuth struct {
	started bool
	ctx     context.Context
}

func (f *fakeAuth) StartDeviceFlow(ctx context.Context) (gh.DeviceCodeResponse, error) {
	f.started = true
	f.ctx = ctx
	return gh.DeviceCodeResponse{}, nil
}
func (f *fakeAuth) PollForToken(context.Context, gh.DeviceCodeResponse) (string, error) {
	return "token", nil
}
func (f *fakeAuth) FetchUsername(context.Context, string) (string, error) { return "user", nil }
func TestAuthStartsAsyncAndCancelledResultsAreIgnored(t *testing.T) {
	f := &fakeAuth{}
	m := model{authClient: f}
	next, cmd := m.startGitHubAuth()
	m = next.(model)
	if f.started || cmd == nil {
		t.Fatal("network work ran in Update")
	}
	started := cmd().(gh.DeviceStartedMsg)
	if !f.started {
		t.Fatal("command did not start")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(model)
	if f.ctx.Err() != context.Canceled {
		t.Fatal("polling not cancelled")
	}
	next, cmd = m.Update(started)
	m = next.(model)
	if cmd != nil {
		t.Fatal("stale start launched poll")
	}
	next, _ = m.Update(gh.AuthDoneMsg{ID: started.ID, Token: "token", Username: "user"})
	m = next.(model)
	if m.cfg.IsGitHubConnected() {
		t.Fatal("stale auth mutated state")
	}
}
func TestAuthPersistenceFailureKeepsActiveCredentials(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m := model{state: stateGitHubAuth, authID: 1, preferences: app.NewPreferences(app.PathsAt(t.TempDir()))}
	// An invalid root makes proposed config validation fail deterministically.
	m.cfg = config.Config{Roots: []string{"relative"}}
	next, cmd := m.handleGitHubAuthDone(gh.AuthDoneMsg{ID: 1, Token: "token", Username: "user"})
	m = next.(model)
	if m.cfg.GitHubToken != "" || m.githubAuthScr.err == "" || cmd != nil {
		t.Fatal("failed credential save committed")
	}
}

func TestAuthRetryIgnoresPreviousAttemptWhileNewScreenActive(t *testing.T) {
	f := &fakeAuth{}
	m := model{authClient: f}
	first, cmd := m.startGitHubAuth()
	m = first.(model)
	old := cmd().(gh.DeviceStartedMsg)
	m.cancelAuth()
	next, _ := m.startGitHubAuth()
	m = next.(model)
	// The current attempt already has its presentation clock scheduled.
	m.motion.pending = true
	next, cmd = m.Update(gh.AuthDoneMsg{ID: old.ID, Token: "stale", Username: "stale"})
	m = next.(model)
	if m.state != stateGitHubAuth || m.cfg.GitHubToken != "" || cmd != nil {
		t.Fatal("old response affected new attempt")
	}
	m.cancelAuth()
}
