---
# gozer-3ce4
title: Fix invalid tree-sitter query nodes in highlights.scm
status: completed
type: bug
created_at: 2026-01-18T19:00:55Z
updated_at: 2026-01-18T19:00:55Z
---

The highlights.scm files referenced 'break' and 'continue' node types that don't exist in the ngalaiko/tree-sitter-go-template grammar, causing Zed to fail loading the Go HTML Template language with error:

```
Query error at 27:2. Invalid node type "break"
```

Removed the invalid node type references from both gotmpl and gohtml highlights.scm files.