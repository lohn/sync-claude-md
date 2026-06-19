# Contributing

Thanks for your interest in contributing! This document describes the conventions
and workflow for this repository.

This project follows a [Code of Conduct](./CODE_OF_CONDUCT.md). By participating,
you are expected to uphold it.

## Language

**All communication in this repository must be in English.** This includes, but is
not limited to:

- Commit messages
- Issues
- Pull requests (titles and descriptions)
- Code comments and documentation
- Review discussions

English does not have to be your native language, and you are very welcome here
regardless of your fluency. Feel free to use translation tools — we only ask that
your communication is clear and concise. Don't let language hold you back from
contributing.

## Commit messages and pull request titles

**Commit messages and pull request titles must follow the
[Conventional Commits](https://www.conventionalcommits.org/) specification.**

The format is:

```
<type>(<optional scope>): <description>
```

Common types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`,
`ci`, `chore`, `revert`.

Examples:

```
feat: support .config/mise/conf.d/*.toml
fix(mise-lock): skip top-level *.local.toml overrides
docs: document the testing workflow
ci: pin actions to commit SHAs
```

This is enforced by [commitizen](https://commitizen-tools.github.io/commitizen/) via
the pre-commit hooks (`commit-msg` and `pre-push` stages). Since pull requests are
expected to use a Conventional Commit title as well, keep the PR title consistent
with the change.

## Development setup

This project uses [mise](https://mise.jdx.dev) to manage the toolchain and
[pre-commit](https://pre-commit.com) (run via [prek](https://github.com/j178/prek))
for the hooks.

```sh
# Install the pinned toolchain (prek, dprint, commitizen, etc.)
mise install

# Install the git hooks
prek install
```

## Before you open a pull request

- Run the hooks against everything:

  ```sh
  prek run --all-files
  ```

- Run the test suite (see [Testing](#testing)).

`mise.lock` is generated automatically by the `mise-lock` hook whenever a mise
configuration file changes, so commit the updated lockfile together with your
configuration change.

## Project layout

The implementation is entirely in Go; `npm/` and `pypi/` only package and ship
the prebuilt binary.

```
cmd/sync-claude-md/   CLI entry point (flag parsing, target selection, exit codes)
internal/sync/        Core logic, split by responsibility
                      (constants, targets, discover, mutate, sync) with unit tests
npm/, pypi/           Binary-distribution wrappers — no real logic
docs/                 Integration guides (e.g. Husky)
```

See [AGENTS.md](./AGENTS.md) for a more detailed map and architecture notes.

## Testing

The logic in `internal/sync` is covered by Go unit tests
([`internal/sync/*_test.go`](./internal/sync)). Each test runs in an isolated
temporary directory, so the suite never touches your working tree.

```sh
go test ./...            # run all tests
go test ./... -run Gemini -v   # focus on a subset
```

Add or update tests for any behavior change. Mutation tests in `mutate_test.go`
are table-driven over each target's reference line (`@AGENTS.md` for CLAUDE.md,
`@./AGENTS.md` for GEMINI.md), so covering a new target usually means adding a
table entry rather than new test functions.

Before pushing, make sure the build, vet, and linters are clean:

```sh
go build ./...
go vet ./...
golangci-lint run
```

## License

By contributing, you agree that your contributions will be licensed under the
[MIT License](./LICENSE).
