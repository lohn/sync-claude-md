# AGENTS.md — cmd/sync-claude-md

CLI entry point. `main` dispatches on `os.Args[1]` to one of two subcommands —
`sync`, `check` — each with its own `flag.FlagSet`, mapping flags into
`sync.Options` before calling `sync.Run`. All core logic lives in
[`internal/sync`](../../internal/sync/AGENTS.md); keep this package thin. No
subcommand (or `-h`/`--help`/`help`) prints `helpText`; an unrecognized first
argument prints an error plus `helpText` and exits 1.

## Responsibilities

- **Subcommand dispatch and flag parsing.** `runSync` and `runCheck` each
  build their own `flag.FlagSet` and return the process exit code; `main`
  calls `os.Exit` on the result. `bindCommonFlags` registers the
  `--all`/`--gemini`/`--no-claude`/`--no-ignore` flags shared by both;
  `sync` additionally has `--force`/`-f`, `--stage`/`-S`, and
  `--fail-on-change`, none of which apply to `check` (which never writes).
- **The default-target decision.** CLAUDE.md is synced by default; `--gemini`
  adds GEMINI.md; `--no-claude` opts CLAUDE.md out (`--no-claude` without
  `--gemini` is an error — nothing to sync). The shared `selectTargets` helper
  makes this decision for both subcommands.
- **`sync` maps `sync.Result` to messages and an exit code; the package owns
  all git/IO, this layer only formats.** In order: a non-empty
  `DestroyPaths` blocks (refused to overwrite unstaged work) and prints the
  `--force` hint; a non-empty `SyncPaths` (inside a git repository, the
  reference is not staged) prints the `git add` hint and `--stage` hint;
  otherwise, if `--fail-on-change` was passed and `Result.Wrote` is true, exit
  1 anyway — this check runs last and never blocks a write or a stage, it only
  changes the final exit code (see [`internal/sync`](../../internal/sync/AGENTS.md)
  for what populates each field). `--fail-on-change` itself is CLI-only:
  `sync.Options` has no such field, since it does not change what `Run` does,
  only how the CLI reports it.
- **Exit codes.** `sync`: `0` once nothing is left to do (including after a
  successful `--stage`); `1` on a destroy-protection block, an index-sync
  violation (reference not staged, no `--stage`), or `--fail-on-change` after
  a write. `check`: `0` in sync (on disk and, inside a git repository, in the
  git index), `1` on any drift.
- **Usage text.** Each subcommand sets its own `fs.Usage` to a header constant,
  `printFlags(fs, aliases)`, then an examples constant — one aligned line per
  flag, collapsing shorthand aliases (`-f`, `-S`) onto their long form via the
  `aliases` map. Keep the examples aligned (descriptions start at the same
  column) and in sync with the actual flags. `helpText` (no subcommand) is a
  plain string, not tied to a `flag.FlagSet`.

## Notes

- `version` / `commit` / `date` are injected at build time via `-ldflags` (see
  `.goreleaser.yaml`); leave the defaults as `dev` / `none` / `unknown`.
- User-facing flag or behavior changes must be reflected in the three READMEs,
  `docs/husky.md`, and `.pre-commit-hooks.yaml`/`.pre-commit-config.yaml`.
