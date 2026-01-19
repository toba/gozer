---
# gozer-zkdy
title: Remove fileNameToDefinition field
status: completed
type: task
priority: low
created_at: 2026-01-18T22:43:34Z
updated_at: 2026-01-19T00:32:12Z
---

In template_dependencies_analysis.go:214, the `fileNameToDefinition` field is marked for removal 'when appropriate'.

Need to analyze if this field is still necessary or if it can be safely removed.

## Checklist
- [ ] Analyze usage of fileNameToDefinition field
- [ ] Determine if it can be removed or replaced
- [ ] Remove the field if no longer needed
- [ ] Update any dependent code