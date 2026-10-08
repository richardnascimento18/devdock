package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/ui"
	"strings"
)

func (m model) settingsFlowVisible() bool {
	return m.editorRootReturn && m.termW >= 74 && (m.state == stateAddRoot || m.state == stateRemoveRoot)
}

func (m model) viewSettingsFlow() string {
	left := min(28, m.termW/4)
	detail := max(m.termW-left-2, 1)
	rows := max(m.termH-6, 1)
	labels, _ := m.editorScr.collectionDetails()
	choices := ui.PaneTitle("Settings", false, left) + "\n" + editorChoices(labels, m.editorScr.cursor, max(rows-2, 1), left)
	var form modalContent
	if m.state == stateAddRoot {
		form = m.inputScr.content(detail+6, m.termH)
	} else {
		form = m.genericPicker.content(detail+6, m.termH)
	}
	lines := strings.Split(ansi.Hardwrap(form.body, detail, true), "\n")
	offset := min(m.modalScroll, max(len(lines)-max(rows-1, 1), 0))
	content := ui.PaneTitle(form.title, true, detail) + "\n" + strings.Join(lines[offset:], "\n")
	body := lipgloss.JoinHorizontal(lipgloss.Top, paneContent(choices, left, rows), "  ", paneContent(content, detail, rows))
	return strings.Join([]string{ui.Header("Configuration › Settings", m.termW), promptStyle.Render("Roots / Values"), "", body, ui.Footer(form.hint, diagnosticView(m.diagnostic), m.termW)}, "\n")
}
