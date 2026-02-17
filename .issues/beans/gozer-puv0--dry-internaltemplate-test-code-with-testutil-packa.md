---
# gozer-puv0
title: DRY internal/template test code with testutil package
status: completed
type: task
priority: normal
created_at: 2026-01-19T00:52:03Z
updated_at: 2026-01-19T00:55:05Z
sync:
    github:
        issue_number: "21"
        synced_at: "2026-02-17T17:29:35Z"
---

Create shared testutil package to eliminate duplicate test helpers across internal/template tests.

## Checklist
- [x] Create internal/template/testutil/testutil.go with shared assertion helpers
- [x] Create internal/template/testutil/fixtures.go with TempDir helper
- [x] Update lexer/lexer_test.go to use testutil
- [x] Update parser/parser_test.go to use testutil
- [x] Update integration_test.go to use testutil
- [x] Update analyzer/funcmap_scanner_test.go to use TempDir helper
- [x] Run tests to verify
- [x] Run linter to verify
