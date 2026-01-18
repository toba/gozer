# Gozer

<img src="./docs/gozer.png" alt="Gozer gopher" width="100"/>

A Language Server Protocol (LSP) implementation for Go templates (`text/template` and `html/template`), with a Zed editor extension.

> [!NOTE]
> 🤖 This project was initially set up with a carefully guided LLM (Claude Opus 4.5)

## Features

- **Diagnostics**: Real-time syntax error detection as you type
- **Hover**: Type information and documentation on hover over template variables and functions
- **Go to Definition**: Navigate to template definitions (`{{define "name"}}`)
- **Folding Ranges**: Collapse template blocks (`{{if}}...{{end}}`, `{{range}}...{{end}}`) and comments

## Supported File Extensions

| Extension | Type |
|-----------|------|
| `.gotmpl`, `.go.tmpl`, `.gtpl`, `.tpl`, `.tmpl` | Go text templates |
| `.gohtml`, `.go.html` | Go HTML templates |
| `.html` | HTML (with template detection) |

## Installation

### Zed Editor

1. Open Zed
2. Go to Extensions (Cmd+Shift+X)
3. Search for "Go Template LSP"
4. Click Install

The extension automatically downloads the appropriate LSP binary for your platform.

**As a Dev Extension (for local development):**

1. Clone this repository
2. In Zed, open the command palette (Cmd+Shift+P)
3. Run "zed: install dev extension"
4. Select the `zed-ext` directory

This loads the extension from your local checkout, useful for testing changes. See [Zed's extension development docs](https://zed.dev/docs/extensions/developing-extensions#developing-an-extension-locally) for more details.

**Zed Settings Configuration:**

After installing the extension, you can customize file associations in your Zed `settings.json` (Cmd+, → "Open Settings"):

```json
{
  "languages": {
    "Go HTML Template": {
      "language_servers": ["go-template-lsp"]
    },
    "Go Text Template": {
      "language_servers": ["go-template-lsp"]
    }
  }
}
```

The language names "Go Text Template" and "Go HTML Template" are registered by this extension. They automatically apply to these file extensions:

- **Go Text Template**: `.gotmpl`, `.go.tmpl`, `.gtpl`, `.tpl`
- **Go HTML Template**: `.gohtml`, `.go.html`, `.html.tmpl`

To add additional file extensions, use `file_types`:

```json
{
  "file_types": {
    "Go HTML Template": ["tmpl", "html"]
  }
}
```

### Standalone LSP Binary

Download prebuilt binaries from [GitHub Releases](https://github.com/STR-Consulting/gozer/releases), or build from source:

```bash
go install github.com/STR-Consulting/gozer/cmd/go-template-lsp@latest
```

### Other Editors

The LSP binary works with any editor that supports the Language Server Protocol. Configure your editor to run `go-template-lsp` for the supported file extensions.

**Neovim (with nvim-lspconfig):**
```lua
local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

configs.gotmpl = {
  default_config = {
    cmd = { 'go-template-lsp' },
    filetypes = { 'gotmpl', 'gohtml', 'html' },
    root_dir = lspconfig.util.root_pattern('go.mod', '.git'),
  },
}

lspconfig.gotmpl.setup{}
```

## Building from Source

```bash
# Build the LSP binary
go build -o go-template-lsp ./cmd/go-template-lsp

# Or install to $GOBIN
go install ./cmd/go-template-lsp

# Run tests
go test ./...

# Run linter
golangci-lint run
```

### Building the Zed Extension

```bash
cd zed-ext
cargo build --target wasm32-wasip1
```

## Architecture

The LSP server uses a concurrent architecture:

1. **Main loop**: Handles JSON-RPC requests from the editor
2. **Diagnostic goroutine**: Processes file changes and publishes diagnostics

Key dependencies:
- [github.com/yayolande/gota](https://github.com/yayolande/gota) - Go template parser and analyzer

## Credits

This project builds on the work of several open source projects:

### LSP Implementation
**[yayolande/go-template-lsp](https://github.com/yayolande/go-template-lsp)** (MIT License)

The core language server implementation is based on this project by yayolande. It provides the parsing, analysis, hover, go-to-definition, and diagnostic functionality through the [gota](https://github.com/yayolande/gota) library.

### Tree-sitter Grammar
**[ngalaiko/tree-sitter-go-template](https://github.com/ngalaiko/tree-sitter-go-template)** (MIT License)

Provides the tree-sitter grammar for parsing Go template syntax, used by the Zed extension for syntax highlighting and code structure.

### Syntax Highlighting Queries
**[hjr265/zed-gotmpl](https://github.com/hjr265/zed-gotmpl)** (MIT License)

The tree-sitter query patterns for syntax highlighting in Zed are adapted from this project by Mahmud Ridwan.

## License

MIT License - see [LICENSE](LICENSE) for details.

This project is a derivative work that combines and builds upon the MIT-licensed projects listed above. The LICENSE file includes attribution to all original authors.
