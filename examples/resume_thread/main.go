// Example: multi-turn conversation and resuming a persisted thread by id.
//
// Usage:
//
//	go run ./examples/resume_thread                # start a thread, run two turns
//	CODEX_THREAD_ID=<id> go run ./examples/resume_thread   # resume an existing thread
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	codex "github.com/schlunsen/codex-sdk-go"
	"github.com/schlunsen/codex-sdk-go/types"
)

func main() {
	ctx := context.Background()

	client, err := codex.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	opts := types.NewThreadOptions().WithSkipGitRepoCheck(true)

	var thread *codex.Thread
	if id := os.Getenv("CODEX_THREAD_ID"); id != "" {
		// Threads are persisted by codex in ~/.codex/sessions; reconstruct one by id.
		thread = client.ResumeThread(id, opts)
		fmt.Printf("resuming thread %s\n", id)
	} else {
		thread = client.StartThread(opts)
	}

	turn, err := thread.Run(ctx, "Pick a random programming language and name it. Reply with only the name.", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("turn 1:", turn.FinalResponse)

	// The second turn continues the same conversation.
	turn, err = thread.Run(ctx, "What year was that language first released?", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("turn 2:", turn.FinalResponse)

	fmt.Printf("\nresume later with: CODEX_THREAD_ID=%s go run ./examples/resume_thread\n", thread.ID())
}
