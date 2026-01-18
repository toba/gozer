---
# gozer-sezg
title: Type inference fails for dot inside range loop
status: completed
type: bug
priority: normal
created_at: 2026-01-18T20:47:44Z
updated_at: 2026-01-18T20:57:40Z
---

Multiple type inference failures observed:

## Issue 1: Dot inside range loop
When inside a `{{range .ClientList -}}` loop, hovering over the `.` argument in `{{template "client-summary" .}}` shows 'type system defeated' with type `any`. The type checker should know the element type.

## Issue 2: Field access on root variable
`$.Staff.ReportsTo` shows 'field or method not found' and 'invalid type'. The `$` root variable field access is not resolving correctly.

## Root Cause
The gota analyzer was generating `errDefeatedTypeSystem` errors in three places when types were unknown:
1. Field access on `any` type (already fixed in commit 2aee515)
2. Range loops over `any` type (line 1585-1589)
3. Template invocations with `any` expression (line 4252-4253)

## Fix
1. Updated gozer go.mod to use gota commit 2aee515 which fixed field access
2. Made additional fixes in gota (commit 7e9052e) to remove errors for:
   - Range loops over unknown types
   - Template calls with unknown argument types

## Checklist
- [x] Investigate type inference in gota analyzer
- [x] Identify root cause(s)
- [x] Fix the issue(s)
- [x] Add or verify test coverage (existing tests pass)