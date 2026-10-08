// Package process owns the lifecycle policy for non-PTY child commands.
package process

import (
	"context"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

func Command(ctx context.Context, directory, executable string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = directory
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 2 * time.Second
	return cmd
}

// Output keeps a bounded tail. Error reporting must not echo command arguments
// (which may include credentials); presentation separately encodes controls.
func Output(ctx context.Context, directory, executable string, args ...string) ([]byte, error) {
	cmd := Command(ctx, directory, executable, args...)
	output := &tailBuffer{}
	cmd.Stdout, cmd.Stderr = output, output
	err := cmd.Run()
	return output.data, err
}

type tailBuffer struct {
	mu   sync.Mutex
	data []byte
}

func (b *tailBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(data)
	const limit = 64 * 1024
	if len(data) >= limit {
		b.data = append(b.data[:0], data[len(data)-limit:]...)
	} else {
		if extra := len(b.data) + len(data) - limit; extra > 0 {
			b.data = b.data[extra:]
		}
		b.data = append(b.data, data...)
	}
	return n, nil
}
