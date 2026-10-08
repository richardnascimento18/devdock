package main

import (
	"fmt"
	"github.com/richardnascimento18/devdock/internal/ui"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/state"
)

type item struct {
	project  core.Project
	showRoot bool
	verified bool
	indent   int
}

func (i item) Title() string {
	prefix := visualIndent(i.indent)
	name := lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(ui.SafeBlock(i.project.Name))
	domain := domainStyle.Render(ui.SafeBlock(" (" + i.project.Domain + ")"))
	line := prefix + name + domain + RenderGroupBreadcrumb(i.project.GroupPath)
	if i.showRoot {
		line += " " + rootTagStyle.Render(ui.SafeBlock("("+config.RootName(i.project.Root)+")"))
	}
	if i.verified && i.project.GitHubRepo != "" {
		line += " " + ghStyle.Render("(GITHUB)") + " " + clonedStyle.Render("(cloned)")
	}
	if len(i.project.Languages) > 0 {
		line += "  " + RenderLanguageTags(i.project.Languages)
	}
	return line
}
func (i item) Description() string { return "" }
func (i item) FilterValue() string {
	return i.project.Name + " " + i.project.Location.Breadcrumb() + " " + i.project.Root
}

type flatItem struct{ item }

func (f flatItem) Title() string {
	return f.item.Title()
}
func (f flatItem) Description() string { return "" }
func (f flatItem) FilterValue() string { return f.item.FilterValue() }

type githubItem struct{ repo gh.Repo }

func (g githubItem) Title() string {
	name := lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary).Render(ui.SafeBlock(g.repo.Name))
	return name + " " + ghStyle.Render("(GITHUB)") + " " + notClonedStyle.Render("(not cloned)")
}
func (g githubItem) Description() string { return "" }
func (g githubItem) FilterValue() string { return g.repo.Name }

type createProjectItem struct{ name string }

func (c createProjectItem) Title() string {
	return lipgloss.NewStyle().Foreground(theme.Info).Bold(true).
		Render(ui.SafeBlock(fmt.Sprintf("✦  create new project \"%s\"", c.name)))
}
func (c createProjectItem) Description() string { return "" }
func (c createProjectItem) FilterValue() string { return c.name }

type recentItem struct{ entry state.RecentEntry }

func (r recentItem) Title() string {
	name := lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(ui.SafeBlock(r.entry.Name))
	domain := domainStyle.Render(ui.SafeBlock(" (" + r.entry.Domain + ")"))
	ts := dimStyle.Render(ui.SafeBlock("  " + r.entry.OpenedAt.Format("Jan 02  15:04")))
	return name + domain + RenderGroupBreadcrumb(r.entry.GroupPath) + " " + rootTagStyle.Render(ui.SafeBlock(r.entry.Root)) + ts
}
func (r recentItem) Description() string { return "" }
func (r recentItem) FilterValue() string {
	return r.entry.Name + " " + r.entry.Location.Breadcrumb() + " " + r.entry.Root
}

type favoriteItem struct {
	project  core.Project
	showRoot bool
	verified bool
}

func (f favoriteItem) Title() string {
	star := lipgloss.NewStyle().Foreground(theme.Favorite).Bold(true).Render("★ ")
	name := lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(ui.SafeBlock(f.project.Name))
	domain := domainStyle.Render(ui.SafeBlock(" (" + f.project.Domain + ")"))
	line := star + name + domain + RenderGroupBreadcrumb(f.project.GroupPath)
	if f.showRoot {
		line += " " + rootTagStyle.Render(ui.SafeBlock("("+config.RootName(f.project.Root)+")"))
	}
	if f.verified && f.project.GitHubRepo != "" {
		line += " " + ghStyle.Render("(GITHUB)") + " " + clonedStyle.Render("(cloned)")
	}
	if len(f.project.Languages) > 0 {
		line += "  " + RenderLanguageTags(f.project.Languages)
	}
	return line
}
func (f favoriteItem) Description() string { return "" }
func (f favoriteItem) FilterValue() string {
	return f.project.Name + " " + f.project.Location.Breadcrumb() + " " + f.project.Root
}

type groupItem struct {
	nodeKey    core.NodeKey
	name       string
	location   core.Location
	depth      int
	collapsed  bool
	totalCount int
}

func (g groupItem) Title() string {
	icon := "▼"
	if g.collapsed {
		icon = "▶"
	}
	count := fmt.Sprintf(" [%d]", g.totalCount)
	if g.totalCount == 0 {
		count = " (no projects)"
	}
	return visualIndent(g.depth) + groupHeaderStyle.Render(ui.SafeBlock(icon+" "+g.name)) + countBadgeStyle.Render(ui.SafeBlock(count)) + " " + dimStyle.Render(ui.SafeBlock(g.location.Domain+" › "+compactBreadcrumb(g.location.GroupPath))) + " " + rootTagStyle.Render(ui.SafeBlock(g.location.Root))
}
func (g groupItem) Description() string { return "" }
func (g groupItem) FilterValue() string {
	return g.name + " " + g.location.Breadcrumb() + " " + g.location.Root
}
func visualIndent(depth int) string {
	if depth > 4 {
		return strings.Repeat("  ", 4) + "… "
	}
	return strings.Repeat("  ", max(depth, 0))
}

// tmuxSessionItem represents a live tmux session in the TabTmux list.
type tmuxSessionItem struct{ name, display, location, confirmation string }

func (t tmuxSessionItem) displayName() string {
	if t.display != "" {
		return t.display
	}
	return t.name
}

func (t tmuxSessionItem) Title() string {
	bullet := lipgloss.NewStyle().Foreground(theme.Success).Bold(true).Render("● ")
	name := lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(ui.SafeBlock(t.displayName()))
	return bullet + name
}
func (t tmuxSessionItem) Description() string { return "" }
func (t tmuxSessionItem) FilterValue() string { return t.displayName() + " " + t.location }
