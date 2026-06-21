# Husky Integration

sync-claude-md works seamlessly with [husky](https://typicode.github.io/husky/) for Git hooks.

## Setup

### 1. Install sync-claude-md

```bash
# Via npm (recommended for Node.js projects)
npm install --save-dev sync-claude-md

# Or download binary from GitHub Releases
# https://github.com/lohn/sync-claude-md/releases
```

### 2. Configure husky

#### Option A: Staged files only (recommended)

Only process AGENTS.md files that are staged for commit:

```bash
# .husky/pre-commit
sync-claude-md sync
```

With no file arguments, `sync` processes only staged `AGENTS.md` files and
verifies the result against the git index, so the commit is stopped unless the
synced `CLAUDE.md` is staged too. Add `--stage` to stage the synced files
automatically instead of failing:

```bash
# .husky/pre-commit
sync-claude-md sync --stage
```

#### Option B: Full repository scan

Scan all AGENTS.md files in the repository:

```bash
# .husky/pre-commit
sync-claude-md sync --all
```

### 3. Behavior

- If `AGENTS.md` exists but `CLAUDE.md` doesn't → creates `CLAUDE.md` with `@AGENTS.md`
- If `CLAUDE.md` exists without `@AGENTS.md` → adds it at the top
- If `AGENTS.md` is deleted → removes `@AGENTS.md` reference from `CLAUDE.md`
- If `CLAUDE.md` becomes empty → deletes the file
- If `sync` would overwrite a target file that has unstaged changes → exits 1
  without writing instead (pass `--force`/`-f` to overwrite anyway)
- Inside a git repository, it also exits 1 when a synced file is not staged in
  the git index (so the sync is guaranteed to land in the commit); pass
  `--stage` to stage automatically

Pass `--gemini` to also sync a `GEMINI.md` (`@./AGENTS.md`) in each directory,
or `--no-claude` (with `--gemini`) to sync `GEMINI.md` only.

## CI Check

Use the `check` subcommand in CI to verify sync without making changes:

```bash
# .github/workflows/ci.yaml or similar
sync-claude-md check --all
```

Exits with code 1 if any CLAUDE.md is out of sync, on disk or (inside a git
repository) in the git index.
