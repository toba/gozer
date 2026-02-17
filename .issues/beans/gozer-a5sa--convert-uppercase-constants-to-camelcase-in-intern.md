---
# gozer-a5sa
title: Convert uppercase constants to camelCase in internal/template
status: completed
type: task
priority: normal
created_at: 2026-01-18T22:20:20Z
updated_at: 2026-01-18T22:43:42Z
sync:
    github:
        issue_number: "70"
        synced_at: "2026-02-17T17:29:36Z"
---

Convert SCREAMING_SNAKE_CASE constants in /internal/template to idiomatic Go camelCase, using lowercase first letter for package-private constants. Includes fixing typos (ASSIGNEMENT → assignment, RIGTH → right).

## Checklist
- [x] Convert lexer constants to camelCase
- [x] Convert parser constants to camelCase  
- [x] Fix typos in constant names
- [x] Fix compiler error: undefined parser.PrettyFormatter in template.go:445
- [x] Fix Go 1.25 API change: lit.Elements → lit.Elts in funcmap_scanner.go
- [x] Remove entire unused type constants block in analyzer.go
- [x] Fix gofmt/golines formatting issues
- [x] Change value receivers to pointer receivers for heavy structs (hugeParam)
- [x] Fix paramTypeCombine issues
- [x] Fix nilerr issues properly (return error or use _)
- [x] Fix error string capitalization (staticcheck)
- [x] Fix typeUnparen issue
- [x] Fix emptyStringTest issue
- [x] Fix nestingReduce issues
- [x] Run golangci-lint to verify all issues fixed
