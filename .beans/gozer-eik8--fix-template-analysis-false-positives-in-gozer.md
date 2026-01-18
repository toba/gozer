---
# gozer-eik8
title: Fix template analysis false positives in gozer
status: in-progress
type: bug
created_at: 2026-01-18T21:08:34Z
updated_at: 2026-01-18T21:08:34Z
---

False positives still occurring when linting ../core/web:

1. Method calls with arguments like `.Format "2006-01-02"` flagged as 'only function and method accepts arguments'
2. Custom template function `timehtml` flagged as 'field or method not found'  
3. `.CloudLoggingURL $.ProjectID` flagged with both errors

These are valid Go template constructs that should not be flagged.