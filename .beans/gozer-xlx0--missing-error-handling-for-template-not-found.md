---
# gozer-xlx0
title: Missing error handling for template not found
status: todo
type: bug
priority: normal
created_at: 2026-01-18T22:43:34Z
updated_at: 2026-01-18T22:43:34Z
---

In template_dependencies_analysis.go:652, template lookup silently continues when a template isn't found.

This can lead to confusing behavior where missing template references don't produce warnings or errors.

## Checklist
- [ ] Add error/warning when template lookup fails
- [ ] Determine appropriate error severity (warning vs error)
- [ ] Add test case for missing template reference