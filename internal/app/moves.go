package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/richardnascimento18/devdock/internal/core"
)

type MoveResult struct {
	Project core.Project
	Plan    core.MovePlan
	Planned bool
	Err     error
}

// Move owns planning and execution; callers render a rejected plan differently
// from an execution failure. Execute revalidates the filesystem at mutation time.
func Move(ctx context.Context, project core.Project, destination core.Location, plan func(core.Project, core.Location, string) (core.MovePlan, error), execute func(core.MovePlan) (core.Project, error)) MoveResult {
	result := MoveResult{}
	if result.Err = ctx.Err(); result.Err != nil {
		return result
	}
	result.Plan, result.Err = plan(project, destination, filepath.Base(project.Path))
	if result.Err != nil {
		return result
	}
	result.Planned = true
	if result.Err = ctx.Err(); result.Err != nil {
		return result
	}
	result.Project, result.Err = execute(result.Plan)
	return result
}

type MoveRow struct {
	Plan    core.MovePlan
	Project core.Project
	Err     error
}

// Build every plan before any mutation, including collisions within the batch.
func PreflightMoves(projects []core.Project, destination core.Location, plan func(core.Project, core.Location, string) (core.MovePlan, error)) []MoveRow {
	rows := make([]MoveRow, len(projects))
	paths := map[string][]int{}
	for i, p := range projects {
		rows[i].Plan, rows[i].Err = plan(p, destination, filepath.Base(p.Path))
		if rows[i].Plan.Path != "" {
			paths[rows[i].Plan.Path] = append(paths[rows[i].Plan.Path], i)
		}
	}
	for path, indices := range paths {
		if len(indices) > 1 {
			for _, i := range indices {
				rows[i].Err = errors.Join(rows[i].Err, fmt.Errorf("multiple selections target %s", path))
			}
		}
	}
	return rows
}
func MovesValid(rows []MoveRow) bool {
	if len(rows) == 0 {
		return false
	}
	for _, r := range rows {
		if r.Err != nil {
			return false
		}
	}
	return true
}
func ExecuteMoves(rows []MoveRow, plan func(core.Project, core.Location, string) (core.MovePlan, error), move func(core.MovePlan) (core.Project, error)) ([]MoveRow, bool) {
	if len(rows) == 0 {
		return rows, false
	}
	projects := make([]core.Project, len(rows))
	for i, r := range rows {
		projects[i] = r.Plan.Source
	}
	checked := PreflightMoves(projects, rows[0].Plan.Destination, plan)
	if !MovesValid(checked) {
		return checked, false
	}
	for i := range checked {
		checked[i].Project, checked[i].Err = move(checked[i].Plan)
	}
	return checked, true
}
