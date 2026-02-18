---
# afc-vlg
title: HTML coloring breaks after hx-vals attribute with JSON containing Go template expressions
status: in-progress
type: bug
priority: high
created_at: 2026-02-18T23:02:39Z
updated_at: 2026-02-18T23:08:09Z
sync:
    github:
        issue_number: "75"
        synced_at: "2026-02-18T23:35:26Z"
---

## Description

After an HTML attribute containing JSON with embedded Go template expressions, the tree-sitter HTML injection loses all HTML coloring for the rest of the file. Template expressions (from LSP semantic tokens) remain colored, but HTML tags, attributes, and strings appear as plain text.

## Reproducer

From `pacer/core/web/portfolio/portfolio.gohtml`, line 76:

```html
hx-vals='{"entity_kind":"{{.EntityKind}}","entity_id":"{{.EntityID}}","issue_type":"{{.IssueType}}"}'
```

## Root Cause

The gotmpl grammar's `text` rule splits the attribute into separate text nodes (at `{` and `{{` boundaries). These get injected (`injection.combined`) into the HTML tree-sitter parser, but the HTML parser can't handle a single-quoted attribute value split across multiple disjoint byte ranges with gaps (where template actions were). It enters an error state and loses HTML coloring for the rest of the file.

## Fix Options

1. Change the gotmpl grammar's `text` rule to keep attribute values more intact despite template actions
2. Adjust the injection queries to handle quoted attributes containing template actions

## Acceptance Criteria

- [ ] Add a test case with JSON-in-attribute containing Go template expressions
- [ ] HTML coloring continues correctly after the hx-vals line
- [ ] Existing highlighting tests continue to pass
