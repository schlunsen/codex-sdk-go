package types

import (
	"encoding/json"
	"fmt"
)

// Item type discriminators as emitted by codex exec.
const (
	ItemTypeAgentMessage     = "agent_message"
	ItemTypeReasoning        = "reasoning"
	ItemTypeCommandExecution = "command_execution"
	ItemTypeFileChange       = "file_change"
	ItemTypeMcpToolCall      = "mcp_tool_call"
	ItemTypeWebSearch        = "web_search"
	ItemTypeTodoList         = "todo_list"
	ItemTypeError            = "error"
)

// CommandExecutionStatus is the status of a command execution.
type CommandExecutionStatus string

const (
	CommandExecutionInProgress CommandExecutionStatus = "in_progress"
	CommandExecutionCompleted  CommandExecutionStatus = "completed"
	CommandExecutionFailed     CommandExecutionStatus = "failed"
)

// PatchChangeKind indicates the type of a file change.
type PatchChangeKind string

const (
	PatchChangeAdd    PatchChangeKind = "add"
	PatchChangeDelete PatchChangeKind = "delete"
	PatchChangeUpdate PatchChangeKind = "update"
)

// PatchApplyStatus is the status of a file change.
type PatchApplyStatus string

const (
	PatchApplyCompleted PatchApplyStatus = "completed"
	PatchApplyFailed    PatchApplyStatus = "failed"
)

// McpToolCallStatus is the status of an MCP tool call.
type McpToolCallStatus string

const (
	McpToolCallInProgress McpToolCallStatus = "in_progress"
	McpToolCallCompleted  McpToolCallStatus = "completed"
	McpToolCallFailed     McpToolCallStatus = "failed"
)

// ThreadItem is implemented by every item that can appear in a thread.
//
// Use a type switch to access the concrete payload:
//
//	switch it := item.(type) {
//	case *types.AgentMessageItem:
//	    fmt.Println(it.Text)
//	case *types.CommandExecutionItem:
//	    fmt.Println(it.Command, it.Status)
//	}
type ThreadItem interface {
	// ItemType returns the type discriminator (e.g. "agent_message").
	ItemType() string
	// ItemID returns the unique id of the item within the thread.
	ItemID() string
}

// AgentMessageItem is a response from the agent. Either natural-language text
// or JSON when structured output is requested.
type AgentMessageItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Text string `json:"text"`
}

func (i *AgentMessageItem) ItemType() string { return ItemTypeAgentMessage }
func (i *AgentMessageItem) ItemID() string   { return i.ID }

// ReasoningItem is the agent's reasoning summary.
type ReasoningItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Text string `json:"text"`
}

func (i *ReasoningItem) ItemType() string { return ItemTypeReasoning }
func (i *ReasoningItem) ItemID() string   { return i.ID }

// CommandExecutionItem is a command executed by the agent.
type CommandExecutionItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// Command is the command line executed by the agent.
	Command string `json:"command"`
	// AggregatedOutput is the stdout and stderr captured while the command ran.
	AggregatedOutput string `json:"aggregated_output"`
	// ExitCode is set when the command exits; nil while still running.
	ExitCode *int `json:"exit_code,omitempty"`
	// Status is the current status of the command execution.
	Status CommandExecutionStatus `json:"status"`
}

func (i *CommandExecutionItem) ItemType() string { return ItemTypeCommandExecution }
func (i *CommandExecutionItem) ItemID() string   { return i.ID }

// FileUpdateChange is a single file change within a patch.
type FileUpdateChange struct {
	Path string          `json:"path"`
	Kind PatchChangeKind `json:"kind"`
}

// FileChangeItem is a set of file changes by the agent. Emitted once the
// patch succeeds or fails.
type FileChangeItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// Changes are the individual file changes that comprise the patch.
	Changes []FileUpdateChange `json:"changes"`
	// Status reports whether the patch ultimately succeeded or failed.
	Status PatchApplyStatus `json:"status"`
}

func (i *FileChangeItem) ItemType() string { return ItemTypeFileChange }
func (i *FileChangeItem) ItemID() string   { return i.ID }

// McpToolCallResult is the result payload returned by an MCP server for a
// successful call.
type McpToolCallResult struct {
	Content           []json.RawMessage `json:"content"`
	Meta              json.RawMessage   `json:"_meta,omitempty"`
	StructuredContent json.RawMessage   `json:"structured_content,omitempty"`
}

// McpToolCallError is the error reported for a failed MCP tool call.
type McpToolCallError struct {
	Message string `json:"message"`
}

// McpToolCallItem represents a call to an MCP tool. The item starts when the
// invocation is dispatched and completes when the MCP server reports success
// or failure.
type McpToolCallItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// Server is the name of the MCP server handling the request.
	Server string `json:"server"`
	// Tool is the tool invoked on the MCP server.
	Tool string `json:"tool"`
	// Arguments are the arguments forwarded to the tool invocation.
	Arguments json.RawMessage `json:"arguments,omitempty"`
	// Result is the result payload for successful calls.
	Result *McpToolCallResult `json:"result,omitempty"`
	// Error is the error reported for failed calls.
	Error *McpToolCallError `json:"error,omitempty"`
	// Status is the current status of the tool invocation.
	Status McpToolCallStatus `json:"status"`
}

func (i *McpToolCallItem) ItemType() string { return ItemTypeMcpToolCall }
func (i *McpToolCallItem) ItemID() string   { return i.ID }

// WebSearchItem captures a web search request. Completes when results are
// returned to the agent.
type WebSearchItem struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Query string `json:"query"`
}

func (i *WebSearchItem) ItemType() string { return ItemTypeWebSearch }
func (i *WebSearchItem) ItemID() string   { return i.ID }

// TodoItem is an entry in the agent's to-do list.
type TodoItem struct {
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

// TodoListItem tracks the agent's running to-do list. Starts when the plan is
// issued, updates as steps change, and completes when the turn ends.
type TodoListItem struct {
	ID    string     `json:"id"`
	Type  string     `json:"type"`
	Items []TodoItem `json:"items"`
}

func (i *TodoListItem) ItemType() string { return ItemTypeTodoList }
func (i *TodoListItem) ItemID() string   { return i.ID }

// ErrorItem describes a non-fatal error surfaced as an item.
type ErrorItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (i *ErrorItem) ItemType() string { return ItemTypeError }
func (i *ErrorItem) ItemID() string   { return i.ID }

// UnknownItem is returned for item types this SDK does not know about yet,
// so that newer CLI versions do not break consumers. Raw holds the original
// JSON payload.
type UnknownItem struct {
	ID   string
	Type string
	Raw  json.RawMessage
}

func (i *UnknownItem) ItemType() string { return i.Type }
func (i *UnknownItem) ItemID() string   { return i.ID }

// ParseThreadItem decodes a single thread item from its JSON representation.
// Unknown item types are returned as *UnknownItem rather than an error.
func ParseThreadItem(data []byte) (ThreadItem, error) {
	var probe struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, &ParseError{Line: string(data), Err: err}
	}

	var item ThreadItem
	switch probe.Type {
	case ItemTypeAgentMessage:
		item = &AgentMessageItem{}
	case ItemTypeReasoning:
		item = &ReasoningItem{}
	case ItemTypeCommandExecution:
		item = &CommandExecutionItem{}
	case ItemTypeFileChange:
		item = &FileChangeItem{}
	case ItemTypeMcpToolCall:
		item = &McpToolCallItem{}
	case ItemTypeWebSearch:
		item = &WebSearchItem{}
	case ItemTypeTodoList:
		item = &TodoListItem{}
	case ItemTypeError:
		item = &ErrorItem{}
	case "":
		return nil, &ParseError{Line: string(data), Err: fmt.Errorf("item is missing \"type\"")}
	default:
		raw := make(json.RawMessage, len(data))
		copy(raw, data)
		return &UnknownItem{ID: probe.ID, Type: probe.Type, Raw: raw}, nil
	}

	if err := json.Unmarshal(data, item); err != nil {
		return nil, &ParseError{Line: string(data), Err: err}
	}
	return item, nil
}
