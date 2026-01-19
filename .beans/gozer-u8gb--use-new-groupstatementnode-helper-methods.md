---
# gozer-u8gb
title: Use new GroupStatementNode helper methods
status: completed
type: task
priority: low
created_at: 2026-01-18T22:43:34Z
updated_at: 2026-01-19T00:40:21Z
---

In analyzer_statements.go:140, the code now uses the newer helper methods:
- `node.IsGroupWithNoVariableReset()`
- `node.IsGroupWithDollarAndDotVariableReset()`

This improves code readability and consistency.

## Checklist
- [x] Identify all places that should use the new helpers
- [x] Replace manual checks with helper method calls
- [x] Verify behavior is unchanged