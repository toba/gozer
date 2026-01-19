package lsp

// LSP protocol constants.
const (
	// JSONRPCVersion is the JSON-RPC protocol version.
	JSONRPCVersion = "2.0"

	// SeverityError indicates an error diagnostic per LSP spec.
	SeverityError   = 1
	SeverityWarning = 2
	SeverityInfo    = 3
	SeverityHint    = 4

	// TextDocumentSyncFull indicates full document sync mode.
	TextDocumentSyncFull = 1

	// ErrorInvalidRequest is the JSON-RPC error code for invalid requests.
	ErrorInvalidRequest = -32600
)

// LSP method names.
const (
	MethodInitialize          = "initialize"
	MethodInitialized         = "initialized"
	MethodShutdown            = "shutdown"
	MethodExit                = "exit"
	MethodDidOpen             = "textDocument/didOpen"
	MethodDidChange           = "textDocument/didChange"
	MethodDidClose            = "textDocument/didClose"
	MethodHover               = "textDocument/hover"
	MethodDefinition          = "textDocument/definition"
	MethodFoldingRange        = "textDocument/foldingRange"
	MethodDocumentHighlight   = "textDocument/documentHighlight"
	MethodPublishDiagnostics  = "textDocument/publishDiagnostics"
	MethodSemanticTokensFull  = "textDocument/semanticTokens/full"
	MethodSemanticTokensRange = "textDocument/semanticTokens/range"
)

// Semantic token types (indices into the legend).
const (
	SemanticTokenKeyword = iota
	SemanticTokenVariable
	SemanticTokenFunction
	SemanticTokenProperty // for fields like .FieldName
	SemanticTokenString
	SemanticTokenNumber
	SemanticTokenOperator
	SemanticTokenComment
)

// SemanticTokenTypes is the legend for token types.
var SemanticTokenTypes = []string{
	"keyword",
	"variable",
	"function",
	"property",
	"string",
	"number",
	"operator",
	"comment",
}

// Semantic token modifiers (bit flags).
const (
	SemanticModifierDeclaration = 1 << iota
	SemanticModifierDefinition
	SemanticModifierReadonly
)

// SemanticTokenModifiers is the legend for token modifiers.
var SemanticTokenModifiers = []string{
	"declaration",
	"definition",
	"readonly",
}

// LSP header constants.
const (
	ContentLengthHeader = "Content-Length"
	HeaderDelimiter     = "\r\n\r\n"
	LineDelimiter       = "\r\n"
)

// File and logging constants.
const (
	DirPermissions  = 0750
	FilePermissions = 0600
	MaxLogFileSize  = 5_000_000 // 5MB
)
