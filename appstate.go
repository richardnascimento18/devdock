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
