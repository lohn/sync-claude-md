# sync-claude-md

[![CI](https://github.com/lohn/sync-claude-md/actions/workflows/ci.yaml/badge.svg)](https://github.com/lohn/sync-claude-md/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lohn/sync-claude-md)](https://goreportcard.com/report/github.com/lohn/sync-claude-md)
[![npm](https://img.shields.io/npm/v/sync-claude-md.svg)](https://www.npmjs.com/package/sync-claude-md)
[![PyPI](https://img.shields.io/pypi/v/sync-claude-md.svg)](https://pypi.org/project/sync-claude-md/)

> **English** | [日本語](README.ja.md) | [한국어](README.ko.md)

Keeps `CLAUDE.md` in sync with `AGENTS.md` for multi-agent development workflows.

## Why

AI coding agents disagree on which instruction file to read — Claude Code wants
`CLAUDE.md`, most others (GitHub Copilot, Cursor, etc.) want `AGENTS.md`.
Maintaining both by hand is tedious and error-prone.

`sync-claude-md` keeps them in sync automatically: for every `AGENTS.md` it
finds, it ensures a sibling `CLAUDE.md` containing an `@AGENTS.md` reference
exists — creating the file, or adding the reference to an existing one while
preserving its content — and removes the reference (deleting the file if it's
now empty) once `AGENTS.md` is gone. Pass `--gemini` to do the same for
`GEMINI.md` (`@./AGENTS.md`).

Works as a **pre-commit hook** or standalone CLI.

## Installation

### via npm

```bash
npm install --save-dev sync-claude-md
npx sync-claude-md --help
```

### via PyPI

```bash
pip install sync-claude-md
sync-claude-md --help
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
sync-claude-md sync           # sync staged AGENTS.md files (default), verified against the git index
sync-claude-md sync --all     # scan the entire repository instead
sync-claude-md sync --stage   # also stage the synced files (succeeds in one pass)
sync-claude-md check --all    # dry-run: report drift without writing
sync-claude-md sync --gemini  # also sync GEMINI.md (@./AGENTS.md)
```

Running `sync-claude-md` with no command prints help. With no file arguments,
only staged `AGENTS.md` files are processed — the intended git-hook use.
Outside a git repository, "staged" is meaningless, so the default falls back
to a full scan too.

**Flags:**

| Flag               | Effect                                                                            |
| ------------------ | --------------------------------------------------------------------------------- |
| `--all`            | Scan the entire repository instead of only staged files                           |
| `--stage`, `-S`    | `git add` the synced target files (inside a git repository only)                  |
| `--force`, `-f`    | Overwrite targets with unstaged changes, or write at all outside a git repository |
| `--gemini`         | Also sync `GEMINI.md` (`@./AGENTS.md`) in each directory                          |
| `--no-claude`      | Skip `CLAUDE.md` (use with `--gemini` to sync `GEMINI.md` only)                   |
| `--no-ignore`      | Also process target files that are git-ignored (skipped by default)               |
| `--fail-on-change` | Exit `1` if any file was written, even after a successful sync/stage              |

You can also pass specific files instead of relying on `--all`/staged
discovery, e.g. `sync-claude-md sync path/to/AGENTS.md another/AGENTS.md`.

**`sync` enforces three safety guarantees:**

- **Destroy protection** — refuses to overwrite a target file that has
  unstaged changes, which would discard your work; exits `1` without writing
  unless `--force` is passed.
- **No writes outside a git repository** — there's no git history to recover
  from, so it refuses to write anything, even a brand-new file; exits `1`
  unless `--force` is passed.
- **Index sync** (inside a git repository only) — the `@AGENTS.md` reference
  must be **staged** so the sync actually lands in the next commit. If it
  isn't (including a freshly created but untracked `CLAUDE.md`), it exits `1`
  and asks you to `git add` the file. Pass `--stage` to stage the synced
  files automatically and succeed in a single pass.

> **Note**: `--stage` adds the whole target file, so it does not play well
> with partial staging (`git add -p`). Omit `--stage` and stage manually if
> you rely on partially staged commits.

**Exit codes:** `0` when there's nothing left to do (everything up to date
and, inside a git repository, staged); `1` on a guarantee violation above, on
drift detected by `check`, or — with `--fail-on-change` — on any write at all.

### Pre-commit / [prek](https://github.com/pre-commit/prek)

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/lohn/sync-claude-md
    rev: v1.0.0
    hooks:
      - id: sync-claude-md
```

`sync-claude-md` is written in Go, and the default `sync-claude-md` hook uses
`language: golang`, so prek/pre-commit builds it from source on first run —
which requires a Go toolchain. If you'd rather not require Go on every
machine, swap in one of the other hook ids; all are defined in this repo's
[`.pre-commit-hooks.yaml`](.pre-commit-hooks.yaml):

| Hook id                 | Installs via                                | Requires      |
| ----------------------- | ------------------------------------------- | ------------- |
| `sync-claude-md`        | Go toolchain (build from source)            | Go            |
| `sync-claude-md-pip`    | PyPI wheel                                  | Python        |
| `sync-claude-md-npm`    | npm package                                 | Node.js       |
| `sync-claude-md-system` | a `sync-claude-md` binary already on `PATH` | nothing extra |

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

For each `AGENTS.md` found, a `CLAUDE.md` is created in the **same directory**
containing just:

```markdown
@AGENTS.md
```

The `@path/to/file` syntax resolves relative to the `CLAUDE.md` file itself
(not CWD), so `@AGENTS.md` always points to the correct file. With `--gemini`,
a `GEMINI.md` is created the same way using Gemini's import syntax
`@./AGENTS.md`.

It's idempotent and safe:

- Adds the reference (at the top) only if it isn't already present anywhere
  in the file, and preserves all existing content
- Removes the reference automatically when `AGENTS.md` is deleted, and
  deletes the file if that leaves it empty
- Refuses to read a target file larger than 10 MiB, capping how much it will
  ever load into memory at once — no effect on normal-sized files

## License

MIT © [lohn](https://github.com/lohn)
