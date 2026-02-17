---
# gozer-6c4m
title: Move CreateTemplateDefinition near its type
status: completed
type: task
priority: low
created_at: 2026-01-18T22:43:34Z
updated_at: 2026-01-19T00:30:46Z
sync:
    github:
        issue_number: "13"
        synced_at: "2026-02-17T17:29:35Z"
---

Code organization improvement in template_dependencies_analysis.go:587.

The `CreateTemplateDefinition` function should be moved closer to its associated type definition for better code organization.

## Checklist
- [ ] Identify the associated type definition
- [ ] Move the function near the type
- [ ] Ensure no circular dependencies are introduced
