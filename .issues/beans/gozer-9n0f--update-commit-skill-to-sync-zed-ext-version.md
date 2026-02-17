---
# gozer-9n0f
title: Update commit skill to sync zed-ext version
status: completed
type: task
priority: normal
created_at: 2026-01-18T18:37:33Z
updated_at: 2026-01-18T18:38:04Z
sync:
    github:
        issue_number: "52"
        synced_at: "2026-02-17T17:29:36Z"
---

When releasing with PUSH=true and NEW_VERSION, the commit script should update zed-ext/extension.toml version field to match before pushing.

## Checklist
- [x] Read current commit.sh script
- [x] Add logic to update extension.toml version when NEW_VERSION is set
