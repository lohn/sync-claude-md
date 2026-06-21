# AGENTS.md — cmd/sync-claude-md

CLI entry point. `main` dispatches on `os.Args[1]` to one of three subcommands —
`sync`, `check`, `pre-commit` — each with its own `flag.FlagSet`, mapping flags
into `sync.Options` before calling `sync.Run` (`sync.RunPreCommit` for
`pre-commit`). All core logic lives in
[`internal/sync`](../../internal/sync/AGENTS.md); keep this package thin. No
subcommand (or `-h`/`--help`/`help`) prints `helpText`; an unrecognized first
argument prints an error plus `helpText` and exits 1.

## Responsibilities

- **Subcommand dispatch and flag parsing.** `runSync`, `runCheck`, and
  `runPreCommit` each build their own `flag.FlagSet` and return the process exit
  code; `main` calls `os.Exit` on the result. `bindCommonFlags` registers the
  `--all`/`--gemini`/`--no-claude` flags shared by `sync` and `check`;
  `pre-commit` adds its own (it has no `--all`, since it always operates on
  staged `AGENTS.md`).
- **The default-target decision.** CLAUDE.md is synced by default; `--gemini`
  adds GEMINI.md; `--no-claude` opts CLAUDE.md out (`--no-claude` without
  `--gemini` is an error — nothing to sync). The shared `selectTargets` helper
  makes this decision for all three subcommands.
- **Destroy protection (`sync` and `pre-commit`).** Both pass `--force`/`-f`
  through as `Options.Force`. `sync` calls `sync.Run`, which returns a
  `*sync.DestroyError` when it refuses to overwrite a target with unstaged
  changes; `runSync` type-asserts via `errors.As` to print the path list and the
  `--force` hint, the same wording `runPreCommit` already uses for
  `PreCommitResult.DestroyPaths`. Do not let the two message strings drift apart.
- **The `pre-commit` subcommand.** Adds `--stage`/`-S` (auto `git add`) on top of
  the shared destroy protection, and verifies the result against the git index
  (see [`internal/sync`](../../internal/sync/AGENTS.md)). It calls
  `sync.RunPreCommit` and turns the returned `PreCommitResult` into messages and
  an exit code. The package owns all git/IO; this layer only formats.
- **Exit codes.** `sync`: `0` up to date, `1` when changes were made or a
  destroy-protection block occurred. `check`: `0` in sync, `1` on drift.
  `pre-commit`: `1` on a destroy-protection block or an index-sync violation
  (reference not staged, no `--stage`); `0` once the reference is staged. With
  `--stage` it stages and returns `0` in one pass.
- **Usage text.** Each subcommand sets its own `fs.Usage` to a header constant,
  `printFlags(fs, aliases)`, then an examples constant — one aligned line per
  flag, collapsing shorthand aliases (`-f`, `-S`) onto their long form via the
  `aliases` map. Keep the examples aligned (descriptions start at the same
  column) and in sync with the actual flags. `helpText` (no subcommand) is a
  plain string, not tied to a `flag.FlagSet`.

## Notes

- `version` / `commit` / `date` are injected at build time via `-ldflags` (see
  `.goreleaser.yaml`); leave the defaults as `dev` / `none` / `unknown`.
- User-facing flag or behavior changes must be reflected in the three READMEs and
  `docs/husky.md`.
