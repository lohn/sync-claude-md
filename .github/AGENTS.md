# GitHub Actions Conventions

These conventions apply to both workflow files (`workflows/`) and composite actions (`actions/`).

## Composite Actions

Reusable step sequences shared across multiple workflows must be defined as composite actions
under `.github/actions/`. Before adding new steps to a workflow, check `.github/actions/` for
an existing composite action that already provides the required functionality.

## Action Version Pinning

Follow these rules for `uses:` directives.

- **All external actions**, including official GitHub actions (`actions/*`): Pin to a full commit SHA. Append a trailing comment with the corresponding semver version.
  ```yaml
  uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
  ```
- **New actions**: Look up and use the latest available version. Existing pinned versions do not need to be updated proactively.
  - Resolve the latest tag via the GitHub API instead of relying on memory:
    ```bash
    gh api repos/<owner>/<repo>/releases/latest --jq '.tag_name'
    # Example: gh api repos/actions/cache/releases/latest --jq '.tag_name'
    ```
  - When the resolved tag bumps the major version of the action, delegate the
    investigation to a subagent (e.g. `librarian`) instead of inspecting it
    yourself. This delegation is required even when the resolved version
    matches your own training data or prior knowledge — never skip the
    investigation on the assumption that you already know the release contents.
    The subagent must read the release notes for that major (and any
    intervening minor releases) and confirm that no breaking change affects
    this repository before the new version is pinned:
    ```bash
    gh api repos/<owner>/<repo>/releases/tags/<tag> --jq '.body'
    ```
- **Consistency**: If the same action appears multiple times within a workflow file, all occurrences must use the same version.
- **Marketplace Link**: Add a Marketplace link as a comment above the step block (before `- name:` or `- uses:`), not between `name:` and `uses:`.
  ```yaml
  # https://github.com/marketplace/actions/checkout
  - name: Checkout repository
    uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
  ```

## Script Language Hints

When writing inline scripts in `run:` blocks, add a language hint comment on the line immediately before `run: |`.

```yaml
# language=bash
run: |
  echo "Hello, World"
```
