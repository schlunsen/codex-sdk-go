//go:build live

// Package livetest exercises the SDK against a real Codex CLI and the live
// service. It is excluded from the default build; run it with
// `make test-live` (needs `codex login` or CODEX_API_KEY).
package livetest

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	codex "github.com/schlunsen/codex-sdk-go"
	"github.com/schlunsen/codex-sdk-go/types"
)

const turnTimeout = 3 * time.Minute

func newClient(t *testing.T) *codex.Codex {
	t.Helper()
	c, err := codex.New(nil)
	if err != nil {
		if types.IsCLINotFoundError(err) {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	t.Logf("codex: %s", c.ExecutablePath())
	return c
}

func readOnlyThread() *types.ThreadOptions {
	return types.NewThreadOptions().
		WithSandboxMode(types.SandboxReadOnly).
		WithApprovalPolicy(types.ApprovalNever).
		WithSkipGitRepoCheck(true)
}

// TestRunAndResume runs one turn, then resumes the thread by id from a
// fresh Thread and checks the agent remembers the first turn.
func TestRunAndResume(t *testing.T) {
	c := newClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*turnTimeout)
	defer cancel()

	th := c.StartThread(readOnlyThread())
	turn, err := th.Run(ctx, "Pick one of these words and reply with only that word: apple, river, copper.", nil)
	if err != nil {
		t.Fatal(err)
	}
	word := strings.ToLower(strings.TrimSpace(turn.FinalResponse))
	t.Logf("turn 1: %q, %d items, usage=%+v", word, len(turn.Items), turn.Usage)
	if turn.Usage == nil {
		t.Error("turn.completed usage missing")
	}
	if th.ID() == "" {
		t.Fatal("thread id not populated")
	}
	if word != "apple" && word != "river" && word != "copper" {
		t.Fatalf("unexpected answer %q", turn.FinalResponse)
	}

	resumed := c.ResumeThread(th.ID(), readOnlyThread())
	turn, err = resumed.Run(ctx, "Which word did you pick? Reply with only that word.", nil)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.ToLower(strings.TrimSpace(turn.FinalResponse))
	t.Logf("resumed: %q", got)
	if !strings.Contains(got, word) {
		t.Fatalf("resumed thread answered %q, want %q", turn.FinalResponse, word)
	}
}

// TestCloseKillsRunningCommand starts a long shell command through the agent,
// closes the stream mid-command, and checks both that Close reports ErrClosed
// promptly and that the command did not outlive the turn.
func TestCloseKillsRunningCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on pgrep and Unix process groups")
	}
	c := newClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), turnTimeout)
	defer cancel()

	const marker = "sleep 37"
	th := c.StartThread(readOnlyThread())
	st, err := th.RunStreamed(ctx, "Run the shell command `"+marker+"` and wait for it to finish, then reply done.", nil)
	if err != nil {
		t.Fatal(err)
	}

	started := false
	for ev := range st.Events() {
		is, ok := ev.(*types.ItemStartedEvent)
		if !ok {
			continue
		}
		if _, ok := is.Item.(*types.CommandExecutionItem); ok {
			started = true
			break
		}
	}
	if !started {
		t.Fatalf("agent never started a command: %v", st.Err())
	}
	if !waitFor(5*time.Second, func() bool { return processRunning(marker) }) {
		t.Fatalf("%q did not start within 5s", marker)
	}

	t0 := time.Now()
	err = st.Close()
	took := time.Since(t0)
	t.Logf("Close() = %v in %s", err, took.Round(time.Millisecond))
	if !errors.Is(err, codex.ErrClosed) {
		t.Fatalf("Close returned %v, want ErrClosed", err)
	}
	if took > 3*time.Second {
		t.Errorf("Close took %s; expected codex to exit within its SIGINT grace", took)
	}
	if !waitFor(3*time.Second, func() bool { return !processRunning(marker) }) {
		_ = exec.Command("pkill", "-f", marker).Run()
		t.Fatalf("%q survived Close", marker)
	}
}

func processRunning(pattern string) bool {
	return exec.Command("pgrep", "-f", "^"+pattern).Run() == nil
}

func waitFor(d time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(d)
	for {
		if cond() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
}
