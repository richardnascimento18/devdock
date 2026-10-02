package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/preset"
)

// Commit the preset collection and any default-name change before replacing
// live values. Separate atomic files use compensation, as root removal does.
func savePresetProposal(current config.Config, values []preset.Preset, oldName, newName string) (config.Config, error) {
	if errs := preset.ValidatePresetFile(preset.PresetFile{Presets: values}); len(errs) > 0 {
		return current, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	proposed := current.Clone()
	renameDefault := oldName != "" && oldName != newName && current.DefaultPreset == oldName
	if renameDefault {
		proposed.DefaultPreset = newName
		if err := config.Save(proposed); err != nil {
			return current, fmt.Errorf("save default preset: %w", err)
		}
	}
	if err := preset.Save(config.Dir(), values); err != nil {
		if renameDefault {
			rollback := config.Save(current)
			if rollback != nil {
				rollback = fmt.Errorf("configuration rollback failed: %w", rollback)
			}
			return current, errors.Join(err, rollback)
		}
		return current, err
	}
	return proposed, nil
}
