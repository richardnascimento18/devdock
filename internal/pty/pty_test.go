package pty

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestCompletionPreservesExitAndDrainsOutput(t *testing.T) {
	for _, tc := range []struct {
		name, script string
		code         int
	}{{"success", "printf output", 0}, {"failure", "printf output; exit 7", 7}} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NewSession([]string{"sh", "-c", tc.script}, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := s.Close(); err != nil {
					t.Error(err)
				}
			})
			var output strings.Builder
			exits := 0
			deadline := time.After(5 * time.Second)
			for {
				select {
				case msg, ok := <-s.events:
					if !ok {
						if exits != 1 || output.String() != "output" {
							t.Fatalf("completion %d output %q", exits, output.String())
						}
						return
					}
					switch msg := msg.(type) {
					case OutputMsg:
						if exits != 0 {
							t.Fatal("output after completion")
						}
						output.Write(msg.Data)
					case ExitMsg:
						exits++
						if tc.code == 0 && msg.Err != nil {
							t.Fatal(msg.Err)
						}
						if tc.code != 0 {
							var exit *exec.ExitError
							if !errors.As(msg.Err, &exit) || exit.ExitCode() != tc.code {
								t.Fatalf("exit: %v", msg.Err)
							}
						}
					}
				case <-deadline:
					t.Fatal("PTY completion hung")
				}
			}
		})
	}
}
func TestCloseReapsChild(t *testing.T) {
	s, err := NewSession([]string{"sh", "-c", "sleep 60"}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-s.waited:
	case <-time.After(5 * time.Second):
		t.Fatal("child not reaped")
	}
}
func TestStartErrors(t *testing.T) {
	for _, args := range [][]string{nil, {""}, {"/does/not/exist"}} {
		if s, err := NewSession(args, t.TempDir()); err == nil {
			s.Close()
			t.Fatal("expected error")
		}
	}
}
