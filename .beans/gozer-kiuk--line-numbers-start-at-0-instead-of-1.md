---
# gozer-kiuk
title: Line numbers start at 0 instead of 1
status: todo
type: bug
priority: normal
created_at: 2026-01-18T22:43:34Z
updated_at: 2026-01-18T22:43:34Z
---

Bug in lexer.go:224 affecting error reporting.

Line numbers should start at 1 (human-friendly) but currently start at 0.

## Checklist
- [ ] Fix line number initialization in lexer
- [ ] Verify error messages show correct line numbers
- [ ] Update any tests that depend on line numbers