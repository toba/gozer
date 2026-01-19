---
name: commit
description: Stage all changes and commit with a descriptive message. Use when the user asks to commit, save changes, or says "/commit".
args: "[push]"
---

## Workflow

**IMPORTANT**: Only use `PUSH=true` when the user explicitly says "/commit push" or asks to push. Plain "/commit" should NEVER push.

1. Review changes to determine commit message:
   ```bash
   git diff
   ```

2. Run commit script with subject and description:
   ```bash
   # Local commit only (no push):
   .claude/skills/commit/commit.sh "subject line" "description body"

   # Push after commit:
   PUSH=true .claude/skills/commit/commit.sh "subject line" "description body"
   ```

   - **Subject**: Lowercase, imperative mood (e.g., "add feature" not "Added feature")
   - **Description**: Explain the "why" and context. What problem does this solve? What approach was taken?

The script handles: stage, commit, and beanup sync. Push happens when `PUSH=true`.

**Note**: Version is set manually in `extension.toml` and `Cargo.toml`. Update these files before committing when releasing a new version.
