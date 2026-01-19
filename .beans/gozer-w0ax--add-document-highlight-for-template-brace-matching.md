---
# gozer-w0ax
title: Add document highlight for template brace matching
status: completed
type: feature
priority: normal
created_at: 2026-01-19T02:56:39Z
updated_at: 2026-01-19T03:00:40Z
---

Implement textDocument/documentHighlight LSP capability to highlight matching template control flow keywords (e.g., clicking {{end}} highlights its corresponding {{if}}/{{range}}/etc.).

## Background
The parser already links matching template blocks via `NextLinkedSibling` pointers in `GroupStatementNode`. This creates a linked list connecting related control flow statements:
```
{{if .foo}} → {{else if .bar}} → {{else}} → {{end}}
```

## Checklist
- [ ] Add DocumentHighlightProvider capability to server initialization
- [ ] Add textDocument/documentHighlight method registration in protocol.go
- [ ] Implement ProcessDocumentHighlightRequest in methods.go
- [ ] Create helper to find GroupStatementNode at cursor position
- [ ] Traverse NextLinkedSibling chain to collect all related keywords
- [ ] Return highlight ranges for all matched keywords
- [ ] Add tests for document highlight functionality