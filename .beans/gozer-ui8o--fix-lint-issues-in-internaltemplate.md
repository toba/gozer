---
# gozer-ui8o
title: Fix lint issues in internal/template
status: todo
type: task
created_at: 2026-01-18T21:47:26Z
updated_at: 2026-01-18T21:47:26Z
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