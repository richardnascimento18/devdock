package main

import (
	"fmt"
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
	prefix := strings.Repeat("  ", i.indent)
	if i.indent == 1 {
		prefix = "  · "
	} else if i.indent == 2 {
		prefix = "    · "
	}
	name := lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(i.project.Name)
	domain := domainStyle.Render(" (" + i.project.Domain + ")")
	line := prefix + name + domain
	if i.showRoot {
		line += " " + rootTagStyle.Render("("+config.RootName(i.project.Root)+")")
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
func (i item) FilterValue() string { return i.project.Name }

type flatItem struct{ item }

func (f flatItem) Title() string {
	crumb := RenderGroupBreadcrumb(f.item.project.Group, f.item.project.Subgroup)
	return crumb + f.item.Title()
}
func (f flatItem) Description() string { return "" }
func (f flatItem) FilterValue() string { return f.item.project.Name }

type githubItem struct{ repo gh.Repo }

func (g githubItem) Title() string {
	name := lipgloss.NewStyle().Bold(true).Foreground(colorGray).Render(g.repo.Name)
	return name + " " + ghStyle.Render("(GITHUB)") + " " + notClonedStyle.Render("(not cloned)")
}
func (g githubItem) Description() string { return "" }
func (g githubItem) FilterValue() string { return g.repo.Name }

type createProjectItem struct{ name string }

func (c createProjectItem) Title() string {
	return lipgloss.NewStyle().Foreground(colorCyan).Bold(true).
		Render(fmt.Sprintf("✦  create new project \"%s\"", c.name))
}
func (c createProjectItem) Description() string { return "" }
func (c createProjectItem) FilterValue() string { return c.name }

type recentItem struct{ entry state.RecentEntry }

func (r recentItem) Title() string {
	name := lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(r.entry.Name)
	domain := domainStyle.Render(" (" + r.entry.Domain + ")")
	ts := dimStyle.Render("  " + r.entry.OpenedAt.Format("Jan 02  15:04"))
	return name + domain + ts
}
func (r recentItem) Description() string { return "" }
func (r recentItem) FilterValue() string { return r.entry.Name }

type favoriteItem struct {
	project  core.Project
	showRoot bool
	verified bool
}

func (f favoriteItem) Title() string {
	star := lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Render("★ ")
	name := lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(f.project.Name)
	domain := domainStyle.Render(" (" + f.project.Domain + ")")
	line := star + name + domain
	if f.showRoot {
		line += " " + rootTagStyle.Render("("+config.RootName(f.project.Root)+")")
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
func (f favoriteItem) FilterValue() string { return f.project.Name }

type groupItem struct {
	groupKey   string
	name       string
	domain     string
	root       string
	collapsed  bool
	totalCount int
	shownCount int
	empty      bool
}

func (g groupItem) Title() string {
	icon := "▼"
	if g.collapsed {
		icon = "▶"
	}
	var count string
	if g.empty {
		count = countBadgeStyle.Render(" (no projects)")
	} else if g.collapsed {
		count = countBadgeStyle.Render(fmt.Sprintf(" [%d]", g.totalCount))
	} else if g.shownCount < g.totalCount {
		count = countBadgeStyle.Render(fmt.Sprintf(" [%d/%d]", g.shownCount, g.totalCount))
	} else {
		count = countBadgeStyle.Render(fmt.Sprintf(" [%d]", g.totalCount))
	}
	return groupHeaderStyle.Render(icon+" "+g.name) + count + " " + domainStyle.Render("("+g.domain+")")
}
func (g groupItem) Description() string { return "" }
func (g groupItem) FilterValue() string { return g.name }

type subgroupItem struct {
	subgroupKey string
	name        string
	parentGroup string
	domain      string
	root        string
	collapsed   bool
	totalCount  int
	shownCount  int
}

func (s subgroupItem) Title() string {
	icon := "  ▼"
	if s.collapsed {
		icon = "  ▶"
	}
	var count string
	if s.collapsed {
		count = countBadgeStyle.Render(fmt.Sprintf(" [%d]", s.totalCount))
	} else if s.shownCount < s.totalCount {
		count = countBadgeStyle.Render(fmt.Sprintf(" [%d/%d]", s.shownCount, s.totalCount))
	} else {
		count = countBadgeStyle.Render(fmt.Sprintf(" [%d]", s.totalCount))
	}
	return subgroupHeaderStyle.Render(icon+" "+s.name) + count
}
func (s subgroupItem) Description() string { return "" }
func (s subgroupItem) FilterValue() string { return s.name }

func groupKey(root, domain, group string) string {
	return root + "::" + domain + "::" + group
}

func subgroupKey(root, domain, group, subgroup string) string {
	return root + "::" + domain + "::" + group + "::" + subgroup
}

type emptyGroupFlatItem struct {
	name     string
	domain   string
	root     string
	showRoot bool
}

func (e emptyGroupFlatItem) Title() string {
	nameStr := lipgloss.NewStyle().Bold(true).Foreground(colorPurpleLight).Render(e.name)
	domainStr := domainStyle.Render(" (" + e.domain + ")")
	empty := lipgloss.NewStyle().Foreground(colorPurpleDim).Render(" (no projects)")
	line := nameStr + domainStr + empty
	if e.showRoot {
		line += " " + rootTagStyle.Render("("+config.RootName(e.root)+")")
	}
	return line
}
func (e emptyGroupFlatItem) Description() string { return "" }
func (e emptyGroupFlatItem) FilterValue() string { return e.name }

// tmuxSessionItem represents a live tmux session in the TabTmux list.
type tmuxSessionItem struct{ name string }

func (t tmuxSessionItem) Title() string {
	bullet := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("● ")
	name := lipgloss.NewStyle().Bold(true).Foreground(colorWhite).Render(t.name)
	return bullet + name
}
func (t tmuxSessionItem) Description() string { return "" }
func (t tmuxSessionItem) FilterValue() string { return t.name }
