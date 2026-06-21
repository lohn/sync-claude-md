# AGENTS.md — internal/sync

Core synchronization logic. Pure library code with no CLI concerns; the entry
point in `cmd/sync-claude-md` wires each subcommand's flags into `Options` and
calls `Run` for both `sync` and `check` (`Options.Check` switches between
them).

## Files

```
constants.go    Target definitions (CLAUDE.md/GEMINI.md), file mode, skip-dir set
targets.go      target / targetFile types, resolveTargets
discover.go     AGENTS.md + target-file discovery (filesystem walk, git, explicit args)
mutate.go       Pure plan/apply: planSync / planCleanup decide a plannedAction; applyAction writes it
sync.go         Run orchestration + Options + Result; planActions (decide all) → applyActions (write all)
gitstate.go     Git-index/worktree checks: checkDestroy (axisDestroy), checkIndexSync (axisSync), isIgnored, inGitRepo, gitAdd, etc.
*_test.go       Per-file unit tests (gitstate_test.go and parts of sync_test.go use real git repos)
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
- **`plannedAction.wantRef` records the desired ref presence, separately from
  `kind`.** `kind` alone cannot distinguish a satisfied sync from a satisfied
  cleanup — both collapse to `actionNone`. `planSync` always sets `wantRef:
  true`, `planCleanup` always leaves it `false`. `checkIndexSync` relies on
  this to catch a target that is already correct on disk but never staged
  (see below), including orphans cleaned up by `planStaleTargets`.
- **`readTargetFile` (in `mutate.go`) bounds how much of a target file
  `planSync`/`planCleanup` will read.** It rejects a file over
  `maxTargetFileSize` (10 MiB) outright via `Stat`, then reads through an
  `io.LimitReader` rather than a single unbounded `os.ReadFile`, so a file
  that grows after the `Stat` (or whose reported size can't be trusted) still
  reads at most `maxTargetFileSize`+1 bytes. This bounds the read, not the
  memory footprint: `io.ReadAll` still materializes the result as one
  contiguous `[]byte`, so up to ~10 MiB ends up in memory at once either way —
  a deliberate simplification, since 10 MiB is already a safe ceiling and no
  real target file approaches it. A true file-size-independent streaming
  rewrite (line-by-line, write-time transform) was considered and rejected as
  unnecessary complexity for that ceiling. Separately, `isRefLine` rejects any
  line over `maxLineLength` (1 KiB) before running `TrimSpace`/compare on it,
  since the reference itself is at most 12 bytes — this trades correctness on
  a pathological whitespace-padded line (one that would trim down to exactly
  `ref` but exceeds the cap) for not paying full-line-scan cost on arbitrarily
  long lines. Both constants live in `constants.go`. `readTargetFile`'s errors
  deliberately omit the path — every call site (`planActions` in `sync.go`)
  already wraps the returned error with it.
- **`planCleanup` no-ops on a missing file.** `planActions` calls it for every
  deleted AGENTS.md across each selected target, so a directory that never had a
  given target file must not produce an action.
- **Ignored targets are skipped by default, everywhere.** Every site that
  turns an AGENTS.md directory into a target path — the main sync loop, the
  cleanup loop, and `planStaleTargets`'s `--all` orphan sweep — checks
  `!opts.NoIgnore && isIgnored(targetPath)` and skips entirely (no
  `plannedAction` at all) rather than just skipping the write. This is what
  makes `checkIndexSync` correct for ignored targets without any special
  casing there: they are simply never in `actions`. `Options.NoIgnore`
  overrides this everywhere at once.
- **Plan first, then apply.** `planActions` decides the full set of
  `plannedAction`s without touching disk; `applyActions` writes them. This lets
  `Run` verify the git index before any write happens and means no file is
  written because of a _decision_ that later proves wrong. It is not
  transactional, though: if a later `applyActions` write fails mid-way,
  earlier writes remain on disk.
- **Before writing, `Run` picks exactly one of two pre-write guards depending
  on `inGitRepo()` — never both.**
  - Inside a git repository: `checkDestroy` (axisDestroy, in `gitstate.go`) —
    any planned write to an existing file with unstaged changes, which would
    clobber the user's uncommitted work; a create is exempt since there is
    nothing to destroy. Violations populate `Result.DestroyPaths`.
  - Outside a git repository: there is no "unstaged" to check, and no git
    history to recover from at all, so `Run` instead refuses every write
    outright — `anyModifies(actions)` is true → block, full stop, even a
    brand-new file. Violations populate `Result.NoGitPaths` (the full list of
    `modifyingPaths(actions)`).

  Both are cleared by `Options.Force`, and both leave `actions` fully
  unapplied (nothing partially written). Do not add a third "is it safe"
  check elsewhere — gate any new pre-write safety concern through this same
  inside/outside-git branch in `Run`.
- **`checkIndexSync` (axisSync, in `gitstate.go`) is a separate, post-write,
  git-repository-only check.** For every action (including a no-op
  `actionNone`), it asks whether the git index already matches `wantRef`.
  This covers a target already correct on disk but never staged — the
  original "second run silently passes" bug — by checking the staged blob
  (`git cat-file blob :path`) rather than the working tree. Violations
  populate `Result.SyncPaths`, cleared by `Options.Stage` (auto `git add`) or
  a manual `git add`. Outside a git repository this check does not run at
  all — moot, since `Run` already refused to write anything without
  `Options.Force` (and even with `Options.Force`, "staged" still has no
  meaning without a repository).
- **`Options.Check` bypasses both guards and never writes.** It folds
  `anyModifies(actions)` and, inside a git repository, a non-empty
  `checkIndexSync` result into `Result.Changed`. It does not surface
  `DestroyPaths`/`NoGitPaths`, since a content-drift action already implies
  "changed" regardless of whether a real run would later be blocked, and
  `check` has no `Options.Force` concept at all.
- **`findAgentsFiles` falls back to a full scan outside a git repository.**
  When there is no git repository and neither `Options.All` nor
  `Options.Files` was given, the default "staged AGENTS.md" discovery is
  meaningless without an index, so it scans the whole tree instead — the same
  files `--all` would find, just without the orphan-target sweep that
  `Options.All` additionally triggers in `planActions`.

## Testing

```sh
go test ./internal/sync/...
go test ./internal/sync/... -run Gemini -v   # focus on a subset
```

Each test runs in an isolated temp dir (`setupTestDir` + `chdir`), so the suite
never touches the working tree. Mutation tests in `mutate_test.go` are
table-driven over each target's reference line (`refCases`), so covering a new
target is usually a new table entry rather than new test functions.
`initGitRepo`/`runGit` (defined in `gitstate_test.go`) set up a real, isolated
git repo for tests that need one — `sync_test.go`'s destroy-protection tests
use them too rather than duplicating git setup.
