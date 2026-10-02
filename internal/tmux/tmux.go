package tmux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/preset"
)

var ErrNoSession = errors.New("no tmux session")

type Runner interface {
	Run(...string) error
	Output(...string) (string, error)
}
type Client struct{ Runner Runner }
type processRunner struct{}

func (processRunner) Run(args ...string) error {
	if _, err := exec.LookPath("tmux"); err != nil {
		return fmt.Errorf("tmux is required: %w", err)
	}
	cmd := exec.Command("tmux", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
func (processRunner) Output(args ...string) (string, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return "", fmt.Errorf("tmux is required: %w", err)
	}
	out, err := exec.Command("tmux", args...).CombinedOutput()
	if err != nil && len(args) > 0 && (args[0] == "has-session" || args[0] == "list-sessions") {
		message := string(out)
		if strings.Contains(message, "can't find session") || strings.Contains(message, "no server running") || strings.Contains(message, "No such file or directory") {
			return "", ErrNoSession
		}
	}
	return string(out), err
}
func NewClient() Client { return Client{Runner: processRunner{}} }
func (c Client) run(args ...string) error {
	if err := c.Runner.Run(args...); err != nil {
		return fmt.Errorf("tmux %s: %w", args[0], err)
	}
	return nil
}
func (c Client) output(args ...string) (string, error) {
	out, err := c.Runner.Output(args...)
	if err != nil {
		return "", fmt.Errorf("tmux %s: %w", args[0], err)
	}
	return strings.TrimSpace(out), nil
}

func LaunchWorkspace(p core.Project, ps preset.Preset) error {
	return NewClient().LaunchWorkspace(p, ps)
}
func (c Client) LaunchWorkspace(p core.Project, ps preset.Preset) error {
	if err := preset.ValidatePreset(ps); err != "" {
		return fmt.Errorf("invalid preset: %s", err)
	}
	session := SessionName(p)
	_, err := c.output("has-session", "-t", "="+session)
	if err == nil {
		return c.AttachSession(session)
	}
	if !errors.Is(err, ErrNoSession) {
		return err
	}
	first := ps.Windows[0]
	pane, err := c.output("new-session", "-d", "-P", "-F", "#{pane_id}", "-s", session, "-n", sanitizeTmuxName(first.Name), "-c", p.Path)
	if err != nil {
		return err
	}
	if err := c.buildWindow(first, pane, p.Path); err != nil {
		return err
	}
	for _, w := range ps.Windows[1:] {
		pane, err := c.output("new-window", "-P", "-F", "#{pane_id}", "-t", "="+session, "-n", sanitizeTmuxName(w.Name), "-c", p.Path)
		if err != nil {
			return err
		}
		if err := c.buildWindow(w, pane, p.Path); err != nil {
			return err
		}
	}
	if err := c.run("select-window", "-t", "="+session+":"+sanitizeTmuxName(first.Name)); err != nil {
		return err
	}
	return c.AttachSession(session)
}
func (c Client) buildWindow(w preset.Window, pane, path string) error {
	if !validPaneID(pane) {
		return fmt.Errorf("tmux returned invalid pane ID")
	}
	if w.Layout != nil {
		return c.buildPaneTree(pane, *w.Layout, path)
	}
	return c.sendCommand(pane, w.Command)
}
func (c Client) sendCommand(pane, command string) error {
	if command == "" {
		return nil
	}
	if err := c.run("send-keys", "-t", pane, "-l", "--", command); err != nil {
		return err
	}
	return c.run("send-keys", "-t", pane, "Enter")
}
func (c Client) buildPaneTree(target string, layout preset.PaneLayout, path string) error {
	if layout.IsLeaf() {
		return c.sendCommand(target, layout.Command)
	}
	// Split siblings before recursing so nested children do not change which pane
	// the next sibling splits. Pane IDs are globally unique tmux targets.
	targets := []string{target}
	for _, child := range layout.Panes[1:] {
		flag := "-h"
		if layout.Direction == "vertical" {
			flag = "-v"
		}
		args := []string{"split-window", flag, "-P", "-F", "#{pane_id}", "-t", target, "-c", path}
		if child.Size > 0 {
			args = append(args, "-p", strconv.Itoa(child.Size))
		}
		pane, err := c.output(args...)
		if err != nil {
			return err
		}
		if !validPaneID(pane) {
			return fmt.Errorf("tmux split returned invalid pane ID")
		}
		targets = append(targets, pane)
	}
	for i, child := range layout.Panes {
		if err := c.buildPaneTree(targets[i], child, path); err != nil {
			return err
		}
	}
	return nil
}
func validPaneID(id string) bool {
	if !strings.HasPrefix(id, "%") {
		return false
	}
	_, err := strconv.ParseUint(strings.TrimPrefix(id, "%"), 10, 64)
	return err == nil
}
func sanitizeTmuxName(s string) string { return strings.NewReplacer(".", "_", ":", "_").Replace(s) }
func ListSessions() ([]string, error)  { return NewClient().ListSessions() }
func (c Client) ListSessions() ([]string, error) {
	out, err := c.output("list-sessions", "-F", "#{session_name}")
	if errors.Is(err, ErrNoSession) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var sessions []string
	for _, line := range strings.Split(out, "\n") {
		if line != "" {
			sessions = append(sessions, line)
		}
	}
	return sessions, nil
}
func AttachSession(name string) error { return NewClient().AttachSession(name) }
func (c Client) AttachSession(name string) error {
	action := "attach-session"
	if os.Getenv("TMUX") != "" {
		action = "switch-client"
	}
	return c.run(action, "-t", "="+name)
}
func KillSession(name string) error { return NewClient().run("kill-session", "-t", "="+name) }

// SessionName is location based; marker display names cannot alias sessions.
func SessionName(p core.Project) string {
	var readable strings.Builder
	for _, r := range filepath.Base(p.Path) {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			readable.WriteRune(r)
		} else {
			readable.WriteByte('_')
		}
		if readable.Len() >= 32 {
			break
		}
	}
	name := readable.String()
	if name == "" {
		name = "project"
	}
	return "devdock-" + name + "-" + core.PathID(p.Path)
}
