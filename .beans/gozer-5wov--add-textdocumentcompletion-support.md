---
# gozer-5wov
title: Add textDocument/completion support
status: draft
type: feature
priority: low
created_at: 2026-01-19T03:29:15Z
updated_at: 2026-01-19T03:29:35Z
---

Implement auto-completion for Go templates. This is a high-value LSP feature that would significantly improve the editing experience.

## Completion contexts to support:
- **Variables**: Complete `.FieldName` from the current dot context type
- **Functions**: Complete built-in and custom template functions (e.g., `len`, `printf`, `eq`)
- **Template names**: Complete template names in `{{template "..." }}` and `{{block "..." }}`
- **Keywords**: Complete template keywords (`if`, `range`, `with`, `define`, `template`, `block`, `end`)
- **Variables in scope**: Complete `$varName` for declared variables

## Implementation notes:
- Use existing type inference from analyzer for context-aware field completion
- Leverage WorkspaceTemplateManager for template name completion
- Consider trigger characters: `.`, `$`, `"` (inside template calls)

## LSP methods:
- `textDocument/completion` - provide completion items
- `completionItem/resolve` (optional) - lazily resolve details