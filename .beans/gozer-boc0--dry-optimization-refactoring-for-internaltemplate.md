---
# gozer-boc0
title: DRY & Optimization Refactoring for internal/template Package
status: completed
type: task
created_at: 2026-01-18T23:39:34Z
updated_at: 2026-01-18T23:39:34Z
---

Refactor the internal/template package to eliminate code duplication, optimize performance, and improve maintainability across lexer, parser, and analyzer sub-packages.

## Checklist

### Phase 1: Lexer Package (Performance Critical)
- [x] 1.1 Pre-compile regex patterns (HIGH PRIORITY)
- [x] 1.2 Position/Range helper methods
- [x] 1.3 Extract duplicate lookahead logic
- [ ] 1.4 Use go:generate stringer for Kind (skipped - low priority)
- [x] 1.5 Standardize byte comparisons

### Phase 2: Parser Package (Maintainability)
- [x] 2.1 Expression validation helper
- [x] 2.2 Named constants for magic numbers
- [ ] 2.3 Keyword dispatch map (skipped - low priority, high effort)
- [x] 2.4 Consolidated variable parsing

### Phase 3: Analyzer Package & template.go (DRY)
- [x] 3.1 Consolidate analysis chain functions (HIGH PRIORITY)
- [x] 3.2 Error appending helper
- [ ] 3.3 Remove pure pass-through wrapper (skipped - would break external callers)

## Summary of Changes

### Lexer Package
- Pre-compiled all regex patterns at package init time (18 patterns per template block)
- Added Position.Offset(), Range.AdjustStart(), Range.AdjustEnd(), Range.Shrink() helpers
- Extracted `templateExtractor` struct with `processLoneDelimiter()` helper
- Standardized single-byte comparisons to use direct byte comparison

### Parser Package
- Added `validControlFlowKinds` map and `validateControlFlowExpression()` helper
- Added named constants: `maxRecursionDepth`, `maxExpressionTokens`, `maxVariablesPerDeclaration`
- Extracted `parseVariableNames()` helper to deduplicate variable parsing logic

### Template Package
- Extracted `analyzeAffectedFiles()` to consolidate analysis chain loops
- Added `appendAnalysisErrors()` helper for error collection
- Added `validateFileInWorkspace()` helper for file validation

## Verification
All tests pass:
- `go test ./internal/template/...` ✓
- `golangci-lint run ./internal/template/...` ✓
- `go build ./...` ✓