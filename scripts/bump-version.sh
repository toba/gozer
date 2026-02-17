#!/usr/bin/env bash
# Bump version in extension.toml, Cargo.toml, and Cargo.lock.
# Usage: scripts/bump-version.sh <version>
# Example: scripts/bump-version.sh 0.14.0
set -euo pipefail

VERSION="${1:?Usage: bump-version.sh <version>}"
VERSION="${VERSION#v}" # strip leading v if present

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"

sed -i'' -e "s/^version = \".*\"/version = \"$VERSION\"/" "$REPO_DIR/extension.toml"

# Update only the [package] version, not dependency versions
sed -i'' -e "/^\[package\]/,/^\[/{s/^version = \".*\"/version = \"$VERSION\"/;}" "$REPO_DIR/Cargo.toml"

# Regenerate lockfile if cargo is available
if command -v cargo &>/dev/null; then
  (cd "$REPO_DIR" && cargo generate-lockfile)
fi

echo "Bumped to $VERSION"
