package app

import (
	"context"
	"errors"
	"testing"

	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
)

func TestCreationMarkerFailureRetainsExistingLocalAndRemotePolicy(t *testing.T) {
	failure := errors.New("marker failure")
	service := NewWorkspaces()
	service.CreateProject = func(core.Location, string) (core.Project, error) {
		return core.Project{Path: "/workspace/example"}, nil
	}
	service.WriteMarker = func(string) error { return failure }
	service.Git.Run = func(context.Context, string, ...string) error { t.Fatal("Git ran after marker failure"); return nil }
	result, err := service.Create(context.Background(), CreateRequest{Name: "example"})
	if err != nil || !errors.Is(result.Warning, failure) || result.Project.Path == "" {
		t.Fatal("local warning policy changed")
	}
	result, err = service.Create(context.Background(), CreateRequest{Name: "example", Repo: gh.Repo{FullName: "owner/example"}})
	if !errors.Is(err, failure) || result.Project.Path == "" {
		t.Fatal("remote stop policy changed")
	}
}

func TestCreationCancellationDoesNotMutateFilesystem(t *testing.T) {
	service := NewWorkspaces()
	service.CreateProject = func(core.Location, string) (core.Project, error) {
		t.Fatal("creation after cancellation")
		return core.Project{}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Create(ctx, CreateRequest{Name: "example"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
