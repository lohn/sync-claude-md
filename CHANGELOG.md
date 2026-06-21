# Changelog

## [0.1.2](https://github.com/lohn/sync-claude-md/compare/v1.0.0...v0.1.2) (2026-06-21)


### ⚠ BREAKING CHANGES

* merge pre-commit into sync, add index check to check ([#23](https://github.com/lohn/sync-claude-md/issues/23))
* split CLI into sync/check subcommands with unstaged-change guard ([#22](https://github.com/lohn/sync-claude-md/issues/22))

### Features

* add opt-in GEMINI.md sync via --gemini ([#14](https://github.com/lohn/sync-claude-md/issues/14)) ([114769e](https://github.com/lohn/sync-claude-md/commit/114769e69b60be9a291630785ce96c916b35d34e))
* add pip/npm/system pre-commit hook variants and self-sync ([873996c](https://github.com/lohn/sync-claude-md/commit/873996cc47c0d2fa4926c950dbaaf804028b0470))
* add pre-commit subcommand with index-based staged verification ([#21](https://github.com/lohn/sync-claude-md/issues/21)) ([302974c](https://github.com/lohn/sync-claude-md/commit/302974cf8eecc57b42c25502ba95f3c14f62bebe))
* add project config files and husky integration docs ([89535d5](https://github.com/lohn/sync-claude-md/commit/89535d5e35fb5a9c4fbfd03b7b302569fa276cb6))
* add sync-claude-md CLI and AGENTS.md sync logic ([a4aa9b4](https://github.com/lohn/sync-claude-md/commit/a4aa9b43fe315160b854a2ea9f09ec16f7843da8))
* bound target-file read size and per-line length ([#24](https://github.com/lohn/sync-claude-md/issues/24)) ([40ccc75](https://github.com/lohn/sync-claude-md/commit/40ccc75399490557f5f5ceb0f332d3d170b040c1))
* install prek git hooks via mise postinstall hook ([#18](https://github.com/lohn/sync-claude-md/issues/18)) ([50e3413](https://github.com/lohn/sync-claude-md/commit/50e3413f1b0aee3818424c0b9aa46d3369082630))
* merge pre-commit into sync, add index check to check ([#23](https://github.com/lohn/sync-claude-md/issues/23)) ([b03df3e](https://github.com/lohn/sync-claude-md/commit/b03df3e4d823206055810456ef04bf3fdffbbef9))
* split CLI into sync/check subcommands with unstaged-change guard ([#22](https://github.com/lohn/sync-claude-md/issues/22)) ([f99da9b](https://github.com/lohn/sync-claude-md/commit/f99da9b4caa21e2578fc14bad5a6a4ece1b747bb))


### Bug Fixes

* **release:** add version-bearing PR title pattern ([5902a02](https://github.com/lohn/sync-claude-md/commit/5902a0250f759e5c1b1faa20be3d134137d98c0c))
* **release:** bump npm and pypi versions via extra-files ([34576ea](https://github.com/lohn/sync-claude-md/commit/34576ea083bad61f8468a4928257769bd9f560cb))
* **release:** drop component/package-name so grouped release PR matches ([469aeb1](https://github.com/lohn/sync-claude-md/commit/469aeb1332bed687549afdf2f0df0a423178dea6))
* **release:** fix package README links and empty release body ([7ab12df](https://github.com/lohn/sync-claude-md/commit/7ab12df7051f650e9e0d22112be779291815f892))
* **release:** let goreleaser own the GitHub release, not release-please ([52cb737](https://github.com/lohn/sync-claude-md/commit/52cb737a54c278eebec0038d762a011a4e5ad83e))
* **release:** publish via draft release to satisfy immutable releases ([46be0b6](https://github.com/lohn/sync-claude-md/commit/46be0b6f6769913f8c2402b86129fa4e80b335cc))
* **release:** set grouped PR title pattern to carry version ([1f58130](https://github.com/lohn/sync-claude-md/commit/1f58130153f8aed0e378843c9627525bcb24a020))
* **release:** split release-please and tag-triggered release workflows ([5309d7a](https://github.com/lohn/sync-claude-md/commit/5309d7a9d6c2af9cf108e0a7fd54aa584455aae6))
* **release:** use correct PyPI project name in failure cleanup ([7e2cf89](https://github.com/lohn/sync-claude-md/commit/7e2cf8964d941d23c7ae85fdd13a130bb1759737))
* **release:** write release notes outside the repo for a clean tree ([8665de9](https://github.com/lohn/sync-claude-md/commit/8665de93549faa609341c9229903e89802bd0636))
* remove AGENTS.md reference wherever it appears in target file ([#19](https://github.com/lohn/sync-claude-md/issues/19)) ([534d9eb](https://github.com/lohn/sync-claude-md/commit/534d9eb5ef2052f3b699fa97144204ea494a13bd))
* **renovate:** apply npm versioning to all mise tools, not just github-tags ([89bc211](https://github.com/lohn/sync-claude-md/commit/89bc211c1d5984cd2380884005bfcc847b15d7b6))
* **renovate:** keep mise.toml partial pins, refresh mise.lock via maintenance ([bc5bd9e](https://github.com/lohn/sync-claude-md/commit/bc5bd9e525b8025228158bd3d91ead510c82aea7))


### Miscellaneous Chores

* release 0.1.0 ([f925a82](https://github.com/lohn/sync-claude-md/commit/f925a822284dea04940da74c0b81875261fd563c))

## 1.0.0 (2026-06-21)

Initial release of `sync-claude-md`, a CLI and pre-commit hook that keeps
`CLAUDE.md` in sync with its sibling `AGENTS.md`, so Claude Code and other AI
coding agents (GitHub Copilot, Cursor, etc.) read consistent instructions.
Pass `--gemini` to do the same for `GEMINI.md`.
