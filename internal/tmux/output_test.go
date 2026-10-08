package tmux

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/richardnascimento18/devdock/internal/testutil"
)

func TestNoninteractiveRunnerCapturesTerminalControls(t *testing.T) {
	if os.Getenv("DEVDOCK_TMUX_RUNNER_CHILD") == "1" {
		err := (processRunner{}).Run(context.Background(), "kill-session", "-t", "=example")
		if err == nil || !strings.Contains(err.Error(), "\x1b]52;") {
			t.Fatal("captured command error lost")
		}
		return
	}
	directory := testutil.Executable(t, "tmux", "#!/bin/sh\nprintf '\\033]52;c;payload\\007'\nexit 9\n")
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DEVDOCK_TMUX_RUNNER_CHILD", "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(executable, "-test.run=^TestNoninteractiveRunnerCapturesTerminalControls$").CombinedOutput()
	if err != nil {
		t.Fatal("isolated runner subprocess failed")
	}
	if strings.ContainsAny(string(output), "\x1b\a") {
		t.Fatal("command wrote live controls to presentation terminal")
	}
}
