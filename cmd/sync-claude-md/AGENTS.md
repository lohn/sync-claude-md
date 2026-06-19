# AGENTS.md — cmd/sync-claude-md

CLI entry point. Parses flags, decides which targets to sync, and maps them into
`sync.Options` before calling `sync.Run` (or `sync.RunPreCommit` for the
`pre-commit` subcommand). All core logic lives in
[`internal/sync`](../../internal/sync/AGENTS.md); keep this package thin.

## Responsibilities

- **Flag parsing and the default-target decision.** CLAUDE.md is synced by
  default; `--gemini` adds GEMINI.md; `--no-claude` opts CLAUDE.md out
  (`--no-claude` without `--gemini` is an error — nothing to sync). The shared
  `selectTargets` helper makes this decision for both the top-level command and
  the subcommand.
- **The `pre-commit` subcommand.** `os.Args[1] == "pre-commit"` dispatches to
  `runPreCommit`, which has its own `flag.FlagSet` adding `--stage` (auto
  `git add`) and `--force`/`-f` (overwrite targets with unstaged changes). It
  calls `sync.RunPreCommit` and turns the returned `PreCommitResult` into
  messages and an exit code. The package owns all git/IO; this layer only formats.
- **Exit codes.** Top-level: `0` up to date, `1` when changes were made (or, with
  `--check`, drift detected) so it stops a commit and prompts a re-stage.
  `pre-commit`: `1` on a destroy-protection block (unstaged changes, no
  `--force`) or an index-sync violation (reference not staged, no `--stage`);
  `0` once the reference is staged. With `--stage` it stages and returns `0` in
  one pass.
- **Usage text.** `usageHeader` / `usageExamples` and `preCommitUsageHeader` plus
  custom `flag.Usage`. Keep the examples aligned (descriptions start at the same
  column) and in sync with the actual flags.

## Notes

- `version` / `commit` / `date` are injected at build time via `-ldflags` (see
  `.goreleaser.yaml`); leave the defaults as `dev` / `none` / `unknown`.
- User-facing flag or behavior changes must be reflected in the three READMEs and
  `docs/husky.md`.
