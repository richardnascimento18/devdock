package app

import (
	"context"
	"errors"
	"testing"

	"github.com/richardnascimento18/devdock/internal/core"
)

func TestClonePreflightFailureNeverExecutesGit(t *testing.T) {
	s := NewWorkspaces()
	failure := errors.New("prepare failure")
	s.Prepare = func(core.Location, string, bool) (string, string, error) { return "", "", failure }
	s.Git.Run = func(context.Context, string, ...string) error { t.Fatal("git after failed preflight"); return nil }
	if _, err := s.Clone(context.Background(), "unused", core.Location{}, "example"); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}

func TestCancelledCloneNeverPreparesFilesystem(t *testing.T) {
	s := NewWorkspaces()
	s.Prepare = func(core.Location, string, bool) (string, string, error) {
		t.Fatal("prepare after cancellation")
		return "", "", nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Clone(ctx, "unused", core.Location{}, "example"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
