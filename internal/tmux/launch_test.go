package tmux

import (
	"errors"
	"testing"

	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/preset"
)

func TestLaunchPlanSnapshotsPresetAndReportsOwnedResources(t *testing.T) {
	t.Setenv("TMUX", "")
	ps := preset.Clone(preset.DefaultPresets)[2]
	plan, err := PlanWorkspace(core.Project{Path: "/workspace/example"}, ps)
	if err != nil {
		t.Fatal(err)
	}
	ps.Windows[0].Layout.Panes[0].Command = "changed"
	if plan.Windows[0].Layout.Panes[0].Command == "changed" {
		t.Fatal("plan shares editable preset")
	}
	for _, test := range []struct {
		failAt          int
		existing, owned bool
	}{
		{1, false, false}, {2, false, false}, {3, false, true}, {2, true, false},
	} {
		fake := &fakeRunner{failAt: test.failAt, existing: test.existing}
		err := (Client{Runner: fake}).ExecuteLaunch(plan)
		var partial *LaunchError
		if !errors.As(err, &partial) || partial.Created != test.owned || partial.Session != plan.Session || partial.Cause == nil {
			t.Fatalf("incorrect ownership: %v", err)
		}
		for _, call := range fake.calls {
			if call[0] == "kill-session" {
				t.Fatal("launch destroyed session after failure")
			}
		}
	}
}
