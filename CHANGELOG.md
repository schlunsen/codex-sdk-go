# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project
adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.2] - 2026-09-06

### Fixed
- `StreamedTurn.Close` no longer blocks when a terminal event is waiting behind
  a full event buffer, and continues draining output during session persistence.
- Oversized JSONL output terminates the child process and preserves the scanner
  error instead of waiting indefinitely for a blocked writer.
- Clean process exits without a terminal turn event return `ErrIncompleteTurn`
  from both collected and streamed turns.
- Relative working directories resolve once, avoiding applying the directory
  twice through the process working directory and `--cd`.

## [0.1.1] - 2026-09-06

### Added
- `StreamedTurn.Close` to release a streamed turn without cancelling the
  caller's context: kills the codex process while the turn is in progress,
  or waits for a clean exit once `turn.completed` / `turn.failed` was delivered
- `types.IsConfigError` helper, matching the other `Is*` error helpers
- `codex.ErrClosed`, returned by `StreamedTurn.Err`/`Close` when `Close` stopped
  an in-progress turn, so it is distinguishable from the caller's own context
  being cancelled

### Changed
- On Unix, codex now runs in its own process group and cancellation sends
  `SIGINT` to that group (codex aborts the turn and kills the shell command it
  is running), escalating to `SIGKILL` after one second. Agent-spawned
  commands no longer outlive a cancelled turn, and cancellation no longer
  waits out `WaitDelay` on a pipe held by a grandchild.
- `Thread.Run` returns the context error if the context was cancelled before
  `turn.completed` arrived, even when codex exited cleanly.

## [0.1.0] - 2026-09-06

Initial release. Feature parity with `@openai/codex-sdk` (TypeScript).

### Added
- `codex.New`, `Codex.StartThread`, `Codex.ResumeThread`
- `Thread.Run` / `Thread.RunInputs` returning a completed `Turn`
- `Thread.RunStreamed` / `Thread.RunStreamedInputs` streaming `types.ThreadEvent`s over a channel
- Typed events (`thread.started`, `turn.*`, `item.*`, `error`) and items
  (`agent_message`, `reasoning`, `command_execution`, `file_change`,
  `mcp_tool_call`, `web_search`, `todo_list`, `error`) with forward-compatible
  `UnknownEvent` / `UnknownItem`
- Thread options: model, sandbox mode, working directory, additional
  directories, skip git repo check, reasoning effort, network access, web
  search mode, approval policy, thread source
- Turn options: JSON output schema (`--output-schema`)
- Structured `--config` overrides flattened to TOML literals, plus raw overrides
- Explicit environment control, `CODEX_API_KEY` injection, `openai_base_url` override
- Local image inputs (`--image`)
- Context-based cancellation that kills the codex process
- Codex CLI discovery via `CODEX_PATH`, `PATH`, and common install locations
- Typed errors: `CLINotFoundError`, `ExecError`, `TurnFailedError`,
  `ThreadStreamError`, `ParseError`, `ConfigError`

[Unreleased]: https://github.com/schlunsen/codex-sdk-go/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/schlunsen/codex-sdk-go/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/schlunsen/codex-sdk-go/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/schlunsen/codex-sdk-go/releases/tag/v0.1.0
