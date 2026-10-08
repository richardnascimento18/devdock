package app

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/preset"
	"github.com/richardnascimento18/devdock/internal/state"
	"github.com/richardnascimento18/devdock/internal/testutil"
)

func TestStoresRemainBoundToResolvedPaths(t *testing.T) {
	dir := t.TempDir()
	paths := PathsAt(dir)
	p := NewPreferences(paths)
	t.Setenv("HOME", t.TempDir())
	value := config.Config{DefaultPreset: "example"}
	if err := p.Config.Save(value); err != nil {
		t.Fatal(err)
	}
	got, err := (config.Store{File: filepath.Join(dir, "config.toml")}).Load()
	if err != nil || got.DefaultPreset != "example" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestCommitFailureAndUnchangedProposal(t *testing.T) {
	failure := errors.New("write failed")
	calls := 0
	p := Preferences{Config: testutil.ConfigSave(func(config.Config) error { calls++; return failure })}
	current := config.Config{DefaultPreset: "before"}
	got, changed, err := p.CommitConfig(current, current)
	if err != nil || changed || calls != 0 || !reflect.DeepEqual(got, current) {
		t.Fatal("unchanged proposal wrote")
	}
	proposed := current.Clone()
	proposed.DefaultPreset = "after"
	got, changed, err = p.CommitConfig(current, proposed)
	if !errors.Is(err, failure) || changed || !reflect.DeepEqual(got, current) {
		t.Fatal("failed proposal became active")
	}
}

func TestPresetFailureCompensatesAndReportsRollbackFailure(t *testing.T) {
	writeFailure, rollbackFailure := errors.New("preset write"), errors.New("config rollback")
	var saved []config.Config
	p := Preferences{Config: testutil.ConfigSave(func(value config.Config) error {
		saved = append(saved, value.Clone())
		if len(saved) == 2 {
			return rollbackFailure
		}
		return nil
	}), Presets: testutil.PresetSave(func([]preset.Preset) error { return writeFailure })}
	current := config.Config{DefaultPreset: "before"}
	values := []preset.Preset{{Name: "after", Windows: []preset.Window{{Name: "shell"}}}}
	got, err := p.SavePresetProposal(current, values, "before", "after")
	if !errors.Is(err, writeFailure) || !errors.Is(err, rollbackFailure) || !reflect.DeepEqual(got, current) {
		t.Fatalf("%+v %v", got, err)
	}
	if len(saved) != 2 || saved[0].DefaultPreset != "after" || saved[1].DefaultPreset != "before" {
		t.Fatal(saved)
	}
}

func TestRootRemovalFailureRetainsLiveValues(t *testing.T) {
	root := t.TempDir()
	current := config.Config{Roots: []string{root}}
	s := state.UIState{Favorites: map[string]bool{filepath.Join(root, "apps", "example"): true}}
	failure := errors.New("state write")
	var saved []config.Config
	p := Preferences{Config: testutil.ConfigSave(func(value config.Config) error { saved = append(saved, value.Clone()); return nil }),
		State: testutil.StateSave(func(state.UIState) error { return failure })}
	got, gotState, err := p.RemoveRoot(current, s, root)
	if !errors.Is(err, failure) || !reflect.DeepEqual(got, current) || !reflect.DeepEqual(gotState, s) || len(s.Favorites) != 1 {
		t.Fatal("failed removal changed live values")
	}
	if len(saved) != 2 || len(saved[0].ActiveRoots()) != 0 || !reflect.DeepEqual(saved[1], current) {
		t.Fatal(saved)
	}
}
