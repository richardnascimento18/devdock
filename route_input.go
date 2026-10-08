package main

import tea "github.com/charmbracelet/bubbletea"

func (m model) routeInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateNewProjectName:
		return m.updateNewProjectName(msg)
	case statePickRoot:
		return m.updatePickRoot(msg)
	case statePickDomain:
		return m.updateDomainPicker(msg)
	case stateNewDomainName:
		return m.updateNewDomainName(msg)
	case stateCreateDomainOnly:
		return m.updateCreateDomainOnly(msg)
	case statePickRootForDomain:
		return m.updatePickRootForDomain(msg)
	case statePickPreset:
		return m.updatePickPreset(msg)
	case stateAskCreateGitHub:
		return m.updateAskCreateGitHub(msg)
	case stateAskRepoPrivacy:
		return m.updateAskRepoPrivacy(msg)
	case stateBulkPreflight:
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
			m.state = stateList
			m.bulk = bulkWorkflow{}
		}
		return m, nil
	case stateChangingScope, statePreparingProject, stateBulkMoving, stateCreatingGitHub, stateCloningRepo, stateMovingProject, stateDeletingWorkspace:
		return m, nil
	case statePickTemplate:
		return m.updatePickTemplate(msg)
	case statePTYExecution:
		return m.updatePTYExecution(msg)
	case stateDeleteProject:
		return m.updateDeleteProject(msg)
	case stateDeleteDomain:
		return m.updateDeleteDomain(msg)
	case stateAddRoot:
		return m.updateAddRoot(msg)
	case stateRemoveRoot:
		return m.updateRemoveRoot(msg)
	case stateGitHubAuth:
		return m.updateGitHubAuth(msg)
	case statePickRootForClone:
		return m.updatePickRootForClone(msg)
	case statePickDomainForClone:
		return m.updatePickDomainForClone(msg)
	case stateMovePickRoot:
		return m.updateMovePickRoot(msg)
	case stateMovePickDomain:
		return m.updateMovePickDomain(msg)
	case stateMovePickPlacement:
		return m.updatePlacement(msg)
	case stateStatusDetails:
		if key, ok := msg.(tea.KeyMsg); ok && (key.String() == "esc" || key.String() == "enter") {
			m.state = stateList
		}
		return m, nil
	case stateBulkConfirm, stateBulkResult:
		return m.updateBulk(msg)
	case stateHelp:
		return m.updateHelp(msg)
	case statePalette:
		return m.updatePalette(msg)
	case stateCreateGroup:
		return m.updateCreateGroup(msg)
	case stateConfirmDeleteGroup:
		return m.updateConfirmDeleteGroup(msg)
	case stateDeleteGroup:
		return m.updateDeleteGroup(msg)
	case stateEditor:
		return m.updateEditor(msg)
	case stateDeleteTmuxSession:
		return m.updateDeleteTmuxSession(msg)
	default:
		return m.updateList(msg)
	}
}
