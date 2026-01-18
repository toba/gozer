---
# gozer-u8gb
title: Use new GroupStatementNode helper methods
status: todo
type: task
priority: low
created_at: 2026-01-18T22:43:34Z
updated_at: 2026-01-18T22:43:34Z
---

In analyzer.go:1000, the code should use the newer helper methods:
- `node.IsGroupWithNoVariableReset()`
- Similar helpers on GroupStatementNode

This would improve code readability and consistency.

## Checklist
- [ ] Identify all places that should use the new helpers
- [ ] Replace manual checks with helper method calls
- [ ] Verify behavior is unchanged