---
# gozer-ta6v
title: Remove log import from parser
status: scrapped
type: task
priority: deferred
created_at: 2026-01-18T22:43:34Z
updated_at: 2026-01-19T00:28:59Z
---

The log import is used for debugging before panics at lines 181 and 196. These provide useful context when fatal errors occur. Not removing.