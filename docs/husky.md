# Husky Integration

sync-claude-md works seamlessly with [husky](https://typicode.github.io/husky/) for Git hooks.

## Setup

Install:

```bash
npm install --save-dev sync-claude-md
```

Add to `.husky/pre-commit`:

```bash
# Only staged AGENTS.md files; fails the commit unless the synced CLAUDE.md is staged too
sync-claude-md sync

# Or auto-stage the synced files instead of failing
sync-claude-md sync --stage

# Or scan the whole repository instead of just staged files
sync-claude-md sync --all
```

Pass `--gemini` to also sync `GEMINI.md` (`@./AGENTS.md`), or add `--no-claude`
to sync `GEMINI.md` only. See the [README](../README.md#usage) for the full
flag reference and safety guarantees.

## CI Check

Use `check` in CI to verify sync without writing:

```bash
sync-claude-md check --all
```

Exits `1` if any `CLAUDE.md` is out of sync, on disk or (inside a git
repository) in the index.
