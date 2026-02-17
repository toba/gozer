---
# gozer-x69k
title: Move source_code.go to testdata/analyzer directory
status: scrapped
type: task
priority: normal
created_at: 2026-01-19T00:49:45Z
updated_at: 2026-01-19T00:57:18Z
sync:
    github:
        issue_number: "73"
        synced_at: "2026-02-17T17:29:36Z"
---

The file is actively used by analyzer_test.go (lines 160, 788). Moving it would require updating test paths. This is just organizational preference, not a bug. The current location works fine.
