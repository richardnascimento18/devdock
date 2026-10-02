package main

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/core"
)

type placementIntent uint8

const (
	placeProject placementIntent = iota
	placeClone
	placeMove
	placeGroup
)

func (m model) currentLocation() core.Location {
	if m.pendingLocation.Root == m.pendingRoot && m.pendingLocation.Domain == m.pendingDomain {
		return m.pendingLocation
	}
	return core.Location{Root: m.pendingRoot, Domain: m.pendingDomain}
}
func (m model) openPlacement(root, domain string, intent placementIntent) model {
	m.pendingRoot, m.pendingDomain = root, domain
	m.placementIntent = intent
	locations := m.workspaceTree.Locations(root, domain)
	if len(locations) == 0 {
		locations = []core.Location{{Root: root, Domain: domain}}
	}
	opts := make([]movePlacementOption, 0, len(locations))
	for _, loc := range locations {
		label := "(place directly in domain)"
		if len(loc.GroupPath) > 0 {
			label = strings.Join(loc.GroupPath, " › ")
		}
		opts = append(opts, movePlacementOption{label: label, location: loc})
	}
	m.movePlacementOpts = opts
	labels := make([]string, len(opts))
	for i, opt := range opts {
		labels[i] = opt.label
	}
	m.genericPicker = newLocationPicker("Select workspace location:", locations, labels, "↑/↓ • enter • esc")
	m.state = stateMovePickPlacement
	return m
}
func (m model) updatePlacement(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if key.String() == "esc" {
		m.state = stateList
		return m, nil
	}
	if navPicker(&m.genericPicker, key.String()) {
		return m, nil
	}
	if key.String() != "enter" {
		return m, nil
	}
	idx := m.genericPicker.cursor
	if idx < 0 || idx >= len(m.movePlacementOpts) {
		return m, nil
	}
	loc := m.movePlacementOpts[idx].location
	if _, err := loc.Resolve(); err != nil {
		m.genericPicker.err = err.Error()
		return m, nil
	}
	m.pendingLocation = loc
	switch m.placementIntent {
	case placeProject:
		path, err := loc.ProjectPath(m.pendingProjectName)
		if err == nil {
			err = core.CheckDestination(loc.Root, path)
		}
		if err != nil {
			m.genericPicker.err = err.Error()
			return m, nil
		}
		return m.openPresetPicker(), nil
	case placeClone:
		return m.beginClone(loc.Domain)
	case placeGroup:
		m.groupFlow.parent = loc
		m.groupFlow.phase = groupEnterName
		m.inputScr = newInputScreen("New Group in "+loc.Breadcrumb()+" — enter path:", "group/nested-group", "enter confirm • esc cancel")
		m.state = stateCreateGroup
		return m, nil
	case placeMove:
		if len(m.bulk.projects) > 0 {
			return m.beginBulkPreflight(loc)
		}
		plan, err := core.PlanMove(m.moveTarget, loc, filepath.Base(m.moveTarget.Path))
		if err != nil {
			m.genericPicker.err = fmt.Sprintf("%s: %v", plan.Status, err)
			return m, nil
		}
		m.spinnerID++
		m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Moving %q...", m.moveTarget.Name))
		m.state = stateMovingProject
		return m, tea.Batch(func() tea.Msg { p, err := core.ExecuteMove(plan); return moveProjectDoneMsg{newProject: p, err: err} }, spinnerTick(m.spinnerID))
	}
	return m, nil
}
