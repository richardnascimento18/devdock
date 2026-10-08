package template

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/richardnascimento18/devdock/internal/process"
)

// PreparedStep is shared by interactive and ordinary execution. Arguments,
// expanded paths and output handling have one owner; adapters choose transport.
type PreparedStep struct {
	Step                                            TemplateStep
	Args                                            []string
	WorkDir, ProjectPath, Path, Output, Description string
	PostStarted                                     bool
}

func PrepareStep(step TemplateStep, workDir string, vars Vars) (PreparedStep, error) {
	p := PreparedStep{Step: step, WorkDir: workDir, ProjectPath: vars.ProjectPath,
		Path: ExpandVars(step.Path, vars), Output: ExpandVars(step.Output, vars), Description: ExpandVars(step.Run, vars)}
	switch step.Type {
	case "builtin":
		if _, err := safeJoin(vars.ProjectPath, p.Path); err != nil {
			return p, err
		}
		p.Description = step.Action + " " + step.Path
	case "command":
		args, err := CommandArgs(step, vars)
		if err != nil {
			return p, err
		}
		p.Args = args
		if p.Output != "" {
			if _, err := safeJoin(vars.ProjectPath, p.Output); err != nil {
				return p, err
			}
		}
	default:
		return p, fmt.Errorf("unknown step type %q", step.Type)
	}
	return p, nil
}

func ExecutePrepared(ctx context.Context, p PreparedStep, input io.Reader, output, stderr io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.Step.Type == "builtin" {
		if err := ExecuteBuiltin(p.Step.Action, p.Path, p.ProjectPath); err != nil {
			return fmt.Errorf("builtin %s: %w", p.Step.Action, err)
		}
		return nil
	}
	cmd := process.Command(ctx, p.WorkDir, p.Args[0], p.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = input, output, stderr
	var file *os.File
	if p.Output != "" {
		var err error
		file, err = CommandOutput(p.Output, p.ProjectPath)
		if err != nil {
			return err
		}
		cmd.Stdout = file
	}
	err := cmd.Run()
	if file != nil {
		err = errors.Join(err, file.Close())
	}
	if err != nil {
		return fmt.Errorf("command %q: %w", p.Step.Run, err)
	}
	return nil
}

// Execution owns step/post-step sequencing without terminal or process state.
type Execution struct {
	steps, postSteps []TemplateStep
	vars             Vars
	workDir          string
	index            int
	post             bool
}

func NewExecution(steps, postSteps []TemplateStep, vars Vars, workDir string) Execution {
	return Execution{steps: append([]TemplateStep(nil), steps...), postSteps: append([]TemplateStep(nil), postSteps...), vars: vars, workDir: workDir}
}
func (e *Execution) Next() (PreparedStep, bool, error) {
	postStarted := false
	if e.index >= len(e.steps) && !e.post && len(e.postSteps) > 0 {
		e.steps, e.index, e.post = e.postSteps, 0, true
		e.workDir = e.vars.ProjectPath
		postStarted = true
	}
	if e.index >= len(e.steps) {
		return PreparedStep{}, false, nil
	}
	prepared, err := PrepareStep(e.steps[e.index], e.workDir, e.vars)
	e.index++
	prepared.PostStarted = postStarted
	return prepared, true, err
}
func (e Execution) Progress() (int, int, bool) { return e.index, len(e.steps), e.post }
