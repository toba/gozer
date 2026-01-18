---
# gozer-q90j
title: Vendor gota source into gozer
status: in-progress
type: task
created_at: 2026-01-18T21:17:56Z
updated_at: 2026-01-18T21:17:56Z
---

Copy gota source code into gozer as internal package to simplify development workflow.

## Checklist
- [ ] Copy gota source into internal/gota
- [ ] Update package declarations
- [ ] Update internal imports within gota packages
- [ ] Update gozer imports to use internal/gota
- [ ] Remove replace directive from go.mod
- [ ] Run tests to verify everything works
- [ ] Simplify code if possible