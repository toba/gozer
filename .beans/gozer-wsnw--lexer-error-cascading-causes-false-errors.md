---
# gozer-wsnw
title: Lexer error cascading causes false errors
status: todo
type: bug
priority: high
created_at: 2026-01-18T22:43:33Z
updated_at: 2026-01-18T22:43:33Z
---

Architecture issue in template.go:22-57: when lexer drops error lines, it causes false errors on valid syntax.

Example: `{{ if -- }} hello {{ end }}`
- The error on `if` causes a false error on the valid `{{ end }}`

The detailed design notes in template.go explain the issue:
- Option 1: Improve current LexerError system
- Option 2: Abandon lexer/error line drop system
- Option 3: Have error reporting only show lexer errors when present

## Checklist
- [ ] Analyze the root cause of cascading errors
- [ ] Choose an approach from the options in template.go
- [ ] Implement the fix
- [ ] Add test cases for error cascading scenarios