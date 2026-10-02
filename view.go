package main

import (
	"github.com/richardnascimento18/devdock/internal/ui"
)

func (m model) View() string {
	if ui.Measure(m.termW, m.termH).Mode == ui.Tiny {
		return ui.TooSmall(m.termW, m.termH)
	}
	return ui.Fit(m.viewScreen(), m.termW, m.termH)
}

func (m model) viewScreen() string {
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
		return renderHelpOverlay(w, h, m.helpScroll)
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
