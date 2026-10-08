package main

import (
	"github.com/richardnascimento18/devdock/internal/app"

	"fmt"
	"github.com/richardnascimento18/devdock/internal/ui"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/core"
)

// Selection is ephemeral and keyed by canonical project path, never row index.
func (m model) toggleSelection() model {
	p, ok := m.actionProject()
	if !ok {
		return m
	}
	next := make(map[string]bool, len(m.selected)+1)
	for key, value := range m.selected {
		next[key] = value
	}
	if next[p.Path] {
		delete(next, p.Path)
	} else {
		next[p.Path] = true
	}
	m.selected = next
	return m
}
func (m model) selectedProjects() []core.Project {
	var projects []core.Project
	for _, p := range m.navigation.projects {
		if m.selected[p.Path] {
			projects = append(projects, p)
		}
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].Path < projects[j].Path })
	return projects
}
func (m *model) reconcileSelection() {
	valid := make(map[string]bool, len(m.selected))
	for _, p := range m.navigation.projects {
		if m.selected[p.Path] {
			valid[p.Path] = true
		}
	}
	m.selected = valid
}
func (m model) hiddenSelection() int {
	visible := make(map[string]bool)
	for _, it := range m.list.Items() {
		if p, ok := projectFromItem(it); ok && m.selected[p.Path] {
			visible[p.Path] = true
		}
	}
	return max(len(m.selected)-len(visible), 0)
}
func (m model) selectionSignature() string {
	keys := make([]string, 0, len(m.selected))
	for key := range m.selected {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, "\x00")
}
func (m model) selectionStatus() string {
	s := fmt.Sprintf("%d selected", len(m.selected))
	if hidden := m.hiddenSelection(); hidden > 0 {
		s += fmt.Sprintf(" · %d hidden (clear filter before bulk actions)", hidden)
	}
	return s
}
func (m model) scopeIdentity() string {
	if m.navigation.scope != nil {
		return string(m.navigation.scope.Key())
	}
	return m.activeRoot()
}
func (m model) bulkFavorites() model {
	if m.hiddenSelection() > 0 {
		m.diagnostic = app.Diagnostic{Severity: app.Warning, Summary: "! " + m.selectionStatus()}
		return m
	}
	projects := m.selectedProjects()
	if len(projects) == 0 {
		return m
	}
	allFavorite := true
	for _, p := range projects {
		if !m.uiState.Favorites[p.Path] {
			allFavorite = false
		}
	}
	for _, p := range projects {
		if allFavorite {
			delete(m.uiState.Favorites, p.Path)
		} else {
			if m.uiState.Favorites == nil {
				m.uiState.Favorites = map[string]bool{}
			}
			m.uiState.Favorites[p.Path] = true
		}
	}
	if m.saveState() {
		m.diagnostic = app.Diagnostic{Severity: app.Success, Summary: fmt.Sprintf("✓ Updated favorites for %d projects", len(projects))}
		m = m.applySearch()
	}
	return m
}

type bulkMoveRow = app.MoveRow
type bulkWorkflow struct {
	id               uint64
	projects         []core.Project
	destination      core.Location
	rows             []bulkMoveRow
	executed         bool
	persistenceError string
}
type bulkPreflightMsg struct {
	id   uint64
	rows []bulkMoveRow
}
type bulkDoneMsg struct {
	id       uint64
	rows     []bulkMoveRow
	executed bool
}

func (m model) startBulkMove() model {
	if m.hiddenSelection() > 0 {
		m.diagnostic = app.Diagnostic{Severity: app.Warning, Summary: "! " + m.selectionStatus()}
		return m
	}
	m.bulk = bulkWorkflow{projects: m.selectedProjects()}
	if len(m.bulk.projects) == 0 {
		return m
	}
	m.bulkID++
	m.bulk.id = m.bulkID
	m.moveTarget = m.bulk.projects[0]
	m.domainIntent = domainForMove
	roots := m.cfg.ActiveRoots()
	if len(roots) == 1 {
		m.pendingRoot = roots[0]
		return m.openMoveDestDomainPicker()
	}
	m.genericPicker = newRootPicker(fmt.Sprintf("Move %d selected projects — destination root", len(m.bulk.projects)), roots, "enter choose · esc cancel")
	m.state = stateMovePickRoot
	return m
}
func (m model) beginBulkPreflight(loc core.Location) (tea.Model, tea.Cmd) {
	m.bulk.destination = loc
	m.bulkID++
	m.bulk.id = m.bulkID
	id := m.bulk.id
	projects := append([]core.Project(nil), m.bulk.projects...)
	m.state = stateBulkPreflight
	m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Checking %d move destinations…", len(projects)))
	return m, func() tea.Msg { return bulkPreflightMsg{id, app.PreflightMoves(projects, loc, core.PlanMove)} }
}
func (m model) handleBulkPreflight(msg bulkPreflightMsg) (tea.Model, tea.Cmd) {
	if m.state != stateBulkPreflight || msg.id != m.bulk.id {
		return m, nil
	}
	m.bulk.rows = msg.rows
	m.state = stateBulkConfirm
	return m, nil
}
func (m model) updateBulk(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if key.String() == "esc" {
		m.state = stateList
		m.bulk = bulkWorkflow{}
		return m, nil
	}
	if key.String() == "enter" && m.state == stateBulkResult {
		m.state = stateList
		return m, nil
	}
	if key.String() == "enter" && m.state == stateBulkConfirm && app.MovesValid(m.bulk.rows) {
		m.state = stateBulkMoving
		m.spinnerScr = newSpinnerScreen(fmt.Sprintf("Moving %d selected projects…", len(m.bulk.rows)))
		id := m.bulk.id
		rows := append([]bulkMoveRow(nil), m.bulk.rows...)
		return m, func() tea.Msg {
			result, executed := app.ExecuteMoves(rows, core.PlanMove, m.moveFilesystem.Execute)
			return bulkDoneMsg{id, result, executed}
		}
	}
	return m, nil
}
func (m model) handleBulkDone(msg bulkDoneMsg) (tea.Model, tea.Cmd) {
	if m.state != stateBulkMoving || msg.id != m.bulk.id {
		return m, nil
	}
	m.bulk.rows = msg.rows
	m.bulk.executed = msg.executed
	if msg.executed {
		changed := false
		for _, row := range msg.rows {
			if row.Project.Path != "" {
				m.uiState.MoveProject(row.Plan.Source.Path, row.Project)
				changed = true
			}
		}
		if changed && !m.saveFilesystemState() {
			m.bulk.persistenceError = "State save failed; filesystem results remain in memory and will be retried. " + m.persistenceErr.Error()
		}
		m.selected = nil
		m = m.rescan()
	}
	m.state = stateBulkResult
	return m, nil
}
func (m model) bulkModal() modalContent {
	title := fmt.Sprintf("Move %d selected projects", len(m.bulk.projects))
	hint := "enter move all · esc cancel"
	lines := []string{"Destination: " + ui.SafeText(m.bulk.destination.Path()), "", "Every source and destination is checked before moving.", "Filesystem moves are separate operations; failures may be partial.", ""}
	if m.state == stateBulkResult {
		title = "Bulk move results"
		hint = "enter done · esc back"
		lines = nil
		if !m.bulk.executed {
			lines = append(lines, warningStyle.Render("! Preflight changed. No projects were moved."))
		} else {
			success, partial, failed := 0, 0, 0
			for _, r := range m.bulk.rows {
				if r.Err == nil {
					success++
				} else if r.Project.Path != "" {
					partial++
				} else {
					failed++
				}
			}
			lines = append(lines, fmt.Sprintf("%d moved · %d cleanup warnings · %d failed", success, partial, failed), "Selection cleared; review failures before retrying.")
		}
	}
	if m.state == stateBulkConfirm && !app.MovesValid(m.bulk.rows) {
		lines = append([]string{warningStyle.Render("! Resolve these conflicts. Nothing has moved."), ""}, lines...)
		hint = "esc choose again"
	}
	for _, r := range m.bulk.rows {
		label := "✓ Ready"
		if m.state == stateBulkResult {
			label = "✓ Moved"
		}
		if r.Err != nil {
			label = "! " + r.Err.Error()
		}
		lines = append(lines, "", ui.SafeText(r.Plan.Source.Name), "From: "+ui.SafeText(r.Plan.Source.Path), "To:   "+ui.SafeText(r.Plan.Path), ui.SafeBlock(label))
	}
	if m.bulk.persistenceError != "" {
		lines = append(lines, "", warningStyle.Render(ui.SafeBlock("! "+m.bulk.persistenceError)))
	}
	return modalContent{title, strings.Join(lines, "\n"), hint}
}
