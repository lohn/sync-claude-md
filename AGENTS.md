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
cmd/sync-claude-md/   CLI entry point        → see cmd/sync-claude-md/AGENTS.md
internal/sync/        Core logic             → see internal/sync/AGENTS.md
npm/, pypi/           Binary-distribution wrappers — no real logic
docs/                 Integration guides (e.g. Husky)
README.md, README.ja.md, README.ko.md        User docs (keep all three in sync)
```

The implementation lives entirely in `cmd/` and `internal/`. `npm/` and `pypi/`
only package and ship the prebuilt binary for their ecosystems. Each code
directory has its own `AGENTS.md` with the details for that package.

## Development

This project uses [mise](https://mise.jdx.dev) for the toolchain and
[pre-commit](https://pre-commit.com) (run via [prek](https://github.com/j178/prek)).

```sh
mise install      # install pinned Go toolchain, linters, etc.
prek install      # install git hooks
```

mise provides the pinned toolchain. If it is activated in your shell the tools
are on `PATH`; otherwise prefix commands with `mise exec --`.

Day-to-day commands:

```sh
go build ./...
go test ./...
go vet ./...
```

The git hooks (assuming `prek install`) cover the rest automatically:
formatting, `golangci-lint`, and `go mod tidy` on `pre-commit`, and
`go test ./...` on `pre-push`. To run the pre-commit hooks on demand:

```sh
prek run          # against staged files
```

`golangci-lint run` / `golangci-lint fmt` are still handy while iterating.

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
- Add or update unit tests in `internal/sync/*_test.go` for behavior changes.

## Release

Versioning/changelog is handled by release-please; releases are built and
published by goreleaser (binaries to GitHub Releases, plus npm and PyPI
packages). Do not bump versions by hand.
