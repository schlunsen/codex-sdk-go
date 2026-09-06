# Contributing

Thanks for helping improve codex-sdk-go.

## Development setup

```bash
git clone https://github.com/schlunsen/codex-sdk-go
cd codex-sdk-go
make test-race   # unit tests use a fake codex binary; no API key needed
make examples    # compile the examples
make lint        # go vet + golangci-lint if installed
```

Go 1.24+ is required. The unit tests do **not** call the real Codex CLI or
OpenAI; they run against `internal/transport/testdata/fake-codex`, a small
shell script that emits the same JSONL protocol.

To exercise the real CLI, install it (`npm install -g @openai/codex`), log in
(`codex login`) or export `CODEX_API_KEY`, and run any example:

```bash
go run ./examples/simple_run "What does this repo do?"
make test-live   # build-tagged smoke tests in internal/livetest (costs tokens)
```

CI never runs these; they are local-only and cost tokens.

## Guidelines

- Keep the public API aligned with the official TypeScript SDK
  (`@openai/codex-sdk`). New options should map 1:1 to CLI flags or
  `--config` keys; see `docs/PARITY.md`.
- Argument ordering in `internal/transport/args.go` is load-bearing: it
  defines precedence between structured config, raw overrides, SDK-managed
  settings and thread options. Add a test in `args_test.go` for any change.
- Every exported identifier gets a doc comment.
- Add or update tests, and note user-facing changes in `CHANGELOG.md` under
  "Unreleased".
- Commit messages follow `type: subject` (`feat`, `fix`, `docs`, `refactor`,
  `test`, `chore`).

## Releasing

1. Update `VERSION` and move the "Unreleased" section in `CHANGELOG.md` to the new version.
2. Open a PR (`main` is protected: PR + green CI required), merge it, then tag
   the merge commit: `git tag v0.x.y && git push origin v0.x.y`.
3. The release workflow verifies the tag matches `VERSION`, runs tests, and
   publishes a GitHub release with the changelog section.
