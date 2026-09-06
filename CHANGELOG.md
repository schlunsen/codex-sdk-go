# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project
adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

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

[Unreleased]: https://github.com/schlunsen/codex-sdk-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/schlunsen/codex-sdk-go/releases/tag/v0.1.0
