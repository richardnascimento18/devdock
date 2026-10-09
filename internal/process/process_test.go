package process

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestCaptureBoundAndExitStatus(t *testing.T) {
	output, err := Output(context.Background(), t.TempDir(), "sh", "-c", "head -c 100000 /dev/zero; printf tail; exit 7")
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 7 || len(output) != 65536 || !strings.HasSuffix(string(output), "tail") {
		t.Fatalf("exit %v size %d", err, len(output))
	}
}

func TestCancelledCommandDoesNotWaitForChild(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := Command(ctx, t.TempDir(), "sh", "-c", "sleep 60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	cancel()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancel succeeded as normal exit")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancellation hung")
	}
}
