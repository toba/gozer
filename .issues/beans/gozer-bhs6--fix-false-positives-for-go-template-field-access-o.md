---
# gozer-bhs6
title: Fix false positives for Go template field access on any type
status: completed
type: bug
priority: normal
created_at: 2026-01-18T20:32:29Z
updated_at: 2026-01-18T20:34:04Z
sync:
    github:
        issue_number: "41"
        synced_at: "2026-02-17T17:29:36Z"
---

Remove errDefeatedTypeSystem error when accessing fields on `any` type in gota analyzer.

## Problem
The gozer LSP produces false positive errors for Go template field access chains when type information is unknown:
- `$.Staff.ReportsTo` → 'type system defeated'
- `.Staff.CreatedAt.Format` → 'field or method not found'

## Solution
Modify `/Users/jason/Developer/pacer/gota/analyzer/analyzer.go`:
1. Lines 2548-2550: Remove error for multi-field access on `any` type
2. Lines 2683-2689: Remove error when empty interface encountered mid-chain

## Checklist
- [x] Change 1: Lines 2548-2550 - Return TYPE_ANY without error
- [x] Change 2: Lines 2683-2689 - Return TYPE_ANY without error
- [x] Run gota tests (analyzer package builds; pre-existing test failures unrelated to this change)
- [x] Run gozer tests (pass)
- [x] Run golangci-lint (0 issues)
