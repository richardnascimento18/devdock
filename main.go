package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/preset"
	uistate "github.com/richardnascimento18/devdock/internal/state"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/tmux"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Println(versionInfo())
		return
	}
	cfg, err := config.Load()
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		cfg = runSetup()
	}

	if err := os.MkdirAll(config.Dir(), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not create config dir: %v\n", err)
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

	uiSt, stateErr := uistate.Load(config.Dir())
	if stateErr != nil {
		fmt.Fprintln(os.Stderr, stateErr)
	}

	m := newModel(nil, cfg, presets, templates, uiSt)
	m.startupCmd = m.scanCommand()
	m.motion.pending = !m.motion.reduced
	m.motion.generation++
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

	fm.cancelAuth()
	if fm.ptyScr.cancel != nil {
		fm.ptyScr.cancel()
	}
	if fm.persistenceErr != nil {
		fmt.Fprintln(os.Stderr, fm.persistenceErr)
	}
	if fm.ptyScr.session != nil {
		if err := fm.ptyScr.session.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	// Launch the workspace AFTER the TUI exits so the alt-screen is torn down
	// and tmux can attach cleanly.
	if fm.pendingLaunchReady {
		if err := tmux.LaunchWorkspace(fm.pendingLaunch, fm.pendingLaunchPreset); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if fm.pendingTmuxAttach != "" {
		if err := tmux.AttachSession(fm.pendingTmuxAttach); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
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

	root, err = filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	info, serr := os.Stat(root)
	if serr != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "error: %q does not exist or is not a directory\n", root)
		os.Exit(1)
	}

	cfg := config.Config{Roots: []string{root}}
	if err := config.Save(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("DevDock configured!")
	return cfg
}
