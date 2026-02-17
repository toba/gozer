---
# gozer-92qw
title: Add textDocument/documentSymbol support
status: draft
type: feature
priority: low
created_at: 2026-01-19T03:29:15Z
updated_at: 2026-01-19T03:29:36Z
sync:
    github:
        issue_number: "28"
        synced_at: "2026-02-17T17:29:34Z"
---

Implement document symbols to provide an outline/symbol tree for templates.

## Symbols to expose:
- **Template definitions**: `{{define "name"}}` and `{{block "name"}}`
- **Variable declarations**: `$var := ...` 
- **Control flow blocks**: `if`, `range`, `with` (as a hierarchy)

## Symbol kinds (LSP SymbolKind):
- Template definitions → Function (12) or Module (2)
- Variables → Variable (13)
- Control flow → Struct (23) or Namespace (3)

## Implementation notes:
- Walk AST and collect symbols with their ranges
- Preserve hierarchy (variables inside templates, etc.)
- Include symbol details like template parameter type if known

## LSP methods:
- `textDocument/documentSymbol` - return symbol hierarchy
