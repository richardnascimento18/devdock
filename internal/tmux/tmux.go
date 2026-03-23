package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/preset"
)

func run(args ...string) error {
	cmd := exec.Command("tmux", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runOutput(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.Output()
	return string(out), err
}

func LaunchWorkspace(p core.Project, ps preset.Preset) {
	if len(ps.Windows) == 0 {
		ps = preset.DefaultPresets[0]
	}
	session := sanitizeTmuxName(fmt.Sprintf("%s-%s", p.Domain, p.Name))
	if exec.Command("tmux", "has-session", "-t", session).Run() == nil {
		run("attach-session", "-t", session)
		return
	}
	first := ps.Windows[0]
	firstName := sanitizeTmuxName(first.Name)
	run("new-session", "-d", "-s", session, "-n", firstName, "-c", p.Path)
	buildWindowLayout(session, first, p.Path)
	for _, w := range ps.Windows[1:] {
		run("new-window", "-t", session, "-n", sanitizeTmuxName(w.Name), "-c", p.Path)
		buildWindowLayout(session, w, p.Path)
	}
	run("select-window", "-t", session+":"+firstName)
	run("attach-session", "-t", session)
}

func buildWindowLayout(session string, w preset.Window, projectPath string) {
	target := session + ":" + sanitizeTmuxName(w.Name)
	if w.Layout != nil {
		buildPaneTree(target, *w.Layout, projectPath, true)
		run("select-layout", "-t", target, "tiled")
		return
	}
	if w.Command != "" {
		run("send-keys", "-t", target, w.Command, "C-m")
	}
}

func buildPaneTree(target string, pl preset.PaneLayout, projectPath string, isFirst bool) {
	if pl.IsLeaf() {
		if pl.Command != "" {
			run("send-keys", "-t", target, pl.Command, "C-m")
		}
		return
	}
	for i, child := range pl.Panes {
		var paneTarget string
		if i == 0 {
			paneTarget = target
		} else {
			splitFlag := "-h"
			if pl.Direction == "vertical" {
				splitFlag = "-v"
			}
			args := []string{"split-window", splitFlag, "-t", target, "-c", projectPath}
			if child.Size > 0 && child.Size < 100 {
				args = append(args, "-p", strconv.Itoa(child.Size))
			}
			run(args...)
			id, err := runOutput("display-message", "-t", target, "-p", "#{pane_id}")
			if err == nil && len(id) > 0 {
				paneTarget = sessionPart(target) + "." + trimNewline(id)
			} else {
				paneTarget = target
			}
		}
		buildPaneTree(paneTarget, child, projectPath, i == 0)
	}
}

func sessionPart(target string) string {
	for i, c := range target {
		if c == '.' {
			return target[:i]
		}
	}
	return target
}

func trimNewline(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		return s[:len(s)-1]
	}
	return s
}

// sanitizeTmuxName replaces characters that tmux interprets specially in target
// names (. and :) with underscores so session and window names are unambiguous.
func sanitizeTmuxName(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '.' || r == ':' {
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ListSessions returns the names of all active tmux sessions, or nil if tmux
// is not running or not installed.
func ListSessions() []string {
	out, err := runOutput("list-sessions", "-F", "#{session_name}")
	if err != nil {
		return nil
	}
	var sessions []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			sessions = append(sessions, line)
		}
	}
	return sessions
}

// AttachSession attaches the current terminal to the named tmux session.
// This replaces the current process image, so it must be called after the TUI
// has fully torn down its alt-screen.
func AttachSession(name string) {
	run("attach-session", "-t", name)
}

// KillSession destroys the named tmux session and all its windows.
func KillSession(name string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
