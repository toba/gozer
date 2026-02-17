---
# gozer-r2ui
title: Variadic function type checking bugs
status: completed
type: bug
priority: high
created_at: 2026-01-18T22:43:33Z
updated_at: 2026-01-19T00:21:03Z
sync:
    github:
        issue_number: "45"
        synced_at: "2026-02-17T17:29:35Z"
---

Two related bugs in the type checker (analyzer_test.go:561,582):

1. Passing `[]int` to `...int` variadic incorrectly errors - should allow slice expansion with `slice...` syntax
2. Omitting variadic args incorrectly errors - variadic args are optional in Go

Current tests expect the wrong behavior (expect errors instead of success). The type checker needs to be fixed to:
- Allow passing a slice to a variadic parameter when using the `...` expansion operator
- Allow calling variadic functions without providing any variadic arguments

## Checklist
- [x] Fix slice expansion to variadic parameter (`[]int` to `...int`)
- [x] Fix optional variadic argument handling
- [x] Update tests to expect correct behavior
