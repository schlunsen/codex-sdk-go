# Parity with `@openai/codex-sdk`

This SDK tracks the official TypeScript SDK. The table maps each TS concept to
its Go equivalent.

| TypeScript | Go |
|---|---|
| `new Codex(options)` | `codex.New(&types.CodexOptions{...})` |
| `codex.startThread(opts)` | `client.StartThread(opts)` |
| `codex.resumeThread(id, opts)` | `client.ResumeThread(id, opts)` |
| `thread.run(input, turnOpts)` | `thread.Run(ctx, prompt, turnOpts)` / `thread.RunInputs(ctx, inputs, turnOpts)` |
| `thread.runStreamed(input)` → `{ events }` | `thread.RunStreamed(ctx, prompt, nil)` → `*StreamedTurn` (`Events()`, `Err()`, `Close()`) |
| `thread.id` | `thread.ID()` |
| `Turn { items, finalResponse, usage }` | `codex.Turn { Items, FinalResponse, Usage }` |
| `Input = string \| UserInput[]` | `string` for `Run`, `[]types.UserInput` for `RunInputs` |
| `{ type: "text", text }` | `types.TextInput(text)` |
| `{ type: "local_image", path }` | `types.LocalImageInput(path)` |
| `TurnOptions.outputSchema` | `types.TurnOptions.OutputSchema` (`map[string]any`) |
| `TurnOptions.signal` (AbortSignal) | `context.Context` passed to `Run*` |
| `CodexOptions.codexPathOverride` | `CodexOptions.CodexPathOverride` |
| `CodexOptions.baseUrl` | `CodexOptions.BaseURL` |
| `CodexOptions.apiKey` | `CodexOptions.APIKey` |
| `CodexOptions.config` | `CodexOptions.Config` (`types.ConfigObject`) |
| `CodexOptions.configOverrides` | `CodexOptions.ConfigOverrides` |
| `CodexOptions.env` | `CodexOptions.Env` |
| `ThreadOptions.model` | `ThreadOptions.Model` |
| `ThreadOptions.threadSource` | `ThreadOptions.ThreadSource` |
| `ThreadOptions.sandboxMode` | `ThreadOptions.SandboxMode` |
| `ThreadOptions.workingDirectory` | `ThreadOptions.WorkingDirectory` |
| `ThreadOptions.additionalDirectories` | `ThreadOptions.AdditionalDirectories` |
| `ThreadOptions.skipGitRepoCheck` | `ThreadOptions.SkipGitRepoCheck` |
| `ThreadOptions.modelReasoningEffort` | `ThreadOptions.ModelReasoningEffort` |
| `ThreadOptions.networkAccessEnabled` | `ThreadOptions.NetworkAccessEnabled` (`*bool`) |
| `ThreadOptions.webSearchMode` | `ThreadOptions.WebSearchMode` |
| `ThreadOptions.webSearchEnabled` | `ThreadOptions.WebSearchEnabled` (`*bool`) |
| `ThreadOptions.approvalPolicy` | `ThreadOptions.ApprovalPolicy` |
| `ThreadEvent` union | `types.ThreadEvent` interface + concrete `*types.XxxEvent` |
| `ThreadItem` union | `types.ThreadItem` interface + concrete `*types.XxxItem` |
| `Usage` | `types.Usage` |
| `CODEX_INTERNAL_ORIGINATOR_OVERRIDE=codex_sdk_ts` | `=codex_sdk_go` |

## Differences

- **Errors are typed.** `turn.failed` becomes `*types.TurnFailedError`, a
  non-zero exit becomes `*types.ExecError` (with exit code and stderr), and
  malformed JSONL becomes `*types.ParseError`. The TS SDK throws plain `Error`s.
- **Forward compatibility.** Unknown event/item types decode to
  `*types.UnknownEvent` / `*types.UnknownItem` carrying the raw JSON instead of
  failing.
- **CLI discovery.** The TS SDK resolves the binary bundled in the
  `@openai/codex` npm package. This SDK checks `CODEX_PATH`, `PATH`, then
  common install locations (npm global, Homebrew, bun, cargo).
- **`cache_write_input_tokens`** defaults to `0` when absent, matching the TS SDK.
- **Process cleanup.** Cancelling the context kills the codex process and
  unblocks readers even if grandchildren keep stdout open.
- **Early close.** `StreamedTurn.Close` stops a streamed turn and reaps the
  process without cancelling the caller's context (or, once `turn.completed` /
  `turn.failed` was delivered, waits for a clean exit). The TS SDK has only
  `AbortSignal`.
