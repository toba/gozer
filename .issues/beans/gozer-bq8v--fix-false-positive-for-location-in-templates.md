---
# gozer-bq8v
title: Fix false positive for $.Location in templates
status: completed
type: bug
priority: normal
created_at: 2026-01-19T01:30:00Z
updated_at: 2026-01-19T01:32:41Z
sync:
    github:
        issue_number: "50"
        synced_at: "2026-02-17T17:29:35Z"
---

The linter is incorrectly flagging $.Location as 'field or method not found' and 'var $.Location invalid type' in Go templates. The $ variable refers to the root template data and should be valid when accessing fields like $.Location, even when inside a range loop or when using dict to create a new context.
