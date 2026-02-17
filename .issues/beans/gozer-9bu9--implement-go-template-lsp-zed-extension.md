---
# gozer-9bu9
title: Implement Go Template LSP Zed Extension
status: completed
type: feature
priority: normal
created_at: 2026-01-18T16:08:51Z
updated_at: 2026-01-18T16:08:51Z
sync:
    github:
        issue_number: "53"
        synced_at: "2026-02-17T17:29:35Z"
---

Create a Zed extension with LSP support for Go templates (`text/template` and `html/template`). Port code from yayolande/go-template-lsp with full attribution.

## Features

- **Diagnostics**: Real-time error detection for template syntax
- **Hover**: Type information and documentation on hover
- **Go to Definition**: Navigate to template definitions
- **Folding Ranges**: Collapse template blocks and comments
- **Syntax Highlighting**: Full highlighting via tree-sitter grammar

## Supported File Extensions

- `.gotmpl`, `.go.tmpl`, `.gtpl`, `.tpl` - Go text templates
- `.gohtml`, `.go.html`, `.html.tmpl` - Go HTML templates

## Architecture

**Binary Distribution**: The Go LSP binary is built via goreleaser and published to GitHub releases. The Zed extension downloads the appropriate binary for the user's platform on first use.

**Data Flow**: `main.go` → `lsp.ReceiveInput()` → LSP handlers → `gota` package (parsing/analysis) → `lsp.SendToLspClient()`

## Files Created

### Go LSP Server (`cmd/go-template-lsp/`)
- `main.go` - Server entry point with workspace management, async diagnostic processing
- `lsp/methods.go` - LSP handlers (initialize, hover, definition, folding ranges, diagnostics)
- `lsp/parsing.go` - Content-Length protocol encoding/decoding

### Zed Extension (`zed-ext/`)
- `extension.toml` - Zed extension manifest
- `Cargo.toml` - Rust dependencies (zed_extension_api 0.7.0)
- `src/lib.rs` - Extension logic that downloads LSP binary from GitHub releases
- `languages/gotmpl/` - Go Text Template language config, highlights.scm, brackets.scm
- `languages/gohtml/` - Go HTML Template language config, highlights.scm, brackets.scm
- `LICENSE` - MIT license with attribution
- `README.md` - Installation instructions and credits

### Project Files
- `go.mod` / `go.sum` - Go module with gota dependency
- `.goreleaser.yaml` - Updated with go-template-lsp binary build

## Dependencies

- **Go**: `github.com/yayolande/gota` - Template parsing and semantic analysis
- **Rust**: `zed_extension_api 0.7.0` - Zed extension WASM API
- **Tree-sitter**: `ngalaiko/tree-sitter-go-template` - Grammar for syntax highlighting

## Credits

- **LSP implementation**: Based on [yayolande/go-template-lsp](https://github.com/yayolande/go-template-lsp) (MIT License)
- **Tree-sitter grammar**: [ngalaiko/tree-sitter-go-template](https://github.com/ngalaiko/tree-sitter-go-template) (MIT License)
- **Syntax highlighting queries**: Adapted from [hjr265/zed-gotmpl](https://github.com/hjr265/zed-gotmpl) (MIT License)

## Checklist

- [x] Create Go LSP server (cmd/go-template-lsp)
  - [x] main.go - Server entry point
  - [x] lsp/methods.go - LSP handlers
  - [x] lsp/parsing.go - Content-Length protocol encoding
- [x] Create Zed extension (zed-ext/)
  - [x] extension.toml - Zed extension manifest
  - [x] Cargo.toml - Rust dependencies
  - [x] src/lib.rs - Extension binary download logic
- [x] Create language configs
  - [x] languages/gotmpl/config.toml
  - [x] languages/gotmpl/highlights.scm
  - [x] languages/gotmpl/brackets.scm
  - [x] languages/gohtml/config.toml
  - [x] languages/gohtml/highlights.scm
  - [x] languages/gohtml/brackets.scm
- [x] Create documentation
  - [x] LICENSE (MIT)
  - [x] README.md
- [x] Update .goreleaser.yaml for the new binary
- [x] Build and verify Zed extension

## Testing

```bash
# Build LSP binary
go build -o go-template-lsp ./cmd/go-template-lsp

# Test version flag
./go-template-lsp --version

# Build Zed extension WASM
cd zed-ext && cargo build --target wasm32-wasip1

# Install as dev extension in Zed
# Extensions > "Install Dev Extension" > select zed-ext/
```
