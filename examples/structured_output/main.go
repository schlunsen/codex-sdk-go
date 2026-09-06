// Example: request JSON output that conforms to a schema.
//
// Usage:
//
//	go run ./examples/structured_output
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	codex "github.com/schlunsen/codex-sdk-go"
	"github.com/schlunsen/codex-sdk-go/types"
)

type repoStatus struct {
	Summary string `json:"summary"`
	Status  string `json:"status"`
}

func main() {
	client, err := codex.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"summary": map[string]any{"type": "string"},
			"status":  map[string]any{"type": "string", "enum": []string{"ok", "action_required"}},
		},
		"required":             []string{"summary", "status"},
		"additionalProperties": false,
	}

	thread := client.StartThread(types.NewThreadOptions().
		WithSandboxMode(types.SandboxReadOnly).
		WithSkipGitRepoCheck(true))

	turn, err := thread.Run(context.Background(),
		"Summarize the repository status.",
		types.NewTurnOptions().WithOutputSchema(schema))
	if err != nil {
		log.Fatal(err)
	}

	var result repoStatus
	if err := json.Unmarshal([]byte(turn.FinalResponse), &result); err != nil {
		log.Fatalf("response was not valid JSON: %v\n%s", err, turn.FinalResponse)
	}
	fmt.Printf("status:  %s\nsummary: %s\n", result.Status, result.Summary)
}
