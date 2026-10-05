package main

import (
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/ui"
)

type animationClock struct {
	generation       uint64
	frame            int
	pending, reduced bool
	label            string
}
type animationTickMsg struct{ generation uint64 }

func reducedMotion() bool {
	value := strings.ToLower(os.Getenv("DEVDOCK_REDUCED_MOTION"))
	return value == "1" || value == "true" || value == "on" || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
}
func animationTick(generation uint64, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg { return animationTickMsg{generation} })
}
func (m model) activityLabel() string {
	switch {
	case m.loading():
		return m.spinnerScr.message
	case m.state == statePTYExecution && !m.ptyScr.completed:
		return "Running template steps…"
	case m.state == stateGitHubAuth && m.githubAuthScr.verificationURI == "" && m.githubAuthScr.err == "":
		return "Connecting to GitHub…"
	case m.state == stateGitHubAuth && m.githubAuthScr.done && m.repoLoading:
		return "Loading GitHub repositories…"
	case m.state != stateList:
		return ""
	case m.scanInFlight:
		return "Scanning workspace…"
	case m.repoLoading:
		return "Loading GitHub repositories…"
	case m.tmuxRefreshing || m.tmuxDeleting:
		return "Updating tmux sessions…"
	}
	return ""
}

// Only this presentation clock schedules animation ticks. Async service IDs and
// deadlines remain independent; animation never starts or completes operations.
func (m *model) scheduleAnimation() tea.Cmd {
	label := m.activityLabel()
	if label != m.motion.label {
		if label != "" && (m.motion.label != "" || !m.motion.pending) {
			m.motion.frame = 0
		}
		if m.motion.pending && m.motion.label != "" {
			m.motion.generation++
			m.motion.pending = false
		}
		m.motion.label = label
	}
	if m.motion.reduced || label == "" {
		if m.motion.pending {
			m.motion.generation++
			m.motion.pending = false
		}
		return nil
	}
	if m.motion.pending {
		return nil
	}
	m.motion.pending = true
	m.motion.generation++
	delay := 160 * time.Millisecond
	if m.motion.frame == 0 {
		delay = 320 * time.Millisecond
	}
	return animationTick(m.motion.generation, delay)
}
func (m model) handleAnimationTick(msg animationTickMsg) (tea.Model, tea.Cmd) {
	if msg.generation != m.motion.generation || !m.motion.pending || m.motion.reduced || m.activityLabel() == "" {
		return m, nil
	}
	m.motion.pending = false
	m.motion.frame = (m.motion.frame + 1) % 10000
	return m, nil
}
func (m model) activityView() string {
	return ui.Activity(m.activityLabel(), m.motion.frame, m.motion.reduced)
}
