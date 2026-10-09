package main

import (
	"context"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/app"
	"github.com/richardnascimento18/devdock/internal/core"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/preset"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/ui"
)

// Substates own one lifecycle each. The route enum remains exclusive: a modal
// or operation replaces input routing, so independent overlay flags cannot
// accidentally accept dashboard keys during a destructive workflow.
type navigationState struct {
	focus              ui.Pane
	rows               []workspaceRow
	cursor             int
	selection          core.NodeKey
	scope              *core.Location
	inspectorScroll    int
	tree               core.Workspace
	domains            map[string][]string
	projects           []core.Project
	projectIndex       map[string]core.Project
	duplicateLocations map[string]bool
}

type creationState struct {
	failure      app.Diagnostic
	returnState  appState
	name         string
	preset       preset.Preset
	template     *tmpl.Template
	repo         gh.Repo
	createGitHub bool
	private      bool
}

type launchState struct {
	project core.Project
	ready   bool
	preset  preset.Preset
	session string // session name to attach after TUI exits
}

type authState struct {
	id      uint64
	context context.Context
	cancel  context.CancelFunc
	client  authClient
	screen  githubAuthScreen
	code    gh.DeviceCodeResponse
}

type scanState struct {
	startup      tea.Cmd
	inFlight     bool
	warnings     string
	warningCount int
	requested    bool
	id           uint64
}

type repositoriesState struct {
	loading bool
	id      uint64
}

type tmuxState struct {
	requested  bool
	id         uint64
	deleting   bool
	refreshing bool
	cached     map[string]bool
	sessions   []string
}

type queryState struct {
	input   textinput.Model
	editing bool
	restore string
	value   string
}
