package transport

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/schlunsen/codex-sdk-go/types"
)

func fakeCodex(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake codex script requires a POSIX shell")
	}
	p, err := filepath.Abs(filepath.Join("testdata", "fake-codex"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func collect(t *testing.T, s *Stream) []string {
	t.Helper()
	var lines []string
	for l := range s.Lines() {
		lines = append(lines, l)
	}
	return lines
}

func TestRunSuccess(t *testing.T) {
	ex, err := NewExec(fakeCodex(t), types.NewCodexOptions().WithAPIKey("sk-test"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := ex.Run(context.Background(), RunArgs{Input: "hello", APIKey: "sk-test"})
	if err != nil {
		t.Fatal(err)
	}
	lines := collect(t, s)
	if err := s.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 8 {
		t.Fatalf("expected 8 lines, got %d: %v", len(lines), lines)
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{`\"prompt\": \"hello\"`, `\"originator\": \"codex_sdk_go\"`, `\"api_key\": \"sk-test\"`, `\"args\": \"exec --experimental-json\"`} {
		if !strings.Contains(joined, want) {
			t.Errorf("output missing %s:\n%s", want, joined)
		}
	}
}

func TestRunExplicitEnvNotInherited(t *testing.T) {
	t.Setenv("CODEX_API_KEY", "from-parent")
	ex, err := NewExec(fakeCodex(t), types.NewCodexOptions().WithEnv(map[string]string{"PATH": os.Getenv("PATH")}))
	if err != nil {
		t.Fatal(err)
	}
	s, err := ex.Run(context.Background(), RunArgs{Input: "x"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(collect(t, s), "\n")
	if err := s.Err(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(joined, `\"api_key\": \"\"`) {
		t.Errorf("api key leaked from parent env:\n%s", joined)
	}
}

func TestRunNonZeroExit(t *testing.T) {
	ex, _ := NewExec(fakeCodex(t), nil)
	s, err := ex.Run(context.Background(), RunArgs{Input: "crash"})
	if err != nil {
		t.Fatal(err)
	}
	collect(t, s)
	err = s.Err()
	var execErr *types.ExecError
	if !errors.As(err, &execErr) {
		t.Fatalf("expected ExecError, got %v", err)
	}
	if execErr.ExitCode != 2 || !strings.Contains(execErr.Stderr, "boom") {
		t.Fatalf("unexpected exec error: %+v", execErr)
	}
}

func TestRunCancellation(t *testing.T) {
	ex, _ := NewExec(fakeCodex(t), nil)
	ctx, cancel := context.WithCancel(context.Background())
	s, err := ex.Run(ctx, RunArgs{Input: "hang"})
	if err != nil {
		t.Fatal(err)
	}
	<-s.Lines() // thread.started
	cancel()
	done := make(chan error, 1)
	go func() { collect(t, s); done <- s.Err() }()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stream did not terminate after cancel")
	}
}

func TestRunCloseKillsProcess(t *testing.T) {
	ex, _ := NewExec(fakeCodex(t), nil)
	s, err := ex.Run(context.Background(), RunArgs{Input: "hang"})
	if err != nil {
		t.Fatal(err)
	}
	<-s.Lines()
	done := make(chan struct{})
	go func() { collect(t, s); close(done) }()
	_ = s.Close()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not terminate the stream")
	}
}

func TestNewExecNotFound(t *testing.T) {
	_, err := NewExec("/nonexistent/codex-binary", nil)
	if err != nil {
		t.Fatalf("NewExec should not stat explicit path: %v", err)
	}
	ex := &Exec{ExecutablePath: "/nonexistent/codex-binary"}
	_, err = ex.Run(context.Background(), RunArgs{Input: "x"})
	if !types.IsCLINotFoundError(err) {
		t.Fatalf("expected CLINotFoundError, got %v", err)
	}
}

func TestFindCLIEnvOverride(t *testing.T) {
	p := fakeCodex(t)
	t.Setenv("CODEX_PATH", p)
	got, err := FindCLI()
	if err != nil || got != p {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestFindCLINotFound(t *testing.T) {
	t.Setenv("CODEX_PATH", "")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	if runtime.GOOS != "windows" {
		if isFile("/usr/local/bin/codex") || isFile("/opt/homebrew/bin/codex") {
			t.Skip("codex installed system-wide")
		}
	}
	_, err := FindCLI()
	if !types.IsCLINotFoundError(err) {
		t.Fatalf("expected CLINotFoundError, got %v", err)
	}
}
