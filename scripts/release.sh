#!/usr/bin/env bash
set -euo pipefail

# PiHolster — release tagger.
# Usage: bash scripts/release.sh v1.2.3
#
# Checks that the working tree is clean, creates a signed annotated tag,
# and pushes it to origin. The CI release-binary workflow picks up the tag
# and builds cross-compiled binaries automatically.

VERSION="${1:-}"

if [ -z "$VERSION" ]; then
    echo "ERROR: version argument required." >&2
    echo "Usage: $0 v<major>.<minor>.<patch>" >&2
    exit 1
fi

# Validate semver-with-v format
if ! echo "$VERSION" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "ERROR: version must be in the form vX.Y.Z (e.g. v1.2.3), got: $VERSION" >&2
    exit 1
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# Ensure git working tree is clean (no uncommitted changes)
if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "ERROR: git working tree is not clean. Commit or stash your changes before releasing." >&2
    git status --short >&2
    exit 1
fi

# Ensure we are on main or develop
BRANCH="$(git rev-parse --abbrev-ref HEAD)"
if [ "$BRANCH" != "main" ] && [ "$BRANCH" != "develop" ]; then
    echo "WARNING: you are releasing from branch '$BRANCH', not 'main' or 'develop'." >&2
    read -r -p "Continue anyway? [y/N] " confirm
    case "$confirm" in
        [yY]) ;;
        *) echo "Aborted."; exit 1 ;;
    esac
fi

echo "==> Creating tag $VERSION on branch $BRANCH"
git tag -a "$VERSION" -m "Release $VERSION"

echo "==> Pushing tag $VERSION to origin"
git push origin "$VERSION"

echo ""
echo "Tag $VERSION pushed. GitHub Actions will now build and publish the release binaries."
echo "Monitor progress at: https://github.com/piholster/piholster/actions"
