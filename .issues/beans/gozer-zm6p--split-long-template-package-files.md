---
# gozer-zm6p
title: Split long template package files
status: completed
type: task
priority: normal
created_at: 2026-01-18T22:46:15Z
updated_at: 2026-01-18T23:31:43Z
sync:
    github:
        issue_number: "48"
        synced_at: "2026-02-17T17:29:35Z"
---

## Description
Split oversized Go files in internal/template/ into smaller, focused files with idiomatic names. Includes:
- analyzer/analyzer.go (5,436 lines) → 11 files
- lexer/lexer.go (834 lines) → 5 files  
- parser/parser.go (1,495 lines) → 5 files
- Delete deprecated_checker.go (build-ignored)

## Checklist
- [x] Split lexer/lexer.go into 5 files
  - [x] lexer.go - Core types and entry point
  - [x] lexer_extract.go - Template extraction
  - [x] lexer_position.go - Position utilities
  - [x] lexer_tokenize.go - Line tokenization
  - [x] lexer_patterns.go - Pattern definitions
  - [x] Verify lexer builds and tests pass
- [x] Split parser/parser.go into 5 files
  - [x] parser.go - Core parser
  - [x] parser_scope.go - Scope management  
  - [x] parser_statement.go - Statement parsing
  - [x] parser_expression.go - Expression parsing
  - [x] parser_util.go - Utilities
  - [x] Verify parser builds and tests pass
- [ ] Split analyzer/analyzer.go into 11 files
  - [x] analyzer_types.go - Type definitions (CREATED)
  - [x] analyzer_statements.go - Statement analysis (CREATED)
  - [x] analyzer_variables.go - Variable analysis (CREATED)
  - [x] analyzer_expression.go - Expression analysis (CREATED)
  - [x] analyzer_inference.go - Type inference (CREATED)
  - [x] analyzer_typecheck.go - Function type checking (CREATED)
  - [x] analyzer_implicit.go - Implicit type trees (CREATED)
  - [ ] analyzer_errors.go - Error utilities (PARTIALLY CREATED - interrupted)
  - [ ] analyzer_lsp.go - IDE features (NOT CREATED)
  - [ ] analyzer_compat.go - Type compatibility (NOT CREATED)
  - [ ] analyzer.go - Core entry points (NOT UPDATED - still has all content)
  - [ ] Delete deprecated_checker.go
  - [ ] Verify analyzer builds and tests pass
- [ ] Run full verification: go build ./... && go test ./... && golangci-lint run

## Resume Notes

**CRITICAL PROBLEM DISCOVERED:** The split files have INCOMPATIBLE implementations with the original analyzer.go:
- `NewFileDefinition` returns 3 values in analyzer.go but 1 in analyzer_types.go
- `TemplateScopeID` is referenced in analyzer_types.go but never defined anywhere
- Type signatures differ (e.g., `BasicSymbolDefinition.typ` is `*types.Basic` vs `types.Type`)
- Const values differ (e.g., `OperatorType` vs `operation`, different constant orderings)

**The split files cannot be used as-is.** They need to be overwritten with correct code from analyzer.go.

## Recommended Approach

**Option A (Recommended):** Delete all split files and re-split correctly from analyzer.go:
```bash
rm internal/template/analyzer/analyzer_{types,statements,variables,expression,inference,typecheck,implicit}.go
```
Then split analyzer.go correctly, one file at a time.

**Option B:** Overwrite each split file with the correct code sections from analyzer.go.

## Line Ranges in analyzer.go (5436 lines total)

For correct splitting, here's what goes where:
- Lines 1-66: Package, imports, vars, const, init → KEEP in analyzer.go
- Lines 67-585: Type definitions → analyzer_types.go (BUT with correct signatures from analyzer.go, not the broken ones)
- Lines 587-688: getBuiltinFunctionDefinition, NewGlobalAndLocalVariableDefinition → KEEP in analyzer.go
- Lines 690-877: NewFileDefinition, DefinitionAnalysis, definitionAnalysisRecursive → KEEP in analyzer.go
- Lines 879-958: analyzeGroupStatementHeader → analyzer_statements.go
- Lines 959-1831: definitionAnalysisGroupStatement, definitionAnalysisTemplatateStatement, definitionAnalysisComment → analyzer_statements.go
- Lines 1832-2509: definitionAnalysisVariableDeclaration, definitionAnalysisVariableAssignment → analyzer_variables.go
- Lines 2510-3044: definitionAnalysisMultiExpression, definitionAnalysisExpression, definitionAnalyzer type → analyzer_expression.go
- Lines 3045-3596: makeTypeInferenceWhenPossible, splitVariableNameFields, etc. → analyzer_inference.go
- Lines 3597-3873: unTuple, makeExpressionTypeCheck, makeFunctionTypeCheck → analyzer_typecheck.go
- Lines 3874-4227: markVariableAsUsed, updateVariableImplicitType, etc. → analyzer_inference.go (continued)
- Lines 4229-4553: nodeImplicitType, buildTypeFromTreeOfType, etc. → analyzer_implicit.go
- Lines 4554-4725: remapRangeFromCommentGoCodeToSource, NewParseErrorFromErrorType, etc. → analyzer_errors.go
- Lines 4727-5134: FindSourceDefinitionFromPosition, findAstNodeRelatedToPosition, GoToDefinition, Hover → analyzer_lsp.go
- Lines 5215-5436: TypeCheckAgainstConstraint, TypeCheckCompatibilityWithConstraint, etc. → analyzer_compat.go

## Error Variables (keep in analyzer.go or move to analyzer_errors.go)
- Lines 3130-3136: errEmptyVariableName, errVariableUndefined, etc.
- Lines 3618-3636: errTemplateUndefined, errTypeMismatch, etc.
