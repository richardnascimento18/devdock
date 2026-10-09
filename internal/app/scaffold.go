package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/richardnascimento18/devdock/internal/git"
	"github.com/richardnascimento18/devdock/internal/process"
	"github.com/richardnascimento18/devdock/internal/pty"
	"github.com/richardnascimento18/devdock/internal/template"
)

// Scaffold coordinates the sequence, step transport and successful finalization.
// PTY interaction/rendering belongs to the TUI; execution has no Tea dependency.
type Scaffold struct {
	Execution           template.Execution
	ProjectPath, Remote string
	Git                 git.Client
	RunStep             func(context.Context, template.PreparedStep) error
	StartPTY            func(context.Context, []string, string) (*pty.Session, error)
	WriteMarker         func(string) error
}
type StepResult struct {
	Session *pty.Session
	Text    string
}

func NewScaffold(t *template.Template, projectPath string, vars template.Vars, steps []template.TemplateStep, workDir, remote string) Scaffold {
	var post []template.TemplateStep
	if t != nil {
		post = t.PostSteps
	}
	return Scaffold{Execution: template.NewExecution(steps, post, vars, workDir), ProjectPath: projectPath, Remote: remote,
		Git: git.NewClient(), StartPTY: pty.NewSessionContext, WriteMarker: template.WriteDevDockMarkerFile,
		RunStep: func(ctx context.Context, p template.PreparedStep) error {
			var output process.Capture
			err := template.ExecutePrepared(ctx, p, nil, &output, &output)
			if err != nil {
				return fmt.Errorf("%w: %s", err, output.Bytes())
			}
			return nil
		}}
}
func (s Scaffold) Execute(ctx context.Context, step template.PreparedStep, rows, cols uint16) (StepResult, error) {
	if err := ctx.Err(); err != nil {
		return StepResult{}, err
	}
	if step.Step.Type == "builtin" || step.Output != "" {
		if err := s.RunStep(ctx, step); err != nil {
			return StepResult{}, err
		}
		text := step.Description
		if step.Output != "" {
			text += " → " + step.Step.Output
		}
		return StepResult{Text: text}, nil
	}
	session, err := s.StartPTY(ctx, step.Args, step.WorkDir)
	if err != nil {
		return StepResult{}, fmt.Errorf("start command %q: %w", step.Step.Run, err)
	}
	if err := session.Resize(rows, cols); err != nil {
		return StepResult{}, errors.Join(err, session.Close())
	}
	return StepResult{Session: session, Text: step.Description}, nil
}
func (s Scaffold) Finish(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.WriteMarker(s.ProjectPath); err != nil {
		return fmt.Errorf("write project marker: %w", err)
	}
	if s.Remote != "" {
		return s.Git.Init(ctx, s.ProjectPath, s.Remote)
	}
	return nil
}
