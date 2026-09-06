// Package codex is an unofficial Go SDK for the OpenAI Codex agent.
//
// It mirrors the official TypeScript SDK (@openai/codex-sdk): it spawns the
// `codex` CLI with `exec --experimental-json`, writes the prompt to stdin and
// streams structured JSONL events back over stdout.
//
// Basic usage:
//
//	client, err := codex.New(nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	thread := client.StartThread(nil)
//	turn, err := thread.Run(ctx, "Diagnose the test failure and propose a fix", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(turn.FinalResponse)
//
// Call Run repeatedly on the same Thread to continue the conversation, or use
// RunStreamed to react to events as they are produced.
package codex
