---
# gozer-hiiq
title: LSP incorrectly reports 'invalid type' and 'field or method not found' for valid template expressions
status: completed
type: bug
created_at: 2026-01-19T01:13:55Z
updated_at: 2026-01-19T01:23:55Z
---

The LSP reports false positive errors for valid Go template expressions like `$.Location` and field accesses like `.StartedAt`. Seen in ../core/web/job/job-logs.html.

## Errors observed
- `var $.Location invalid type`
- `field or method not found` for `.StartedAt`, `$.Location`

## Root Cause
Two issues in `analyzer_statements.go`:

1. **Shared definition for $ and .**: In template definitions (`IsGroupWithDollarAndDotVariableReset`), `$` was set to point to the same `*VariableDefinition` as `.`. When type inference modified `.`, it also changed `$`, causing field access errors when `$` no longer had type `any`.

2. **$ not preserved in range/with blocks**: When entering a range or with block, a new `localVariables` map was created with only `.` defined. The `$` variable from the parent scope wasn't copied, so `$.Field` couldn't resolve.

## Fix
1. Create a separate `$` definition in template scopes (line 294) so type inference on `.` doesn't affect `$`.
2. Preserve `$` from parent scope when entering range blocks (lines 162-166) and with blocks (lines 280-284).

## Checklist
- [x] Read the problematic template file to understand the context
- [x] Create a test case that reproduces the issue
- [x] Identify the root cause in the template analyzer
- [x] Fix the bug
- [x] Verify the fix
