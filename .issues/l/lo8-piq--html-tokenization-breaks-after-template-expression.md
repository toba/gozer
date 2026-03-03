---
# lo8-piq
title: HTML tokenization breaks after template expression in attribute value
status: completed
type: bug
priority: high
created_at: 2026-03-03T17:18:17Z
updated_at: 2026-03-03T17:25:02Z
blocked_by:
    - afc-vlg
---

## Description

HTML syntax highlighting breaks after an `<input>` tag with a Go template expression in an attribute value:

```html
<input type="email" id="email" name="email" value="{{.Email}}" required autofocus>
```

Everything after this line loses HTML coloring — tags, attributes, and strings appear as plain text. Only template expressions (from LSP semantic tokens) remain colored.

## Reproducer

From the partner login page, the `value="{{.Email}}"` attribute is enough to trigger the break. Unlike afc-vlg (which requires JSON-in-attribute), this is a minimal case — a single template expression in a standard HTML attribute value.

## Root Cause

Same as afc-vlg — the gotmpl grammar splits the attribute value into disjoint text nodes around the `{{.Email}}` template action. With `injection.combined`, the HTML parser receives non-contiguous ranges inside a quoted attribute value, enters an error state, and stops coloring.

## Acceptance Criteria

- [ ] `value="{{.Email}}"` and similar simple template-in-attribute patterns do not break HTML coloring
- [ ] Fix for this also addresses afc-vlg (same root cause)
- [ ] Existing highlighting tests pass

\n## Summary of Changes\n\nResolved by parent issue afc-vlg fix: switching injection queries from `(text)` + `injection.combined` to `(template)` + `injection.include-children`.
