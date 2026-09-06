# codex-sdk-go — development guide

Unofficial Go SDK for the OpenAI Codex agent. Mirrors `@openai/codex-sdk`
(TypeScript): spawns `codex exec --experimental-json`, writes the prompt to
stdin, parses JSONL events from stdout.

## Layout

```
codex.go                  Codex client: New, StartThread, ResumeThread
thread.go                 Thread: Run/RunInputs, RunStreamed/RunStreamedInputs, Turn, StreamedTurn
types/                    Public types: events, items, options, errors
internal/config/          Flatten config map -> `--config key=value` TOML literals
internal/transport/       CLI discovery, arg building, subprocess + JSONL streaming
internal/transport/testdata/fake-codex   Fake CLI used by tests (bash + python3)
internal/livetest/        Build-tagged (`live`) tests against a real codex; local-only, never in CI
examples/                 Runnable examples (need a real codex CLI)
docs/PARITY.md            Feature map vs the TypeScript SDK
```

## Commands

```bash
make test-race     # unit tests (no network, no codex install needed)
make test-live     # live smoke test vs a real codex CLI (login or CODEX_API_KEY)
make examples      # build all examples
make lint          # vet + golangci-lint
make coverage      # HTML coverage report
```

## Rules

- Never commit directly to `main`; use `feature/`, `fix/`, `docs/`, `chore/` branches and open a PR.
- Keep `internal/transport/args.go` ordering identical to `exec.ts` in the TS SDK.
- Unknown event/item types must parse into `UnknownEvent`/`UnknownItem`, never error.
- Update `CHANGELOG.md` and bump `VERSION` on release; the release workflow rejects mismatches.
- Commit messages: `type: subject`. No "co-authored-by" trailers.
