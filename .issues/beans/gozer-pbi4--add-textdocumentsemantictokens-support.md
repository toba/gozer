---
# gozer-pbi4
title: Add textDocument/semanticTokens support
status: completed
type: feature
priority: low
created_at: 2026-01-19T03:29:15Z
updated_at: 2026-01-19T16:43:09Z
sync:
    github:
        issue_number: "31"
        synced_at: "2026-02-17T17:29:35Z"
---

Implement semantic tokens for rich, context-aware syntax highlighting.

## Token types to distinguish:
- **Keywords**: `if`, `else`, `range`, `with`, `define`, `template`, `block`, `end`
- **Variables**: `$varName`
- **Functions**: `len`, `printf`, custom functions
- **Fields**: `.FieldName`
- **Strings**: String literals
- **Numbers**: Numeric literals
- **Operators**: `:=`, `=`, `|`
- **Types**: Type names in comments/hints

## Token modifiers:
- `declaration` for variable declarations
- `definition` for template definitions
- `readonly` for range loop variables

## Implementation notes:
- Walk AST and emit tokens with types/modifiers
- Coordinate with tree-sitter grammar for base highlighting
- Semantic tokens override/enhance textmate scopes

## LSP methods:
- `textDocument/semanticTokens/full` - all tokens in document
- `textDocument/semanticTokens/delta` (optional) - incremental updates

## Checklist

- [x] Add semantic token types/modifiers constants to `lsp/protocol.go`
- [x] Add `SemanticTokensProvider` to `ServerCapabilities` struct
- [x] Add method constant `MethodSemanticTokensFull` to protocol.go
- [x] Create `ProcessSemanticTokensRequest` handler in `lsp/methods.go`
- [x] Create `SemanticTokens` function in `internal/template/template.go`
- [x] Add method routing in `cmd/go-template-lsp/main.go`
- [x] Run lint and tests
