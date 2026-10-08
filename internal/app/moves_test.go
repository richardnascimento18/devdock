package app

import (
	"context"
	"errors"
	"testing"

	"github.com/richardnascimento18/devdock/internal/core"
)

func TestMovePlanningFailureAndCancellationPreventMutation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	failure := errors.New("destination collision")
	mutate := func(core.MovePlan) (core.Project, error) {
		t.Fatal("mutation after rejected plan or cancellation")
		return core.Project{}, nil
	}
	result := Move(ctx, core.Project{}, core.Location{}, func(core.Project, core.Location, string) (core.MovePlan, error) {
		return core.MovePlan{Status: core.MoveDestinationExists}, failure
	}, mutate)
	if result.Planned || !errors.Is(result.Err, failure) || result.Plan.Status != core.MoveDestinationExists {
		t.Fatal("planning failure lost")
	}
	result = Move(ctx, core.Project{}, core.Location{}, func(core.Project, core.Location, string) (core.MovePlan, error) {
		cancel()
		return core.MovePlan{}, nil
	}, mutate)
	if !result.Planned || !errors.Is(result.Err, context.Canceled) {
		t.Fatal("cancellation between planning and mutation lost")
	}
}
