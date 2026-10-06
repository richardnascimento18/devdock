package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/tmux"
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
			add("Tmux", ui.Foreground(theme.Tmux).Render(x.displayName()+" · active"))
			add("Location", x.location)
			add("Technical session", dimStyle.Render(x.name))
			add("Actions", "enter attach · x kill with confirmation")
		default:
			lines = append(lines, dimStyle.Render("Select a project to inspect.\nctrl+p opens available actions."))
		}
		return lines
	}
	lines = append(lines, selectedStyle.Render(p.Name))
	if len(p.Languages) > 0 {
		lines = append(lines, RenderLanguageTags(p.Languages))
	}
	lines = append(lines, "")
	add("Location", dimStyle.Render(p.Location.Breadcrumb()+"\n"+p.Path))
	repo := "⌂ Local project"
	if p.LocalGit {
		repo = "⑂ Local Git repository"
	}
	if p.GitHubRepo != "" {
		repo = "⑂ GitHub · " + p.GitHubRepo
	}
	add("Git", ui.Foreground(theme.Git).Render(repo))
	session := "Not checked · visit tmux tab"
	if m.cachedTmux != nil {
		session = "No cached session"
		if m.cachedTmux[tmux.SessionName(p)] {
			session = "● " + p.Name + " · active (cached)"
		}
	}
	add("Tmux", ui.Foreground(theme.Tmux).Render(session)+"\n"+dimStyle.Render("Preset · "+m.presetSel.SelectedName()))
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
		if len(m.selected) > 0 {
			add("Actions", "m move selected · f favorites\nesc clears selection before deletion")
		} else {
			add("Actions", "enter open · f favorite\nm move · x delete with confirmation")
		}
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
