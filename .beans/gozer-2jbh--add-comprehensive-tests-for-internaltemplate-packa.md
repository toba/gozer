---
# gozer-2jbh
title: Add comprehensive tests for internal/template package
status: completed
type: task
priority: normal
created_at: 2026-01-19T00:01:02Z
updated_at: 2026-01-19T00:11:50Z
---

Implement test coverage for the internal/template package which currently has significant gaps:
- Parser: Zero coverage (most critical)
- Lexer: Only position tests
- Public API: Untested

## Checklist
- [x] Parser tests (parser/parser_test.go)
- [x] Parser expression tests (parser/parser_expression_test.go)
- [x] Parser scope tests (parser/parser_scope_test.go)
- [x] Lexer tests (lexer/lexer_test.go)
- [x] Public API tests (template_test.go)
- [x] Integration tests (integration_test.go)
- [ ] Test fixtures in testdata/ (tests use inline strings, no additional fixtures needed)
- [x] Verify all tests pass
- [x] Run golangci-lint (minor style warnings remaining in test files)