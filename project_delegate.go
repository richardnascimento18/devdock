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
	sessions  map[string]bool
	favorites map[string]bool
	selected  map[string]bool
	focused   bool
}

func (projectDelegate) Height() int                         { return 2 }
func (projectDelegate) Spacing() int                        { return 0 }
func (projectDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (d projectDelegate) Render(w io.Writer, m list.Model, index int, it list.Item) {
	title := ""
	detail := ""
	if p, ok := projectFromItem(it); ok {
		title = p.Name
		if d.selected[p.Path] {
			title = "● " + title
		}
		if d.favorites[p.Path] {
			title = "★ " + title
		}
		detail = config.RootName(p.Root) + " › " + p.Location.Breadcrumb() + " · " + core.PathID(p.Path)[:6]
		if m.Width() >= 45 && d.sessions[tmux.SessionName(p)] {
			detail += " · ● tmux"
		}
		if m.Width() >= 55 && len(p.Languages) > 0 {
			detail += " · " + strings.Join(p.Languages, "/")
		}
		if m.Width() >= 65 && p.GitHubRepo != "" {
			detail += " · GitHub"
		}
	} else if entry, ok := it.(interface{ Title() string }); ok {
		title = ansi.Strip(entry.Title())
	}
	prefix := "  "
	style := ui.Foreground(theme.Primary)
	if index == m.Index() {
		prefix = "› "
		style = style.Bold(true)
		if d.focused {
			prefix = "> "
			style = selectedStyle
		}
	}
	fmt.Fprint(w, ansi.Truncate(prefix+style.Render(title), m.Width(), "…"))
	if detail != "" {
		fmt.Fprint(w, "\n"+dimStyle.Render(ansi.Truncate("  "+detail, m.Width(), "…")))
	} else {
		fmt.Fprint(w, "\n")
	}
}
