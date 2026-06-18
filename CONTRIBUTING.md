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

## Testing

The routing logic in `mise-lock.sh` is covered by [bats](https://github.com/bats-core/bats-core)
tests in [`test/`](./test). They stub `mise` on `PATH` so each test asserts which
`mise lock` calls the script makes, without invoking a real toolchain.

`mise-lock.sh` must run across the bash versions our users have — from bash 3.2 (the
version shipped with macOS) to the latest. So [`test/run-bats.sh`](./test/run-bats.sh)
runs the suite inside a `bash:<version>` Docker image, and CI runs it as a matrix over
**bash 3.2, 4.4, and 5.3**.

Run it locally (requires [Docker](https://www.docker.com); no local bats install
needed). `mise run test` runs every supported bash version; the per-version tasks run
just one:

```sh
mise run test          # all of the below
mise run test:bash32   # bash 3.2
mise run test:bash44   # bash 4.4
mise run test:bash53   # bash 5.3
```

If you touch `mise-lock.sh`, keep it free of bash 4+ only features (associative
arrays, etc.) so the bash 3.2 job stays green.

## License

By contributing, you agree that your contributions will be licensed under the
[MIT License](./LICENSE).
