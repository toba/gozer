---
# gozer-9up1
title: Add textDocument/rename support
status: draft
type: feature
priority: low
created_at: 2026-01-19T03:29:15Z
updated_at: 2026-01-19T03:29:36Z
sync:
    github:
        issue_number: "56"
        synced_at: "2026-02-17T17:29:35Z"
---

Implement rename refactoring to rename variables and templates across files.

## Rename targets:
- **Variables**: Rename `$varName` and all references within scope
- **Templates**: Rename template definition and all `{{template "name"}}` calls

## Implementation notes:
- For variables: Scope-aware rename (only within declaring scope)
- For templates: Cross-file rename using WorkspaceTemplateManager
- Validate new name doesn't conflict with existing symbols
- Use `prepareRename` to validate rename is possible at cursor

## LSP methods:
- `textDocument/rename` - perform the rename, return WorkspaceEdit
- `textDocument/prepareRename` - validate rename is possible, return range
