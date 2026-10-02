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
	if m.flowModal() {
		return m.currentModal().View(w, h, m.modalScroll)
	}
	switch m.state {
	case stateBulkPreflight, stateBulkMoving, stateCreatingGitHub, stateCloningRepo, stateMovingProject, stateDeletingWorkspace:
		return m.spinnerScr.View(w, h, m.motion.frame, m.motion.reduced, m.state == stateBulkPreflight)
	case statePTYExecution:
		p := m.ptyScr
		p.motionFrame = m.motion.frame
		p.reducedMotion = m.motion.reduced
		return p.View()
	case stateHelp:
		return renderHelpOverlay(w, h, m.helpScroll, m.availableActions())
	case statePalette:
		return m.palette.View(w, h)
	case stateEditor:
		return m.editorScr.View()
	default:
		return m.viewList(w, h)
	}
}
