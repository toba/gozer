---
# gozer-45gq
title: Hover on keywords returns wrong/unexpected content
status: completed
type: bug
created_at: 2026-01-19T17:18:47Z
updated_at: 2026-01-19T17:18:47Z
---

When hovering over control flow keywords like `if`, `else`, `range`, `end`, the hover handler returns incorrect content (e.g., showing 'else' when hovering on 'if').

## Root Cause
In `analyzer_lsp.go`, `FindSourceDefinitionFromPosition` sets `IsKeyword = true` when the cursor is on a keyword (lines 303-308), but then doesn't check this flag - it proceeds to try matching the keyword as a variable/template definition, producing wrong results.

## Expected Behavior
Keywords should either:
1. Return no hover content (preferred - they're not identifiers with type info)
2. Return documentation about the keyword

## Checklist
- [x] Create failing test demonstrating the bug
- [x] Fix `FindSourceDefinitionFromPosition` to return nil when `IsKeyword` is true
- [x] Run tests and linter