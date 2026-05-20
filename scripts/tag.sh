#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_DIR"

VERSION=$(cat .version | tr -d '[:space:]')

if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "Error: not a git repository"
    exit 1
fi

if git tag | grep -q "^${VERSION}$"; then
    echo "Tag $VERSION already exists. Skipping."
    exit 0
fi

echo "Creating git tag: $VERSION"
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"

echo "Tag $VERSION created and pushed."
