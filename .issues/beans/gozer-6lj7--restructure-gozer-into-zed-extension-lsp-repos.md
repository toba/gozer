---
# gozer-6lj7
title: Restructure gozer into Zed Extension + LSP repos
status: completed
type: task
priority: normal
created_at: 2026-01-19T17:40:51Z
updated_at: 2026-01-19T17:40:51Z
sync:
    github:
        issue_number: "10"
        synced_at: "2026-02-17T17:29:35Z"
---

Split the monorepo into:
- **gozer** (current repo) → Zed extension only
- **go-template-lsp** (new repo at ../go-template-lsp) → LSP binary + template library

## Checklist

### go-template-lsp repo setup
- [x] Create new directory at ../go-template-lsp
- [x] Copy go.mod and update module path to github.com/toba/go-template-lsp
- [x] Copy go.sum
- [x] Copy cmd/ and internal/ directories
- [x] Copy .goreleaser.yaml and update repo references
- [x] Copy .golangci.yaml
- [x] Copy .github/workflows/release.yml
- [x] Copy mise.toml
- [x] Copy LICENSE
- [x] Create README.md for LSP
- [x] Create CLAUDE.md for LSP development
- [x] Initialize git repo (already initialized)
- [x] Copy .claude and .zed directories

### gozer repo (Zed extension)
- [x] Move zed-ext/* contents to root
- [x] Delete empty zed-ext/ directory
- [x] Update src/lib.rs: change GitHub release URL to toba/go-template-lsp
- [x] Update README.md for Zed extension focus
- [x] Update CLAUDE.md for extension development
- [x] Remove Go-related files (cmd/, internal/, go.mod, go.sum, .goreleaser.yaml, .golangci.yaml, mise.toml, .github/workflows/release.yml)
- [x] Keep docs/gozer.png for README

### Verification
- [x] In go-template-lsp: go build ./... and go test ./...
- [x] In go-template-lsp: golangci-lint run
- [x] In gozer: extension.toml at root, valid structure
