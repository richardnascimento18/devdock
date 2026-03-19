package pty

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
)

// Session represents an active pseudo-terminal session.
type Session struct {
	cmd    *exec.Cmd
	ptmx   *os.File
	done   chan error
	output []byte
	mu     sync.Mutex
}

// OutputMsg is sent when new output is available from the PTY.
type OutputMsg struct {
	Data []byte
}

// ExitMsg is sent when the PTY process exits.
type ExitMsg struct {
	Err error
}

func isTerminalEOF(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF {
		return true
	}
	var errno syscall.Errno
	if errors.As(err, &errno) && errno == syscall.EIO {
		return true
	}
	if errors.Is(err, os.ErrClosed) {
		return true
	}
	return false
}

// NewSession creates and starts a PTY session for the given command and arguments.
func NewSession(args []string, workDir string) (*Session, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no command provided")
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 40, Cols: 120})
	s := &Session{
		cmd:    cmd,
		ptmx:   ptmx,
		done:   make(chan error, 1),
		output: make([]byte, 0),
	}
	go func() { s.done <- s.cmd.Wait() }()
	return s, nil
}

// CmdRead returns a tea.Cmd that reads one chunk from the PTY.
func CmdRead(s *Session) tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 4096)
		n, err := s.ptmx.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			s.mu.Lock()
			s.output = append(s.output, data...)
			s.mu.Unlock()
			return OutputMsg{Data: data}
		}
		if err != nil {
			if isTerminalEOF(err) {
				select {
				case <-s.done:
				default:
				}
				return ExitMsg{Err: nil}
			}
			return ExitMsg{Err: err}
		}
		return OutputMsg{Data: nil}
	}
}

func (s *Session) Write(input []byte) error {
	_, err := s.ptmx.Write(input)
	return err
}

func (s *Session) Resize(rows, cols uint16) error {
	return pty.Setsize(s.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

func (s *Session) Close() error {
	if s.ptmx != nil {
		return s.ptmx.Close()
	}
	return nil
}

func (s *Session) Output() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return string(s.output)
}
