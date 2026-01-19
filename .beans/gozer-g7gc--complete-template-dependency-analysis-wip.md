---
# gozer-g7gc
title: Complete template dependency analysis WIP
status: scrapped
type: task
priority: normal
created_at: 2026-01-19T00:49:20Z
updated_at: 2026-01-19T00:56:15Z
---

The TODOs were stale markers. The functionality is already implemented:
- Lines 177-196 find files affected by template changes
- ContainerFileAnalysisForDefinedTemplates separates static (TemplateErrs) from dynamic (CycleTemplateErrs) errors  
- Root vs local template error handling is implemented via separate container types