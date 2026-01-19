# Template Package

This package provides lexing, parsing, and semantic analysis for Go templates (`text/template` / `html/template`). It powers IDE features like diagnostics, go-to-definition, and hover for template files.

## Architecture

The package follows a three-stage pipeline:

```
Source Code → Lexer → Parser → Analyzer → Diagnostics/LSP Features
```

### Package Structure

```
internal/template/
├── template.go          # Public API: parsing and analysis entry points
├── lexer/               # Tokenization of template syntax
├── parser/              # AST construction from tokens
└── analyzer/            # Semantic analysis and type checking
```

## Sub-packages

### lexer/

Tokenizes Go template source code, extracting `{{...}}` blocks and breaking them into tokens.

**Key types:**
- `Token` - A single lexical element (keyword, variable, operator, etc.)
- `StreamToken` - All tokens from one `{{...}}` block
- `Kind` - Token type identifier (Keyword, DollarVariable, DotVariable, etc.)
- `Position` / `Range` - Source locations

**Entry point:**
```go
func Tokenize(content []byte) ([]*StreamToken, []Error)
```

**Files:**
- `lexer.go` - Core types and `Tokenize()` function
- `lexer_tokenize.go` - Token stream processing
- `lexer_extract.go` - Extracts `{{...}}` blocks from source
- `lexer_patterns.go` - Regex patterns for token recognition
- `lexer_enum.go` - Token kind constants
- `lexer_position.go` - Position/range utilities

### parser/

Builds an Abstract Syntax Tree (AST) from token streams. Handles control flow (`if`, `range`, `with`), template definitions, variable declarations, and expressions.

**Key types:**
- `AstNode` - Interface for all AST nodes
- `GroupStatementNode` - Block with nested statements (if, range, with, define)
- `ExpressionNode` - Single expression (function call, variable, literal)
- `MultiExpressionNode` - Pipeline of expressions (`a | b | c`)
- `VariableDeclarationNode` - Variable declaration (`$x := expr`)
- `TemplateStatementNode` - Template invocation (`{{template "name" .}}`)
- `CommentNode` - Template comments, including `go:code` directives

**Entry point:**
```go
func Parse(streams []*lexer.StreamToken) (*GroupStatementNode, []lexer.Error)
```

**Files:**
- `parser.go` - Core `Parser` type and `Parse()` function
- `ast.go` - AST node type definitions
- `parser_expression.go` - Expression parsing
- `parser_keywords.go` - Control flow keyword parsing
- `parser_statement.go` - Statement-level parsing
- `parser_scope.go` - Scope management during parsing
- `parser_util.go` - Parser utilities (peek, expect, etc.)

### analyzer/

Performs semantic analysis: type checking, variable resolution, template dependency analysis, and provides data for LSP features.

**Key types:**
- `FileDefinition` - Analysis results for a single file
- `VariableDefinition` - Variable with its type and scope
- `FunctionDefinition` - Template function signature
- `TemplateDefinition` - Defined template with input type
- `NodeDefinition` - Interface for all symbol definitions

**Key concepts:**
- **Type inference**: Variables declared as `any` have their types inferred from usage
- **Template dependencies**: Tracks which templates call which others
- **go:code directives**: Parses Go code in comments to extract type hints

**Entry points:**
```go
// Single file analysis
func DefinitionAnalysisSingleFile(fileName string, workspace map[string]*parser.GroupStatementNode) (*FileDefinition, []Error)

// Workspace-wide analysis
func DefinitionAnalysisWithinWorkspace(workspace map[string]*parser.GroupStatementNode) []FileAnalysisAndError
```

**Files:**
- `analyzer.go` - Built-in functions, entry points
- `analyzer_types.go` - Type definitions (FileDefinition, VariableDefinition, etc.)
- `analyzer_statements.go` - Statement analysis (groups, templates, comments)
- `analyzer_expression.go` - Expression analysis
- `analyzer_variables.go` - Variable declaration/assignment analysis
- `analyzer_inference.go` - Type inference logic
- `analyzer_typecheck.go` - Type compatibility checking
- `analyzer_implicit.go` - Implicit type tree for inference
- `analyzer_lsp.go` - LSP feature support (hover, go-to-definition)
- `template_dependencies_analysis.go` - Cross-template dependency tracking
- `funcmap_scanner.go` - Scans Go files for custom template functions

## Public API (template.go)

The main `template` package provides high-level functions:

```go
// File operations
func OpenProjectFiles(rootDir string, extensions []string) map[string][]byte

// Parsing
func ParseSingleFile(source []byte) (*parser.GroupStatementNode, []Error)
func ParseFilesInWorkspace(files map[string][]byte) (map[string]*parser.GroupStatementNode, []Error)

// Analysis
func DefinitionAnalysisSingleFile(fileName string, workspace ...) (*FileDefinition, []Error)
func DefinitionAnalysisWithinWorkspace(workspace ...) []FileAnalysisAndError
func DefinitionAnalysisChainTriggeredBySingleFileChange(workspace ..., fileName string) []FileAnalysisAndError

// LSP features
func GoToDefinition(file *FileDefinition, position lexer.Position) ([]string, []lexer.Range, error)
func Hover(file *FileDefinition, position lexer.Position) (string, lexer.Range)
func FoldingRange(root *parser.GroupStatementNode) ([]*parser.GroupStatementNode, []*parser.CommentNode)

// Custom functions
func SetWorkspaceCustomFunctions(funcs map[string]*FunctionDefinition)
```

## Type Inference

The analyzer supports type inference for variables with unknown types (declared as `any`). When a variable is used in a context that requires a specific type, the analyzer records this constraint and resolves it at scope end.

Example:
```html
{{/* go:code
type Input struct {
    Users []User
}
*/}}

{{range .Users}}        <!-- .Users inferred as []User -->
    {{.Name}}           <!-- . inferred as User, .Name checked against User fields -->
{{end}}
```

## go:code Directives

Templates can include Go code in comments to provide type hints:

```html
{{/* go:code
type Input struct {
    Title string
    Items []Item
}

func formatDate(t time.Time) string
*/}}
```

The analyzer parses this Go code to:
1. Set the type of `.` (the Input type)
2. Register custom functions with their signatures
