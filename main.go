package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/richardnascimento18/devdock/internal/app"
	"github.com/richardnascimento18/devdock/internal/ui"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, ui.SafeBlock(err.Error()))
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	paths, err := app.ResolvePaths()
	if err != nil {
		return err
	}
	preferences := app.NewPreferences(paths)
	cfg, err := (config.Store{File: paths.Config}).Load()
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		cfg, err = runSetup(paths, os.Stdin, os.Stdout)
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(paths.Directory, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not create config dir: %s\n", ui.SafeBlock(err.Error()))
	}

	presets, presetErr := (preset.Store{Directory: paths.Directory}).Load()
	if presetErr != nil {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "=== presets.json warning ===")
		fmt.Fprintln(os.Stderr, ui.SafeBlock(presetErr.Error()))
		fmt.Fprintln(os.Stderr, "============================")
		fmt.Fprintln(os.Stderr, "")
	}

	templates, tmplErr := (tmpl.Store{Directory: paths.Directory}).Load()
	if tmplErr != nil {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "=== templates.json warning ===")
		fmt.Fprintln(os.Stderr, ui.SafeBlock(tmplErr.Error()))
		fmt.Fprintln(os.Stderr, "==============================")
		fmt.Fprintln(os.Stderr, "")
	}

	uiSt, stateErr := (uistate.Store{Directory: paths.Directory}).Load()
	if stateErr != nil {
		fmt.Fprintln(os.Stderr, ui.SafeBlock(stateErr.Error()))
	}

	m := newModelWithPreferences(nil, cfg, presets, templates, uiSt, preferences)
	m.appContext = ctx
	m.tmuxClient = tmux.NewClientContext(ctx, "")
	m.startupCmd = m.scanCommand()
	m.motion.pending = !m.motion.reduced
	m.motion.generation++
	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx))
	finalModel, err := prog.Run()
	fm, ok := finalModel.(model)
	if !ok {
		return err
	}

	fm.cancelAuth()
	if fm.ptyScr.cancel != nil {
		fm.ptyScr.cancel()
	}
	if fm.persistenceErr != nil {
		fmt.Fprintln(os.Stderr, ui.SafeBlock(fm.persistenceErr.Error()))
	}
	if fm.ptyScr.session != nil {
		if err := fm.ptyScr.session.Close(); err != nil {
			fmt.Fprintln(os.Stderr, ui.SafeBlock(err.Error()))
		}
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}

	// Launch the workspace AFTER the TUI exits so the alt-screen is torn down
	// and tmux can attach cleanly.
	if fm.pendingLaunchReady {
		if err := fm.tmuxClient.LaunchWorkspace(fm.pendingLaunch, fm.pendingLaunchPreset); err != nil {
			return err
		}
	}
	if fm.pendingTmuxAttach != "" {
		if err := fm.tmuxClient.AttachSession(fm.pendingTmuxAttach); err != nil {
			return err
		}
	}
	return nil
}

// runSetup is the first-run configuration wizard.
func runSetup(paths app.Paths, input io.Reader, output io.Writer) (config.Config, error) {
	reader := bufio.NewReader(input)
	fmt.Fprintln(output, "DevDock setup")
	fmt.Fprintln(output, "Enter root directory for your projects:")
	fmt.Fprint(output, "> ")
	root, err := reader.ReadString('\n')
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to read input: %w", err)
	}
	root = strings.TrimSpace(root)

	if root == "~" {
		root = paths.Home
	} else if strings.HasPrefix(root, "~/") {
		root = filepath.Join(paths.Home, root[2:])
	}

	root, err = filepath.Abs(root)
	if err != nil {
		return config.Config{}, err
	}
	info, serr := os.Stat(root)
	if serr != nil || !info.IsDir() {
		return config.Config{}, fmt.Errorf("error: %q does not exist or is not a directory", root)
	}

	cfg := config.Config{Roots: []string{root}}
	if err := (config.Store{File: paths.Config}).Save(cfg); err != nil {
		return config.Config{}, err
	}

	fmt.Fprintln(output, "DevDock configured!")
	return cfg, nil
}
