# AGENTS.md

Guidance for AI coding agents working in this repository.

## What this project is

`sync-claude-md` is a small Go CLI (and pre-commit hook) that keeps each
`CLAUDE.md` in sync with its sibling `AGENTS.md`. For every directory containing
an `AGENTS.md`, it ensures a `CLAUDE.md` referencing it (`@AGENTS.md`) exists,
and removes the reference (deleting the file if empty) when `AGENTS.md` is gone.
With `--gemini`, it does the same for `GEMINI.md` using Gemini's import syntax
(`@./AGENTS.md`).

## Repository layout

```
cmd/sync-claude-md/main.go   CLI entry point: flag parsing, target selection, exit codes
internal/sync/               Core logic, split by responsibility:
  constants.go               Target definitions (CLAUDE.md/GEMINI.md), file mode, skip-dir set
  targets.go                 target / targetFile types, resolveTargets
  discover.go                AGENTS.md + target-file discovery (filesystem walk, git, explicit args)
  mutate.go                  Single-file create / update / remove (parameterized by reference line)
  sync.go                    Run orchestration + Options
  *_test.go                  Per-file unit tests
npm/, pypi/                  Binary-distribution wrappers only — no real logic lives here
docs/husky.md                Husky integration guide
README.md, README.ja.md, README.ko.md   User docs (keep all three in sync)
```

`npm/` and `pypi/` package and ship the prebuilt binary for their ecosystems;
the actual implementation is entirely in `cmd/` and `internal/`.

## Architecture notes

- `mutate.go` is generic over a target's reference line (`ref`). The only
  difference between CLAUDE.md and GEMINI.md is the `target{filename, ref}` data
  in `constants.go` — do not fork the read/write logic per agent.
- The default-target decision lives in the **CLI** (`main.go`): CLAUDE.md is on
  by default, GEMINI.md is opt-in via `--gemini`, `--no-claude` opts CLAUDE.md
  out. `resolveTargets` is a literal mapping of the `Options` booleans.

## Development

This project uses [mise](https://mise.jdx.dev) for the toolchain and
[pre-commit](https://pre-commit.com) (run via [prek](https://github.com/j178/prek)).

```sh
mise install      # install pinned Go toolchain, linters, etc.
prek install      # install git hooks
```

Common commands (run directly — the PATH is already configured):

```sh
go build ./...
go test ./...
go vet ./...
golangci-lint run         # lint
golangci-lint fmt         # format (gofumpt + goimports)
prek run --all-files      # run every hook
```

## Conventions

- **Commit messages and PR titles must follow [Conventional Commits](https://www.conventionalcommits.org/)**
  (`feat`, `fix`, `docs`, `refactor`, `test`, `chore`, …). Enforced by commitizen
  via the `commit-msg` / `pre-push` hooks. See [CONTRIBUTING.md](./CONTRIBUTING.md).
- All repository communication (commits, PRs, comments, code) is in **English**.
- Go code is formatted with **gofumpt + goimports** (local prefix
  `github.com/lohn/sync-claude-md`); non-Go files with **dprint**. Let the hooks
  format — don't hand-format.
- Keep the three READMEs (`README.md`, `README.ja.md`, `README.ko.md`) consistent
  when changing user-facing behavior.
- Add or update unit tests in `internal/sync/*_test.go` for behavior changes;
  mutation tests are table-driven over each target's reference line.

## Release

Versioning/changelog is handled by release-please; releases are built and
published by goreleaser (binaries to GitHub Releases, plus npm and PyPI
packages). Do not bump versions by hand.
