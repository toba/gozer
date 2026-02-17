---
# gozer-ewcb
title: Implement remaining refactoring items for internal/template
status: completed
type: task
priority: normal
created_at: 2026-01-18T23:51:14Z
updated_at: 2026-01-18T23:54:43Z
sync:
    github:
        issue_number: "8"
        synced_at: "2026-02-17T17:29:35Z"
---

Implement three refactoring items:
1. Remove pass-through wrapper (ScanWorkspaceForFuncMap)
2. Use go:generate stringer for Kind
3. Create keyword dispatch map for parser

## Checklist
- [x] Remove ScanWorkspaceForFuncMap wrapper from template.go and update caller in main.go
- [x] Add go:generate stringer directive for Kind and remove manual String() method
- [x] Create keyword dispatch map to replace if-else chain in parser_statement.go
