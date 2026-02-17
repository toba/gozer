---
# gozer-tp5u
title: Parallelize ParseFilesInWorkspace for performance
status: completed
type: task
priority: normal
created_at: 2026-01-19T01:00:16Z
updated_at: 2026-01-19T01:03:03Z
sync:
    github:
        issue_number: "61"
        synced_at: "2026-02-17T17:29:35Z"
---

Implement goroutine-based parallelization for ParseFilesInWorkspace() to improve performance on multi-core systems.

## Checklist
- [x] Fix thread-unsafe counter in parser/parser.go using atomic.Int64
- [x] Parallelize ParseFilesInWorkspace in template.go using sync.WaitGroup
- [x] Add concurrency tests in template_test.go
- [x] Run tests with race detector
- [x] Run linter
