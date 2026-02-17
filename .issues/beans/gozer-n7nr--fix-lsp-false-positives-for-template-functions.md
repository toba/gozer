---
# gozer-n7nr
title: Fix LSP false positives for template functions
status: completed
type: bug
priority: normal
created_at: 2026-01-18T19:11:12Z
updated_at: 2026-01-18T19:39:46Z
sync:
    github:
        issue_number: "65"
        synced_at: "2026-02-17T17:29:35Z"
---

# Fix LSP false positives for template functions

The LSP incorrectly flags template functions as 'function undefined'.

## Symptoms

1. Custom functions like `lower`, `upper`, `platformKnown` are flagged because they're not in gota's builtin list
2. Even builtins like `and`, `not`, `eq` are flagged despite being defined

## Root Cause Found

The bug is in **github.com/yayolande/gota** (v0.8.3) - the Go Template LSP analyzer library that gozer depends on.

### Location
`analyzer/analyzer.go` in the `getBuiltinFunctionDefinition()` function (lines 546-586)

### The Bug

1. `FunctionDefinition.typ` is declared as `*types.Signature` (pointer type, line 252)
2. `VariableDefinition.typ` is declared as `types.Type` (interface type, line 291)
3. For builtins, `def.typ` is NEVER SET - it remains nil `*types.Signature`
4. When copied via `fakeVarDef.typ = def.typ`, Go creates a **non-nil interface with nil value**
5. The nil check `if parentType == nil` (line 2495) fails because the interface itself is not nil
6. The nil `*types.Signature` flows through to `makeFunctionTypeCheck` (line 2814) causing "function undefined"

This is the classic Go nil interface gotcha:
```go
var p *types.Signature = nil  // typed nil
var i types.Type = p          // interface is NOT nil! It has type info.
fmt.Println(i == nil)         // FALSE
```

### Code Flow

1. `getBuiltinFunctionDefinition()` creates FunctionDefinition with `typ = nil` (line 578-584)
2. Function lookup succeeds: `def := p.file.functions[functionName]` (line 2068)
3. `fakeVarDef.typ = def.typ` copies nil pointer to interface (line 2089)
4. `getRealTypeAssociatedToVariable()` doesn't catch nil because interface != nil (line 2494-2496)
5. Returns the typed nil which becomes `processedTypes[0]`
6. In `makeExpressionTypeCheck()`, type assertion `typs[0].(*types.Signature)` succeeds but value is nil
7. `makeFunctionTypeCheck(funcType, ...)` receives nil funcType (line 2806)
8. Check `if funcType == nil` is true, generates "function undefined" (line 2815)

## Debug Evidence

Created local fork at `/Users/jason/Developer/pacer/gota-debug` (now deleted) with debug logging:

```
[DEBUG] Looking up function 'and', found: true, functions map size: 23
[DEBUG] Function 'and' found, def.typ=*types.Signature (<nil>), fakeVarDef.typ=*types.Signature (<nil>)
[DEBUG] After getRealTypeAssociatedToVariable: symbolType=*types.Signature (<nil>)
[DEBUG] Before makeExpressionTypeCheck: len(processedToken)=3, len(processedTypes)=3
[DEBUG]   token[0]: 'and', type: *types.Signature (<nil>)   <-- typed nil!
[DEBUG]   token[1]: 'true', type: *types.Basic (bool)
[DEBUG]   token[2]: 'false', type: *types.Basic (bool)
[DEBUG] makeFunctionTypeCheck: funcType is nil for 'and'
```

## Fix Options

### Option A: Fix in gota library (Recommended)
- Set proper `*types.Signature` for each builtin function in `getBuiltinFunctionDefinition()`
- Builtins need real signatures, e.g.:
  - `and`: variadic, returns last arg type
  - `or`: variadic, returns first non-empty arg type  
  - `eq`, `ne`, `lt`, `le`, `gt`, `ge`: 2+ args, returns bool
  - `not`: 1 arg, returns bool
  - `len`: 1 arg, returns int
  - `print`, `printf`, `println`: variadic, returns string
- Requires forking gota or contributing upstream

### Option B: Suppress errors in gozer
- Filter out "function undefined" errors for known builtins
- Doesn't fix root cause but provides workaround

### Option C: Custom functions configuration
- Add `.gozer.json` or similar config to specify additional functions
- Useful for Sprig functions and project-specific template functions
- Could also integrate with gopls to discover `template.FuncMap` definitions

## Test Files

- `/Users/jason/Developer/pacer/gozer/testdata/credential-form.html` - real template demonstrating issue
- `/Users/jason/Developer/pacer/gozer/cmd/go-template-lsp/main_test.go` - unit tests

## Test Case

```go
func TestSimpleExpression(t *testing.T) {
    content := []byte(`{{and true false}}`)
    // ... parses and analyzes, produces "function undefined" for 'and'
}
```

## Related Files in gota (v0.8.3)

- `analyzer/analyzer.go:244-253` - FunctionDefinition struct
- `analyzer/analyzer.go:285-296` - VariableDefinition struct  
- `analyzer/analyzer.go:546-586` - getBuiltinFunctionDefinition()
- `analyzer/analyzer.go:2062-2100` - FUNCTION token handling
- `analyzer/analyzer.go:2479-2512` - getRealTypeAssociatedToVariable()
- `analyzer/analyzer.go:2797-2811` - makeExpressionTypeCheck() type assertion
- `analyzer/analyzer.go:2814-2820` - makeFunctionTypeCheck() nil check
