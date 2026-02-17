---
# gozer-q90j
title: Vendor gota source into gozer
status: completed
type: task
priority: normal
created_at: 2026-01-18T21:17:56Z
updated_at: 2026-01-19T00:23:26Z
sync:
    github:
        issue_number: "18"
        synced_at: "2026-02-17T17:29:35Z"
---

Copy gota source code into gozer as internal package to simplify development workflow.

## Checklist
- [x] Copy gota source into internal/template (note: used internal/template not internal/gota)
- [x] Update package declarations
- [x] Update internal imports within gota packages
- [x] Update gozer imports to use internal/template
- [x] Remove replace directive from go.mod
- [x] Run tests to verify everything works
- [x] Simplify code if possible (refactored for maintainability)

## Resolution
Gota source was vendored into `internal/template` (per commit bb87107). The replace directive has been removed from go.mod.
