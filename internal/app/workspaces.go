package app

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/detect"
	"github.com/richardnascimento18/devdock/internal/git"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/template"
)

// Workspaces coordinates discovery and local repository operations. Its effects
// are explicit functions/clients, allowing failures without real external tools.
type Workspaces struct {
	Git           git.Client
	Prepare       func(core.Location, string, bool) (string, string, error)
	Delete        func(string, string) error
	CreateProject func(core.Location, string) (core.Project, error)
	WriteMarker   func(string) error
	CreateDomain  func(string, string) error
	CreateGroups  func(core.Workspace, core.Location, []string) error
}

func NewWorkspaces() Workspaces {
	return Workspaces{Git: git.NewClient(), Prepare: core.PrepareProject, Delete: core.DeletePath,
		CreateProject: core.CreateProject, WriteMarker: template.WriteDevDockMarkerFile,
		CreateDomain: core.CreateDomain, CreateGroups: func(tree core.Workspace, parent core.Location, names []string) error {
			return tree.CreateGroups(parent, names)
		}}
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
	// Snapshot-local only: a later association always re-reads Git metadata and
	// remotes, including directories that have become repositories since a scan.
	identities := make(map[string]string)

	for i, project := range projects {
		project.GitHubRepo = ""
		if len(byFullName) > 0 && ctx.Err() == nil && project.Path != "" {
			path := filepath.Clean(project.Path)
			identity, inspected := identities[path]
			if !inspected {
				if s.Git.Initialized(ctx, path) {
					identity = s.Git.Remote(ctx, path)
				}
				identities[path] = identity
			}
			if repo, ok := byFullName[strings.ToLower(identity)]; ok {
				project.GitHubRepo = repo.FullName
			}
		}
		linked[i] = project
	}
	return linked
}
