---
# gozer-txnj
title: Add HTMX Attribute Highlighting to Go HTML Template Extension
status: completed
type: feature
priority: normal
created_at: 2026-01-18T19:27:30Z
updated_at: 2026-01-18T19:28:57Z
sync:
    github:
        issue_number: "54"
        synced_at: "2026-02-17T17:29:35Z"
---

Add HTMX attribute highlighting to the gohtml language with two variants: gohtml-htmx2 (HTMX 2.x) and gohtml-htmx4 (HTMX 4.x). Creates separate language definitions with version-specific attribute support.

## Checklist
- [x] Read existing gohtml files to understand structure
- [x] Create gohtml-htmx2/ directory with config.toml, highlights.scm, injections.scm, brackets.scm
- [x] Create gohtml-htmx4/ directory with config.toml, highlights.scm, injections.scm, brackets.scm
- [x] Update extension.toml to include the new languages
- [x] Verify with golangci-lint run
- [x] Verify with go test ./...
