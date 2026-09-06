// Package transport spawns the codex CLI and streams its JSONL output.
package transport

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/schlunsen/codex-sdk-go/internal/log"
	"github.com/schlunsen/codex-sdk-go/types"
)

const (
	// OriginatorEnv is the environment variable codex uses to identify the
	// client that spawned it.
	OriginatorEnv = "CODEX_INTERNAL_ORIGINATOR_OVERRIDE"
	// Originator identifies this SDK to the codex CLI.
	Originator = "codex_sdk_go"
	// APIKeyEnv is the environment variable codex reads its API key from.
	APIKeyEnv = "CODEX_API_KEY" //nolint:gosec // G101: this is the name of an env var, not a credential

	// maxLineSize bounds a single JSONL line. Agent messages can be large
	// (aggregated command output, big diffs), so allow up to 64 MiB.
	maxLineSize = 64 * 1024 * 1024
)

// Exec spawns `codex exec` processes.
type Exec struct {
	ExecutablePath  string
	Env             map[string]string // nil = inherit
	Config          types.ConfigObject
	ConfigOverrides []string
	Logger          *log.Logger
}

// NewExec creates an Exec for the given executable. If executablePath is
// empty the codex binary is located via FindCLI.
func NewExec(executablePath string, opts *types.CodexOptions) (*Exec, error) {
	if opts == nil {
		opts = &types.CodexOptions{}
	}
	if executablePath == "" {
		var err error
		executablePath, err = FindCLI()
		if err != nil {
			return nil, err
		}
	}
	return &Exec{
		ExecutablePath:  executablePath,
		Env:             opts.Env,
		Config:          opts.Config,
		ConfigOverrides: opts.ConfigOverrides,
		Logger:          log.NewLogger(opts.Verbose),
	}, nil
}

// Stream is a running codex exec process. Read lines from Lines until it is
// closed, then check Err for the process exit status.
type Stream struct {
	lines chan string
	done  chan struct{}
	err   error
	mu    sync.Mutex

	cmd    *exec.Cmd
	cancel context.CancelFunc
}

// Lines returns a channel that yields one JSONL line per message. It is
// closed once the process exits or the context is cancelled.
func (s *Stream) Lines() <-chan string { return s.lines }

// Err returns the terminal error of the stream. It blocks until the process
// has exited. It returns nil on a clean exit, a *types.ExecError on a
// non-zero exit, or the context error on cancellation.
func (s *Stream) Err() error {
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Close terminates the process if it is still running and waits for cleanup.
func (s *Stream) Close() error {
	s.cancel()
	return s.Err()
}

func (s *Stream) setErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err == nil {
		s.err = err
	}
}

// Run starts a codex exec process for args and returns a Stream of JSONL lines.
func (e *Exec) Run(ctx context.Context, args RunArgs) (*Stream, error) {
	cmdArgs, err := BuildArgs(e.Config, e.ConfigOverrides, args)
	if err != nil {
		return nil, err
	}

	env := e.buildEnv(args.APIKey)

	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, e.ExecutablePath, cmdArgs...)
	cmd.Env = env
	if args.Thread != nil && args.Thread.WorkingDirectory != "" {
		// codex receives --cd, but also start the process there so relative
		// paths in --image / --output-schema behave consistently.
		if info, statErr := os.Stat(args.Thread.WorkingDirectory); statErr == nil && info.IsDir() {
			cmd.Dir = args.Thread.WorkingDirectory
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	// If the process is killed (context cancelled) bound how long Wait blocks
	// on lingering pipe holders.
	cmd.WaitDelay = 2 * time.Second

	e.Logger.Debugf("spawning %s %s", e.ExecutablePath, strings.Join(cmdArgs, " "))

	if err := cmd.Start(); err != nil {
		cancel()
		if errors.Is(err, exec.ErrNotFound) || os.IsNotExist(err) {
			return nil, types.NewCLINotFoundError(fmt.Sprintf("codex executable not found at %s", e.ExecutablePath))
		}
		return nil, fmt.Errorf("failed to start codex: %w", err)
	}

	s := &Stream{
		lines:  make(chan string, 64),
		done:   make(chan struct{}),
		cmd:    cmd,
		cancel: cancel,
	}

	// Write the prompt and close stdin so codex knows input is complete.
	go func() {
		defer func() { _ = stdin.Close() }()
		if _, werr := io.WriteString(stdin, args.Input); werr != nil {
			e.Logger.Debugf("stdin write error: %v", werr)
		}
	}()

	go e.pump(ctx, s, stdout, &stderr)

	return s, nil
}

func (e *Exec) pump(ctx context.Context, s *Stream, stdout io.ReadCloser, stderr *bytes.Buffer) {
	defer close(s.done)
	defer s.cancel()

	// Scan stdout in its own goroutine so that a cancelled context can
	// unblock the read by closing the pipe. This matters when codex spawns
	// grandchildren that keep the pipe open after codex itself is killed.
	scanDone := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				continue
			}
			e.Logger.Debugf("<- %s", truncate(line, 300))
			select {
			case s.lines <- line:
			case <-ctx.Done():
				scanDone <- ctx.Err()
				return
			}
		}
		scanDone <- scanner.Err()
	}()

	var scanErr error
	select {
	case scanErr = <-scanDone:
	case <-ctx.Done():
		_ = stdout.Close()
		<-scanDone
		scanErr = ctx.Err()
	}
	close(s.lines)

	waitErr := s.cmd.Wait()

	// All output was read to EOF and the process exited 0: that is a
	// successful run even if the context was cancelled in the meantime (e.g.
	// a caller closing the stream right after the final event).
	if scanErr == nil && waitErr == nil {
		return
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		s.setErr(ctxErr)
		return
	}
	if scanErr != nil {
		s.setErr(fmt.Errorf("failed to read codex output: %w", scanErr))
		return
	}
	if waitErr != nil {
		execErr := &types.ExecError{ExitCode: -1, Stderr: stderr.String(), Err: waitErr}
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			execErr.ExitCode = exitErr.ExitCode()
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				execErr.Signal = ws.Signal().String()
			}
		}
		s.setErr(execErr)
	}
}

// buildEnv assembles the process environment following the TypeScript SDK's
// rules: inherit the current environment unless an explicit Env is given,
// always set the originator, and inject the API key when provided.
func (e *Exec) buildEnv(apiKey string) []string {
	env := map[string]string{}
	if e.Env != nil {
		for k, v := range e.Env {
			env[k] = v
		}
	} else {
		for _, kv := range os.Environ() {
			if i := strings.IndexByte(kv, '='); i > 0 {
				env[kv[:i]] = kv[i+1:]
			}
		}
	}
	if env[OriginatorEnv] == "" {
		env[OriginatorEnv] = Originator
	}
	if apiKey != "" {
		env[APIKeyEnv] = apiKey
	}

	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+env[k])
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
