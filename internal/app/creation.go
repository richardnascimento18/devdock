package app

import (
	"context"

	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/template"
)

type CreateRequest struct {
	Location core.Location
	Name     string
	Template *template.Template
	Repo     gh.Repo
}
type CreateResult struct {
	Project  core.Project
	Scaffold *Scaffold
	Warning  error
}

// CheckCreate runs before remote repository creation and is repeated by the
// filesystem implementation at mutation time. No name-based identity matching.
func (s Workspaces) CheckCreate(ctx context.Context, location core.Location, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := location.ProjectPath(name)
	if err != nil {
		return err
	}
	return core.CheckDestination(location.Root, path)
}

func (s Workspaces) Create(ctx context.Context, request CreateRequest) (CreateResult, error) {
	if err := ctx.Err(); err != nil {
		return CreateResult{}, err
	}
	if request.Template != nil {
		path, workDir, err := s.Prepare(request.Location, request.Name, request.Template.CreatesProjectFolder)
		if err != nil {
			return CreateResult{}, err
		}
		vars := template.Vars{ProjectName: request.Name, Domain: request.Location.Domain, Root: request.Location.Root, ProjectPath: path}
		scaffold := NewScaffold(request.Template, path, vars, request.Template.Steps, workDir, request.Repo.CloneURL)
		scaffold.Git = s.Git
		return CreateResult{Scaffold: &scaffold}, nil
	}
	project, err := s.CreateProject(request.Location, request.Name)
	if err != nil {
		return CreateResult{}, err
	}
	result := CreateResult{Project: project}
	if err := s.WriteMarker(project.Path); err != nil {
		// Preserve the existing distinction: local creation warns; a remote
		// workflow must stop before initialization when its marker fails.
		if request.Repo.FullName != "" {
			return result, err
		}
		result.Warning = err
	}
	if request.Repo.FullName != "" {
		result.Project.GitHubRepo = request.Repo.FullName
		if err := s.Git.Init(ctx, project.Path, request.Repo.CloneURL); err != nil {
			return result, err
		}
	}
	return result, nil
}
