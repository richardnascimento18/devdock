package main

import "github.com/richardnascimento18/devdock/internal/core"

// appState identifies which screen/flow is currently active.
type appState int

const (
	stateList appState = iota
	stateNewProjectName
	statePickRoot
	statePickDomain
	stateNewDomainName
	stateCreateDomainOnly
	statePickRootForDomain
	statePickPreset
	stateAskCreateGitHub
	stateAskRepoPrivacy
	stateCreatingGitHub
	stateDeleteProject
	stateDeleteDomain
	stateAddRoot
	stateRemoveRoot
	stateGitHubAuth
	statePickRootForClone
	statePickDomainForClone
	stateCloningRepo
	stateMovePickRoot
	stateMovePickDomain
	stateMovePickPlacement
	stateHelp
	statePickTemplate
	statePTYExecution
	stateCreateGroup
	stateDeleteGroup
	stateEditor
	stateDeleteTmuxSession
	stateConfirmDeleteGroup
	stateMovingProject
	stateDeletingWorkspace
	statePalette
	stateBulkPreflight
	stateBulkConfirm
	stateBulkMoving
	stateBulkResult
	stateStatusDetails
	statePreparingProject
	stateChangingScope
)

const (
	TabSearch    = 0
	TabRecents   = 1
	TabFavorites = 2
	TabTmux      = 3
)

var tabNames = []string{"Search", "Recents", "Favorites", "tmux-sessions"}

// Picker intent is independent of user-entered names.
type rootPickerIntent uint8

const (
	rootForProject rootPickerIntent = iota
	rootForCreateGroup
	rootForDeleteGroup
)

type domainIntent uint8

const (
	domainForProject domainIntent = iota
	domainOnly
	domainForClone
	domainForMove
)

type groupPhase uint8

const (
	groupPickDomain groupPhase = iota
	groupEnterName
)

type groupWorkflow struct {
	phase           groupPhase
	domain          string
	parent          core.Location
	deleteLocations []core.Location
	deletePath      string
}

// Route classes describe input ownership without independent overlay flags.
// The exclusive state value prevents a destructive operation and modal from
// both consuming dashboard keys.
type routeKind uint8

const (
	routeDashboard routeKind = iota
	routeOverlay
	routeOperation
	routeEditor
	routeTerminal
	routeHelp
	routePalette
)

func (s appState) kind() routeKind {
	switch s {
	case stateList:
		return routeDashboard
	case stateEditor:
		return routeEditor
	case statePTYExecution:
		return routeTerminal
	case stateHelp:
		return routeHelp
	case statePalette:
		return routePalette
	case stateChangingScope, statePreparingProject, stateBulkPreflight, stateBulkMoving, stateCreatingGitHub, stateCloningRepo, stateMovingProject, stateDeletingWorkspace:
		return routeOperation
	default:
		return routeOverlay
	}
}
