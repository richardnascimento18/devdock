package main

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/app"
	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
)

type ReposLoadedMsg struct {
	Repos []gh.Repo
	Err   error
	ID    uint64
}
type DeviceStartedMsg struct {
	Code gh.DeviceCodeResponse
	Err  error
	ID   uint64
}
type AuthDoneMsg struct {
	Token, Username string
	Err             error
	ID              uint64
}
type RepoCreatedMsg struct {
	Repo gh.Repo
	Err  error
}
type CloneDoneMsg struct {
	Project core.Project
	Err     error
}

func cmdFetchRepos(ctx context.Context, token string, id uint64) tea.Cmd {
	return func() tea.Msg {
		repos, err := gh.NewClient().FetchRepos(ctx, token)
		return ReposLoadedMsg{Repos: repos, Err: err, ID: id}
	}
}
func cmdCreateRepo(ctx context.Context, token, name string, private bool) tea.Cmd {
	return func() tea.Msg {
		repo, err := gh.NewClient().CreateRepo(ctx, token, name, private)
		return RepoCreatedMsg{Repo: repo, Err: err}
	}
}
func cmdCloneRepo(ctx context.Context, service app.Workspaces, cloneURL string, location core.Location, repoName string) tea.Cmd {
	return func() tea.Msg {
		project, err := service.Clone(ctx, cloneURL, location, repoName)
		return CloneDoneMsg{Project: project, Err: err}
	}
}
