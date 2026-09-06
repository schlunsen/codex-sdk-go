package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

// StreamedTurn is the result of Thread.RunStreamed. Read events from Events
// until it is closed, then call Err to learn how the turn ended.
type StreamedTurn struct {
	events <-chan types.ThreadEvent
	done   chan struct{}
	err    error
	mu     sync.Mutex
}

// Events yields thread events as they are produced. It is closed when the
// turn ends, the process exits, or the context is cancelled.
func (s *StreamedTurn) Events() <-chan types.ThreadEvent { return s.events }

// Err blocks until the stream has finished and returns the terminal error:
// nil on success, *types.ExecError if codex exited non-zero, *types.ParseError
// on malformed output, or the context error on cancellation.
func (s *StreamedTurn) Err() error {
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
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
		cleanup()
		return nil, err
	}

	events := make(chan types.ThreadEvent, 64)
	st := &StreamedTurn{events: events, done: make(chan struct{})}

	go func() {
		defer close(st.done)
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
			}
			select {
			case events <- ev:
			case <-ctx.Done():
				st.setErr(ctx.Err())
				_ = stream.Close()
				return
			}
		}
		// Drain in case we broke early.
		for range stream.Lines() {
		}
		if err := stream.Err(); err != nil {
			st.setErr(err)
		}
	}()

	return st, nil
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
