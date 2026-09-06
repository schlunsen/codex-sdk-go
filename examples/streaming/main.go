// Example: stream events as the agent works.
//
// Usage:
//
//	go run ./examples/streaming "Run the tests and tell me what fails"
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	codex "github.com/schlunsen/codex-sdk-go"
	"github.com/schlunsen/codex-sdk-go/types"
)

func main() {
	prompt := "List the files in this directory and describe the project layout."
	if len(os.Args) > 1 {
		prompt = strings.Join(os.Args[1:], " ")
	}

	// Ctrl-C cancels the turn and kills the codex process.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	client, err := codex.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	thread := client.StartThread(types.NewThreadOptions().
		WithSandboxMode(types.SandboxWorkspaceWrite).
		WithApprovalPolicy(types.ApprovalNever).
		WithSkipGitRepoCheck(true))

	stream, err := thread.RunStreamed(ctx, prompt, nil)
	if err != nil {
		log.Fatal(err)
	}

	for ev := range stream.Events() {
		switch e := ev.(type) {
		case *types.ThreadStartedEvent:
			fmt.Printf("▶ thread %s\n", e.ThreadID)
		case *types.ItemStartedEvent:
			printItem("⏳", e.Item)
		case *types.ItemUpdatedEvent:
			printItem("↻", e.Item)
		case *types.ItemCompletedEvent:
			printItem("✔", e.Item)
		case *types.TurnCompletedEvent:
			fmt.Printf("■ done: %d input / %d output tokens\n", e.Usage.InputTokens, e.Usage.OutputTokens)
		case *types.TurnFailedEvent:
			fmt.Printf("✖ turn failed: %s\n", e.Error.Message)
		case *types.ThreadErrorEvent:
			fmt.Printf("✖ stream error: %s\n", e.Message)
		}
	}
	if err := stream.Err(); err != nil {
		log.Fatal(err)
	}
}

func printItem(prefix string, item types.ThreadItem) {
	switch it := item.(type) {
	case *types.AgentMessageItem:
		fmt.Printf("%s agent: %s\n", prefix, it.Text)
	case *types.ReasoningItem:
		fmt.Printf("%s reasoning: %s\n", prefix, it.Text)
	case *types.CommandExecutionItem:
		fmt.Printf("%s $ %s [%s]\n", prefix, it.Command, it.Status)
		if it.Status != types.CommandExecutionInProgress && it.AggregatedOutput != "" {
			fmt.Println(indent(it.AggregatedOutput))
		}
	case *types.FileChangeItem:
		for _, c := range it.Changes {
			fmt.Printf("%s %s %s\n", prefix, c.Kind, c.Path)
		}
	case *types.McpToolCallItem:
		fmt.Printf("%s mcp %s/%s [%s]\n", prefix, it.Server, it.Tool, it.Status)
	case *types.WebSearchItem:
		fmt.Printf("%s search: %s\n", prefix, it.Query)
	case *types.TodoListItem:
		for _, todo := range it.Items {
			mark := " "
			if todo.Completed {
				mark = "x"
			}
			fmt.Printf("%s [%s] %s\n", prefix, mark, todo.Text)
		}
	case *types.ErrorItem:
		fmt.Printf("%s error: %s\n", prefix, it.Message)
	default:
		fmt.Printf("%s %s (%s)\n", prefix, item.ItemType(), item.ItemID())
	}
}

func indent(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = "    " + l
	}
	return strings.Join(lines, "\n")
}
