package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// Executable installs a controlled script in an isolated directory. Callers
// explicitly add it to PATH; it never replaces an installed workstation tool.
func Executable(t testing.TB, name, script string) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, name), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return directory
}
