package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/ui"
	"strings"
	"time"
)

func projectFromItem(it list.Item) (core.Project, bool) {
	switch x := it.(type) {
	case item:
		return x.project, true
	case flatItem:
		return x.project, true
	case favoriteItem:
		return x.project, true
	case recentItem:
		return core.Project{Location: x.entry.Location, Name: x.entry.Name, Path: x.entry.Path}, true
	}
	return core.Project{}, false
}

// Recents can outlive a scan result. Mutable project actions use the loaded
// canonical project when it still exists; stale recents remain openable only.
func (m model) actionProject() (core.Project, bool) {
	if recent, ok := m.list.SelectedItem().(recentItem); ok {
		for _, p := range m.rawProjects {
			if p.Path == recent.entry.Path {
				return p, true
			}
		}
		return core.Project{}, false
	}
	return projectFromItem(m.list.SelectedItem())
}

func (m model) currentLocationLabel() string {
	if m.focus == ui.Workspace && m.workspaceCursor > 0 && m.workspaceCursor < len(m.workspaceRows) {
		return m.workspaceRows[m.workspaceCursor].node.Path()
	}
	if p, ok := projectFromItem(m.list.SelectedItem()); ok {
		return p.Path
	}
	return ""
}

// Inspector renders only reliable loaded data. No Git, tmux or network probes
// occur here; unknown state is explicitly identified rather than inferred.
func (m model) inspectorLines(width int) []string {
	var lines []string
	add := func(label, value string) {
		lines = append(lines, promptStyle.Render(label))
		lines = append(lines, strings.Split(ansi.Hardwrap(value, max(width-2, 1), true), "\n")...)
		lines = append(lines, "")
	}
	if m.focus == ui.Workspace && m.workspaceCursor > 0 && m.workspaceCursor < len(m.workspaceRows) {
		n := m.workspaceRows[m.workspaceCursor].node
		add("Workspace", n.Name+" · "+string(n.Kind))
		add("Location", n.Path())
		add("Projects", fmt.Sprint(n.ProjectCount))
		add("Actions", "enter show projects\n←/→ collapse / expand\nG create group")
		return lines
	}
	p, ok := projectFromItem(m.list.SelectedItem())
	if cached, available := m.actionProject(); available {
		p = cached
	}
	if !ok {
		switch x := m.list.SelectedItem().(type) {
		case githubItem:
			add("GitHub", x.repo.FullName)
			add("Actions", "enter clone repository")
		case tmuxSessionItem:
			add("Tmux", x.name)
			add("Actions", "enter attach · x kill with confirmation")
		default:
			lines = append(lines, dimStyle.Render("Select a project to inspect.\nctrl+p opens available actions."))
		}
		return lines
	}
	add("Project", p.Name)
	add("Location", p.Root+"\n"+p.Location.Breadcrumb()+"\n"+p.Path)
	if len(p.Languages) > 0 {
		add("Detected stack", strings.Join(p.Languages, " · "))
	} else {
		add("Detected stack", "No recognized stack markers")
	}
	if p.GitHubRepo != "" {
		add("GitHub", p.GitHubRepo)
	}
	// The backend does not cache local branch/dirty state or project-session
	// attachment. Avoid inventing these facts in a presentation redesign.
	add("Tmux", "Preset: "+m.presetSel.SelectedName()+"\nSession state is available in the tmux tab.")
	favorite := "☆ Not a favorite"
	if m.uiState.Favorites[p.Path] {
		favorite = "★ Favorite"
	}
	add("Favorite", favorite)
	for _, recent := range m.uiState.Recents {
		if recent.Path == p.Path {
			add("Last opened", recent.OpenedAt.Format(time.RFC822))
			break
		}
	}
	if _, mutable := m.actionProject(); mutable {
		add("Actions", "enter open · f favorite\nm move · x delete with confirmation")
	} else {
		add("Actions", "enter open\nSelect a current project to move/delete/favorite.")
	}
	return lines
}

func (m model) viewInspector(width, height int) string {
	lines := m.inspectorLines(width)
	rows := max(height-1, 1)
	start := min(max(m.inspectorScroll, 0), max(len(lines)-rows, 0))
	body := strings.Join(lines[start:min(start+rows, len(lines))], "\n")
	return paneContent(ui.PaneTitle("Inspector", m.focus == ui.Inspector, width)+"\n"+body, width, height)
}
