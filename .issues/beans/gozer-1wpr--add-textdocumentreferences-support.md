---
# gozer-1wpr
title: Add textDocument/references support
status: draft
type: feature
priority: low
created_at: 2026-01-19T03:29:15Z
updated_at: 2026-01-19T03:29:36Z
sync:
    github:
        issue_number: "67"
        synced_at: "2026-02-17T17:29:35Z"
---

Implement find-all-references to locate all usages of variables and templates.

## Reference types to support:
- **Variables**: Find all uses of `$varName` within its scope
- **Dot fields**: Find all uses of `.FieldName` (where type context matches)
- **Templates**: Find all `{{template "name"}}` calls for a defined template

## Implementation notes:
- For variables: Walk AST within scope, match by name
- For templates: Use existing WorkspaceTemplateManager which tracks template calls
- Consider cross-file references for templates

## LSP methods:
- `textDocument/references` - return list of locations
