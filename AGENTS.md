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
mise install      # install pinned toolchain AND wire up git hooks
```

`mise install` runs a `postinstall` hook that also installs the prek git hooks,
so this one command is enough on a fresh clone. Run `prek install` by hand only
if you need to re-wire the hooks.

mise provides the pinned toolchain. If it is activated in your shell the tools
are on `PATH`; otherwise prefix commands with `mise exec --`.

Day-to-day commands:

```sh
go build ./...
go test ./...
go vet ./...
```

> **Working in a git worktree?** `go build`/`go test`/`go vet` may fail with
> `error obtaining VCS status: exit status 128`. Go's VCS stamping does not
> handle a worktree whose checkout lives under the parent repo's tree. Add
> `-buildvcs=false` (e.g. `go build -buildvcs=false ./...`) when running these
> commands from a worktree. Releases are unaffected — goreleaser stamps version
> info via `-ldflags`, not VCS stamping.

The git hooks (installed by `mise install`) cover the rest automatically:
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
