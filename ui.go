package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/richardnascimento18/devdock/internal/detect"
)

var (
	colorPurple      = lipgloss.Color("135")
	colorPurpleLight = lipgloss.Color("183")
	colorPurpleDim   = lipgloss.Color("61")
	colorPink        = lipgloss.Color("213")
	colorCyan        = lipgloss.Color("87")
	colorGreen       = lipgloss.Color("84")
	colorYellow      = lipgloss.Color("227")
	colorOrange      = lipgloss.Color("215")
	colorRed         = lipgloss.Color("203")
	colorWhite       = lipgloss.Color("255")
	colorGray        = lipgloss.Color("245")
	colorDimGray     = lipgloss.Color("238")
	colorDark        = lipgloss.Color("235")
)

var (
	promptStyle    = lipgloss.NewStyle().Bold(true).Foreground(colorPurple)
	errorStyle     = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
	warningStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorOrange)
	cursorStyle    = lipgloss.NewStyle().Foreground(colorPink)
	activeStyle    = lipgloss.NewStyle().Bold(true).Foreground(colorDark).Background(colorPurple)
	dimStyle       = lipgloss.NewStyle().Foreground(colorDimGray)
	successStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorGreen)
	hintStyle      = lipgloss.NewStyle().Foreground(colorPurpleDim).Italic(true)
	ghStyle        = lipgloss.NewStyle().Bold(true).Foreground(colorPink)
	clonedStyle    = lipgloss.NewStyle().Foreground(colorGreen)
	notClonedStyle = lipgloss.NewStyle().Foreground(colorOrange)
	rootTagStyle   = lipgloss.NewStyle().Foreground(colorCyan)
	domainStyle    = lipgloss.NewStyle().Foreground(colorPurpleDim)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")).
			Background(colorPurple).
			PaddingLeft(1).
			PaddingRight(1)

	listNormalStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			PaddingLeft(2)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPurple).
			Padding(0, 2)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPurpleDim).
			Padding(1, 3).
			Width(62)

	warningBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorRed).
			Padding(1, 3).
			Width(62)

	groupHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPurpleLight)

	countBadgeStyle = lipgloss.NewStyle().
			Foreground(colorDimGray)

	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(colorPurple)
	subtitleStyle = lipgloss.NewStyle().Foreground(colorPurpleDim).Italic(true)
)

const asciiTitle = `
██████╗ ███████╗██╗   ██╗██████╗  ██████╗  ██████╗██╗  ██╗
██╔══██╗██╔════╝██║   ██║██╔══██╗██╔═══██╗██╔════╝██║ ██╔╝
██║  ██║█████╗  ██║   ██║██║  ██║██║   ██║██║     █████╔╝ 
██║  ██║██╔══╝  ╚██╗ ██╔╝██║  ██║██║   ██║██║     ██╔═██╗ 
██████╔╝███████╗ ╚████╔╝ ██████╔╝╚██████╔╝╚██████╗██║  ██╗
╚═════╝ ╚══════╝  ╚═══╝  ╚═════╝  ╚═════╝  ╚═════╝╚═╝  ╚═╝`

func RenderTitle() string {
	title := titleStyle.Render(asciiTitle)
	sub := subtitleStyle.Render("  your terminal workspace manager")
	return lipgloss.JoinVertical(lipgloss.Center, title, sub, "")
}

func centerInTerminal(w, h int, content string) string {
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 40
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}

func wrapInBox(title, content string) string {
	header := promptStyle.Render("  " + title)
	body := inputBoxStyle.Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}

func wrapInWarningBox(title, content string) string {
	header := warningStyle.Render("  ⚠  " + title)
	body := warningBoxStyle.Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}

// RenderLanguageTags returns a compact line of colored badge pills.
func RenderLanguageTags(langs []string) string {
	if len(langs) == 0 {
		return ""
	}
	var parts []string
	for _, label := range langs {
		t := detect.Tech(label)
		color := lipgloss.Color(t.Color)
		badge := lipgloss.NewStyle().
			Foreground(colorDark).
			Background(color).
			Bold(true).
			Padding(0, 1).
			Render(t.Badge)
		parts = append(parts, badge)
	}
	return strings.Join(parts, " ")
}

// RenderGroupBreadcrumb renders complete ancestry using a compact suffix.
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
