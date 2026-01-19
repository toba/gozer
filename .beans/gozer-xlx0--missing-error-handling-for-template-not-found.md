---
# gozer-xlx0
title: Missing error handling for template not found
status: completed
type: bug
priority: normal
created_at: 2026-01-18T22:43:34Z
updated_at: 2026-01-19T00:28:25Z
---

In template_dependencies_analysis.go:652, template lookup silently continues when a template isn't found.

This can lead to confusing behavior where missing template references don't produce warnings or errors.

## Checklist
- [x] Add error/warning when template lookup fails
- [x] Determine appropriate error severity (warning vs error)
- [x] Add test case for missing template reference

## Resolution
Error handling for undefined templates was already in place at the type-check level (`errTemplateUndefined` in analyzer_typecheck.go:33). Added additional error reporting at the dependency analysis level (template_dependencies_analysis.go:652) to ensure coverage in all code paths. Test added in integration_test.go.