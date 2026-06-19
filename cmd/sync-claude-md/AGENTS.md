# AGENTS.md — cmd/sync-claude-md

CLI entry point. Parses flags, decides which targets to sync, and maps them into
`sync.Options` before calling `sync.Run`. All core logic lives in
[`internal/sync`](../../internal/sync/AGENTS.md); keep this package thin.

## Responsibilities

- **Flag parsing and the default-target decision.** CLAUDE.md is synced by
  default; `--gemini` adds GEMINI.md; `--no-claude` opts CLAUDE.md out
  (`--no-claude` without `--gemini` is an error — nothing to sync).
- **Exit codes.** `0` when everything is up to date; `1` when changes were made
  (or, with `--check`, when drift is detected) so it stops a commit and prompts a
  re-stage.
- **Usage text.** `usageHeader` / `usageExamples` plus a custom `flag.Usage`. Keep
  the examples aligned (descriptions start at the same column) and in sync with
  the actual flags.

## Notes

- `version` / `commit` / `date` are injected at build time via `-ldflags` (see
  `.goreleaser.yaml`); leave the defaults as `dev` / `none` / `unknown`.
- User-facing flag or behavior changes must be reflected in the three READMEs and
  `docs/husky.md`.
