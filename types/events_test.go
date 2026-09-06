package types

import (
	"testing"
)

func TestParseThreadEventAllTypes(t *testing.T) {
	cases := map[string]string{
		"thread.started": `{"type":"thread.started","thread_id":"t1"}`,
		"turn.started":   `{"type":"turn.started"}`,
		"turn.completed": `{"type":"turn.completed","usage":{"input_tokens":1,"cached_input_tokens":2,"cache_write_input_tokens":3,"output_tokens":4,"reasoning_output_tokens":5}}`,
		"turn.failed":    `{"type":"turn.failed","error":{"message":"nope"}}`,
		"item.started":   `{"type":"item.started","item":{"id":"i","type":"agent_message","text":"hi"}}`,
		"item.updated":   `{"type":"item.updated","item":{"id":"i","type":"todo_list","items":[{"text":"a","completed":false}]}}`,
		"item.completed": `{"type":"item.completed","item":{"id":"i","type":"web_search","query":"go"}}`,
		"error":          `{"type":"error","message":"fatal"}`,
		"something.new":  `{"type":"something.new","x":1}`,
	}
	for wantType, line := range cases {
		ev, err := ParseThreadEvent([]byte(line))
		if err != nil {
			t.Fatalf("%s: %v", wantType, err)
		}
		if ev.EventType() != wantType {
			t.Errorf("%s: got %s", wantType, ev.EventType())
		}
	}

	ev, _ := ParseThreadEvent([]byte(cases["thread.started"]))
	if ev.(*ThreadStartedEvent).ThreadID != "t1" {
		t.Error("thread id not parsed")
	}
	ev, _ = ParseThreadEvent([]byte(cases["turn.completed"]))
	if u := ev.(*TurnCompletedEvent).Usage; u.CacheWriteInputTokens != 3 || u.ReasoningOutputTokens != 5 {
		t.Errorf("usage not parsed: %+v", u)
	}
	ev, _ = ParseThreadEvent([]byte(cases["turn.failed"]))
	if ev.(*TurnFailedEvent).Error.Message != "nope" {
		t.Error("turn.failed message not parsed")
	}
	ev, _ = ParseThreadEvent([]byte(cases["item.updated"]))
	todo := ev.(ItemEvent).ThreadItem().(*TodoListItem)
	if len(todo.Items) != 1 || todo.Items[0].Text != "a" {
		t.Errorf("todo list not parsed: %+v", todo)
	}
	ev, _ = ParseThreadEvent([]byte(cases["something.new"]))
	if _, ok := ev.(*UnknownEvent); !ok {
		t.Error("expected UnknownEvent")
	}
}

func TestParseThreadEventErrors(t *testing.T) {
	for _, line := range []string{
		`not json`,
		`{"x":1}`,
		`{"type":"item.completed"}`,
		`{"type":"item.completed","item":{"id":"i"}}`,
		`{"type":"item.completed","item":{"id":"i","type":"agent_message","text":5}}`,
	} {
		if _, err := ParseThreadEvent([]byte(line)); !IsParseError(err) {
			t.Errorf("%s: expected ParseError, got %v", line, err)
		}
	}
}

func TestParseThreadItemAllTypes(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{`{"id":"1","type":"agent_message","text":"hi"}`, ItemTypeAgentMessage},
		{`{"id":"1","type":"reasoning","text":"hmm"}`, ItemTypeReasoning},
		{`{"id":"1","type":"command_execution","command":"ls","aggregated_output":"","exit_code":0,"status":"completed"}`, ItemTypeCommandExecution},
		{`{"id":"1","type":"file_change","changes":[{"path":"a.go","kind":"update"}],"status":"completed"}`, ItemTypeFileChange},
		{`{"id":"1","type":"mcp_tool_call","server":"s","tool":"t","arguments":{"a":1},"result":{"content":[{"type":"text","text":"ok"}],"structured_content":null},"status":"completed"}`, ItemTypeMcpToolCall},
		{`{"id":"1","type":"web_search","query":"q"}`, ItemTypeWebSearch},
		{`{"id":"1","type":"todo_list","items":[]}`, ItemTypeTodoList},
		{`{"id":"1","type":"error","message":"m"}`, ItemTypeError},
		{`{"id":"1","type":"brand_new","foo":"bar"}`, "brand_new"},
	}
	for _, c := range cases {
		item, err := ParseThreadItem([]byte(c.line))
		if err != nil {
			t.Fatalf("%s: %v", c.line, err)
		}
		if item.ItemType() != c.want || item.ItemID() != "1" {
			t.Errorf("%s: got %s/%s", c.line, item.ItemType(), item.ItemID())
		}
	}

	item, _ := ParseThreadItem([]byte(cases[2].line))
	cmd := item.(*CommandExecutionItem)
	if cmd.ExitCode == nil || *cmd.ExitCode != 0 || cmd.Status != CommandExecutionCompleted {
		t.Errorf("command not parsed: %+v", cmd)
	}
	item, _ = ParseThreadItem([]byte(cases[4].line))
	mcp := item.(*McpToolCallItem)
	if mcp.Result == nil || len(mcp.Result.Content) != 1 || string(mcp.Arguments) != `{"a":1}` {
		t.Errorf("mcp not parsed: %+v", mcp)
	}
	item, _ = ParseThreadItem([]byte(cases[8].line))
	if u := item.(*UnknownItem); string(u.Raw) != cases[8].line {
		t.Errorf("unknown raw not preserved: %s", u.Raw)
	}
}
