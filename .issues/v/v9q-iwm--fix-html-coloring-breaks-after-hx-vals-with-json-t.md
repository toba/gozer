---
# v9q-iwm
title: Fix HTML coloring breaks after hx-vals with JSON + template expressions
status: completed
type: bug
priority: normal
created_at: 2026-02-18T23:22:48Z
updated_at: 2026-02-18T23:29:21Z
sync:
    github:
        issue_number: "74"
        synced_at: "2026-02-18T23:35:27Z"
---

After an HTML attribute containing JSON with Go template expressions (e.g. hx-vals='{"key":"{{.Value}}}'), HTML coloring breaks for the rest of the file. Fix by modifying the text rule to merge lone { with surrounding text.

- [x] Modify text rule in make_grammar.js
- [x] Update test expectations
- [x] Add regression test
- [x] Regenerate parser and run tests (89/89 pass)
- [x] Build wasm target

## Summary of Changes

Added a new `token(seq(...))` alternative to the `text` rule in `make_grammar.js` that merges lone `{` characters with surrounding text into a single token. This reduces fragmentation for injected HTML parsers — e.g. `<div hx-vals='{"key":"` is now one token instead of three. The original `/[^{]+/` and `/\{/` alternatives are preserved for cases where text doesn't contain `{` or has a lone `{` at EOF/before `{{`. Updated test expectations for the curly brackets test (7 text nodes → 1) and added a regression test for JSON-in-attribute with template expressions.
