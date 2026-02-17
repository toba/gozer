---
# gozer-mnk7
title: Redo comment token detection
status: scrapped
type: task
priority: low
created_at: 2026-01-18T22:43:34Z
updated_at: 2026-01-19T00:34:10Z
sync:
    github:
        issue_number: "2"
        synced_at: "2026-02-17T17:29:35Z"
---

In lexer.go:589,609, comment parsing needs improvement.

Two related TODOs:
1. Improve comment token detection logic
2. Use bytes instead of slices for comparison (performance)

## Checklist
- [ ] Review current comment detection implementation
- [ ] Refactor to use bytes for comparison
- [ ] Improve comment parsing logic
- [ ] Add tests for edge cases in comment handling
