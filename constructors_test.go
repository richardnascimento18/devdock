package main

import (
	"github.com/richardnascimento18/devdock/internal/app"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/preset"
	"github.com/richardnascimento18/devdock/internal/state"
	"github.com/richardnascimento18/devdock/internal/template"
)

// Existing characterization fixtures set an isolated HOME. Production always
// constructs stores at startup and passes them explicitly to both models.
func newModel(projects []core.Project, cfg config.Config, presets []preset.Preset, templates []template.Template, uiState state.UIState) model {
	paths, _ := app.ResolvePaths()
	return newModelWithPreferences(projects, cfg, presets, templates, uiState, app.NewPreferences(paths))
}
func newEditorScreen(presets []preset.Preset, templates []template.Template, w, h int) editorScreen {
	paths, _ := app.ResolvePaths()
	return newEditorScreenWithPreferences(presets, templates, w, h, app.NewPreferences(paths))
}
