---
# gozer-y8wz
title: Change generate.go log output from stdout to file
status: scrapped
type: task
priority: normal
created_at: 2026-01-19T00:49:29Z
updated_at: 2026-01-19T00:56:40Z
---

This is a //go:build ignore code generator tool. Logging to stdout is actually preferable for manual code generators - you see feedback immediately. Writing to a file would obscure it. Not a bug, just an optional enhancement.