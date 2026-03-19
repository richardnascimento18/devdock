package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/pty"
	tmpl "github.com/richardnascimento18/devdock/internal/template"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Message types
// ---------------------------------------------------------------------------

type ptyStepStartMsg struct {
	session *pty.Session
	cmdStr  string
}

type ptyBuiltinCompleteMsg struct{}

type ptyDoneMsg struct{}

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

type virtualScreen struct {
	lines  []string
	cursor int
	col    int
}

func (v *virtualScreen) ensureLine(row int) {
	for len(v.lines) <= row {
		v.lines = append(v.lines, "")
	}
}

func (v *virtualScreen) currentLines() []string {
	return v.lines
}

func (v *virtualScreen) reset() {
	v.lines = nil
	v.cursor = 0
	v.col = 0
}

func (v *virtualScreen) write(text string) {
	if text == "" {
		return
	}
	v.ensureLine(v.cursor)
	if v.col == 0 {
		// Overwrite from start — replace the line content
		v.lines[v.cursor] = text
	} else {
		// Append to existing content
		existing := v.lines[v.cursor]
		if v.col <= len(existing) {
			v.lines[v.cursor] = existing[:v.col] + text
		} else {
			v.lines[v.cursor] = existing + text
		}
	}
	v.col += len(text)
}

type ptyScreen struct {
	viewport viewport.Model
	// committed lines — permanent log
	lines []logLine
	// liveBlock holds lines currently being redrawn in-place (e.g. interactive
	// prompts that use \r to overwrite themselves). When we receive a bare \r
	// (without \n), we reset the live block and start fresh. When we receive \n,
	// the current live line is promoted to the committed log.
	vscreen virtualScreen

	session     *pty.Session
	workDir     string
	width       int
	height      int
	completed   bool
	interrupted bool
	exitErr     error

	tmpl           *tmpl.Template
	projectPath    string
	currentStepIdx int
	allSteps       []tmpl.TemplateStep
	vars           tmpl.Vars
	isPostSteps    bool
	githubRepo     gh.Repo
}

func newPTYScreen(w, h int, t *tmpl.Template, projectPath string, vars tmpl.Vars, steps []tmpl.TemplateStep, workDir string, ghRepo gh.Repo) ptyScreen {
	vp := viewport.New(vpW(w), vpH(h))
	vp.Style = lipgloss.NewStyle()
	ps := ptyScreen{
		viewport:    vp,
		workDir:     workDir,
		width:       w,
		height:      h,
		tmpl:        t,
		projectPath: projectPath,
		allSteps:    steps,
		vars:        vars,
		githubRepo:  ghRepo,
	}
	ps.addLine("Starting template setup...", lineSystem)
	return ps
}

func vpW(total int) int {
	w := total - 6
	if w < 20 {
		w = 20
	}
	return w
}

func vpH(total int) int {
	h := total - 9
	if h < 5 {
		h = 5
	}
	return h
}

func (p *ptyScreen) addLine(text string, kind lineKind) {
	text = strings.TrimRightFunc(text, unicode.IsSpace)
	if text == "" {
		return
	}
	p.lines = append(p.lines, logLine{ts: time.Now(), text: text, kind: kind})
	p.refreshViewport()
}

func (p *ptyScreen) flushPending() {
	for _, l := range p.vscreen.lines {
		t := strings.TrimSpace(l)
		if t != "" {
			p.lines = append(p.lines, logLine{ts: time.Now(), text: t, kind: lineNormal})
		}
	}
	p.vscreen.reset()
	p.refreshViewport()
}

// buildViewportContent renders all committed lines plus the live block into a string.
func (p *ptyScreen) buildViewportContent(extraPending string) string {
	w := vpW(p.width)
	tsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	cmdStyle := lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	sysStyle := lipgloss.NewStyle().Foreground(colorGray)
	okStyle := lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	errStyle := lipgloss.NewStyle().Foreground(colorRed)
	liveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	const tsWidth = 19
	const sepWidth = 4
	msgW := w - tsWidth - sepWidth - 2
	if msgW < 10 {
		msgW = 10
	}

	var sb strings.Builder

	renderCommitted := func(l logLine) {
		ts := tsStyle.Render(l.ts.Format("2006-01-02 15:04:05"))
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
				sb.WriteString(ts + sep + msgStyle.Render(wl) + "\n")
			} else {
				sb.WriteString(strings.Repeat(" ", tsWidth+sepWidth) + msgStyle.Render(wl) + "\n")
			}
		}
	}

	for _, l := range p.lines {
		renderCommitted(l)
	}

	// Replace the live block rendering section with:
	ts := tsStyle.Render(time.Now().Format("2006-01-02 15:04:05"))
	sep := sepStyle.Render(" -- ")

	firstLive := true
	for _, line := range p.vscreen.currentLines() {
		if strings.TrimSpace(line) == "" {
			continue
		}
		wrapped := hardWrap(line, msgW)
		parts := strings.Split(wrapped, "\n")
		for j, wl := range parts {
			if firstLive && j == 0 {
				sb.WriteString(ts + sep + liveStyle.Render(wl) + "\n")
				firstLive = false
			} else {
				sb.WriteString(strings.Repeat(" ", tsWidth+sepWidth) + liveStyle.Render(wl) + "\n")
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
	if maxWidth <= 0 || len(s) <= maxWidth {
		return s
	}
	var sb strings.Builder
	for len(s) > 0 {
		if len(s) <= maxWidth {
			sb.WriteString(s)
			break
		}
		cut := maxWidth
		for cut > 0 && s[cut-1] != ' ' {
			cut--
		}
		if cut == 0 {
			cut = maxWidth
		}
		sb.WriteString(s[:cut])
		s = strings.TrimLeft(s[cut:], " ")
		if len(s) > 0 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// ingestPTYData processes raw PTY bytes into committed lines and live block.
//
// The strategy:
//   - Split on \n first — each \n-terminated chunk is a "paragraph" of output.
//   - Within each paragraph, \r means "overwrite from column 0", i.e. restart
//     drawing the current live block.
//   - The last chunk (after the last \n) is incomplete and stored as livePartial.
//
// For a multi-line interactive prompt like Next.js's option picker, the program
// emits the entire prompt, then on navigation re-emits the entire prompt
// prefixed with \r (or uses cursor-up ANSI codes — which we strip). The result
// after ANSI stripping is just \r + new block text. We detect this by checking
// whether the incoming data starts with \r (or contains \r before any \n),
// which signals "replace the live block".
func (p *ptyScreen) ingestPTYData(raw []byte) {
	s := string(raw)
	i := 0
	for i < len(s) {
		c := s[i]

		if c == '\x1b' {
			i++
			if i >= len(s) {
				break
			}
			if s[i] != '[' {
				i++
				continue
			}
			i++ // skip '['
			params := []int{}
			cur := 0
			hasCur := false
			for i < len(s) {
				ch := s[i]
				if ch >= '0' && ch <= '9' {
					cur = cur*10 + int(ch-'0')
					hasCur = true
					i++
				} else if ch == ';' {
					params = append(params, cur)
					cur = 0
					hasCur = false
					i++
				} else if ch == '?' {
					// skip mode prefix
					i++
				} else {
					break
				}
			}
			if hasCur {
				params = append(params, cur)
			}
			if i >= len(s) {
				break
			}
			final := s[i]
			i++

			param1 := 0
			if len(params) > 0 {
				param1 = params[0]
			}

			switch final {
			case 'A': // cursor up
				n := param1
				if n == 0 {
					n = 1
				}
				p.vscreen.cursor -= n
				if p.vscreen.cursor < 0 {
					p.vscreen.cursor = 0
				}
				p.vscreen.col = 0
			case 'B': // cursor down
				n := param1
				if n == 0 {
					n = 1
				}
				p.vscreen.cursor += n
				p.vscreen.col = 0
			case 'C': // cursor right — ignore
			case 'D': // cursor left / large value = go to col 1
				p.vscreen.col = 0
			case 'G': // cursor to column 1
				p.vscreen.col = 0
			case 'K': // erase line
				p.vscreen.ensureLine(p.vscreen.cursor)
				p.vscreen.lines[p.vscreen.cursor] = ""
				p.vscreen.col = 0
			case 'J': // erase display
				if p.vscreen.cursor < len(p.vscreen.lines) {
					p.vscreen.lines = p.vscreen.lines[:p.vscreen.cursor]
				}
			case 'h', 'l', 'm': // mode/color — ignore
			}
			continue
		}

		switch c {
		case '\r':
			// Carriage return: move to column 0, do NOT erase.
			// Erasing is done by explicit \x1b[K sequences.
			p.vscreen.col = 0
			i++
		case '\n':
			p.vscreen.cursor++
			p.vscreen.col = 0
			i++
		default:
			start := i
			for i < len(s) && s[i] >= 0x20 && s[i] != '\x1b' {
				i++
			}
			p.vscreen.write(s[start:i])
		}
	}

	p.refreshViewport()
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (p ptyScreen) Init() tea.Cmd { return nil }

func (p ptyScreen) Update(msg tea.Msg) (ptyScreen, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if p.completed || p.session == nil {
			return p, nil
		}
		switch msg.Type {
		case tea.KeyEnter:
			_ = p.session.Write([]byte("\r"))
		case tea.KeyBackspace, tea.KeyDelete:
			_ = p.session.Write([]byte{127})
		case tea.KeyCtrlC:
			// Close the session and signal an interrupt — do NOT proceed with steps.
			_ = p.session.Write([]byte{3})
			_ = p.session.Close()
			p.session = nil
			p.interrupted = true
			p.completed = true
			return p, func() tea.Msg { return ptyInterruptMsg{} }
		case tea.KeyCtrlD:
			_ = p.session.Write([]byte{4})
		case tea.KeyUp:
			_ = p.session.Write([]byte("\x1b[A"))
		case tea.KeyDown:
			_ = p.session.Write([]byte("\x1b[B"))
		case tea.KeyLeft:
			_ = p.session.Write([]byte("\x1b[D"))
		case tea.KeyRight:
			_ = p.session.Write([]byte("\x1b[C"))
		case tea.KeySpace:
			_ = p.session.Write([]byte(" "))
		case tea.KeyTab:
			_ = p.session.Write([]byte("\t"))
		case tea.KeyRunes:
			_ = p.session.Write([]byte(string(msg.Runes)))
		}
		return p, pty.CmdRead(p.session)

	case ptyStepStartMsg:
		p.session = msg.session
		p.vscreen.reset()
		p.addLine(fmt.Sprintf("$ %s", msg.cmdStr), lineCmd)
		return p, pty.CmdRead(p.session)

	case ptyBuiltinCompleteMsg:
		return p, p.startNextStep()

	case ptyDoneMsg:
		p.flushPending()
		p.completed = true
		return p, nil

	case pty.OutputMsg:
		if len(msg.Data) > 0 {
			p.ingestPTYData(msg.Data)
		}
		if p.session != nil && !p.completed {
			return p, pty.CmdRead(p.session)
		}

	case pty.ExitMsg:
		if p.session != nil {
			_ = p.session.Close()
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
			_ = p.session.Resize(uint16(vpH(msg.Height)), uint16(vpW(msg.Width)))
		}
	}
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

// ---------------------------------------------------------------------------
// startNextStep
// ---------------------------------------------------------------------------

func (p *ptyScreen) startNextStep() tea.Cmd {
	if !p.isPostSteps && p.currentStepIdx >= len(p.allSteps) {
		if p.tmpl != nil && len(p.tmpl.PostSteps) > 0 {
			p.isPostSteps = true
			p.currentStepIdx = 0
			p.allSteps = p.tmpl.PostSteps
			p.workDir = p.projectPath
			return func() tea.Msg {
				return pty.OutputMsg{Data: []byte("\n── running post-setup steps ──\n")}
			}
		}
	}

	if p.currentStepIdx >= len(p.allSteps) {
		if p.githubRepo.FullName != "" {
			linkMsg := fmt.Sprintf("🔗 Linking to GitHub repository %s...", p.githubRepo.FullName)
			projectPath := p.projectPath
			cloneURL := p.githubRepo.CloneURL
			p.githubRepo = gh.Repo{}
			return func() tea.Msg {
				if err := gh.InitRepoWithRemote(projectPath, cloneURL); err != nil {
					warn := fmt.Sprintf("⚠️  Warning: Could not link to GitHub: %v", err)
					return tea.Batch(
						func() tea.Msg { return pty.OutputMsg{Data: []byte(linkMsg + "\n" + warn)} },
						func() tea.Msg { return ptyDoneMsg{} },
					)()
				}
				return tea.Batch(
					func() tea.Msg {
						return pty.OutputMsg{Data: []byte(linkMsg + "\n✅ Successfully linked to GitHub!")}
					},
					func() tea.Msg { return ptyDoneMsg{} },
				)()
			}
		}
		return func() tea.Msg {
			return tea.Batch(
				func() tea.Msg { return pty.OutputMsg{Data: []byte("✓ Setup completed successfully!")} },
				func() tea.Msg { return ptyDoneMsg{} },
			)()
		}
	}

	step := p.allSteps[p.currentStepIdx]
	p.currentStepIdx++

	switch step.Type {
	case "builtin":
		action := step.Action
		path := tmpl.ExpandVars(step.Path, p.vars)
		projectPath := p.vars.ProjectPath
		statusMsg := fmt.Sprintf("⚙  %s %s", action, path)
		return func() tea.Msg {
			if err := tmpl.ExecuteBuiltin(action, path, projectPath); err != nil {
				return pty.ExitMsg{Err: fmt.Errorf("builtin %s %q: %w", action, path, err)}
			}
			return tea.Batch(
				func() tea.Msg { return pty.OutputMsg{Data: []byte(statusMsg)} },
				func() tea.Msg { return ptyBuiltinCompleteMsg{} },
			)()
		}

	case "command":
		cmdStr := tmpl.ExpandVars(step.Run, p.vars)
		var args []string
		if step.Shell {
			args = []string{"sh", "-c", cmdStr}
		} else {
			args = strings.Fields(cmdStr)
		}
		workDir := p.workDir
		width, height := p.width, p.height
		return func() tea.Msg {
			session, err := pty.NewSession(args, workDir)
			if err != nil {
				return pty.ExitMsg{Err: fmt.Errorf("failed to start command %q: %w", cmdStr, err)}
			}
			if width > 0 && height > 0 {
				_ = session.Resize(uint16(vpH(height)), uint16(vpW(width)))
			}
			return ptyStepStartMsg{session: session, cmdStr: cmdStr}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (p ptyScreen) View() string {
	titleBarStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorWhite).
		Background(colorPurple).
		Padding(0, 2)

	title := "Template Setup"
	if p.tmpl != nil {
		title = fmt.Sprintf("Template Setup: %s", p.tmpl.Name)
	}
	titleBar := titleBarStyle.Render(title)

	progressStyle := lipgloss.NewStyle().Foreground(colorGray).Padding(0, 1)
	var progress string
	if p.completed {
		progress = progressStyle.Render("✓ Complete")
	} else if len(p.allSteps) > 0 {
		total := len(p.allSteps)
		if p.tmpl != nil && !p.isPostSteps {
			total = len(p.tmpl.Steps)
		}
		progress = progressStyle.Render(fmt.Sprintf("Step %d/%d", p.currentStepIdx, total))
	} else {
		progress = progressStyle.Render("Running...")
	}

	innerW := vpW(p.width)
	innerH := vpH(p.height)
	terminalStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple).
		Padding(0, 1).
		Width(innerW).
		Height(innerH)
	terminal := terminalStyle.Render(p.viewport.View())

	var footer string
	if p.completed {
		footer = lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Padding(1, 0).
			Render("Opening workspace...")
	} else {
		footer = lipgloss.NewStyle().Foreground(colorGray).Italic(true).
			Render("Interactive terminal • Type to send input • Ctrl+C to interrupt")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		titleBar, progress, "", terminal, "", footer,
	)
}
