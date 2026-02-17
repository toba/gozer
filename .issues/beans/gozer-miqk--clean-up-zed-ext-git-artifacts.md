---
# gozer-miqk
title: Clean up zed-ext git artifacts
status: completed
type: task
priority: normal
created_at: 2026-01-18T18:33:27Z
updated_at: 2026-01-18T18:34:16Z
sync:
    github:
        issue_number: "64"
        synced_at: "2026-02-17T17:29:35Z"
---

Fix git issues in zed-ext directory:
1. Convert embedded grammars/gotmpl repo to proper submodule
2. Remove previously committed target/ directory from repo

## Checklist
- [x] Remove embedded repo and add as submodule
- [x] Remove zed-ext/target from git tracking
