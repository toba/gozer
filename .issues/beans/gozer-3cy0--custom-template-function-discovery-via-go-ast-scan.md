---
# gozer-3cy0
title: Custom Template Function Discovery via Go AST Scanning
status: completed
type: feature
priority: normal
created_at: 2026-01-18T20:00:15Z
updated_at: 2026-01-18T20:03:56Z
sync:
    github:
        issue_number: "63"
        synced_at: "2026-02-17T17:29:35Z"
---

Automatically discover custom template functions from Go source code to prevent false 'function undefined' errors.

## Checklist

- [x] Create `gota/analyzer/funcmap_scanner.go` with AST scanning logic
- [x] Modify `gota/analyzer/analyzer.go` to merge custom functions with builtins
- [x] Modify `gota/gota.go` to add custom function API
- [x] Modify `gozer/cmd/go-template-lsp/main.go` to scan on initialize
- [x] Add tests for funcmap scanner
- [x] Run tests in both repos to verify
