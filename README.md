# Gozer

<img src="./docs/gozer.png" alt="Gozer gopher" width="100"/>

A Zed editor extension for Go templates (`text/template` and `html/template`), powered by [go-template-lsp](https://github.com/STR-Consulting/go-template-lsp).

## Features

- **Diagnostics**: Real-time syntax error detection as you type
- **Hover**: Type information and documentation on hover over template variables and functions
- **Go to Definition**: Navigate to template definitions (`{{define "name"}}`)
- **Folding Ranges**: Collapse template blocks (`{{if}}...{{end}}`, `{{range}}...{{end}}`) and comments
- **Semantic Tokens**: Enhanced syntax highlighting
- **Document Highlight**: Highlight matching template keywords

## Supported File Extensions

| Extension | Type |
|-----------|------|
| `.gotmpl`, `.go.tmpl`, `.gtpl`, `.tpl`, `.tmpl` | Go text templates |
| `.gohtml`, `.go.html` | Go HTML templates |
| `.html` | HTML (with template detection) |
| `.htmx2.gohtml` | Go HTML templates with HTMX 2.x |
| `.htmx4.gohtml` | Go HTML templates with HTMX 4.x |

### HTMX Support

The extension includes variants for [HTMX](https://htmx.org/) projects:

- **Go HTML Template (HTMX 2)**: For HTMX 2.x projects (stable)
- **Go HTML Template (HTMX 4)**: For HTMX 4.x projects (alpha, expected stable early-mid 2026)

To enable HTMX support, either:

1. **Use the file extension**: Name your files with `.htmx2.gohtml` or `.htmx4.gohtml`
2. **Select manually**: Use Zed's language picker (click the language name in the status bar) to choose the HTMX variant
3. **Configure in settings**: Add to your Zed `settings.json`:

```json
{
  "file_types": {
    "Go HTMX 2": ["gohtml"]
  }
}
```

The Zed language IDs for HTMX variants are `Go HTMX 2` and `Go HTMX 4`.

## Installation

1. Open Zed
2. Go to Extensions (Cmd+Shift+X)
3. Search for "Go Template LSP"
4. Click Install

The extension automatically downloads the appropriate LSP binary for your platform.

**As a Dev Extension (for local development):**

1. Clone this repository
2. In Zed, open the command palette (Cmd+Shift+P)
3. Run "zed: install dev extension"
4. Select this directory

See [Zed's extension development docs](https://zed.dev/docs/extensions/developing-extensions#developing-an-extension-locally) for more details.

**Configuration:**

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

To add additional file extensions, use `file_types`:

```json
{
  "file_types": {
    "Go HTML Template": ["tmpl", "html"]
  }
}
```

## Building the Extension

```bash
cargo build --target wasm32-wasip1
```

## Credits

### LSP Server

**[STR-Consulting/go-template-lsp](https://github.com/STR-Consulting/go-template-lsp)** (MIT License)

The Go template LSP server that powers this extension.

### Tree-sitter Grammar

**[ngalaiko/tree-sitter-go-template](https://github.com/ngalaiko/tree-sitter-go-template)** (MIT License)

Provides the tree-sitter grammar for parsing Go template syntax.

### Syntax Highlighting Queries

**[hjr265/zed-gotmpl](https://github.com/hjr265/zed-gotmpl)** (MIT License)

The tree-sitter query patterns for syntax highlighting are adapted from this project by Mahmud Ridwan.

## License

MIT License - see [LICENSE](LICENSE) for details.
