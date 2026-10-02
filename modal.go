package main

import "github.com/richardnascimento18/devdock/internal/ui"

type modalScreen struct{ scroll int }
type modalContent struct{ title, body, hint string }

func (c modalContent) View(width, height, offset int) string {
	return ui.ModalAt(c.title, c.body, c.hint, width, height, offset)
}
func (m model) flowModal() bool {
	switch m.state {
	case stateList, stateHelp, statePalette, stateEditor, statePTYExecution,
		stateBulkPreflight, stateBulkMoving, stateCreatingGitHub, stateCloningRepo, stateMovingProject, stateDeletingWorkspace:
		return false
	default:
		return true
	}
}
func (m model) currentModal() modalContent {
	w, h := m.termW, m.termH
	switch m.state {
	case stateBulkConfirm, stateBulkResult:
		return m.bulkModal()
	case stateDeleteDomain:
		if m.pendingDomainName != "" {
			return m.confirmDelDomain.content(w, h)
		}
	case statePickDomain:
		return m.domainPicker.content(w, h)
	case statePickPreset:
		return m.presetPicker.content(w, h)
	case statePickTemplate:
		return m.templatePicker.content(w, h)
	case stateAskCreateGitHub, stateAskRepoPrivacy:
		return m.yesNoScr.content(w, h)
	case stateGitHubAuth:
		return m.githubAuthScr.content(w, h)
	case stateDeleteTmuxSession:
		return m.confirmDelTmux.content(w, h)
	case statePickRoot, statePickRootForDomain, stateRemoveRoot, statePickRootForClone,
		statePickDomainForClone, stateMovePickRoot, stateMovePickDomain, stateMovePickPlacement,
		stateDeleteGroup:
		return m.genericPicker.content(w, h)
	case stateCreateGroup:
		if m.groupFlow.phase == groupPickDomain {
			return m.genericPicker.content(w, h)
		}
	}
	return m.inputScr.content(w, h)
}
func (m model) modalScrollLimit() int {
	c := m.currentModal()
	return ui.ModalScrollLimit(c.body, c.hint, m.termW, m.termH)
}
