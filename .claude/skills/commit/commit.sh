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

# Push (only if PUSH=true)
if [ "$PUSH" = "true" ]; then
    echo ""
    echo "==> Pushing commits..."
    git push
fi

# Sync issues to GitHub
echo ""
echo "==> Syncing issues to GitHub..."
todo sync || echo "Warning: todo sync failed or not available"

# Include sync state changes in the commit
if [ -n "$(git status --porcelain .issues/ 2>/dev/null)" ]; then
    echo "Including .issues/ changes in commit..."
    git add .issues/
    git commit --amend --no-edit
    if [ "$PUSH" = "true" ]; then
        git push --force-with-lease
    fi
fi

echo ""
echo "==> Done!"
