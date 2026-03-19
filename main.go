package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/detect"
	"github.com/richardnascimento18/devdock/internal/preset"
	uistate "github.com/richardnascimento18/devdock/internal/state"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/tmux"

	tea "github.com/charmbracelet/bubbletea"
)

// classifyFn is used by openMovePlacementPicker in update.go.
// collectFn and classifyFn bridge detect → core to avoid import cycles.

var (
	classifyFn core.ClassifyDirFunc
	collectFn  core.ScanFunc
)

// Set here in main and injected where needed.
func init() {
	collectFn = detect.CollectProjects
	classifyFn = detect.ClassifyDir
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		cfg = runSetup()
	}

	if err := os.MkdirAll(config.Dir(), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not create config dir: %v\n", err)
	}
	projects, err := core.ScanRoots(config.Dir(), cfg.ActiveRoots(), detect.CollectProjects)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	presets, presetErr := preset.Load(config.Dir())
	if presetErr != nil {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "=== presets.json warning ===")
		fmt.Fprintln(os.Stderr, presetErr.Error())
		fmt.Fprintln(os.Stderr, "============================")
		fmt.Fprintln(os.Stderr, "")
	}

	templates, tmplErr := tmpl.Load(config.Dir())
	if tmplErr != nil {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "=== templates.json warning ===")
		fmt.Fprintln(os.Stderr, tmplErr.Error())
		fmt.Fprintln(os.Stderr, "==============================")
		fmt.Fprintln(os.Stderr, "")
	}

	uiSt := uistate.Load(config.Dir())

	m := newModel(projects, cfg, presets, templates, uiSt)
	prog := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := prog.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fm, ok := finalModel.(model)
	if !ok {
		return
	}

	// Launch the workspace AFTER the TUI exits so the alt-screen is torn down
	// and tmux can attach cleanly.
	if fm.pendingLaunchReady {
		tmux.LaunchWorkspace(fm.pendingLaunch, fm.pendingLaunchPreset)
	}
}

// runSetup is the first-run configuration wizard.
func runSetup() config.Config {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("DevDock setup")
	fmt.Println("Enter root directory for your projects:")
	fmt.Print("> ")
	root, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read input: %v\n", err)
		os.Exit(1)
	}
	root = strings.TrimSpace(root)

	// Expand ~ since os.Stat does not understand shell shorthand.
	if root == "~" {
		home, herr := os.UserHomeDir()
		if herr == nil {
			root = home
		}
	} else if strings.HasPrefix(root, "~/") {
		home, herr := os.UserHomeDir()
		if herr == nil {
			root = filepath.Join(home, root[2:])
		}
	}

	info, serr := os.Stat(root)
	if serr != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "error: %q does not exist or is not a directory\n", root)
		os.Exit(1)
	}

	cfg := config.Config{Roots: []string{root}}
	config.Save(cfg)

	preset.Load(config.Dir())
	tmpl.Load(config.Dir())

	fmt.Println("DevDock configured!")
	return cfg
}
