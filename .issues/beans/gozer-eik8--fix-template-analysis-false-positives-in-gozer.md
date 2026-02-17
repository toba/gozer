---
# gozer-eik8
title: Fix template analysis false positives in gozer
status: completed
type: bug
priority: normal
created_at: 2026-01-18T21:08:34Z
updated_at: 2026-01-19T00:26:20Z
sync:
    github:
        issue_number: "22"
        synced_at: "2026-02-17T17:29:35Z"
---

False positives still occurring when linting ../core/web:

1. Method calls with arguments like `.Format "2006-01-02"` flagged as 'only function and method accepts arguments'
2. Custom template function `timehtml` flagged as 'field or method not found'
3. `.CloudLoggingURL $.ProjectID` flagged with both errors

These are valid Go template constructs that should not be flagged.

## Resolution
All three issues have been addressed:

1. **Method calls with arguments** - Fixed in `analyzer_typecheck.go:98-104`. When the receiver type is unknown (`any`), the analyzer no longer reports "only functions and methods accept arguments". Tests added in `integration_test.go`.

2. **Custom template functions** - Fixed via `funcmap_scanner.go` which scans Go source files for `template.FuncMap` definitions and registers custom functions. Functions like `timehtml` can be discovered automatically.

3. **Method calls with variable arguments** - Same fix as #1 handles this case.

Tests confirm no false positives for:
- `.CreatedAt.Format "Jan 2, 2006"`
- `.Staff.CreatedAt.Format "Jan 2, 2006"`
- `.CloudLoggingURL $.ProjectID`
