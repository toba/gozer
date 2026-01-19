---
# gozer-1am0
title: Add textDocument/signatureHelp support
status: draft
type: feature
priority: low
created_at: 2026-01-19T03:29:15Z
updated_at: 2026-01-19T03:29:35Z
---

Implement signature help to show function signatures while typing arguments.

## Behavior:
When the cursor is inside a function call like `{{printf "%s" .Name}}`, show:
- Function signature with parameter names and types
- Highlight the current parameter being typed
- Show documentation for the function

## Functions to support:
- Built-in functions: `printf`, `len`, `index`, `slice`, `eq`, `ne`, `lt`, `gt`, etc.
- Custom workspace functions discovered via FuncMap scanning

## Implementation notes:
- Trigger on space after function name inside `{{`
- Use existing function definitions from analyzer
- Track cursor position to highlight active parameter

## LSP methods:
- `textDocument/signatureHelp` - provide signature information