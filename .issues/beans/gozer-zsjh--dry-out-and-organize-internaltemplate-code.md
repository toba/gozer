---
# gozer-zsjh
title: DRY out and organize internal/template code
status: completed
type: task
priority: normal
created_at: 2026-01-18T21:39:27Z
updated_at: 2026-01-18T21:41:27Z
sync:
    github:
        issue_number: "30"
        synced_at: "2026-02-17T17:29:35Z"
---

Consolidate duplicate code and fix typos in the gozer codebase:
- Fix typos: Analisis→Analysis, Trigerred→Triggered, singleFileChange→SingleFileChange
- Remove duplicate intToUint and range conversion from main.go
- Remove unused getBuiltinFunctionDefinition from template.go
- Update all call sites
