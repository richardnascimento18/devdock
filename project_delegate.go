package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/tmux"
	"github.com/richardnascimento18/devdock/internal/ui"
	"io"
	"strings"
)

type projectDelegate struct {
	sessions   map[string]bool
	favorites  map[string]bool
	selected   map[string]bool
	focused    bool
	duplicates map[string]bool
	projects   map[string]core.Project
}

func (projectDelegate) Height() int                         { return 2 }
func (projectDelegate) Spacing() int                        { return 0 }
func (projectDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (d projectDelegate) Render(w io.Writer, m list.Model, index int, it list.Item) {
	title := ""
	detail := ""
	decorations := ""
	if p, ok := projectFromItem(it); ok {
		if cached, exists := d.projects[p.Path]; exists {
			p = cached
		}
		title = ui.SafeText(p.Name)
		if d.selected[p.Path] {
			decorations += selectedStyle.Render(" ●")
		}
		if d.favorites[p.Path] {
			decorations += ui.Foreground(theme.Favorite).Render(" ★")
		}
		width := max(m.Width()-2, 1)
		location := compactProjectLocation(p)
		locationWidth := min(ansi.StringWidth(location), max(width/3, 10))
		if width < 32 {
			locationWidth = min(12, width/2)
		}
		tags := []string{}
		if len(p.Languages) > 0 {
			tags = append(tags, RenderLanguageTags(p.Languages))
		}
		repo := "⌂ local"
		if p.LocalGit {
			repo = "⑂ Git"
		}
		if p.GitHubRepo != "" {
			repo = "⑂ GitHub"
		}
		tags = append(tags, ui.Foreground(theme.Git).Render(ui.SafeBlock(repo)))
		if d.sessions[tmux.SessionName(p)] {
			tags = append(tags, ui.Foreground(theme.Tmux).Render("● tmux"))
		}
		metadata := strings.Join(tags, dimStyle.Render(" · "))
		budget := max(width-locationWidth-3, 1)
		metadata = ansi.Truncate(metadata, budget, "…")
		gap := "   "
		if width >= 50 {
			gap = strings.Repeat(" ", max(budget-ansi.StringWidth(metadata), 0)) + gap
		}
		detail = metadata + gap + dimStyle.Render(ui.SafeBlock(projectLocationAtWidth(p, locationWidth, d.duplicates[p.Path])))
	} else if session, ok := it.(tmuxSessionItem); ok {
		title = ui.SafeText(session.displayName())
		detail = ui.Foreground(theme.Tmux).Render("● active")
		if session.location != "" {
			detail += "   " + dimStyle.Render(ui.SafeBlock(session.location))
		}
	} else if entry, ok := it.(interface{ Title() string }); ok {
		title = ansi.Strip(entry.Title())
	}
	prefix := "  "
	style := ui.Foreground(theme.Primary).Bold(true)
	if index == m.Index() {
		prefix = "› "
		style = style.Bold(true)
		if d.focused {
			prefix = "> "
			style = selectedStyle
		}
	}
	fmt.Fprint(w, ansi.Truncate(prefix+style.Render(ui.SafeBlock(ansi.Truncate(title, max(m.Width()-2-ansi.StringWidth(decorations), 1), "…")))+decorations, m.Width(), "…"))
	if detail != "" {
		fmt.Fprint(w, "\n"+ansi.Truncate("  "+detail, m.Width(), "…"))
	} else {
		fmt.Fprint(w, "\n")
	}
}

// Keep the nearest container visible; full paths remain in the Inspector.
func compactProjectLocation(p core.Project) string {
	parts := []string{config.RootName(p.Root)}
	if p.Domain != "" {
		parts = append(parts, p.Domain)
	}
	parts = append(parts, p.GroupPath...)
	if len(parts) > 3 {
		parts = []string{parts[0], "…", parts[len(parts)-1]}
	}
	return ui.SafeText(strings.Join(parts, " › "))
}

func projectLocationAtWidth(p core.Project, width int, ambiguous bool) string {
	suffix := ""
	if ambiguous && width >= 10 {
		suffix = " · " + core.PathID(p.Path)[:4]
	}
	budget := max(width-ansi.StringWidth(suffix), 1)
	label := compactProjectLocation(p)
	if ansi.StringWidth(label) > budget {
		nearest := ui.SafeText(p.Domain)
		if len(p.GroupPath) > 0 {
			nearest = ui.SafeText(p.GroupPath[len(p.GroupPath)-1])
		}
		if nearest == "" {
			nearest = ui.SafeText(config.RootName(p.Root))
		}
		label = "… › " + nearest
		if ansi.StringWidth(label) > budget {
			label = ansi.TruncateLeft(nearest, max(ansi.StringWidth(nearest)-budget+1, 0), "…")
		}
	}
	return label + suffix
}
