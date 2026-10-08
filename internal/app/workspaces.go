package app

import (
	"context"
	"strings"

	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/detect"
	"github.com/richardnascimento18/devdock/internal/git"
	gh "github.com/richardnascimento18/devdock/internal/github"
)

// Workspaces coordinates discovery and local repository operations. Its effects
// are explicit functions/clients, allowing failures without real external tools.
type Workspaces struct {
	Git     git.Client
	Prepare func(core.Location, string, bool) (string, string, error)
	Delete  func(string, string) error
}

func NewWorkspaces() Workspaces {
	return Workspaces{Git: git.NewClient(), Prepare: core.PrepareProject, Delete: core.DeletePath}
}

type Snapshot struct {
	Projects []core.Project
	Tree     core.Workspace
	Domains  map[string][]string
	Err      error
}

func (s Workspaces) Refresh(ctx context.Context, roots []string, repos []gh.Repo) Snapshot {
	result := Snapshot{Domains: map[string][]string{}}
	if err := ctx.Err(); err != nil {
		result.Err = err
		return result
	}
	detector := detect.New()
	result.Tree, result.Err = core.ScanWorkspace(roots, detector.Inspect)
	result.Projects = result.Tree.Projects
	for _, root := range result.Tree.Roots {
		result.Domains[root.Root] = result.Tree.Domains(root.Root)
	}
	if len(repos) > 0 {
		result.Projects = s.Associate(ctx, result.Projects, repos)
	}
	return result
}

func (s Workspaces) Clone(ctx context.Context, cloneURL string, location core.Location, name string) (core.Project, error) {
	if err := ctx.Err(); err != nil {
		return core.Project{}, err
	}
	destination, _, err := s.Prepare(location, name, true)
	if err != nil {
		return core.Project{}, err
	}
	if err := s.Git.Clone(ctx, cloneURL, destination); err != nil {
		return core.Project{}, err
	}
	return core.Project{Location: location, Name: name, Path: destination, GitHubRepo: s.Git.Remote(ctx, destination)}, nil
}

// Association is canonical owner/repository identity, never a folder/name match.
func (s Workspaces) Associate(ctx context.Context, projects []core.Project, repos []gh.Repo) []core.Project {
	byFullName := make(map[string]gh.Repo, len(repos))
	for _, repo := range repos {
		byFullName[strings.ToLower(repo.FullName)] = repo
	}
	linked := make([]core.Project, len(projects))
	for i, project := range projects {
		project.GitHubRepo = ""
		if s.Git.Initialized(ctx, project.Path) {
			if repo, ok := byFullName[strings.ToLower(s.Git.Remote(ctx, project.Path))]; ok {
				project.GitHubRepo = repo.FullName
			}
		}
		linked[i] = project
	}
	return linked
}
