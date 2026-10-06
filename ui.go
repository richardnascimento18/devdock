package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/richardnascimento18/devdock/internal/detect"
	"github.com/richardnascimento18/devdock/internal/ui"
	"strings"
)

var theme = ui.Default

var (
	promptStyle      = ui.Foreground(theme.Accent).Bold(true)
	errorStyle       = ui.Foreground(theme.Error).Bold(true)
	warningStyle     = ui.Foreground(theme.Warning).Bold(true)
	cursorStyle      = ui.Foreground(theme.Accent)
	activeStyle      = ui.Foreground(theme.Selection).Bold(true).Underline(true)
	dimStyle         = ui.Foreground(theme.Muted)
	successStyle     = ui.Foreground(theme.Success).Bold(true)
	hintStyle        = ui.Foreground(theme.Muted)
	ghStyle          = ui.Foreground(theme.Git).Bold(true)
	clonedStyle      = ui.Foreground(theme.Success)
	notClonedStyle   = ui.Foreground(theme.Warning)
	rootTagStyle     = ui.Foreground(theme.Info)
	domainStyle      = ui.Foreground(theme.Muted)
	selectedStyle    = ui.Foreground(theme.Selection).Bold(true)
	groupHeaderStyle = ui.Foreground(theme.Accent).Bold(true)
	countBadgeStyle  = ui.Foreground(theme.Faint)
	titleStyle       = ui.Foreground(theme.Accent).Bold(true)
)

func transparentInput() textinput.Model {
	input := textinput.New()
	ui.ConfigureInput(&input)
	return input
}

func RenderTitle() string { return titleStyle.Render("DevDock") }

// Technology labels use semantic foregrounds rather than colored badge fills.
func RenderLanguageTags(langs []string) string {
	var parts []string
	for _, label := range langs {
		parts = append(parts, ui.Foreground(theme.Info).Render(detect.Tech(label).Badge))
	}
	return strings.Join(parts, " · ")
}

func RenderGroupBreadcrumb(path []string) string {
	if len(path) == 0 {
		return ""
	}
	return dimStyle.Render(" " + compactBreadcrumb(path) + " › ")
}

func compactBreadcrumb(path []string) string {
	if len(path) > 3 {
		return "… › " + strings.Join(path[len(path)-3:], " › ")
	}
	return strings.Join(path, " › ")
}

// Adapted from the original beta mark. Reserve it for roomy startup/empty states.
const asciiTitle = `██████╗ ███████╗██╗   ██╗██████╗  ██████╗  ██████╗██╗  ██╗
██╔══██╗██╔════╝██║   ██║██╔══██╗██╔═══██╗██╔════╝██║ ██╔╝
██║  ██║█████╗  ██║   ██║██║  ██║██║   ██║██║     █████╔╝
██║  ██║██╔══╝  ╚██╗ ██╔╝██║  ██║██║   ██║██║     ██╔═██╗
██████╔╝███████╗ ╚████╔╝ ██████╔╝╚██████╔╝╚██████╗██║  ██╗
╚═════╝ ╚══════╝  ╚═══╝  ╚═════╝  ╚═════╝  ╚═════╝╚═╝  ╚═╝`

func (m model) showBrandMark() bool {
	return m.termW >= 110 && m.termH >= 36 && (m.scanInFlight && len(m.rawProjects) == 0 || len(m.list.Items()) == 0 && m.lastFilter == "")
}
