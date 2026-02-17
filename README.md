# Gozer

<img src="./docs/gozer.png" alt="Gozer gopher" width="100"/>

A Zed editor extension for Go templates (`text/template` and `html/template`), powered by [go-template-lsp](https://github.com/toba/go-template-lsp).

## Features

- **Diagnostics**: Real-time syntax error detection and semantic analysis (undefined variables, missing fields, unknown functions)
- **Hover**: Type information and documentation on hover over template variables and functions
- **Go to Definition**: Navigate to template definitions (`{{define "name"}}`)
- **Formatting**: Re-indent based on HTML and template nesting, with optional attribute wrapping
- **Folding Ranges**: Collapse template blocks (`{{if}}...{{end}}`, `{{range}}...{{end}}`) and comments
- **Semantic Tokens**: Enhanced syntax highlighting for keywords, variables, functions, strings, numbers, and operators
- **Document Highlight**: Highlight matching template keywords (e.g. click `{{if}}` to highlight its `{{else}}` and `{{end}}`)
- **Document Links**: Clickable links in template documents
- **Custom Function Discovery**: Automatically scans Go source files for `template.FuncMap` definitions so custom functions are recognized

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

### Formatting

The LSP formatter re-indents Go template files based on HTML tag and template action nesting. It uses the standard `tabSize` and `insertSpaces` settings from Zed.

To enable attribute wrapping, add initialization options to your Zed `settings.json`:

```json
{
  "lsp": {
    "go-template-lsp": {
      "initialization_options": {
        "printWidth": 120,
        "attrWrapMode": "overflow"
      }
    }
  }
}
```

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `printWidth` | `int` | `0` (disabled) | Maximum line width before wrapping attributes. Set to `0` to disable. |
| `attrWrapMode` | `string` | `"overflow"` | `"overflow"`: only wrap attributes that push past `printWidth`. `"all"`: wrap every attribute onto its own line. |

**`overflow` mode** keeps attributes on the first line as long as they fit, then wraps the rest:

```html
<button type="button"
   class="map-zoom-btn"
   title="Expand">
```

**`all` mode** puts every attribute on its own line:

```html
<button
   type="button"
   class="map-zoom-btn"
   title="Expand">
```

Continuation lines are indented one level deeper than the tag. Multi-line tags already in the source are joined back into a single line before re-wrapping.

## Building the Extension

```bash
cargo build --target wasm32-wasip1
```

## Credits

**[toba/go-template-lsp](https://github.com/toba/go-template-lsp)** — The Go template LSP server that powers this extension.

**[yayolande/gota](https://github.com/yayolande/gota)** — Template parsing and semantic analysis, by yayolande.

**[yayolande/go-template-lsp](https://github.com/yayolande/go-template-lsp)** — LSP server architecture, by yayolande.

**[hjr265/zed-gotmpl](https://github.com/hjr265/zed-gotmpl)** — Tree-sitter query patterns for syntax highlighting, adapted from this project by Mahmud Ridwan.

## License

MIT License - see [LICENSE](LICENSE) for details.
