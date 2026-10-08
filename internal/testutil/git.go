package testutil

import (
	"os/exec"
	"testing"
)

// Git uses a per-command generic identity, never the owner's configured author.
func Git(t testing.TB, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-c", "user.name=Example", "-c", "user.email=example@example.invalid", "-C", directory}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v", args[0], err)
	}
	return string(output)
}
