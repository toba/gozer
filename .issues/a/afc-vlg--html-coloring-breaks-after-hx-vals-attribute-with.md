---
# afc-vlg
title: HTML coloring breaks after hx-vals attribute with JSON containing Go template expressions
status: review
type: bug
priority: high
created_at: 2026-02-18T23:02:39Z
updated_at: 2026-03-03T17:24:57Z
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

- [x] Add a test case with JSON-in-attribute containing Go template expressions
- [x] HTML coloring continues correctly after the hx-vals line
- [x] Existing highlighting tests continue to pass


## Research Notes (2026-02-24)

### Grammar Fix Status

The toba fork grammar fix (rev `e8e1e41126552259406304462e2c5bba36a990d5`) IS present in the local build and installed extension. Confirmed:
- `grammars/gotmpl/src/parser.c` has 3 text token types (`aux_sym_text_token1/2/3`) matching the merge pattern
- `gotmpl.wasm` built Feb 18 16:47
- Installed extension at `~/Library/Application Support/Zed/extensions/installed/gozer/` is a symlink to the dev repo
- Grammar fix merges lone `{` with surrounding text via `token(seq(/[^{]+/, repeat1(seq(/\{/, /[^{]+/))))` but this only reduces fragmentation — **does not eliminate gaps from template actions inside attribute values**

### Why the Grammar Fix Is Insufficient

For `hx-vals='{"entity_kind":"{{.EntityKind}}","entity_id":"{{.EntityID}}","issue_type":"{{.IssueType}}"}'`:

1. Text token 1: `...hx-vals='{"entity_kind:"` (merged — `{` joined with surrounding text ✓)
2. GAP: `{{.EntityKind}}` (template action)
3. Text token 2: `","entity_id":"` (no `{`, matches `/[^{]+/`)
4. GAP: `{{.EntityID}}`
5. Text token 3: `","issue_type":"`
6. GAP: `{{.IssueType}}`
7. Text token 4: `"}'...` (continues to next `{` or `{{`)

With `injection.combined`, the HTML parser receives these 4 disjoint ranges. The gaps fall inside a single-quoted attribute value. Even though tree-sitter multi-range parsing is supposed to treat them as contiguous, the HTML parser enters an error state and stops coloring HTML for the rest of the file.

### Approaches Investigated

#### 1. `injection.include-children` at template root level

```scheme
((template) @injection.content
  (#set! injection.language "html")
  (#set! injection.include-children))
```

- HTML parser gets entire file as one continuous range (no gaps)
- Template actions (`{{.EntityKind}}`) appear as text content to HTML parser
- **Tradeoff**: Template expressions inside HTML attribute values would get HTML `@string` coloring instead of gotmpl-specific coloring (e.g., `@variable.member`, `@keyword`)
- **Priority concern**: In Zed, innermost (injected) layer captures typically win over outer layer. HTML `@string` for attribute value would override gotmpl `@variable.member` for `.EntityKind`
- **Uncertain**: Whether `injection.include-children` is fully supported in Zed — no Zed extensions found using it at root level
- **NOT YET TESTED**

#### 2. Remove `injection.combined` (per-node injection)

- Each text node parsed as independent HTML
- Error would be contained per-node (no cascading)
- **Major regression**: Common patterns like `class="{{.Foo}}"` would break since `<div class="` and `">` are separate text nodes

#### 3. External scanner tracking quote state

- C scanner in grammar tracks single/double quote state
- Prevents `{{` from being recognized as template delimiter inside HTML attribute quotes
- Preserves all coloring
- **Very complex** — grammar doesn't know about HTML, would need heuristics
- Context-dependent: `{{.Content}}` outside attributes should still be a template action

#### 4. `#set! priority` on gotmpl captures

- Could override HTML captures for template action ranges
- Syntax: `#set! priority 105` (default is 100)
- Would work WITH approach 1 to preserve template coloring in attributes
- **Uncertain**: Whether Zed supports cross-layer priority overrides

### Zed Highlight Priority Model

From research (Zed source, GitHub issues, tree-sitter docs):
- Innermost (injected) layer captures typically win
- Known bug: [zed-industries/zed#42810](https://github.com/zed-industries/zed/issues/42810) — outer captures can incorrectly override injected captures
- `#set! priority N` metadata directive exists but cross-layer behavior unclear
- Tree-sitter issue [#3517](https://github.com/tree-sitter/tree-sitter/issues/3517) discusses rethinking `injection.combined` — recommends quantified captures (`@injection.content+`) but these have the same gap problem

### Other Extensions for Reference

- **PHP**: Has known highlighting issues ([zed#10778](https://github.com/zed-industries/zed/issues/10778)) — only highlights PHP, not embedded HTML
- **ERB**: Basic support, limited HTML injection
- **Handlebars**: Not officially supported in Zed
- **EJS**: Third-party, implementation details not examined

### Recommended Next Steps

1. **Test `injection.include-children` approach** — modify `languages/gohtml/injections.scm` and test in Zed. Quick to try, may solve the cascading error
2. **If include-children works**, test combining with `#set! priority` on gotmpl highlights to preserve template coloring in attributes
3. **If include-children not supported**, investigate external scanner approach or report upstream to Zed/tree-sitter


## Summary of Changes

Changed all three injection queries (`gohtml`, `gohtmx2`, `gohtmx4`) from targeting individual `(text)` nodes with `injection.combined` to targeting the root `(template)` node with `injection.include-children`. This gives the HTML parser the entire file as one continuous range, eliminating the gaps that caused it to enter an error state when template expressions appeared inside attribute values.

Manual verification needed in Zed with files containing `value="{{.Email}}"` and `hx-vals='...{{.Val}}...'` patterns.
