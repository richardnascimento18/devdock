package main

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/ui"
)

type modalScreen struct{ scroll int }
type modalContent struct{ title, body, hint string }

func (c modalContent) View(width, height, offset int) string {
	return ui.ModalAt(c.title, c.body, c.hint, width, height, offset)
}
func (m model) flowModal() bool { return m.state.kind() == routeOverlay }
func (m model) currentModal() (content modalContent) {
	// The retained creation draft owns its failure, including while navigating
	// back through its choices. A retry or a fresh creation clears it.
	defer func() {
		switch m.state {
		case statePickPreset, statePickTemplate, stateAskCreateGitHub, stateAskRepoPrivacy:
			if m.creation.failure.Summary != "" {
				content.body = diagnosticView(m.creation.failure) + "\n\n" + content.body
			}
		}
	}()
	w, h := m.termW, m.termH
	switch m.state {
	case stateStatusDetails:
		return modalContent{"Status details", ui.SafeBlock(m.statusDetails), "esc back"}
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
		if m.auth.screen.err != "" {
			return modalContent{"GitHub connection failed", "! " + ui.SafeBlock(m.auth.screen.err), "esc back"}
		}
		if m.auth.screen.verificationURI == "" || m.auth.screen.done {
			return modalContent{"Connect GitHub", m.activityView(), "esc cancel"}
		}
		return m.auth.screen.content(w, h)
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
	if m.settingsFlowVisible() {
		left := min(28, m.termW/4)
		detail := max(m.termW-left-2, 1)
		copy := m
		copy.termW = detail + 6
		c := copy.currentModal()
		lines := strings.Split(ansi.Hardwrap(c.body, detail, true), "\n")
		return max(len(lines)-max(m.termH-7, 1), 0)
	}
	c := m.currentModal()
	return ui.ModalScrollLimit(c.body, c.hint, m.termW, m.termH)
}
