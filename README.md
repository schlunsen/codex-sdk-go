# Codex SDK for Go

[![CI](https://github.com/schlunsen/codex-sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/schlunsen/codex-sdk-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/schlunsen/codex-sdk-go.svg)](https://pkg.go.dev/github.com/schlunsen/codex-sdk-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/schlunsen/codex-sdk-go)](https://goreportcard.com/report/github.com/schlunsen/codex-sdk-go)
[![Release](https://img.shields.io/github/v/release/schlunsen/codex-sdk-go)](https://github.com/schlunsen/codex-sdk-go/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Unofficial Go port** of the [official TypeScript SDK](https://github.com/openai/codex/tree/main/sdk/typescript) (`@openai/codex-sdk`) for the OpenAI Codex agent.

> ⚠️ Not affiliated with or endorsed by OpenAI.

Embed the Codex coding agent in Go services, CLIs and CI jobs. The SDK spawns the `codex` CLI with `exec --experimental-json`, writes your prompt to stdin, and streams structured JSONL events back — the same mechanism the TypeScript SDK uses, with idiomatic Go on top: `context` for cancellation, channels for streaming, typed events and errors, and zero third-party dependencies.

## Features

- 🧵 **Threads & turns** — `StartThread`, `ResumeThread`, multi-turn conversations persisted by codex
- ⚡ **Streaming** — `RunStreamed` delivers every `thread.*`, `turn.*` and `item.*` event over a channel as it happens
- 🧱 **Typed items** — agent messages, reasoning, command executions, file changes, MCP tool calls, web searches, todo lists
- 🎯 **Structured output** — pass a JSON schema per turn, get JSON back
- 🖼️ **Images** — attach local images alongside text
- 🔒 **Sandbox & approvals** — read-only / workspace-write / full-access, approval policy, network access, extra directories
- ⚙️ **Config overrides** — structured maps flattened to `--config key=value` TOML, plus raw overrides
- 🛑 **Cancellation** — cancel the `context` and the codex process is killed
- 🧪 **Testable** — unit tests run against a fake `codex` binary; no API key required
- 📦 **Zero dependencies** — stdlib only

See [docs/PARITY.md](docs/PARITY.md) for a field-by-field map to the TypeScript SDK.

## Installation

```bash
go get github.com/schlunsen/codex-sdk-go
```

Requires Go 1.24+ and the Codex CLI:

```bash
npm install -g @openai/codex   # or: brew install codex
codex login                    # or: export CODEX_API_KEY=...
```

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"

	codex "github.com/schlunsen/codex-sdk-go"
)

func main() {
	client, err := codex.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	thread := client.StartThread(nil)
	turn, err := thread.Run(context.Background(), "Diagnose the test failure and propose a fix", nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(turn.FinalResponse)
	fmt.Println(len(turn.Items), "items")
}
```

Call `Run` again on the same `Thread` to continue the conversation:

```go
next, err := thread.Run(ctx, "Implement the fix", nil)
```

### Streaming responses

`Run` buffers events until the turn finishes. To react to intermediate progress — tool calls, file changes, reasoning — use `RunStreamed`:

```go
stream, err := thread.RunStreamed(ctx, "Diagnose the test failure and propose a fix", nil)
if err != nil {
	log.Fatal(err)
}

for ev := range stream.Events() {
	switch e := ev.(type) {
	case *types.ItemCompletedEvent:
		switch item := e.Item.(type) {
		case *types.CommandExecutionItem:
			fmt.Printf("$ %s (exit %d)\n", item.Command, *item.ExitCode)
		case *types.AgentMessageItem:
			fmt.Println(item.Text)
		}
	case *types.TurnCompletedEvent:
		fmt.Printf("tokens: %d in / %d out\n", e.Usage.InputTokens, e.Usage.OutputTokens)
	}
}
if err := stream.Err(); err != nil { // nil on success; *types.ExecError, *types.ParseError, or ctx error
	log.Fatal(err)
}
```

To stop a turn early without cancelling your context, call `stream.Close()`. While the turn is in progress it kills the codex process, closes `Events()`, and returns `codex.ErrClosed` (distinct from the context errors, so an `errgroup` or retry loop can tell a deliberate close from an upstream abort). Once `turn.completed` or `turn.failed` has been delivered, `Close` no longer kills anything — codex is left to exit and persist the session — and it returns the same error `Err()` would, so `defer stream.Close()` releases event delivery while allowing session persistence to finish. A clean process exit without `turn.completed` or `turn.failed` returns `codex.ErrIncompleteTurn`.

### Structured output

```go
schema := map[string]any{
	"type": "object",
	"properties": map[string]any{
		"summary": map[string]any{"type": "string"},
		"status":  map[string]any{"type": "string", "enum": []string{"ok", "action_required"}},
	},
	"required":             []string{"summary", "status"},
	"additionalProperties": false,
}

turn, err := thread.Run(ctx, "Summarize repository status",
	types.NewTurnOptions().WithOutputSchema(schema))
// turn.FinalResponse is JSON conforming to the schema
```

### Attaching images

Text entries are concatenated into the prompt; image entries are passed via `--image`.

```go
turn, err := thread.RunInputs(ctx, []types.UserInput{
	types.TextInput("Describe these screenshots"),
	types.LocalImageInput("./ui.png"),
	types.LocalImageInput("./diagram.jpg"),
}, nil)
```

### Resuming a thread

Threads are persisted by codex in `~/.codex/sessions`. If you lose the in-memory `Thread`, reconstruct it by id:

```go
thread := client.ResumeThread(savedThreadID, nil)
turn, err := thread.Run(ctx, "Implement the fix", nil)
```

`thread.ID()` is populated after the first turn starts.

### Thread options

```go
thread := client.StartThread(types.NewThreadOptions().
	WithModel("gpt-5-codex").
	WithSandboxMode(types.SandboxWorkspaceWrite).
	WithApprovalPolicy(types.ApprovalNever).
	WithWorkingDirectory("/path/to/project").
	WithAdditionalDirectories("/path/to/shared").
	WithSkipGitRepoCheck(true).
	WithModelReasoningEffort(types.ReasoningHigh).
	WithNetworkAccess(true).
	WithWebSearchMode(types.WebSearchLive))
```

Codex refuses to run outside a git repository unless `SkipGitRepoCheck` is set.

### Controlling the CLI environment

By default the codex process inherits the parent environment. Provide `Env` to fully control it (useful in sandboxed hosts). The SDK still injects `CODEX_API_KEY` when `APIKey` is set, and identifies itself via `CODEX_INTERNAL_ORIGINATOR_OVERRIDE=codex_sdk_go`.

```go
client, err := codex.New(types.NewCodexOptions().
	WithAPIKey(os.Getenv("OPENAI_API_KEY")).
	WithBaseURL("https://my-proxy.example/v1").   // -> --config openai_base_url=...
	WithEnv(map[string]string{"PATH": "/usr/local/bin"}).
	WithCodexPath("/opt/codex/bin/codex"))
```

### Passing `--config` overrides

`Config` takes a nested map, flattens it to dotted paths and serializes values as TOML literals:

```go
client, err := codex.New(types.NewCodexOptions().
	WithConfig(types.ConfigObject{
		"show_raw_agent_reasoning": true,
		"sandbox_workspace_write":  map[string]any{"network_access": true},
	}))
// --config show_raw_agent_reasoning=true
// --config sandbox_workspace_write.network_access=true
```

For keys that can't be expressed as dotted paths, pass raw TOML with `ConfigOverrides`; they are forwarded unchanged after `Config` and before SDK-managed / thread-specific overrides:

```go
types.NewCodexOptions().
	WithConfig(types.ConfigObject{"default_permissions": "audit"}).
	WithConfigOverrides(`permissions.audit.filesystem={":root"="read","/path/.env"="deny"}`)
```

### Cancellation

Every `Run*` method takes a `context.Context`. Cancelling it stops the codex process and closes the event channel; `stream.Err()` / `Run` return `context.Canceled` or `context.DeadlineExceeded`. On Unix the SDK first sends `SIGINT` to codex's process group — codex aborts the turn and kills the shell command it is running — and escalates to `SIGKILL` of the group if codex has not exited within a second. On Windows the process is killed outright.

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
turn, err := thread.Run(ctx, prompt, nil)
```

### Error handling

| Error | When |
|---|---|
| `*types.CLINotFoundError` | codex binary not found (`types.IsCLINotFoundError`) |
| `*types.TurnFailedError` | agent emitted `turn.failed` |
| `*types.ExecError` | codex exited non-zero; carries `ExitCode`, `Signal`, `Stderr` |
| `*types.ThreadStreamError` | stream emitted a fatal `error` event but exited 0 |
| `*types.ParseError` | a JSONL line couldn't be decoded |
| `*types.ConfigError` | a config override couldn't be serialized |

All support `errors.As`, and `Is*` helpers are provided.

## Events and items

Events implement `types.ThreadEvent`; items implement `types.ThreadItem`. Unknown types from newer CLI versions decode into `*types.UnknownEvent` / `*types.UnknownItem` with the raw JSON preserved, so upgrades never break your consumer.

| Event | Payload |
|---|---|
| `thread.started` | `ThreadID` |
| `turn.started` | — |
| `turn.completed` | `Usage` |
| `turn.failed` | `Error.Message` |
| `item.started` / `item.updated` / `item.completed` | `Item types.ThreadItem` |
| `error` | `Message` |

| Item | Key fields |
|---|---|
| `agent_message` | `Text` |
| `reasoning` | `Text` |
| `command_execution` | `Command`, `AggregatedOutput`, `ExitCode`, `Status` |
| `file_change` | `Changes[]{Path, Kind}`, `Status` |
| `mcp_tool_call` | `Server`, `Tool`, `Arguments`, `Result`, `Error`, `Status` |
| `web_search` | `Query` |
| `todo_list` | `Items[]{Text, Completed}` |
| `error` | `Message` |

## CLI discovery

The binary is resolved in order: `CodexOptions.CodexPathOverride` → `$CODEX_PATH` → `codex` on `PATH` → common install locations (`~/.npm-global/bin`, `/opt/homebrew/bin`, `/usr/local/bin`, `~/.local/bin`, `~/.bun/bin`, `~/.cargo/bin`, …).

## Examples

| Example | Shows |
|---|---|
| [`examples/simple_run`](examples/simple_run) | One-shot turn |
| [`examples/streaming`](examples/streaming) | Live event rendering, Ctrl-C cancellation |
| [`examples/resume_thread`](examples/resume_thread) | Multi-turn + resume by id |
| [`examples/structured_output`](examples/structured_output) | JSON schema output |
| [`examples/with_images`](examples/with_images) | Text + image input |

```bash
go run ./examples/streaming "Run the tests and summarize what fails"
```

## Development

```bash
make test-race   # unit tests against a fake codex binary (no network)
make test-live   # smoke test against a real codex CLI (needs codex login or CODEX_API_KEY)
make examples    # compile examples
make lint        # go vet + golangci-lint
make coverage    # HTML coverage report
```

See [CONTRIBUTING.md](CONTRIBUTING.md). Sibling project: [claude-agent-sdk-go](https://github.com/schlunsen/claude-agent-sdk-go).

## License

MIT — see [LICENSE](LICENSE).
