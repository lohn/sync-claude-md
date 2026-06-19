# AGENTS.md — internal/sync

Core synchronization logic. Pure library code with no CLI concerns; the entry
point in `cmd/sync-claude-md` wires flags into `Options` and calls `Run`.

## Files

```
constants.go    Target definitions (CLAUDE.md/GEMINI.md), file mode, skip-dir set
targets.go      target / targetFile types, resolveTargets
discover.go     AGENTS.md + target-file discovery (filesystem walk, git, explicit args)
mutate.go       Single-file create / update / remove, parameterized by reference line
sync.go         Run orchestration + Options
*_test.go       Per-file unit tests
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
- **`updateTarget` is idempotent on presence anywhere.** It adds the reference
  (at the top) only if it is not already present _anywhere_ in the file, so a
  reference moved lower by the user is left untouched.
- **`removeRef` strips the reference anywhere, symmetric with `updateTarget`.** A
  line counts as the reference when, trimmed, it equals `ref` exactly; every such
  standalone line is removed wherever it sits (so a reference the user moved lower
  is still cleaned up), along with one blank line immediately after each. Lines
  that merely contain `ref` as a substring are left untouched.
- **`removeRef` no-ops on a missing file.** `Run` calls it for every deleted
  AGENTS.md across each selected target, so a directory that never had a given
  target file must not error.

## Testing

```sh
go test ./internal/sync/...
go test ./internal/sync/... -run Gemini -v   # focus on a subset
```

Each test runs in an isolated temp dir (`setupTestDir` + `chdir`), so the suite
never touches the working tree. Mutation tests in `mutate_test.go` are
table-driven over each target's reference line (`refCases`), so covering a new
target is usually a new table entry rather than new test functions.
