# sync-claude-md

[![CI](https://github.com/lohn/sync-claude-md/actions/workflows/ci.yaml/badge.svg)](https://github.com/lohn/sync-claude-md/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lohn/sync-claude-md)](https://goreportcard.com/report/github.com/lohn/sync-claude-md)

> **English** | [日本語](README.ja.md) | [한국어](README.ko.md)

Automatically sync `CLAUDE.md` with `AGENTS.md` for multi-agent development workflows.

## The Problem

Different AI coding agents use different instruction files:

- **Claude Code** reads `CLAUDE.md`
- **Other agents** (GitHub Copilot, Cursor, etc.) read `AGENTS.md`

Managing both files manually is tedious and error-prone, especially in teams with multiple developers.

## The Solution

This tool automatically:

1. **Creates** `CLAUDE.md` with `@AGENTS.md` reference when `AGENTS.md` exists
2. **Updates** existing `CLAUDE.md` by adding the reference at the top
3. **Cleans up** references when `AGENTS.md` is deleted
4. **Preserves** all existing content in `CLAUDE.md`

Works as a **pre-commit hook** or standalone CLI.

## Installation

### via npm (Node.js projects)

```bash
npm install --save-dev sync-claude-md
npx sync-claude-md --help
```

### via GitHub Releases

Download the binary for your platform from [Releases](https://github.com/lohn/sync-claude-md/releases).

### via Go

```bash
go install github.com/lohn/sync-claude-md/cmd/sync-claude-md@latest
```

## Usage

### CLI

```bash
# Process staged AGENTS.md files only (default)
sync-claude-md

# Scan entire repository
sync-claude-md --all

# Dry-run: check without making changes
sync-claude-md --check

# Also sync GEMINI.md (@./AGENTS.md) alongside CLAUDE.md
sync-claude-md --gemini

# Sync GEMINI.md only
sync-claude-md --gemini --no-claude

# Process specific files
sync-claude-md path/to/AGENTS.md another/AGENTS.md

# Pre-commit mode: sync staged AGENTS.md and verify against the git index
sync-claude-md pre-commit

# Pre-commit mode: also stage the synced files (succeeds in one pass)
sync-claude-md pre-commit --stage
```

**Target flags:**

- `CLAUDE.md` is synced by default
- `--gemini` — also sync `GEMINI.md` (`@./AGENTS.md`) in each directory
- `--no-claude` — skip `CLAUDE.md` (use with `--gemini` to sync `GEMINI.md` only)

**Exit codes:**

- `0` — everything is up to date
- `1` — changes were made (or would be made in --check mode)

### `pre-commit` subcommand

`sync-claude-md pre-commit` syncs the staged `AGENTS.md` files and then verifies
the result against the **git index** (not just the working tree). It enforces
two guarantees:

- **Sync** — the `@AGENTS.md` reference must be **staged**, so the sync actually
  lands in the commit. If it is not staged (including a freshly created but
  untracked `CLAUDE.md`), the commit is stopped with exit `1` and you are asked
  to `git add` the file. Pass `--stage` to stage the synced files automatically
  and succeed in a single pass (exit `0`).
- **Destroy protection** — it refuses to overwrite a target file that has
  unstaged changes, which would discard your in-progress work, and exits `1`
  without writing. Pass `--force` (`-f`) to override. (Running under
  pre-commit/prek this rarely triggers, since the framework stashes unstaged
  changes first; it mainly protects manual runs.)

| Flag                       | Effect                                                  |
| -------------------------- | ------------------------------------------------------- |
| `--stage`                  | `git add` the synced target files; exit `0` in one pass |
| `--force`, `-f`            | Overwrite targets even if they have unstaged changes    |
| `--gemini` / `--no-claude` | Same target selection as the top-level command          |

> **Note**: `--stage` adds the whole target file, so it does not play well with
> partial staging (`git add -p`). Omit `--stage` and stage manually if you rely
> on partially staged commits.

### Pre-commit / [prek](https://github.com/pre-commit/prek)

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/lohn/sync-claude-md
    rev: v1.0.0
    hooks:
      - id: sync-claude-md
```

The hook runs `sync-claude-md pre-commit` and, by default, fails the commit when
a synced file is not staged so you re-stage and commit again. To stage the
synced files automatically instead, add `args: ['--stage']`:

```yaml
repos:
  - repo: https://github.com/lohn/sync-claude-md
    rev: v1.0.0
    hooks:
      - id: sync-claude-md
        args: ["--stage"]
```

Or use `repo: local` with a pre-installed binary:

```yaml
repos:
  - repo: local
    hooks:
      - id: sync-claude-md
        name: Sync CLAUDE.md
        entry: sync-claude-md pre-commit
        language: system
        always_run: true
        pass_filenames: false
```

### [Husky](https://typicode.github.io/husky/)

See [docs/husky.md](docs/husky.md) for detailed setup instructions.

Quick example for `.husky/pre-commit`:

```bash
STAGED_AGENTS=$(git diff --cached --name-only --diff-filter=ACMR | grep -E 'AGENTS\.md$' || true)
if [ -n "$STAGED_AGENTS" ]; then
  echo "$STAGED_AGENTS" | xargs sync-claude-md
fi
```

## How It Works

### Per-Directory Strategy

For each `AGENTS.md` found, a `CLAUDE.md` is created in the **same directory** with:

```markdown
@AGENTS.md
```

The `@path/to/file` syntax is resolved relative to the `CLAUDE.md` file itself (not CWD), so `@AGENTS.md` always points to the correct file.

With `--gemini`, a `GEMINI.md` is created the same way using Gemini's import syntax `@./AGENTS.md`.

### Idempotent & Safe

- Adds the reference (at the top) only if it isn't already present anywhere in the file
- Preserves all existing content
- Removes references automatically when `AGENTS.md` is deleted
- Deletes empty instruction files after cleanup

## License

MIT © [lohn](https://github.com/lohn)
