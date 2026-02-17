---
# gozer-ui8o
title: Fix lint issues in internal/template
status: completed
type: task
priority: normal
created_at: 2026-01-18T21:47:26Z
updated_at: 2026-01-18T22:01:24Z
sync:
    github:
        issue_number: "33"
        synced_at: "2026-02-17T17:29:35Z"
---

Fix 129 golangci-lint issues in vendored internal/template code (from gota).

Categories:
- unused: 12 (unused fields, types, functions)
- staticcheck: 50 (bool comparisons, nil checks, type inference)
- misspell: 22 (defintion, comming, sucess, etc.)
- gocritic: 8 (if-else chains, etc.)
- gosec: 8 (potential issues)
- ineffassign: 6
- dupl: 4 (duplicate code blocks)
- prealloc: 5
- other: 14
