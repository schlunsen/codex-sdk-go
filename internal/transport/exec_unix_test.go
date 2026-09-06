//go:build unix

package transport

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// runHangAndCancel starts the fake with prompt, waits until its grandchild
// `sleep 60` exists, cancels, and returns the grandchild pid plus how long
// cancellation took to be fully reaped.
func runHangAndCancel(t *testing.T, prompt string) (int, time.Duration) {
	t.Helper()
	pidFile := filepath.Join(t.TempDir(), "sleep.pid")
	t.Setenv("FAKE_PIDFILE", pidFile)

	ex, _ := NewExec(fakeCodex(t), nil)
	ctx, cancel := context.WithCancel(context.Background())
	s, err := ex.Run(ctx, RunArgs{Input: prompt})
	if err != nil {
		t.Fatal(err)
	}
	<-s.Lines() // thread.started

	var pid int
	deadline := time.Now().Add(5 * time.Second)
	for pid == 0 {
		if data, rerr := os.ReadFile(pidFile); rerr == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(data)))
		}
		if pid == 0 {
			if time.Now().After(deadline) {
				t.Fatal("fake codex never wrote its grandchild pid")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("grandchild %d should be alive before cancel: %v", pid, err)
	}

	start := time.Now()
	cancel()
	collect(t, s)
	if err := s.Err(); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	return pid, time.Since(start)
}

// assertGone fails unless pid is dead (allowing a moment for init to reap it).
func assertGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGKILL) // don't leak it on failure
			t.Fatalf("grandchild %d still alive after cancel (kill -0: %v)", pid, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestRunCancellationKillsProcessGroup verifies that cancelling the context
// interrupts grandchildren codex spawned, not just codex itself, and that
// Wait therefore returns promptly instead of waiting out WaitDelay.
func TestRunCancellationKillsProcessGroup(t *testing.T) {
	pid, elapsed := runHangAndCancel(t, "hang")
	if elapsed >= interruptGrace {
		t.Errorf("cancellation took %s: SIGINT should have stopped the group well within the %s grace", elapsed, interruptGrace)
	}
	assertGone(t, pid)
}

// TestRunCancellationEscalatesToKill verifies that a group ignoring SIGINT is
// SIGKILLed after interruptGrace, still without waiting out WaitDelay.
func TestRunCancellationEscalatesToKill(t *testing.T) {
	pid, elapsed := runHangAndCancel(t, "hangint")
	if elapsed < interruptGrace {
		t.Errorf("cancellation took %s, but the fake ignores SIGINT so it cannot have stopped before the %s grace", elapsed, interruptGrace)
	}
	if elapsed >= interruptGrace+2*time.Second {
		t.Errorf("cancellation took %s: SIGKILL escalation should not wait out WaitDelay", elapsed)
	}
	assertGone(t, pid)
}
