package tmux

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

type contextRunner struct {
	ctx  context.Context
	args []string
}

func (r *contextRunner) Run(ctx context.Context, args ...string) error {
	r.ctx, r.args = ctx, args
	return ctx.Err()
}
func (r *contextRunner) Output(ctx context.Context, args ...string) (string, error) {
	r.ctx, r.args = ctx, args
	return "", ctx.Err()
}

func TestSocketAndCancellationAreExplicit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := &contextRunner{}
	socket := filepath.Join(t.TempDir(), "test-socket")
	client := Client{Runner: runner, Context: ctx, Socket: socket}
	if _, err := client.ListSessions(); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if runner.ctx != ctx || !reflect.DeepEqual(runner.args, []string{"-S", socket, "list-sessions", "-F", "#{session_name}"}) {
		t.Fatal("socket/context not propagated")
	}
	if err := client.KillSession("example"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(runner.args, []string{"-S", socket, "kill-session", "-t", "=example"}) {
		t.Fatal("kill target")
	}
}
