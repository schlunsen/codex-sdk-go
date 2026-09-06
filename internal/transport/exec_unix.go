//go:build unix

package transport

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// interruptGrace is how long codex gets to shut down after SIGINT before the
// whole process group is SIGKILLed. Codex needs a few hundred milliseconds to
// abort the turn and kill the shell command it is running.
const interruptGrace = time.Second

// configureProcessGroup starts codex in its own process group and makes
// context cancellation interrupt that group.
//
// SIGINT rather than SIGKILL, because codex runs each shell command in a
// process group of its own: a kill of codex's group can never reach the
// command, but on SIGINT codex aborts the turn and kills the command itself.
// (On SIGTERM it exits without doing so.) superviseProcessGroup escalates to
// SIGKILL if codex does not exit within interruptGrace.
func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
}

// superviseProcessGroup keeps signalling the process group after ctx is
// cancelled until done is closed (i.e. cmd.Wait has returned): SIGINT until
// interruptGrace has elapsed, SIGKILL afterwards.
//
// Re-signalling matters even in the happy path: a grandchild that codex was
// fork()ing at the instant of the first signal can be linked into the group
// after the kernel enumerated it and survive, holding the pipes open.
// Signalling an empty or zombie-only group is harmless.
func superviseProcessGroup(ctx context.Context, cmd *exec.Cmd, done <-chan struct{}) {
	select {
	case <-done:
		return
	case <-ctx.Done():
	}
	pgid := cmd.Process.Pid
	deadline := time.Now().Add(interruptGrace)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			sig := syscall.SIGINT
			if time.Now().After(deadline) {
				sig = syscall.SIGKILL
			}
			_ = syscall.Kill(-pgid, sig)
		}
	}
}
