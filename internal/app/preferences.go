package app

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/preset"
	"github.com/richardnascimento18/devdock/internal/state"
	"github.com/richardnascimento18/devdock/internal/template"
)

// These write boundaries support deterministic failures without changing HOME,
// chmod, or global functions. Serialization/validation stays with each store.
type ConfigWriter interface{ Save(config.Config) error }
type StateWriter interface{ Save(state.UIState) error }
type PresetWriter interface{ Save([]preset.Preset) error }
type TemplateWriter interface {
	Save([]template.Template) error
}

type Preferences struct {
	Paths     Paths
	Config    ConfigWriter
	State     StateWriter
	Presets   PresetWriter
	Templates TemplateWriter
}

func NewPreferences(paths Paths) Preferences {
	return Preferences{Paths: paths, Config: config.Store{File: paths.Config}, State: state.Store{Directory: paths.Directory},
		Presets: preset.Store{Directory: paths.Directory}, Templates: template.Store{Directory: paths.Directory}}
}

func (p Preferences) CommitConfig(current, proposed config.Config) (config.Config, bool, error) {
	proposed = proposed.Clone()
	if err := config.Validate(proposed); err != nil {
		return current.Clone(), false, err
	}
	if reflect.DeepEqual(current, proposed) {
		return current.Clone(), false, nil
	}
	if err := p.Config.Save(proposed); err != nil {
		return current.Clone(), false, err
	}
	return proposed, true, nil
}

// Separate atomic files retain the existing compensation policy. This is not
// a crash transaction; journaling and recovery belong to durability hardening.
func (p Preferences) SavePresetProposal(current config.Config, values []preset.Preset, oldName, newName string) (config.Config, error) {
	if errs := preset.ValidatePresetFile(preset.PresetFile{Presets: values}); len(errs) > 0 {
		return current, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	proposed := current.Clone()
	renameDefault := oldName != "" && oldName != newName && current.DefaultPreset == oldName
	if renameDefault {
		proposed.DefaultPreset = newName
		if err := p.Config.Save(proposed); err != nil {
			return current, fmt.Errorf("save default preset: %w", err)
		}
	}
	if err := p.Presets.Save(values); err != nil {
		if renameDefault {
			return current, errors.Join(err, rollbackConfig(p.Config, current))
		}
		return current, err
	}
	return proposed, nil
}

func (p Preferences) RemoveRoot(current config.Config, currentState state.UIState, root string) (config.Config, state.UIState, error) {
	proposed, proposedState := current.Clone(), currentState.Clone()
	proposed.RemoveRoot(root)
	proposedState.RemovePath(root)
	if err := p.Config.Save(proposed); err != nil {
		return current, currentState, fmt.Errorf("error saving config: %w", err)
	}
	if err := p.State.Save(proposedState); err != nil {
		return current, currentState, errors.Join(fmt.Errorf("root state cleanup failed: %w", err), rollbackConfig(p.Config, current))
	}
	return proposed, proposedState, nil
}

func rollbackConfig(store ConfigWriter, current config.Config) error {
	if err := store.Save(current); err != nil {
		return fmt.Errorf("configuration rollback failed: %w", err)
	}
	return nil
}
