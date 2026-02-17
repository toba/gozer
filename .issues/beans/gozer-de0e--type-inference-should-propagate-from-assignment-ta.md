---
# gozer-de0e
title: Type inference should propagate from assignment target to source
status: completed
type: bug
priority: normal
created_at: 2026-01-19T02:38:44Z
updated_at: 2026-01-19T02:43:35Z
sync:
    github:
        issue_number: "3"
        synced_at: "2026-02-17T17:29:35Z"
---

In portfolio-integrations.html, the pattern `{{$current := ""}}{{if .Integration}}{{$current = .Integration.Platform}}{{end}}` causes a type mismatch error. The variable $current is inferred as string from initialization, but .Integration.Platform is inferred as 'any'. The type system should propagate the expected type from the assignment target to infer that .Integration.Platform should be string.
