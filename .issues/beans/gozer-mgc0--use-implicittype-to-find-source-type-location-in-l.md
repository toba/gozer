---
# gozer-mgc0
title: Use implicitType to find source type location in LSP
status: scrapped
type: task
priority: normal
created_at: 2026-01-19T00:49:15Z
updated_at: 2026-01-19T00:55:40Z
sync:
    github:
        issue_number: "16"
        synced_at: "2026-02-17T17:29:35Z"
---

This TODO was stale - the functionality already exists in lines 162-168 of analyzer_lsp.go. The code already checks for TreeImplicitType and uses getVariableImplicitRange() to find the source type location.
