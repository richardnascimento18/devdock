package main

import (
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/core"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	w, h := m.termW, m.termH
	switch m.state {
	case stateNewProjectName, stateNewDomainName, stateCreateDomainOnly,
		stateDeleteProject, stateAddRoot, stateConfirmDeleteGroup:
		return m.inputScr.View(w, h)
	case stateDeleteDomain:
		if m.pendingDomainName == "" {
			return m.inputScr.View(w, h)
		}
		return m.confirmDelDomain.View(w, h)
	case statePickDomain:
		return m.domainPicker.View(w, h)
	case statePickPreset:
		return m.presetPicker.View(w, h)
	case stateAskCreateGitHub, stateAskRepoPrivacy:
		return m.yesNoScr.View(w, h)
	case stateCreatingGitHub, stateCloningRepo, stateMovingProject, stateDeletingWorkspace:
		return m.spinnerScr.View(w, h)
	case statePickTemplate:
		return m.templatePicker.View(w, h)
	case statePTYExecution:
		return m.ptyScr.View()
	case stateHelp:
		return RenderHelpOverlay(w, h)
	case stateGitHubAuth:
		return m.githubAuthScr.View(w, h)
	case statePickRoot, statePickRootForDomain, stateRemoveRoot,
		statePickRootForClone, statePickDomainForClone,
		stateMovePickRoot, stateMovePickDomain, stateMovePickPlacement:
		return m.genericPicker.View(w, h)
	case stateCreateGroup:
		if m.groupFlow.phase == groupPickDomain {
			return m.genericPicker.View(w, h)
		}
		return m.inputScr.View(w, h)
	case stateDeleteGroup:
		return m.genericPicker.View(w, h)
	case stateEditor:
		return m.editorScr.View()
	case stateDeleteTmuxSession:
		return m.confirmDelTmux.View(w, h)
	default:
		return m.viewList(w, h)
	}
}

func (m model) viewList(w, h int) string {
	listStr := m.list.View()

	statusLine := ""
	if m.statusMsg != "" {
		statusLine = "\n" + m.statusMsg
	} else if m.isFiltered {
		statusLine = "\n" + dimStyle.Render(fmt.Sprintf("filter: \"%s\"  •  c or esc to clear", m.lastFilter))
	}

	viewMode := ""
	if m.treeMode {
		viewMode = dimStyle.Render("  ·  ") + dimStyle.Render("tree view") +
			dimStyle.Render("  (v to toggle  •  space/enter to collapse)")
	} else {
		viewMode = dimStyle.Render("  ·  ") + dimStyle.Render("flat view") +
			dimStyle.Render("  (v to toggle)")
	}

	ghBadge := ""
	if m.cfg.IsGitHubConnected() {
		ghBadge = "  " + lipgloss.NewStyle().Foreground(colorGreen).Render("●") +
			dimStyle.Render(" github: "+m.cfg.GitHubUsername)
	} else {
		ghBadge = dimStyle.Render("  ○ github: not connected  (g to connect)")
	}

	// Page counter — only shown on the Search tab
	pageIndicator := ""
	if m.activeTab == TabSearch {
		totalPages := m.list.Paginator.TotalPages
		if totalPages < 1 {
			totalPages = 1
		}
		currentPage := m.list.Paginator.Page + 1
		pageIndicator = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorDark).
			Background(colorPurple).
			Padding(0, 2).
			Render(fmt.Sprintf("%d/%d", currentPage, totalPages))
	}

	selectedPath := rowIdentity(m.list.SelectedItem())
	if group, ok := m.list.SelectedItem().(groupItem); ok {
		selectedPath = group.location.Path()
	}
	locationLine := ""
	if selectedPath != "" {
		locationLine = dimStyle.Render(core.PathID(selectedPath) + " " + ansi.Truncate(selectedPath, max(w-25, 10), "…"))
	}

	widgets := lipgloss.JoinVertical(lipgloss.Left,
		"",
		m.rootSel.View()+viewMode,
		m.presetSel.View(),
		ghBadge,
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		RenderTitle(),
		renderTabBar(m.activeTab),
		pageIndicator,
		listStr,
		locationLine,
		statusLine,
		widgets,
	)
	return centerInTerminal(w, h, content)
}
