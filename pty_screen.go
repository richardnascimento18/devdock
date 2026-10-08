package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/richardnascimento18/devdock/internal/app"

	"github.com/richardnascimento18/devdock/internal/pty"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"github.com/richardnascimento18/devdock/internal/terminal"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/ui"
)

// ---------------------------------------------------------------------------
// Message types
// ---------------------------------------------------------------------------

type ptyStepStartMsg struct {
	session *pty.Session
	cmdStr  string
}

type ptyBuiltinCompleteMsg struct{ text string }

type ptyDoneMsg struct{ err error }

type ptyFlowMsg struct {
	id  uint64
	msg tea.Msg
}

func (p ptyScreen) wrap(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	id := p.operationID
	return func() tea.Msg { return ptyFlowMsg{id: id, msg: cmd()} }
}

type ptyInterruptMsg struct{}

// ---------------------------------------------------------------------------
// ptyScreen
// ---------------------------------------------------------------------------

type logLine struct {
	ts   time.Time
	text string
	kind lineKind
}

type lineKind int

const (
	lineNormal lineKind = iota
	lineCmd
	lineSystem
	lineSuccess
	lineError
)

type ptyScreen struct {
	context     context.Context
	cancel      context.CancelFunc
	operationID uint64
	viewport    viewport.Model
	// committed lines — permanent log
	lines []logLine
	// liveBlock holds lines currently being redrawn in-place (e.g. interactive
	// prompts that use \r to overwrite themselves). When we receive a bare \r
	// (without \n), we reset the live block and start fresh. When we receive \n,
	// the current live line is promoted to the committed log.
	vscreen terminal.Screen

	session       *pty.Session
	workDir       string
	width         int
	height        int
	completed     bool
	motionFrame   int
	reducedMotion bool
	interrupted   bool
	exitErr       error

	tmpl     *tmpl.Template
	scaffold app.Scaffold
}

func newPTYScreenWithScaffold(parent context.Context, w, h int, t *tmpl.Template, scaffold app.Scaffold) ptyScreen {
	ctx, cancel := context.WithCancel(parent)
	vp := viewport.New(vpW(w), vpH(h))
	vp.Style = lipgloss.NewStyle()
	ps := ptyScreen{
		context: ctx, cancel: cancel,
		viewport: vp,
		width:    w,
		height:   h,
		tmpl:     t,
		scaffold: scaffold,
	}
	ps.addLine("Starting template setup...", lineSystem)
	return ps
}

func vpW(total int) int { return max(total-2, 1) }
func vpH(total int) int { return max(total-5, 1) }

func (p *ptyScreen) addLine(text string, kind lineKind) {
	text = strings.TrimRightFunc(text, unicode.IsSpace)
	if text == "" {
		return
	}
	p.lines = append(p.lines, logLine{ts: time.Now(), text: text, kind: kind})
	p.refreshViewport()
}

func (p *ptyScreen) flushPending() {
	for _, l := range p.vscreen.Lines() {
		t := strings.TrimSpace(l)
		if t != "" {
			p.lines = append(p.lines, logLine{ts: time.Now(), text: t, kind: lineNormal})
		}
	}
	p.vscreen.Reset()
	p.refreshViewport()
}

// buildViewportContent renders all committed lines plus the live block into a string.
func (p *ptyScreen) buildViewportContent(extraPending string) string {
	w := vpW(p.width)
	tsStyle := lipgloss.NewStyle().Foreground(theme.Faint)
	sepStyle := lipgloss.NewStyle().Foreground(theme.Faint)
	normalStyle := lipgloss.NewStyle().Foreground(theme.Primary)
	cmdStyle := lipgloss.NewStyle().Foreground(theme.Info).Bold(true)
	sysStyle := lipgloss.NewStyle().Foreground(theme.Secondary)
	okStyle := lipgloss.NewStyle().Foreground(theme.Success).Bold(true)
	errStyle := lipgloss.NewStyle().Foreground(theme.Error)
	liveStyle := lipgloss.NewStyle().Foreground(theme.Primary)
	const tsWidth = 19
	const sepWidth = 4
	msgW := w - tsWidth - sepWidth - 2
	if msgW < 10 {
		msgW = 10
	}

	var sb strings.Builder

	renderCommitted := func(l logLine) {
		ts := tsStyle.Render(ui.SafeBlock(l.ts.Format("2006-01-02 15:04:05")))
		sep := sepStyle.Render(" -- ")
		var msgStyle lipgloss.Style
		switch l.kind {
		case lineCmd:
			msgStyle = cmdStyle
		case lineSystem:
			msgStyle = sysStyle
		case lineSuccess:
			msgStyle = okStyle
		case lineError:
			msgStyle = errStyle
		default:
			msgStyle = normalStyle
		}
		wrapped := hardWrap(l.text, msgW)
		for i, wl := range strings.Split(wrapped, "\n") {
			if i == 0 {
				sb.WriteString(ts + sep + msgStyle.Render(ui.SafeBlock(wl)) + "\n")
			} else {
				sb.WriteString(strings.Repeat(" ", tsWidth+sepWidth) + msgStyle.Render(ui.SafeBlock(wl)) + "\n")
			}
		}
	}

	for _, l := range p.lines {
		renderCommitted(l)
	}

	// Replace the live block rendering section with:
	ts := tsStyle.Render(ui.SafeBlock(time.Now().Format("2006-01-02 15:04:05")))
	sep := sepStyle.Render(" -- ")

	firstLive := true
	for _, line := range p.vscreen.Lines() {
		if strings.TrimSpace(line) == "" {
			continue
		}
		wrapped := hardWrap(line, msgW)
		parts := strings.Split(wrapped, "\n")
		for j, wl := range parts {
			if firstLive && j == 0 {
				sb.WriteString(ts + sep + liveStyle.Render(ui.SafeBlock(wl)) + "\n")
				firstLive = false
			} else {
				sb.WriteString(strings.Repeat(" ", tsWidth+sepWidth) + liveStyle.Render(ui.SafeBlock(wl)) + "\n")
			}
		}
	}

	return sb.String()
}

func (p *ptyScreen) refreshViewport() {
	p.viewport.SetContent(p.buildViewportContent(""))
	p.viewport.GotoBottom()
}

func hardWrap(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ui.SafeBlock(s)
	}
	return ansi.Hardwrap(ui.SafeBlock(s), maxWidth, true)
}

// PTY data is interpreted locally; no child control sequences reach the host.
func (p *ptyScreen) ingestPTYData(raw []byte) { p.vscreen.Write(raw); p.refreshViewport() }
func readPTY(session *pty.Session) tea.Cmd    { return func() tea.Msg { return session.Read() } }

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (p ptyScreen) Init() tea.Cmd { return nil }

func (p ptyScreen) Update(msg tea.Msg) (ptyScreen, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			p.cancel()
			if p.session != nil {
				if err := p.session.Close(); err != nil {
					p.addLine(err.Error(), lineError)
				}
			}
			p.session = nil
			p.interrupted = true
			p.completed = true
			return p, p.wrap(func() tea.Msg { return ptyInterruptMsg{} })
		}
		if p.completed || p.session == nil {
			return p, nil
		}
		var cmdErr error
		switch msg.Type {
		case tea.KeyEnter:
			cmdErr = p.session.Write([]byte("\r"))
		case tea.KeyBackspace, tea.KeyDelete:
			cmdErr = p.session.Write([]byte{127})
		case tea.KeyCtrlD:
			cmdErr = p.session.Write([]byte{4})
		case tea.KeyUp:
			cmdErr = p.session.Write([]byte("\x1b[A"))
		case tea.KeyDown:
			cmdErr = p.session.Write([]byte("\x1b[B"))
		case tea.KeyLeft:
			cmdErr = p.session.Write([]byte("\x1b[D"))
		case tea.KeyRight:
			cmdErr = p.session.Write([]byte("\x1b[C"))
		case tea.KeySpace:
			cmdErr = p.session.Write([]byte(" "))
		case tea.KeyTab:
			cmdErr = p.session.Write([]byte("\t"))
		case tea.KeyRunes:
			cmdErr = p.session.Write([]byte(string(msg.Runes)))
		}
		if cmdErr != nil {
			p.addLine(fmt.Sprintf("terminal input: %v", cmdErr), lineError)
		}
		return p, nil

	case ptyStepStartMsg:
		p.session = msg.session
		p.vscreen.Reset()
		p.addLine(fmt.Sprintf("$ %s", msg.cmdStr), lineCmd)
		return p, p.wrap(readPTY(p.session))

	case ptyBuiltinCompleteMsg:
		p.addLine(msg.text, lineSystem)
		return p, p.startNextStep()

	case ptyDoneMsg:
		p.cancel()
		p.flushPending()
		p.completed = true
		p.exitErr = msg.err
		if msg.err != nil {
			p.addLine(msg.err.Error(), lineError)
		} else {
			p.addLine("✓ Setup completed successfully!", lineSuccess)
		}
		return p, nil

	case pty.OutputMsg:
		if msg.Session != p.session || p.completed {
			return p, nil
		}
		if len(msg.Data) > 0 {
			p.ingestPTYData(msg.Data)
		}
		if p.session != nil && !p.completed {
			return p, p.wrap(readPTY(p.session))
		}

	case pty.ExitMsg:
		if msg.Session != nil && msg.Session != p.session {
			return p, nil
		}
		if p.session != nil {
			msg.Err = errors.Join(msg.Err, p.session.Close())
			p.session = nil
		}
		// If we were interrupted, don't proceed — ptyInterruptMsg was already sent.
		if p.interrupted {
			return p, nil
		}
		p.flushPending()
		if msg.Err != nil {
			p.completed = true
			p.exitErr = msg.Err
			p.cancel()
			p.addLine(fmt.Sprintf("✗ Process exited with error: %v", msg.Err), lineError)
			return p, nil
		}
		return p, p.startNextStep()

	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		p.viewport.Width = vpW(msg.Width)
		p.viewport.Height = vpH(msg.Height)
		p.refreshViewport()
		if p.session != nil {
			if err := p.session.Resize(uint16(vpH(msg.Height)), uint16(vpW(msg.Width))); err != nil {
				p.addLine(fmt.Sprintf("resize terminal: %v", err), lineError)
			}
		}
	}
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

// ---------------------------------------------------------------------------
// startNextStep
// ---------------------------------------------------------------------------

func (p *ptyScreen) startNextStep() (cmd tea.Cmd) {
	defer func() { cmd = p.wrap(cmd) }()
	step, ok, err := p.scaffold.Execution.Next()
	if err != nil {
		return func() tea.Msg { return pty.ExitMsg{Err: err} }
	}
	scaffold, ctx := p.scaffold, p.context
	if !ok {
		return func() tea.Msg { return ptyDoneMsg{err: scaffold.Finish(ctx)} }
	}
	if step.PostStarted {
		p.addLine("Running post-setup steps...", lineSystem)
	}
	rows, cols := uint16(vpH(p.height)), uint16(vpW(p.width))
	return func() tea.Msg {
		result, err := scaffold.Execute(ctx, step, rows, cols)
		if err != nil {
			return pty.ExitMsg{Err: err}
		}
		if result.Session != nil {
			return ptyStepStartMsg{session: result.Session, cmdStr: result.Text}
		}
		return ptyBuiltinCompleteMsg{text: result.Text}
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (p ptyScreen) View() string {
	title := "Template Setup"
	if p.tmpl != nil {
		title += ": " + p.tmpl.Name
	}
	progress := "✓ Complete"
	footer := "Opening workspace…"
	if !p.completed {
		label := "Running…"
		index, total, _ := p.scaffold.Execution.Progress()
		if total > 0 {
			label = fmt.Sprintf("Step %d/%d", min(max(index, 1), total), total)
		}

		progress = ui.Activity(label, p.motionFrame, p.reducedMotion)
		footer = "Type to send input · ctrl+c interrupt"
		if p.width < 40 {
			footer = "Type input · ctrl+c stop"
		}
	}
	terminal := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(theme.BorderFocused).Width(vpW(p.width)).Height(vpH(p.height)).Render(p.viewport.View())
	return strings.Join([]string{ui.Header(title, p.width), ui.Fit(progress, p.width, 1), terminal, ui.Fit(dimStyle.Render(ui.SafeBlock(footer)), p.width, 1)}, "\n")
}
