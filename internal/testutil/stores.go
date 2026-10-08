// Package testutil contains small deterministic external-boundary test helpers.
package testutil

import (
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/preset"
	"github.com/richardnascimento18/devdock/internal/state"
)

type ConfigSave func(config.Config) error

func (f ConfigSave) Save(value config.Config) error { return f(value) }

type StateSave func(state.UIState) error

func (f StateSave) Save(value state.UIState) error { return f(value) }

type PresetSave func([]preset.Preset) error

func (f PresetSave) Save(value []preset.Preset) error { return f(value) }
