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
# Process staged AGENTS.md files only (default), verifying against the git index
sync-claude-md sync

# Also stage the synced files (succeeds in one pass)
sync-claude-md sync --stage

# Scan entire repository
sync-claude-md sync --all

# Dry-run: check without making changes
sync-claude-md check --all

# Also sync GEMINI.md (@./AGENTS.md) alongside CLAUDE.md
sync-claude-md sync --gemini

# Sync GEMINI.md only
sync-claude-md sync --gemini --no-claude

# Process specific files
sync-claude-md sync path/to/AGENTS.md another/AGENTS.md

# Overwrite a target even if it has unstaged changes
sync-claude-md sync --force

# Also process target files that are git-ignored
sync-claude-md sync --no-ignore

# Exit 1 if anything was written, even after a successful sync/stage
sync-claude-md sync --fail-on-change
```

Running `sync-claude-md` with no command prints help.

**Target flags:**

- `CLAUDE.md` is synced by default
- `--gemini` — also sync `GEMINI.md` (`@./AGENTS.md`) in each directory
- `--no-claude` — skip `CLAUDE.md` (use with `--gemini` to sync `GEMINI.md` only)
- `--no-ignore` — also process target files that are git-ignored (skipped by default)

**With no file arguments**, only staged `AGENTS.md` files are processed — the
intended git-hook use. Pass `--all` to scan the whole repository instead.
Outside a git repository, "staged" is meaningless, so the default falls back
to a full scan too.

**`sync` enforces three guarantees:**

- **Destroy protection** — it refuses to overwrite an existing target file
  that has unstaged changes, which would discard your work, and exits `1`
  without writing. Pass `--force` (`-f`) to overwrite anyway.
- **Outside a git repository**, it refuses to write anything at all — even a
  brand-new file — since there is no git history to recover from, and exits
  `1`. Pass `--force` (`-f`) to write anyway.
- **Index sync** (inside a git repository only) — the `@AGENTS.md` reference
  must be **staged**, so the sync actually lands in the next commit. If it is
  not (including a freshly created but untracked `CLAUDE.md`), it exits `1`
  and asks you to `git add` the file. Pass `--stage` (`-S`) to stage the
  synced files automatically and succeed in a single pass.

| Flag               | Effect                                                                            |
| ------------------ | --------------------------------------------------------------------------------- |
| `--all`            | Scan the entire repository instead of only staged files                           |
| `--stage`, `-S`    | `git add` the synced target files (inside a git repository only)                  |
| `--force`, `-f`    | Overwrite targets with unstaged changes, or write at all outside a git repository |
| `--no-ignore`      | Also process target files that are git-ignored                                    |
| `--fail-on-change` | Exit `1` if any file was written, even after a successful sync/stage              |

> **Note**: `--stage` adds the whole target file, so it does not play well with
> partial staging (`git add -p`). Omit `--stage` and stage manually if you rely
> on partially staged commits.

**Exit codes:**

- `0` — nothing left to do: everything is up to date and (inside a git
  repository) staged
- `1` — a destroy-protection block, a refusal to write outside a git
  repository, an unstaged index-sync violation, or (with `check`) drift; or,
  with `--fail-on-change`, any write at all

### Pre-commit / [prek](https://github.com/pre-commit/prek)

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/lohn/sync-claude-md
    rev: v1.0.0
    hooks:
      - id: sync-claude-md
```

The hook runs `sync-claude-md sync` and, by default, fails the commit when a
synced file is not staged so you re-stage and commit again. To stage the
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
        entry: sync-claude-md sync
        language: system
        always_run: true
        pass_filenames: false
```

### [Husky](https://typicode.github.io/husky/)

See [docs/husky.md](docs/husky.md) for detailed setup instructions.

Quick example for `.husky/pre-commit`:

```bash
sync-claude-md sync --stage
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
- Refuses to read a target file (`CLAUDE.md`/`GEMINI.md`) larger than 10 MiB,
  rather than loading it into memory — in practice you'd never write a
  `CLAUDE.md`/`GEMINI.md` that large, so this has no effect on normal usage

## License

MIT © [lohn](https://github.com/lohn)
