package main

import (
	"errors"
	tea "github.com/charmbracelet/bubbletea"
	"testing"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/state"
	"github.com/richardnascimento18/devdock/internal/testutil"
)

func TestInjectedStateFailureRetainsFilesystemReconciliation(t *testing.T) {
	m := fixtureModel(t, t.TempDir())
	failure := errors.New("injected persistence failure")
	m.preferences.State = testutil.StateSave(func(state.UIState) error { return failure })
	m.uiState.Favorites["/workspace/apps/example"] = true
	if m.saveFilesystemState() || !errors.Is(m.persistenceErr, failure) || !m.uiState.Favorites["/workspace/apps/example"] {
		t.Fatal("filesystem reconciliation lost on failure")
	}
	m.preferences.State = testutil.StateSave(func(state.UIState) error { return nil })
	if !m.saveState() || m.filesystemStatePending || !m.committedState.Favorites["/workspace/apps/example"] {
		t.Fatal("retry did not commit retained reconciliation")
	}
}

func TestInjectedConfigFailureKeepsEditorDraft(t *testing.T) {
	m := fixtureModel(t, t.TempDir())
	m.presets = append(m.presets, m.presets[0])
	m.presets[len(m.presets)-1].Name = "example"
	m.preferences.Config = testutil.ConfigSave(func(config.Config) error { return errors.New("injected config failure") })
	next, _ := m.startEditor()
	m = next.(model)
	e := m.editorScr
	e.cfg.DefaultPreset = ""
	e.layer = editorLayerConfig
	e.ce = newConfigurationEditor("example")
	nextEditor, _ := e.updateConfigurationEditor(tea.KeyMsg{Type: tea.KeyCtrlS})
	if nextEditor.cfg.DefaultPreset != "" || nextEditor.layer != editorLayerConfig || nextEditor.ce.defaultPreset.Value() != "example" || nextEditor.ce.diagnostic.Summary == "" {
		t.Fatal("failed config lost draft or changed active values")
	}
}
