---
# gozer-wsnw
title: Lexer error cascading causes false errors
status: completed
type: bug
priority: high
created_at: 2026-01-18T22:43:33Z
updated_at: 2026-01-19T00:22:49Z
sync:
    github:
        issue_number: "42"
        synced_at: "2026-02-17T17:29:35Z"
---

Architecture issue in template.go:22-57: when lexer drops error lines, it causes false errors on valid syntax.

Example: `{{ if -- }} hello {{ end }}`
- The error on `if` causes a false error on the valid `{{ end }}`

The detailed design notes in template.go explain the issue:
- Option 1: Improve current LexerError system
- Option 2: Abandon lexer/error line drop system
- Option 3: Have error reporting only show lexer errors when present

## Checklist
- [x] Analyze the root cause of cascading errors
- [x] Choose an approach from the options in template.go
- [x] Implement the fix
- [x] Add test cases for error cascading scenarios

## Resolution
The error cascade bug has been fixed. Tests confirm that lexer errors on invalid syntax (e.g., `{{ if -- }}`) no longer cause false "extraneous end" errors on valid `{{ end }}` statements. Added comprehensive tests in `integration_test.go`.
