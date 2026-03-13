#!/usr/bin/env bash
# Bump version in extension.toml, Cargo.toml, and Cargo.lock.
# Usage: scripts/bump-version.sh <version>
# Example: scripts/bump-version.sh 0.14.0
set -euo pipefail

VERSION="${1:?Usage: bump-version.sh <version>}"
VERSION="${VERSION#v}" # strip leading v if present

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"

# Use temp file for sed portability (macOS sed -i '' vs GNU sed -i)
sed_inplace() {
  local file="$1"; shift
  local tmp="${file}.tmp"
  sed "$@" "$file" > "$tmp" && mv "$tmp" "$file"
}

sed_inplace "$REPO_DIR/extension.toml" -e "s/^version = \".*\"/version = \"$VERSION\"/"

# Update only the [package] version, not dependency versions
sed_inplace "$REPO_DIR/Cargo.toml" -e "/^\[package\]/,/^\[/{s/^version = \".*\"/version = \"$VERSION\"/;}"

# Regenerate lockfile if cargo is available
if command -v cargo &>/dev/null; then
  (cd "$REPO_DIR" && cargo generate-lockfile)
fi

echo "Bumped to $VERSION"
