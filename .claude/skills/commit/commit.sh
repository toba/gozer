#!/bin/bash
set -e

# Pre-commit checks
echo "==> Running pre-commit checks..."
golangci-lint run
go test ./...

# Update zed-ext/extension.toml version if NEW_VERSION is set
if [ -n "$NEW_VERSION" ]; then
    # Strip 'v' prefix for extension.toml (e.g., v0.1.2 -> 0.1.2)
    EXT_VERSION="${NEW_VERSION#v}"
    EXTENSION_TOML="zed-ext/extension.toml"
    if [ -f "$EXTENSION_TOML" ]; then
        echo "==> Updating $EXTENSION_TOML version to $EXT_VERSION..."
        sed -i '' "s/^version = \".*\"/version = \"$EXT_VERSION\"/" "$EXTENSION_TOML"
    fi
fi

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
CURRENT_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [ -n "$CURRENT_TAG" ]; then
    echo ""
    echo "==> Current version: $CURRENT_TAG"

    if [ "$PUSH" = "true" ]; then
        echo "==> Pushing commits..."
        git push

        # If NEW_VERSION is set, create and push tag
        # GoReleaser workflow will create the GitHub release automatically
        if [ -n "$NEW_VERSION" ]; then
            echo "==> Creating tag $NEW_VERSION..."
            git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"

            echo "==> Pushing tag (GoReleaser will create release)..."
            git push origin "$NEW_VERSION"
            echo "==> Tag $NEW_VERSION pushed, GoReleaser workflow will create release"
        else
            echo "==> No NEW_VERSION set, skipping release"
        fi
    else
        echo "==> Commit is local only (use PUSH=true to push and release)"
        if [ -n "$NEW_VERSION" ]; then
            echo "==> NEW_VERSION=$NEW_VERSION will be used when pushed"
        fi
    fi
else
    echo "==> No existing tags, skipping version bump"
    if [ "$PUSH" = "true" ]; then
        echo "==> Pushing commits..."
        git push
    fi
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
