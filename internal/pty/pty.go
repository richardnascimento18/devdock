package pty

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	terminal "github.com/creack/pty"
)

// Session owns one reader and one waiter. Output is delivered before completion.
type Session struct {
	cmd       *exec.Cmd
	ptmx      *os.File
	events    chan tea.Msg
	waited    chan struct{}
	waitErr   error // published by closing waited
	closed    chan struct{}
	closeOnce sync.Once
	closeErr  error
}

type OutputMsg struct {
	Session *Session
	Data    []byte
}
type ExitMsg struct {
	Session *Session
	Err     error
}

func isTerminalEOF(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, syscall.EIO) || errors.Is(err, os.ErrClosed)
}

func NewSession(args []string, workDir string) (*Session, error) {
	return NewSessionContext(context.Background(), args, workDir)
}

func NewSessionContext(ctx context.Context, args []string, workDir string) (*Session, error) {
	if len(args) == 0 || args[0] == "" {
		return nil, fmt.Errorf("no command provided")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	ptmx, err := terminal.StartWithSize(cmd, &terminal.Winsize{Rows: 40, Cols: 120})
	if err != nil {
		return nil, fmt.Errorf("start PTY: %w", err)
	}
	s := &Session{cmd: cmd, ptmx: ptmx, events: make(chan tea.Msg, 16), waited: make(chan struct{}), closed: make(chan struct{})}
	go func() { s.waitErr = cmd.Wait(); close(s.waited) }()
	go s.readOutput()
	return s, nil
}

func (s *Session) send(msg tea.Msg) bool {
	select {
	case s.events <- msg:
		return true
	case <-s.closed:
		return false
	}
}
func (s *Session) readOutput() {
	defer close(s.events)
	buf := make([]byte, 4096)
	for {
		n, err := s.ptmx.Read(buf)
		if n > 0 && !s.send(OutputMsg{Session: s, Data: append([]byte(nil), buf[:n]...)}) {
			return
		}
		if err != nil {
			if !isTerminalEOF(err) {
				err = errors.Join(err, s.Close())
			} // terminate child after a terminal read failure
			<-s.waited
			if isTerminalEOF(err) {
				err = nil
			}
			s.send(ExitMsg{Session: s, Err: errors.Join(s.waitErr, err)})
			return
		}
	}
}

// CmdRead consumes the next event, never starts another terminal reader.
func CmdRead(s *Session) tea.Cmd            { return func() tea.Msg { return <-s.events } }
func (s *Session) Write(input []byte) error { _, err := s.ptmx.Write(input); return err }
func (s *Session) Resize(rows, cols uint16) error {
	return terminal.Setsize(s.ptmx, &terminal.Winsize{Rows: rows, Cols: cols})
}

// Close cancels event delivery, closes the terminal and kills the owned process
// group when still running; cmd.Wait always reaps the child.
func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		close(s.closed)
		select {
		case <-s.waited:
		default:
			if err := syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				s.closeErr = err
			}
		}
		err := s.ptmx.Close()
		if !errors.Is(err, os.ErrClosed) {
			s.closeErr = errors.Join(s.closeErr, err)
		}
	})
	return s.closeErr
}
