---
# gozer-kbqm
title: Save generated source code to file in generate.go
status: scrapped
type: task
priority: normal
created_at: 2026-01-19T00:49:37Z
updated_at: 2026-01-19T00:56:40Z
sync:
    github:
        issue_number: "26"
        synced_at: "2026-02-17T17:29:35Z"
---

This is a //go:build ignore code generator. Printing generated code to stdout (rather than writing to file) is actually better UX - it lets you inspect output before overwriting files. Not a bug.
