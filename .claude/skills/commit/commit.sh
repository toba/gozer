#!/bin/bash
set -e

# Stage and show changes
echo "==> Staging changes..."
git add -A
git status --short
echo ""
echo "==> Staged diff:"
git diff --staged

# Get commit message from arguments
if [ -z "$1" ]; then
    echo ""
    echo "ERROR: Commit subject required as first argument"
    exit 1
fi

SUBJECT="$1"
DESCRIPTION="${2:-}"

# Build commit message
if [ -n "$DESCRIPTION" ]; then
    COMMIT_MSG="$SUBJECT

$DESCRIPTION"
else
    COMMIT_MSG="$SUBJECT"
fi

# Create commit
echo ""
echo "==> Creating commit..."
git commit -m "$COMMIT_MSG"
git status

# Version tagging and release (only if PUSH=true)
if [ "$PUSH" = "true" ]; then
    # Fetch remote tags to ensure we have the latest version info
    echo ""
    echo "==> Fetching remote tags..."
    git fetch --tags

    echo "==> Pushing commits..."
    git push

    CURRENT_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    if [ -n "$CURRENT_TAG" ]; then
        echo "==> Current version: $CURRENT_TAG"

        # Auto-bump patch version if NEW_VERSION not explicitly set
        if [ -z "$NEW_VERSION" ]; then
            # Parse current version (e.g., v0.6.0 -> 0 6 0)
            VERSION_NUMS=$(echo "$CURRENT_TAG" | sed 's/^v//' | tr '.' ' ')
            MAJOR=$(echo "$VERSION_NUMS" | awk '{print $1}')
            MINOR=$(echo "$VERSION_NUMS" | awk '{print $2}')
            PATCH=$(echo "$VERSION_NUMS" | awk '{print $3}')
            NEW_VERSION="v${MAJOR}.${MINOR}.$((PATCH + 1))"

            # Check if this version already exists, keep bumping if so
            while git rev-parse "$NEW_VERSION" >/dev/null 2>&1; do
                echo "==> Version $NEW_VERSION already exists, bumping again..."
                PATCH=$((PATCH + 1))
                NEW_VERSION="v${MAJOR}.${MINOR}.$((PATCH + 1))"
            done
            echo "==> Auto-bumping patch version: $CURRENT_TAG -> $NEW_VERSION"
        fi

        # Update version files with the new version
        EXT_VERSION="${NEW_VERSION#v}"
        EXTENSION_TOML="extension.toml"
        if [ -f "$EXTENSION_TOML" ]; then
            echo "==> Updating $EXTENSION_TOML version to $EXT_VERSION..."
            sed -i '' "s/^version = \".*\"/version = \"$EXT_VERSION\"/" "$EXTENSION_TOML"
            git add "$EXTENSION_TOML"
        fi
        CARGO_TOML="Cargo.toml"
        if [ -f "$CARGO_TOML" ]; then
            echo "==> Updating $CARGO_TOML version to $EXT_VERSION..."
            sed -i '' "s/^version = \".*\"/version = \"$EXT_VERSION\"/" "$CARGO_TOML"
            git add "$CARGO_TOML"
        fi
        # Amend commit if version files were updated
        if [ -n "$(git diff --staged --name-only)" ]; then
            echo "==> Amending commit with version file updates..."
            git commit --amend --no-edit
            git push --force-with-lease
        fi

        echo "==> Creating tag $NEW_VERSION..."
        git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"

        echo "==> Pushing tag..."
        git push origin "$NEW_VERSION"
        echo "==> Tag $NEW_VERSION pushed"
    else
        echo "==> No existing tags, skipping version bump"
    fi
else
    echo "==> Commit is local only (use PUSH=true to push and release)"
fi

# Sync to ClickUp
echo ""
echo "==> Syncing beans to ClickUp..."
beanup sync || echo "Warning: beanup sync failed or not available"

# Include sync state changes in the commit
if [ -n "$(git status --porcelain .beans/.sync.json 2>/dev/null)" ]; then
    echo "Including .beans/.sync.json in commit..."
    git add .beans/.sync.json
    git commit --amend --no-edit
fi

echo ""
echo "==> Done!"
