package types

import (
	"encoding/json"
	"fmt"
)

// Event type discriminators as emitted by codex exec.
const (
	EventTypeThreadStarted = "thread.started"
	EventTypeTurnStarted   = "turn.started"
	EventTypeTurnCompleted = "turn.completed"
	EventTypeTurnFailed    = "turn.failed"
	EventTypeItemStarted   = "item.started"
	EventTypeItemUpdated   = "item.updated"
	EventTypeItemCompleted = "item.completed"
	EventTypeError         = "error"
)

// ThreadEvent is implemented by every top-level JSONL event emitted by
// codex exec during a turn.
type ThreadEvent interface {
	// EventType returns the type discriminator (e.g. "item.completed").
	EventType() string
}

// Usage describes the tokens used during a turn.
type Usage struct {
	// InputTokens is the number of input tokens used during the turn.
	InputTokens int64 `json:"input_tokens"`
	// CachedInputTokens is the number of cached input tokens used during the turn.
	CachedInputTokens int64 `json:"cached_input_tokens"`
	// CacheWriteInputTokens is the number of input tokens written to the prompt cache.
	CacheWriteInputTokens int64 `json:"cache_write_input_tokens"`
	// OutputTokens is the number of output tokens used during the turn.
	OutputTokens int64 `json:"output_tokens"`
	// ReasoningOutputTokens is the number of reasoning output tokens used during the turn.
	ReasoningOutputTokens int64 `json:"reasoning_output_tokens"`
}

// ThreadError is a fatal error emitted by the stream.
type ThreadError struct {
	Message string `json:"message"`
}

// ThreadStartedEvent is emitted when a new thread is started, as the first event.
type ThreadStartedEvent struct {
	Type string `json:"type"`
	// ThreadID is the identifier of the new thread. Use it to resume the thread later.
	ThreadID string `json:"thread_id"`
}

func (e *ThreadStartedEvent) EventType() string { return EventTypeThreadStarted }

// TurnStartedEvent is emitted when a turn is started by sending a new prompt
// to the model. A turn encompasses all events that happen while the agent is
// processing the prompt.
type TurnStartedEvent struct {
	Type string `json:"type"`
}

func (e *TurnStartedEvent) EventType() string { return EventTypeTurnStarted }

// TurnCompletedEvent is emitted when a turn is completed, typically right
// after the assistant's response.
type TurnCompletedEvent struct {
	Type  string `json:"type"`
	Usage Usage  `json:"usage"`
}

func (e *TurnCompletedEvent) EventType() string { return EventTypeTurnCompleted }

// TurnFailedEvent indicates that a turn failed with an error.
type TurnFailedEvent struct {
	Type  string      `json:"type"`
	Error ThreadError `json:"error"`
}

func (e *TurnFailedEvent) EventType() string { return EventTypeTurnFailed }

// ItemStartedEvent is emitted when a new item is added to the thread.
// Typically the item is initially "in progress".
type ItemStartedEvent struct {
	Type string     `json:"type"`
	Item ThreadItem `json:"-"`
}

func (e *ItemStartedEvent) EventType() string { return EventTypeItemStarted }

// ItemUpdatedEvent is emitted when an item is updated.
type ItemUpdatedEvent struct {
	Type string     `json:"type"`
	Item ThreadItem `json:"-"`
}

func (e *ItemUpdatedEvent) EventType() string { return EventTypeItemUpdated }

// ItemCompletedEvent signals that an item has reached a terminal state,
// either success or failure.
type ItemCompletedEvent struct {
	Type string     `json:"type"`
	Item ThreadItem `json:"-"`
}

func (e *ItemCompletedEvent) EventType() string { return EventTypeItemCompleted }

// ThreadErrorEvent represents an unrecoverable error emitted directly by the
// event stream.
type ThreadErrorEvent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (e *ThreadErrorEvent) EventType() string { return EventTypeError }

// UnknownEvent is returned for event types this SDK does not know about yet.
// Raw holds the original JSON payload.
type UnknownEvent struct {
	Type string
	Raw  json.RawMessage
}

func (e *UnknownEvent) EventType() string { return e.Type }

// ItemEvent is implemented by the three item events so callers can handle
// them generically.
type ItemEvent interface {
	ThreadEvent
	// ThreadItem returns the item carried by the event.
	ThreadItem() ThreadItem
}

func (e *ItemStartedEvent) ThreadItem() ThreadItem   { return e.Item }
func (e *ItemUpdatedEvent) ThreadItem() ThreadItem   { return e.Item }
func (e *ItemCompletedEvent) ThreadItem() ThreadItem { return e.Item }

// ParseThreadEvent decodes one JSONL line emitted by codex exec into a
// ThreadEvent. Unknown event types are returned as *UnknownEvent.
func ParseThreadEvent(line []byte) (ThreadEvent, error) {
	var probe struct {
		Type string          `json:"type"`
		Item json.RawMessage `json:"item"`
	}
	if err := json.Unmarshal(line, &probe); err != nil {
		return nil, &ParseError{Line: string(line), Err: err}
	}

	switch probe.Type {
	case EventTypeThreadStarted:
		ev := &ThreadStartedEvent{}
		return ev, unmarshalEvent(line, ev)
	case EventTypeTurnStarted:
		ev := &TurnStartedEvent{}
		return ev, unmarshalEvent(line, ev)
	case EventTypeTurnCompleted:
		ev := &TurnCompletedEvent{}
		return ev, unmarshalEvent(line, ev)
	case EventTypeTurnFailed:
		ev := &TurnFailedEvent{}
		return ev, unmarshalEvent(line, ev)
	case EventTypeError:
		ev := &ThreadErrorEvent{}
		return ev, unmarshalEvent(line, ev)
	case EventTypeItemStarted, EventTypeItemUpdated, EventTypeItemCompleted:
		if len(probe.Item) == 0 {
			return nil, &ParseError{Line: string(line), Err: fmt.Errorf("%s event is missing \"item\"", probe.Type)}
		}
		item, err := ParseThreadItem(probe.Item)
		if err != nil {
			return nil, err
		}
		switch probe.Type {
		case EventTypeItemStarted:
			return &ItemStartedEvent{Type: probe.Type, Item: item}, nil
		case EventTypeItemUpdated:
			return &ItemUpdatedEvent{Type: probe.Type, Item: item}, nil
		default:
			return &ItemCompletedEvent{Type: probe.Type, Item: item}, nil
		}
	case "":
		return nil, &ParseError{Line: string(line), Err: fmt.Errorf("event is missing \"type\"")}
	default:
		raw := make(json.RawMessage, len(line))
		copy(raw, line)
		return &UnknownEvent{Type: probe.Type, Raw: raw}, nil
	}
}

func unmarshalEvent(line []byte, ev ThreadEvent) error {
	if err := json.Unmarshal(line, ev); err != nil {
		return &ParseError{Line: string(line), Err: err}
	}
	return nil
}
