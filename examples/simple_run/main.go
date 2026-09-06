// Example: one-shot turn with the Codex agent.
//
// Usage:
//
//	go run ./examples/simple_run "Explain what this repository does"
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	codex "github.com/schlunsen/codex-sdk-go"
	"github.com/schlunsen/codex-sdk-go/types"
)

func main() {
	prompt := "Summarize what this repository does in two sentences."
	if len(os.Args) > 1 {
		prompt = strings.Join(os.Args[1:], " ")
	}

	client, err := codex.New(nil)
	if err != nil {
		if types.IsCLINotFoundError(err) {
			log.Fatalf("codex CLI not installed: %v", err)
		}
		log.Fatal(err)
	}

	thread := client.StartThread(types.NewThreadOptions().
		WithSandboxMode(types.SandboxReadOnly).
		WithSkipGitRepoCheck(true))

	turn, err := thread.Run(context.Background(), prompt, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(turn.FinalResponse)
	fmt.Printf("\n--- thread %s, %d items", thread.ID(), len(turn.Items))
	if turn.Usage != nil {
		fmt.Printf(", %d in / %d out tokens", turn.Usage.InputTokens, turn.Usage.OutputTokens)
	}
	fmt.Println()
}
