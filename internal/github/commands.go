package github

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/core"
)

type ReposLoadedMsg struct {
	Repos []Repo
	Err   error
	ID    uint64
}
type DeviceStartedMsg struct {
	Code DeviceCodeResponse
	Err  error
	ID   uint64
}
type AuthDoneMsg struct {
	Token, Username string
	Err             error
	ID              uint64
}
type RepoCreatedMsg struct {
	Repo Repo
	Err  error
}
type CloneDoneMsg struct {
	Project core.Project
	Err     error
}

func CmdFetchRepos(token string, id uint64) tea.Cmd {
	return func() tea.Msg {
		repos, err := NewClient().FetchRepos(context.Background(), token)
		return ReposLoadedMsg{Repos: repos, Err: err, ID: id}
	}
}
func CmdCreateRepo(token, name string, private bool) tea.Cmd {
	return func() tea.Msg {
		repo, err := NewClient().CreateRepo(context.Background(), token, name, private)
		return RepoCreatedMsg{Repo: repo, Err: err}
	}
}
func CmdCloneRepo(cloneURL, root, domain, repoName string) tea.Cmd {
	return func() tea.Msg {
		dest, _, err := core.PrepareProject(root, domain, repoName, true)
		if err != nil {
			return CloneDoneMsg{Err: err}
		}
		if err := CloneRepo(cloneURL, dest); err != nil {
			return CloneDoneMsg{Err: err}
		}
		return CloneDoneMsg{Project: core.Project{Name: repoName, Path: dest, Root: root, Domain: domain, GitHubRepo: DetectRemote(dest)}}
	}
}
