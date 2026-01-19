---
# gozer-9tse
title: Implement Template Call-Site Type Propagation
status: completed
type: feature
priority: normal
created_at: 2026-01-19T02:21:16Z
updated_at: 2026-01-19T02:28:19Z
---

Implement type inference improvements for template analysis:

1. **Template Call-Site Type Propagation** - When a template is called with a specific type (e.g., `{{template "foo" .User}}`), propagate that type into the template body analysis so `.` has the specific type instead of `any`.

2. **Partial Struct Types** - When fields are accessed on an `any`-typed variable, track field existence even if their types are unknown.

3. **Type Inference from Comparisons** - When a variable is compared to a literal (e.g., `eq .Field "hello"`), infer that the variable has the same type as the literal.

## Checklist

- [x] Create failing tests for all three improvements
- [x] Add GetTemplate method to FileDefinition
- [x] Implement call-site tracking in TemplateDefinition (added callSiteTypes field)
- [x] Record call-site types during template use (in definitionAnalysisTemplatateStatement)
- [x] Compute unified type from call sites (InferredInputType method)
- [x] Use inferred type when analyzing template body (Type() now delegates to InferredInputType)
- [x] Verify partial struct type inference works (already passing!)
- [x] Implement comparison type inference (isComparisonFunction, inferTypesFromComparison)
- [x] Run full test suite and linter - all pass!