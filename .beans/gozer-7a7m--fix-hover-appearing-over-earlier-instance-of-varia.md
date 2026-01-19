---
# gozer-7a7m
title: Fix hover appearing over earlier instance of variable
status: completed
type: bug
priority: low
created_at: 2026-01-19T02:02:29Z
updated_at: 2026-01-19T03:29:14Z
---

When hovering over a variable like .ErrorCount, the hover tooltip appears over the first instance of that variable on the line rather than where the cursor actually is. In the reported case, the mouse was over the third ErrorCount towards the right but the hover appeared over the first.