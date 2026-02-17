---
# gozer-wt1x
title: Fix 'variable is never used' false positive for template variables
status: completed
type: bug
priority: normal
created_at: 2026-01-19T01:36:41Z
updated_at: 2026-01-19T01:39:26Z
sync:
    github:
        issue_number: "60"
        synced_at: "2026-02-17T17:29:35Z"
---

The analyzer is incorrectly reporting '.Portfolio.ID' as 'variable is never used' when it's used in HTML attributes (like hx-get) and in script tags. The variable usage detection is missing these cases.
