---
# gozer-kiuk
title: Line numbers start at 0 instead of 1
status: scrapped
type: bug
priority: normal
created_at: 2026-01-18T22:43:34Z
updated_at: 2026-01-19T00:16:08Z
---

Positions are correctly 0-indexed for LSP compatibility. The comment at lexer.go:8 explicitly documents this as intentional design. This is not a bug.