# Changelog

## [1.0.1](https://github.com/lohn/sync-claude-md/compare/v1.0.0...v1.0.1) (2026-06-22)


### Bug Fixes

* **ci:** restore release-please labeling so PR title/body stay in sync ([#44](https://github.com/lohn/sync-claude-md/issues/44)) ([78c06c0](https://github.com/lohn/sync-claude-md/commit/78c06c000168fc68bc1a0e59187e2afd3c5c4416))
* **deps:** give single-update Renovate PRs dependency-specific titles ([#38](https://github.com/lohn/sync-claude-md/issues/38)) ([60e181f](https://github.com/lohn/sync-claude-md/commit/60e181f93f91e579210aba8f579b96dde98f7929))
* fall back to module/VCS build info when version ldflags are unset ([#31](https://github.com/lohn/sync-claude-md/issues/31)) ([c3341a6](https://github.com/lohn/sync-claude-md/commit/c3341a686a653f4492c28065b1a1792db668914b))
* **npm:** correct bin entrypoint path to include .js extension ([#42](https://github.com/lohn/sync-claude-md/issues/42)) ([c2cce04](https://github.com/lohn/sync-claude-md/commit/c2cce046d242429d21b3f4185afe337042284830))
* **release-please:** override default beep-boop PR header ([#45](https://github.com/lohn/sync-claude-md/issues/45)) ([f2e5170](https://github.com/lohn/sync-claude-md/commit/f2e5170151153350ccdd5263283a773060bb3e24))

## 1.0.0 (2026-06-21)

Initial release of `sync-claude-md`, a CLI and pre-commit hook that keeps
`CLAUDE.md` in sync with its sibling `AGENTS.md`, so Claude Code and other AI
coding agents (GitHub Copilot, Cursor, etc.) read consistent instructions.
Pass `--gemini` to do the same for `GEMINI.md`.
