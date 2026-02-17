---
# gozer-hcs9
title: Extract magic strings/numbers to constants
status: completed
type: task
priority: normal
created_at: 2026-01-19T00:32:27Z
updated_at: 2026-01-19T00:36:32Z
sync:
    github:
        issue_number: "43"
        synced_at: "2026-02-17T17:29:36Z"
---

Audit found magic strings and numbers across the codebase. Extract to constants based on: multiple uses, semantic meaning, maintainability impact.

## Checklist
- [x] Create internal/template/lexer/keywords.go with template keyword constants
- [x] Update lexer to use keyword constants
- [x] Update parser to use keyword constants
- [x] Create internal/template/analyzer/constants.go with loop/recursion limits
- [x] Update analyzer files to use new constants
- [x] Create cmd/go-template-lsp/lsp/protocol.go with LSP protocol constants
- [x] Update LSP main.go to use protocol constants
- [x] Update LSP methods.go with method name constants (folding kinds already existed)
- [x] Update LSP parsing.go with header constants
- [x] Add file extensions and permissions constants to protocol.go
- [x] Run build, lint, and tests
