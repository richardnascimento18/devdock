package preset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/richardnascimento18/devdock/internal/fileutil"
)

type PaneLayout struct {
	Direction string       `json:"direction,omitempty"`
	Command   string       `json:"command,omitempty"`
	Panes     []PaneLayout `json:"panes,omitempty"`
	Size      int          `json:"size,omitempty"`
}

func (p PaneLayout) IsLeaf() bool { return len(p.Panes) == 0 }

type Window struct {
	Name    string      `json:"name"`
	Command string      `json:"command,omitempty"`
	Layout  *PaneLayout `json:"layout,omitempty"`
}

type Preset struct {
	Name    string   `json:"name"`
	Windows []Window `json:"windows"`
}

type PresetFile struct {
	Presets []Preset `json:"presets"`
}

var DefaultPresets = []Preset{
	{
		Name: "walker",
		Windows: []Window{
			{Name: "nvim", Command: "nvim"},
			{Name: "terminal", Command: ""},
			{Name: "ai-chat", Command: "opencode"},
		},
	},
	{
		Name: "nvim",
		Windows: []Window{
			{Name: "nvim", Command: "nvim"},
			{Name: "terminal", Command: ""},
		},
	},
	{
		Name: "dev-split",
		Windows: []Window{
			{
				Name: "dev",
				Layout: &PaneLayout{
					Direction: "horizontal",
					Panes: []PaneLayout{
						{Command: "nvim"},
						{
							Direction: "vertical",
							Panes: []PaneLayout{
								{Command: ""},
								{Command: ""},
							},
						},
					},
				},
			},
		},
	},
}

func Path(configDir string) string {
	return filepath.Join(configDir, "presets.json")
}

func ValidatePreset(p Preset) string {
	if strings.TrimSpace(p.Name) == "" {
		return "preset has an empty name"
	}
	if len(p.Windows) == 0 {
		return fmt.Sprintf("preset %q has no windows", p.Name)
	}
	seen := map[string]bool{}
	for i, w := range p.Windows {
		if strings.TrimSpace(w.Name) == "" || strings.ContainsAny(w.Name, "\n\r\x00") {
			return fmt.Sprintf("preset %q: window at index %d has an empty name", p.Name, i)
		}
		targetName := strings.NewReplacer(".", "_", ":", "_").Replace(w.Name)
		if seen[targetName] {
			return fmt.Sprintf("preset %q: duplicate window name %q", p.Name, w.Name)
		}
		seen[targetName] = true
		if w.Layout != nil {
			if err := validatePaneLayout(*w.Layout, p.Name, w.Name); err != "" {
				return err
			}
		}
	}
	return ""
}

func validatePaneLayout(pl PaneLayout, presetName, winName string) string {
	if pl.Size < 0 || pl.Size >= 100 {
		return fmt.Sprintf("preset %q window %q: pane size must be 0..99", presetName, winName)
	}
	if !pl.IsLeaf() {
		if pl.Direction != "horizontal" && pl.Direction != "vertical" {
			return fmt.Sprintf("preset %q window %q: pane direction must be \"horizontal\" or \"vertical\", got %q",
				presetName, winName, pl.Direction)
		}
		if len(pl.Panes) < 2 {
			return fmt.Sprintf("preset %q window %q: pane split must have at least 2 child panes",
				presetName, winName)
		}
		for _, child := range pl.Panes {
			if err := validatePaneLayout(child, presetName, winName); err != "" {
				return err
			}
		}
	}
	return ""
}

func ValidatePresetFile(pf PresetFile) []string {
	var errs []string
	if len(pf.Presets) == 0 {
		errs = append(errs, "presets.json: no presets defined")
		return errs
	}
	names := map[string]bool{}
	for _, p := range pf.Presets {
		if msg := ValidatePreset(p); msg != "" {
			errs = append(errs, msg)
		}
		if names[p.Name] {
			errs = append(errs, fmt.Sprintf("duplicate preset name %q", p.Name))
		}
		names[p.Name] = true
	}
	return errs
}

func Load(configDir string) ([]Preset, error) {
	presetsPath := Path(configDir)
	data, err := os.ReadFile(presetsPath)
	if os.IsNotExist(err) {
		if writeErr := writeDefaults(configDir); writeErr != nil {
			return Clone(DefaultPresets), writeErr
		}
		return Clone(DefaultPresets), nil
	}
	if err != nil {
		return Clone(DefaultPresets), fmt.Errorf("could not read presets.json: %w", err)
	}
	var pf PresetFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return Clone(DefaultPresets), fmt.Errorf(
			"presets.json contains invalid JSON:\n  %v\n\nfalling back to built-in presets", err,
		)
	}
	if errs := ValidatePresetFile(pf); len(errs) > 0 {
		msg := "presets.json has errors:\n"
		for _, e := range errs {
			msg += "  - " + e + "\n"
		}
		return Clone(DefaultPresets), fmt.Errorf("%s\nfalling back to built-in presets", msg)
	}
	return pf.Presets, nil
}

func writeDefaults(configDir string) error { return Save(configDir, DefaultPresets) }

func ByName(presets []Preset, name string) Preset {
	for _, p := range presets {
		if p.Name == name {
			return p
		}
	}
	if len(presets) > 0 {
		return presets[0]
	}
	return DefaultPresets[0]
}

// Save validates the entire proposed collection before committing it.
func Save(configDir string, values []Preset) error {
	if !filepath.IsAbs(configDir) {
		return fmt.Errorf("configuration directory must be absolute")
	}
	file := PresetFile{Presets: values}
	if errs := ValidatePresetFile(file); len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	data, err := json.MarshalIndent(file, "", "    ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(Path(configDir), data, 0o644)
}

func CloneLayout(src *PaneLayout) *PaneLayout {
	if src == nil {
		return nil
	}
	dst := *src
	if src.Panes != nil {
		dst.Panes = make([]PaneLayout, len(src.Panes))
	}
	for i := range src.Panes {
		dst.Panes[i] = *CloneLayout(&src.Panes[i])
	}
	return &dst
}
func Clone(src []Preset) []Preset {
	if src == nil {
		return nil
	}
	dst := append([]Preset{}, src...)
	for i := range dst {
		dst[i].Windows = append([]Window(nil), src[i].Windows...)
		for j := range dst[i].Windows {
			dst[i].Windows[j].Layout = CloneLayout(src[i].Windows[j].Layout)
		}
	}
	return dst
}
