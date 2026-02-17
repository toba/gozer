---
# gozer-iynv
title: Fix Zed extension release pipeline
status: completed
type: bug
priority: normal
created_at: 2026-01-18T18:30:20Z
updated_at: 2026-01-18T18:30:49Z
sync:
    github:
        issue_number: "39"
        synced_at: "2026-02-17T17:29:35Z"
---

The Zed extension fails with 404 because:
1. No GitHub Actions workflow exists to run goreleaser
2. lib.rs references wrong repo (pacer/gozer instead of toba/gozer)

## Checklist
- [x] Create .github/workflows/release.yml
- [x] Fix GITHUB_REPO in zed-ext/src/lib.rs
