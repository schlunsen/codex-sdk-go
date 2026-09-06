package codex

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/schlunsen/codex-sdk-go/types"
)

func newTestClient(t *testing.T, opts *types.CodexOptions) *Codex {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake codex script requires a POSIX shell")
	}
	p, err := filepath.Abs(filepath.Join("internal", "transport", "testdata", "fake-codex"))
	if err != nil {
		t.Fatal(err)
	}
	if opts == nil {
		opts = types.NewCodexOptions()
	}
	opts.WithCodexPath(p)
	c, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

type echo struct {
	Args       string `json:"args"`
	Prompt     string `json:"prompt"`
	APIKey     string `json:"api_key"`
	Originator string `json:"originator"`
}

func decodeEcho(t *testing.T, turn *Turn) echo {
	t.Helper()
	var e echo
	if err := json.Unmarshal([]byte(turn.FinalResponse), &e); err != nil {
		t.Fatalf("final response is not echo json: %q: %v", turn.FinalResponse, err)
	}
	return e
}

func TestRunCollectsTurn(t *testing.T) {
	c := newTestClient(t, types.NewCodexOptions().WithAPIKey("k"))
	th := c.StartThread(types.NewThreadOptions().WithModel("m1"))
	if th.ID() != "" {
		t.Fatal("id should be empty before first turn")
	}
	turn, err := th.Run(context.Background(), "hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	if th.ID() != "thread_123" {
		t.Errorf("thread id = %q", th.ID())
	}
	if len(turn.Items) != 4 {
		t.Errorf("expected 4 completed items, got %d", len(turn.Items))
	}
	if turn.Usage == nil || turn.Usage.InputTokens != 10 || turn.Usage.CacheWriteInputTokens != 0 {
		t.Errorf("usage = %+v", turn.Usage)
	}
	e := decodeEcho(t, turn)
	if e.Prompt != "hello" || e.APIKey != "k" || e.Originator != "codex_sdk_go" {
		t.Errorf("echo = %+v", e)
	}
	if !strings.Contains(e.Args, "--model m1") {
		t.Errorf("args = %s", e.Args)
	}

	// Second turn on the same thread resumes it.
	turn, err = th.Run(context.Background(), "again", nil)
	if err != nil {
		t.Fatal(err)
	}
	if e := decodeEcho(t, turn); !strings.HasSuffix(e.Args, "resume thread_123") {
		t.Errorf("second turn should resume: %s", e.Args)
	}
}

func TestResumeThread(t *testing.T) {
	c := newTestClient(t, nil)
	th := c.ResumeThread("thread_xyz", nil)
	turn, err := th.Run(context.Background(), "go", nil)
	if err != nil {
		t.Fatal(err)
	}
	if th.ID() != "thread_xyz" {
		t.Errorf("id = %s", th.ID())
	}
	if e := decodeEcho(t, turn); !strings.Contains(e.Args, "resume thread_xyz") {
		t.Errorf("args = %s", e.Args)
	}
}

func TestRunInputsImagesAndSchema(t *testing.T) {
	c := newTestClient(t, nil)
	th := c.StartThread(nil)
	schema := map[string]any{"type": "object"}
	turn, err := th.RunInputs(context.Background(), []types.UserInput{
		types.TextInput("one"),
		types.LocalImageInput("a.png"),
		types.TextInput("two"),
	}, types.NewTurnOptions().WithOutputSchema(schema))
	if err != nil {
		t.Fatal(err)
	}
	e := decodeEcho(t, turn)
	if e.Prompt != "one\n\ntwo" {
		t.Errorf("prompt = %q", e.Prompt)
	}
	if !strings.Contains(e.Args, "--image a.png") || !strings.Contains(e.Args, "--output-schema ") {
		t.Errorf("args = %s", e.Args)
	}
}

func TestRunEmptyInput(t *testing.T) {
	c := newTestClient(t, nil)
	if _, err := c.StartThread(nil).Run(context.Background(), "", nil); err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestRunTurnFailed(t *testing.T) {
	c := newTestClient(t, nil)
	_, err := c.StartThread(nil).Run(context.Background(), "fail", nil)
	var tf *types.TurnFailedError
	if !errors.As(err, &tf) || tf.Message != "model refused" {
		t.Fatalf("expected TurnFailedError, got %v", err)
	}
}

func TestRunExecError(t *testing.T) {
	c := newTestClient(t, nil)
	_, err := c.StartThread(nil).Run(context.Background(), "crash", nil)
	if !types.IsExecError(err) {
		t.Fatalf("expected ExecError, got %v", err)
	}
}

func TestRunStreamError(t *testing.T) {
	c := newTestClient(t, nil)
	_, err := c.StartThread(nil).Run(context.Background(), "streamerr", nil)
	if !types.IsExecError(err) || !strings.Contains(err.Error(), "stream broke") {
		t.Fatalf("got %v", err)
	}
}

func TestRunParseError(t *testing.T) {
	c := newTestClient(t, nil)
	_, err := c.StartThread(nil).Run(context.Background(), "garbage", nil)
	if !types.IsParseError(err) {
		t.Fatalf("expected ParseError, got %v", err)
	}
}

func TestRunStreamedEvents(t *testing.T) {
	c := newTestClient(t, nil)
	st, err := c.StartThread(nil).RunStreamed(context.Background(), "hi", nil)
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for ev := range st.Events() {
		kinds = append(kinds, ev.EventType())
	}
	if err := st.Err(); err != nil {
		t.Fatal(err)
	}
	want := "thread.started turn.started item.started item.completed item.completed item.completed item.completed turn.completed"
	if got := strings.Join(kinds, " "); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

// startHang starts a streamed turn against the fake CLI's "hang" prompt and
// consumes the first event, leaving the process blocked with the stream open.
func startHang(t *testing.T, ctx context.Context) *StreamedTurn {
	t.Helper()
	st, err := newTestClient(t, nil).StartThread(nil).RunStreamed(ctx, "hang", nil)
	if err != nil {
		t.Fatal(err)
	}
	<-st.Events() // thread.started
	return st
}

// awaitCanceled drains st in the background and asserts it terminates with
// context.Canceled within a bounded time.
func awaitCanceled(t *testing.T, st *StreamedTurn, what string) {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		for range st.Events() {
		}
		done <- st.Err()
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("%s: got %v, want context.Canceled", what, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("%s: stream did not stop", what)
	}
}

func TestRunStreamedCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	st := startHang(t, ctx)
	cancel()
	awaitCanceled(t, st, "after ctx cancel")
}

func TestRunStreamedClose(t *testing.T) {
	// The caller's context is never cancelled; Close alone must stop the turn.
	st := startHang(t, context.Background())
	go func() { _ = st.Close() }()
	awaitCanceled(t, st, "after Close")
	// Close is idempotent once the stream has terminated.
	if err := st.Close(); !errors.Is(err, context.Canceled) {
		t.Fatalf("second Close returned %v", err)
	}
}

func TestRunStreamedCloseAfterTurnCompleted(t *testing.T) {
	// A consumer that stops reading as soon as it sees turn.completed and
	// calls Close must not have the process killed out from under it: the
	// turn succeeded, so Close reports nil, same as Err.
	c := newTestClient(t, nil)
	st, err := c.StartThread(nil).RunStreamed(context.Background(), "hi", nil)
	if err != nil {
		t.Fatal(err)
	}
	for ev := range st.Events() {
		if _, ok := ev.(*types.TurnCompletedEvent); ok {
			break
		}
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close after turn.completed returned %v, want nil", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("second Close returned %v, want nil", err)
	}
}

func TestStreamedTurnZeroValueClose(t *testing.T) {
	var st StreamedTurn
	if err := st.Close(); err == nil {
		t.Fatal("Close on zero-value StreamedTurn should return an error, not panic or block")
	}
}

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Fatal("Version is empty")
	}
}
