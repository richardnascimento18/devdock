package main

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
)

const (
	TabSearch    = 0
	TabRecents   = 1
	TabFavorites = 2
)

var tabNames = []string{"Search", "Recents", "Favorites"}
