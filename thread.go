package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/schlunsen/codex-sdk-go/internal/transport"
	"github.com/schlunsen/codex-sdk-go/types"
)

// Turn is a completed turn of a thread.
type Turn struct {
	// Items are all items that completed during the turn, in order.
	Items []types.ThreadItem
	// FinalResponse is the text of the last agent_message item (JSON when
	// structured output was requested).
	FinalResponse string
	// Usage is the token usage reported for the turn; nil if not reported.
	Usage *types.Usage
}

// ErrClosed is returned by StreamedTurn.Err and StreamedTurn.Close when Close
// stopped a turn that was still in progress. It is distinct from the context
// errors so callers can tell a deliberate close from an upstream cancellation.
var ErrClosed = errors.New("codex: streamed turn closed")

// StreamedTurn is the result of Thread.RunStreamed. Read events from Events
// until it is closed, then call Err to learn how the turn ended.
type StreamedTurn struct {
	events <-chan types.ThreadEvent
	done   chan struct{}
	cancel context.CancelFunc
	// turnEnded is set once turn.completed or turn.failed has been delivered,
	// so Close can let codex finish persisting the session instead of killing it.
	turnEnded atomic.Bool
	err       error
	mu        sync.Mutex
}

// Events yields thread events as they are produced. It is closed when the
// turn ends, the process exits, the context is cancelled, or Close is called.
func (s *StreamedTurn) Events() <-chan types.ThreadEvent { return s.events }

// Err blocks until the stream has finished and returns the terminal error:
// nil on success, *types.ExecError if codex exited non-zero, *types.ParseError
// on malformed output, the context error on cancellation, or ErrClosed if
// Close stopped the turn.
func (s *StreamedTurn) Err() error {
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Close releases the turn and waits for cleanup. If the turn is still in
// progress the codex process (and, on Unix, its whole process group) is
// killed and Close returns ErrClosed.
// If turn.completed or turn.failed has already been delivered, codex is left
// to exit on its own (so the session is persisted) and Close returns the
// turn's terminal error, exactly like Err. Close is safe to call more than
// once and from a goroutine other than the one reading Events.
func (s *StreamedTurn) Close() error {
	if s.cancel == nil {
		return fmt.Errorf("codex: Close on zero-value StreamedTurn")
	}
	if !s.turnEnded.Load() {
		s.cancel()
	}
	return s.Err()
}

func (s *StreamedTurn) setErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err == nil {
		s.err = err
	}
}

// Thread represents a conversation with the agent. One thread can have
// multiple consecutive turns. A Thread is safe to reuse sequentially; do not
// run turns on the same Thread concurrently.
type Thread struct {
	exec          *transport.Exec
	options       *types.CodexOptions
	threadOptions *types.ThreadOptions

	mu sync.RWMutex
	id string
}

// ID returns the thread id. It is empty until the first turn has started.
func (t *Thread) ID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.id
}

func (t *Thread) setID(id string) {
	t.mu.Lock()
	t.id = id
	t.mu.Unlock()
}

// Run sends a text prompt to the agent and returns the completed turn.
func (t *Thread) Run(ctx context.Context, prompt string, turnOptions *types.TurnOptions) (*Turn, error) {
	return t.RunInputs(ctx, []types.UserInput{types.TextInput(prompt)}, turnOptions)
}

// RunInputs sends structured input (text and local images) to the agent and
// returns the completed turn.
func (t *Thread) RunInputs(ctx context.Context, inputs []types.UserInput, turnOptions *types.TurnOptions) (*Turn, error) {
	stream, err := t.RunStreamedInputs(ctx, inputs, turnOptions)
	if err != nil {
		return nil, err
	}

	turn := &Turn{}
	var failure *types.ThreadError
	var streamErr *types.ThreadErrorEvent

	for ev := range stream.Events() {
		switch e := ev.(type) {
		case *types.ItemCompletedEvent:
			if msg, ok := e.Item.(*types.AgentMessageItem); ok {
				turn.FinalResponse = msg.Text
			}
			turn.Items = append(turn.Items, e.Item)
		case *types.TurnCompletedEvent:
			u := e.Usage
			turn.Usage = &u
		case *types.TurnFailedEvent:
			failure = &e.Error
		case *types.ThreadErrorEvent:
			streamErr = e
		}
		if failure != nil {
			break
		}
	}

	// Drain and wait for the process regardless of how we left the loop.
	for range stream.Events() {
	}
	err = stream.Err()

	if failure != nil {
		return nil, &types.TurnFailedError{Message: failure.Message}
	}
	if err != nil {
		if streamErr != nil {
			return nil, fmt.Errorf("%w (stream error: %s)", err, streamErr.Message)
		}
		return nil, err
	}
	if streamErr != nil {
		return nil, &types.ThreadStreamError{Message: streamErr.Message}
	}
	if turn.Usage == nil && ctx.Err() != nil {
		// No turn.completed arrived and the context is done: codex was
		// interrupted, even if it happened to exit cleanly.
		return nil, ctx.Err()
	}
	return turn, nil
}

// RunStreamed sends a text prompt to the agent and streams events as they are
// produced.
func (t *Thread) RunStreamed(ctx context.Context, prompt string, turnOptions *types.TurnOptions) (*StreamedTurn, error) {
	return t.RunStreamedInputs(ctx, []types.UserInput{types.TextInput(prompt)}, turnOptions)
}

// RunStreamedInputs sends structured input to the agent and streams events as
// they are produced.
func (t *Thread) RunStreamedInputs(ctx context.Context, inputs []types.UserInput, turnOptions *types.TurnOptions) (*StreamedTurn, error) {
	if turnOptions == nil {
		turnOptions = types.NewTurnOptions()
	}
	prompt, images := normalizeInputs(inputs)
	if prompt == "" && len(images) == 0 {
		return nil, fmt.Errorf("input cannot be empty")
	}

	schemaPath, cleanup, err := createOutputSchemaFile(turnOptions.OutputSchema)
	if err != nil {
		return nil, err
	}

	// Own a cancel here rather than delegating Close to transport.Stream.Close:
	// the latter only closes stream.Lines(), which cannot unblock the
	// `events <- ev` send below when the consumer has stopped reading. The
	// deferred cancel also releases this child from a long-lived parent ctx.
	parent := ctx
	ctx, cancel := context.WithCancel(ctx)

	stream, err := t.exec.Run(ctx, transport.RunArgs{
		Input:            prompt,
		ThreadID:         t.ID(),
		Images:           images,
		OutputSchemaFile: schemaPath,
		BaseURL:          t.options.BaseURL,
		APIKey:           t.options.APIKey,
		Thread:           t.threadOptions,
	})
	if err != nil {
		cancel()
		cleanup()
		return nil, err
	}

	events := make(chan types.ThreadEvent, 64)
	st := &StreamedTurn{events: events, done: make(chan struct{}), cancel: cancel}

	go func() {
		defer close(st.done)
		defer cancel()
		defer cleanup()
		defer close(events)

		for line := range stream.Lines() {
			ev, perr := types.ParseThreadEvent([]byte(line))
			if perr != nil {
				st.setErr(perr)
				_ = stream.Close()
				break
			}
			switch e := ev.(type) {
			case *types.ThreadStartedEvent:
				t.setID(e.ThreadID)
			case *types.TurnCompletedEvent, *types.TurnFailedEvent:
				// Mark before delivering so a consumer that calls Close as
				// soon as it sees this event never races the flag.
				st.turnEnded.Store(true)
			}
			select {
			case events <- ev:
			case <-ctx.Done():
				st.setErr(closedErr(parent, ctx.Err()))
				_ = stream.Close()
				return
			}
		}
		// Drain in case we broke early.
		for range stream.Lines() {
		}
		if err := stream.Err(); err != nil {
			st.setErr(closedErr(parent, err))
		}
	}()

	return st, nil
}

// closedErr maps a cancellation of the internal context to ErrClosed when the
// caller's own context is still live, i.e. when Close (not the caller) stopped
// the turn. Any other error is returned unchanged.
func closedErr(parent context.Context, err error) error {
	if errors.Is(err, context.Canceled) && parent.Err() == nil {
		return ErrClosed
	}
	return err
}

func normalizeInputs(inputs []types.UserInput) (string, []string) {
	var parts []string
	var images []string
	for _, in := range inputs {
		switch in.Type {
		case types.UserInputText:
			parts = append(parts, in.Text)
		case types.UserInputLocalImage:
			images = append(images, in.Path)
		}
	}
	return strings.Join(parts, "\n\n"), images
}

// createOutputSchemaFile writes schema to a temp file and returns its path
// with a cleanup func. A nil schema yields an empty path and a no-op cleanup.
func createOutputSchemaFile(schema map[string]any) (string, func(), error) {
	if schema == nil {
		return "", func() {}, nil
	}
	dir, err := os.MkdirTemp("", "codex-output-schema-")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create schema temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	data, err := json.Marshal(schema)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to marshal output schema: %w", err)
	}
	path := filepath.Join(dir, "schema.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to write output schema: %w", err)
	}
	return path, cleanup, nil
}
