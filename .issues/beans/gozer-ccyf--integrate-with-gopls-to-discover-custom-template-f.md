---
# gozer-ccyf
title: Integrate with gopls to discover custom template functions
status: completed
type: feature
priority: normal
created_at: 2026-01-18T19:42:01Z
updated_at: 2026-01-19T00:24:04Z
sync:
    github:
        issue_number: "38"
        synced_at: "2026-02-17T17:29:35Z"
---

Add support for discovering custom template functions (like Sprig functions or project-specific functions) so they aren't flagged as 'function undefined' errors.

## Problem
After fixing builtin function false positives, custom functions added via template.FuncMap are still flagged as undefined. Users defining functions like `lower`, `upper`, `default` (Sprig) or custom project functions get errors.

## Potential Approaches
1. **gopls integration**: Query gopls to find template.FuncMap definitions in Go code
2. **Configuration file**: .gozer.json or similar to declare additional functions
3. **Sprig preset**: Built-in knowledge of common Sprig functions
4. **Hybrid**: Combine approaches

## Research Needed
- How does gopls expose symbol/type information?
- Can we find template.FuncMap assignments via LSP?
- What's the best UX for configuration?

## Resolution
Implemented via direct AST parsing in `funcmap_scanner.go` (approach #1 without gopls dependency):
- Scans Go source files for `template.FuncMap` definitions
- Extracts function names from composite literals
- Supports text/template and html/template imports
- Handles aliased imports
- Skips vendor, node_modules, and test files
- Comprehensive tests in `funcmap_scanner_test.go`
