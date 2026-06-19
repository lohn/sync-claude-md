#!/usr/bin/env bash
#MISE description="Unpublish every npm package for a version (skips versions not published)"
#MISE confirm={ message = "This unpublishes every npm package for the given version. Continue?", default = "no" }
#USAGE arg "[version]" help="Version to unpublish; defaults to the release-please manifest version"
#
# Usage: mise run unpublish-npm [version]
#   version defaults to the release-please manifest version.
#
# Used to abandon a failed/aborted release: npm registry versions are immutable,
# so a half-published release is cleaned up by unpublishing it and moving on to
# the next version. Packages that do not have the version published are skipped.
#
# Requires being logged in to npm (`npm login`); unpublish is only allowed
# within 72 hours of publishing.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

version="${usage_version:-$(jq -r '.["."]' .release-please-manifest.json)}"

if ! npm whoami >/dev/null 2>&1; then
  echo "error: not logged in to npm; run 'npm login' first" >&2
  exit 1
fi

echo "Unpublishing version $version (skipping packages where it is not published)"

# Read the package list up front so the loop does not redirect stdin; npm needs
# the terminal (tty) to run its interactive OTP/2FA flow during unpublish.
mapfile -t manifests < <(fd --type f '^package\.json$' npm)

for manifest in "${manifests[@]}"; do
  name="$(jq -r '.name' "$manifest")"
  spec="$name@$version"
  if npm view "$spec" version >/dev/null 2>&1; then
    echo "→ unpublishing $spec"
    npm unpublish "$spec" --force
  else
    echo "→ skip $spec (not published)"
  fi
done
