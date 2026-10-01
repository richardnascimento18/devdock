package tmux

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/preset"
)

type fakeRunner struct {
	calls    [][]string
	failAt   int
	existing bool
	next     int
}

func (f *fakeRunner) call(args []string) error {
	f.calls = append(f.calls, append([]string(nil), args...))
	if len(f.calls) == f.failAt {
		return errors.New("injected failure")
	}
	return nil
}
func (f *fakeRunner) Run(args ...string) error { return f.call(args) }
func (f *fakeRunner) Output(args ...string) (string, error) {
	if err := f.call(args); err != nil {
		return "", err
	}
	if args[0] == "has-session" {
		if f.existing {
			return "", nil
		}
		return "", ErrNoSession
	}
	f.next++
	return fmt.Sprintf("%%%d\n", f.next), nil
}
func TestWorkspaceCommandsAndEveryFailure(t *testing.T) {
	t.Setenv("TMUX", "")
	p := core.Project{Domain: "apps", Name: "demo", Path: "/workspace/demo"}
	ps := preset.DefaultPresets[2]
	good := &fakeRunner{}
	if err := (Client{good}).LaunchWorkspace(p, ps); err != nil {
		t.Fatal(err)
	}
	var flags []string
	for _, args := range good.calls {
		if args[0] == "split-window" {
			flags = append(flags, args[1])
			if args[6] == "%1.%2" {
				t.Fatal("incorrect pane target")
			}
		}
	}
	if !reflect.DeepEqual(flags, []string{"-h", "-v"}) {
		t.Fatalf("splits: %v", flags)
	}
	for i := 1; i <= len(good.calls); i++ {
		fake := &fakeRunner{failAt: i}
		err := (Client{fake}).LaunchWorkspace(p, ps)
		if err == nil || !strings.Contains(err.Error(), "tmux") {
			t.Fatalf("failure %d lost: %v", i, err)
		}
		if len(fake.calls) != i {
			t.Fatalf("continued after failure %d", i)
		}
	}
}
func TestExistingSessionOnlyAttaches(t *testing.T) {
	t.Setenv("TMUX", "")
	f := &fakeRunner{existing: true}
	if err := (Client{f}).LaunchWorkspace(core.Project{Domain: "a", Name: "b"}, preset.DefaultPresets[0]); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 2 || f.calls[1][0] != "attach-session" || f.calls[1][2] != "=a-b" {
		t.Fatalf("commands: %v", f.calls)
	}
}
func TestWindowAndLiteralCommand(t *testing.T) {
	t.Setenv("TMUX", "")
	f := &fakeRunner{}
	if err := (Client{f}).LaunchWorkspace(core.Project{Domain: "a", Name: "b", Path: "/p"}, preset.DefaultPresets[0]); err != nil {
		t.Fatal(err)
	}
	windows, sends := 0, 0
	for _, args := range f.calls {
		if args[0] == "new-window" {
			windows++
		}
		if args[0] == "send-keys" {
			sends++
		}
	}
	if windows != 2 || sends != 4 {
		t.Fatalf("windows %d sends %d", windows, sends)
	}
}
