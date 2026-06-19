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
```

**Target flags:**

- `CLAUDE.md` is synced by default
- `--gemini` — also sync `GEMINI.md` (`@./AGENTS.md`) in each directory
- `--no-claude` — skip `CLAUDE.md` (use with `--gemini` to sync `GEMINI.md` only)

**Exit codes:**

- `0` — everything is up to date
- `1` — changes were made (or would be made in --check mode)

> **Note on partial failures**: When processing multiple files, if an error occurs midway,
> files processed before the error may remain modified. The tool does not roll back changes.
> Review your working directory if an error is reported.

### Pre-commit / [prek](https://github.com/pre-commit/prek)

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/lohn/sync-claude-md
    rev: v1.0.0
    hooks:
      - id: sync-claude-md
```

Or use `repo: local` with a pre-installed binary:

```yaml
repos:
  - repo: local
    hooks:
      - id: sync-claude-md
        name: Sync CLAUDE.md
        entry: sync-claude-md
        language: system
        files: AGENTS\.md$
        pass_filenames: true
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

- Only adds the reference if not already present at the top
- Preserves all existing content
- Removes references automatically when `AGENTS.md` is deleted
- Deletes empty instruction files after cleanup

## License

MIT © [lohn](https://github.com/lohn)
