package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/richardnascimento18/devdock/internal/template"
)

func TestScaffoldSharesPostStepSequenceAndOutputExecution(t *testing.T) {
	path := t.TempDir()
	tmpl := template.Template{PostSteps: []template.TemplateStep{{Type: "builtin", Action: "touch", Path: "post"}}}
	s := NewScaffold(&tmpl, path, template.Vars{ProjectPath: path}, []template.TemplateStep{{Type: "command", Run: "printf hello", Output: "output.txt"}}, path, "")
	step, ok, err := s.Execution.Next()
	if err != nil || !ok {
		t.Fatal(err)
	}
	if result, err := s.Execute(context.Background(), step, 24, 80); err != nil || result.Session != nil {
		t.Fatalf("%+v %v", result, err)
	}
	step, ok, err = s.Execution.Next()
	if err != nil || !ok || !step.PostStarted || step.WorkDir != path {
		t.Fatal("post ordering or path")
	}
	if _, err := s.Execute(context.Background(), step, 24, 80); err != nil {
		t.Fatal(err)
	}
	_, ok, err = s.Execution.Next()
	if err != nil || ok {
		t.Fatal("sequence did not end")
	}
	if err := s.Finish(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(path, "output.txt"))
	if err != nil || string(data) != "hello" {
		t.Fatal("output handling")
	}
	for _, name := range []string{"post", ".devdock"} {
		if _, err := os.Stat(filepath.Join(path, name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestScaffoldMarkerFailureAndCancellationPreventGit(t *testing.T) {
	s := NewScaffold(nil, t.TempDir(), template.Vars{}, nil, "", "https://github.com/owner/repo")
	failure := errors.New("marker write")
	s.WriteMarker = func(string) error { return failure }
	s.Git.Run = func(context.Context, string, ...string) error {
		t.Fatal("git after marker failure/cancellation")
		return nil
	}
	if err := s.Finish(context.Background()); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.WriteMarker = func(string) error { t.Fatal("marker after cancellation"); return nil }
	if err := s.Finish(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
