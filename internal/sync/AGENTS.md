# AGENTS.md — internal/sync

Core synchronization logic. Pure library code with no CLI concerns; the entry
point in `cmd/sync-claude-md` wires flags into `Options` and calls `Run`.

## Files

```
constants.go    Target definitions (CLAUDE.md/GEMINI.md), file mode, skip-dir set
targets.go      target / targetFile types, resolveTargets
discover.go     AGENTS.md + target-file discovery (filesystem walk, git, explicit args)
mutate.go       Pure plan/apply: planSync / planCleanup decide a plannedAction; applyAction writes it
sync.go         Run orchestration + Options; planActions (decide all) → applyActions (write all)
precommit.go    pre-commit subcommand: git index checks, RunPreCommit, CheckPreCommit
*_test.go       Per-file unit tests (precommit_test.go uses real git repos)
```

## Architecture notes

- **One generic mutation path, data-driven targets.** `mutate.go` operates on a
  target's reference line (`ref`); the only thing that differs between CLAUDE.md
  and GEMINI.md is the `target{filename, ref}` data in `constants.go`
  (`@AGENTS.md` vs. `@./AGENTS.md`). Do **not** fork the read/write logic per
  agent — add a target as data instead.
- **The default-target decision lives in the CLI, not here.** `resolveTargets`
  is a literal mapping of the `Options.Claude` / `Options.Gemini` booleans to the
  selected targets. `cmd/sync-claude-md` decides the defaults (CLAUDE.md on
  unless `--no-claude`, GEMINI.md only with `--gemini`).
- **`withRefPrepended` is idempotent on presence anywhere.** It adds the reference
  (at the top) only if it is not already present _anywhere_ in the file, so a
  reference moved lower by the user is left untouched.
- **`withRefRemoved` strips the reference anywhere, symmetric with
  `withRefPrepended`.** A line counts as the reference when, trimmed, it equals
  `ref` exactly; every such standalone line is removed wherever it sits (so a
  reference the user moved lower is still cleaned up), along with one blank line
  immediately after each. Lines that merely contain `ref` as a substring are left
  untouched.
- **`planCleanup` no-ops on a missing file.** `planActions` calls it for every
  deleted AGENTS.md across each selected target, so a directory that never had a
  given target file must not produce an action.
- **Plan first, then apply.** `planActions` decides the full set of
  `plannedAction`s without touching disk; `applyActions` writes them. This lets
  `pre-commit` verify before any write happens and means no file is written
  because of a _decision_ that later proves wrong. It is not transactional,
  though: if a later `applyActions` write fails mid-way, earlier writes remain on
  disk.
- **`pre-commit` verifies against the git index, not the worktree.** `precommit.go`
  enforces two independent axes: **destroy protection** (`axisDestroy`, refuse to
  overwrite a target with unstaged changes — cleared by `--force`) and **index
  sync** (`axisSync`, the `@AGENTS.md` reference must be staged so the sync lands
  in the commit — cleared by `--stage` or a manual `git add`). The check looks at
  the staged blob (`git cat-file blob :path`), so a file present on disk but
  untracked still fails — fixing the original "second run silently passes" bug.
  Git-ignored targets are a complete no-op (skipped in `planActions` and
  `CheckPreCommit`).

## Testing

```sh
go test ./internal/sync/...
go test ./internal/sync/... -run Gemini -v   # focus on a subset
```

Each test runs in an isolated temp dir (`setupTestDir` + `chdir`), so the suite
never touches the working tree. Mutation tests in `mutate_test.go` are
table-driven over each target's reference line (`refCases`), so covering a new
target is usually a new table entry rather than new test functions.
