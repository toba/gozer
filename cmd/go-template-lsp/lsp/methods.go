// Package lsp implements LSP message types and handlers for Go templates.
//
// Based on https://github.com/yayolande/go-template-lsp (MIT License)
package lsp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"sync"

	tmpl "github.com/pacer/gozer/internal/template"
	"github.com/pacer/gozer/internal/template/analyzer"
	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

var filesOpenedByEditor = make(map[string]string)

// ID represents a JSON-RPC request ID that can be either a string or number.
type ID int

func (id *ID) UnmarshalJSON(data []byte) error {
	length := len(data)
	if data[0] == '"' && data[length-1] == '"' {
		data = data[1 : length-1]
	}

	number, err := strconv.Atoi(string(data))
	if err != nil {
		return errors.New("'ID' expected either a string or an integer")
	}

	*id = ID(number)
	return nil
}

func (id *ID) MarshalJSON() ([]byte, error) {
	val := strconv.Itoa(int(*id))
	return []byte(val), nil
}

// RequestMessage represents a JSON-RPC request.
type RequestMessage[T any] struct {
	JsonRpc string `json:"jsonrpc"`
	Id      ID     `json:"id"`
	Method  string `json:"method"`
	Params  T      `json:"params"`
}

// ResponseMessage represents a JSON-RPC response.
type ResponseMessage[T any] struct {
	JsonRpc string         `json:"jsonrpc"`
	Id      ID             `json:"id"`
	Result  T              `json:"result"`
	Error   *ResponseError `json:"error"`
}

// ResponseError represents a JSON-RPC error.
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NotificationMessage represents a JSON-RPC notification (no response expected).
type NotificationMessage[T any] struct {
	JsonRpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  T      `json:"params"`
}

// InitializeParams holds parameters for the initialize request.
type InitializeParams struct {
	ProcessId    int            `json:"processId"`
	Capabilities map[string]any `json:"capabilities"`
	ClientInfo   struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
	Locale                string `json:"locale"`
	RootUri               string `json:"rootUri"`
	Trace                 any    `json:"trace"`
	WorkspaceFolders      any    `json:"workspaceFolders"`
	InitializationOptions any    `json:"initializationOptions"`
}

// ServerCapabilities describes the capabilities this server supports.
type ServerCapabilities struct {
	TextDocumentSync     int  `json:"textDocumentSync"`
	HoverProvider        bool `json:"hoverProvider"`
	DefinitionProvider   bool `json:"definitionProvider"`
	FoldingRangeProvider bool `json:"foldingRangeProvider"`
}

// InitializeResult is the response to the initialize request.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

// PublishDiagnosticsParams holds parameters for publishing diagnostics.
type PublishDiagnosticsParams struct {
	Uri         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Diagnostic represents a diagnostic message (error, warning, etc.).
type Diagnostic struct {
	Range    Range  `json:"range"`
	Message  string `json:"message"`
	Severity int    `json:"severity"`
}

// Position represents a position in a text document.
type Position struct {
	Line      uint `json:"line"`
	Character uint `json:"character"`
}

// Range represents a range in a text document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// TextDocumentItem represents a text document.
type TextDocumentItem struct {
	Uri        string `json:"uri"`
	Version    int    `json:"version"`
	LanguageId string `json:"languageId"`
	Text       string `json:"text"`
}

// TextDocumentIdentifier identifies a text document.
type TextDocumentIdentifier struct {
	Uri string `json:"uri"`
}

// TextDocumentPositionParams combines a document identifier with a position.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// Location represents a location in a text document.
type Location struct {
	Uri   string `json:"uri"`
	Range Range  `json:"range"`
}

// MarkupContent represents markup content (markdown or plaintext).
type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// intToUint safely converts int to uint, returning 0 for negative values.
func intToUint(v int) uint {
	if v < 0 {
		return 0
	}
	return uint(v) //nolint:gosec // bounds checked above
}

// uintToInt safely converts uint to int, clamping to max int for overflow.
func uintToInt(v uint) int {
	const maxInt = int(^uint(0) >> 1)
	if v > uint(maxInt) {
		return maxInt
	}
	return int(v) //nolint:gosec // bounds checked above
}

// ConvertParserRangeToLspRange converts a parser range to an LSP range.
func ConvertParserRangeToLspRange(parserRange lexer.Range) Range {
	if parserRange.IsEmpty() {
		return Range{}
	}

	return Range{
		Start: Position{
			Line:      intToUint(parserRange.Start.Line),
			Character: intToUint(parserRange.Start.Character),
		},
		End: Position{
			Line:      intToUint(parserRange.End.Line),
			Character: intToUint(parserRange.End.Character),
		},
	}
}

// ProcessInitializeRequest handles the initialize request.
func ProcessInitializeRequest(
	data []byte,
	lspName, lspVersion string,
) (response []byte, root string) {
	req := RequestMessage[InitializeParams]{}

	err := json.Unmarshal(data, &req)
	if err != nil {
		msg := "error while unmarshalling data during 'initialize' phase: " + err.Error()
		slog.Error(msg,
			slog.Group("details",
				slog.Any("unmarshalled_req", req),
				slog.String("received_req", string(data)),
			),
		)
		panic(msg)
	}

	res := ResponseMessage[InitializeResult]{
		JsonRpc: "2.0",
		Id:      req.Id,
		Result: InitializeResult{
			Capabilities: ServerCapabilities{
				TextDocumentSync:     1,
				HoverProvider:        true,
				DefinitionProvider:   true,
				FoldingRangeProvider: true,
			},
		},
	}

	res.Result.ServerInfo.Name = lspName
	res.Result.ServerInfo.Version = lspVersion

	response, err = json.Marshal(res)
	if err != nil {
		msg := "error while marshalling data during 'initialize' phase: " + err.Error()
		slog.Error(msg)
		panic(msg)
	}

	return response, req.Params.RootUri
}

// ProcessInitializedNotification handles the initialized notification.
func ProcessInitializedNotification(data []byte) {
	slog.Info("Received 'initialized' notification", slog.String("data", string(data)))
}

// ProcessShutdownRequest handles the shutdown request.
func ProcessShutdownRequest(jsonVersion string, requestId ID) []byte {
	response := ResponseMessage[any]{
		JsonRpc: jsonVersion,
		Id:      requestId,
		Result:  nil,
		Error:   nil,
	}

	responseText, err := json.Marshal(response)
	if err != nil {
		msg := "Error while marshalling shutdown response: " + err.Error()
		slog.Error(msg)
		panic(msg)
	}

	return responseText
}

// ProcessIllegalRequestAfterShutdown returns an error for requests after shutdown.
func ProcessIllegalRequestAfterShutdown(jsonVersion string, requestId ID) []byte {
	response := ResponseMessage[any]{
		JsonRpc: jsonVersion,
		Id:      requestId,
		Result:  nil,
		Error: &ResponseError{
			Code:    -32600,
			Message: "illegal request while server shutting down",
		},
	}

	responseText, err := json.Marshal(response)
	if err != nil {
		msg := "Error while marshalling error response: " + err.Error()
		slog.Error(msg)
		panic(msg)
	}

	return responseText
}

// DidOpenTextDocumentParams holds parameters for textDocument/didOpen.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// ProcessDidOpenTextDocumentNotification handles textDocument/didOpen.
func ProcessDidOpenTextDocumentNotification(
	data []byte,
) (fileURI string, fileContent []byte) {
	request := RequestMessage[DidOpenTextDocumentParams]{}

	err := json.Unmarshal(data, &request)
	if err != nil {
		msg := "error while unmarshalling 'textDocument/didOpen': " + err.Error()
		slog.Error(msg,
			slog.Group("details",
				slog.Any("unmarshalled_req", request),
				slog.String("received_req", string(data)),
			),
		)
		panic(msg)
	}

	documentURI := request.Params.TextDocument.Uri
	documentContent := request.Params.TextDocument.Text
	filesOpenedByEditor[documentURI] = documentContent

	return documentURI, []byte(documentContent)
}

// TextDocumentContentChangeEvent represents a content change event.
type TextDocumentContentChangeEvent struct {
	Range       Range  `json:"range"`
	RangeLength uint   `json:"rangeLength"`
	Text        string `json:"text"`
}

// DidChangeTextDocumentParams holds parameters for textDocument/didChange.
type DidChangeTextDocumentParams struct {
	TextDocument   TextDocumentItem                 `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// ProcessDidChangeTextDocumentNotification handles textDocument/didChange.
func ProcessDidChangeTextDocumentNotification(
	data []byte,
) (fileURI string, fileContent []byte) {
	var request RequestMessage[DidChangeTextDocumentParams]

	err := json.Unmarshal(data, &request)
	if err != nil {
		msg := "error while unmarshalling 'textDocument/didChange': " + err.Error()
		slog.Error(msg,
			slog.Group("details",
				slog.Any("unmarshalled_req", request),
				slog.String("received_req", string(data)),
			),
		)
		panic(msg)
	}

	documentChanges := request.Params.ContentChanges
	if len(documentChanges) > 1 {
		msg := "server doesn't handle incremental changes yet"
		slog.Error(msg,
			slog.Group("details",
				slog.Any("unmarshalled_req", request),
				slog.String("received_req", string(data)),
			),
		)
		panic(msg)
	}

	if len(documentChanges) == 0 {
		slog.Warn("'documentChanges' field is empty")
		return "", nil
	}

	documentContent := documentChanges[0].Text
	documentURI := request.Params.TextDocument.Uri
	filesOpenedByEditor[documentURI] = documentContent

	return documentURI, []byte(documentContent)
}

// DidCloseTextDocumentParams holds parameters for textDocument/didClose.
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// ProcessDidCloseTextDocumentNotification handles textDocument/didClose.
func ProcessDidCloseTextDocumentNotification(
	data []byte,
) (fileURI string, fileContent []byte) {
	var request RequestMessage[DidCloseTextDocumentParams]

	err := json.Unmarshal(data, &request)
	if err != nil {
		msg := "error while unmarshalling 'textDocument/didClose': " + err.Error()
		slog.Error(msg,
			slog.Group("details",
				slog.Any("unmarshalled_req", request),
				slog.String("received_req", string(data)),
			),
		)
		panic(msg)
	}

	documentPath := request.Params.TextDocument.Uri
	documentContent := request.Params.TextDocument.Text
	delete(filesOpenedByEditor, documentPath)

	return documentPath, []byte(documentContent)
}

// ProcessHoverRequest handles textDocument/hover.
func ProcessHoverRequest(
	data []byte,
	openFiles map[string]*analyzer.FileDefinition,
) []byte {
	type HoverParams struct {
		TextDocument TextDocumentItem `json:"textDocument"`
		Position     Position         `json:"position"`
	}

	var request RequestMessage[HoverParams]

	err := json.Unmarshal(data, &request)
	if err != nil {
		slog.Warn("Error unmarshalling hover request: " + err.Error())
		return nil
	}

	position := lexer.Position{
		Line:      uintToInt(request.Params.Position.Line),
		Character: uintToInt(request.Params.Position.Character),
	}

	file := openFiles[request.Params.TextDocument.Uri]
	if file == nil {
		msg := "file not found on server for hover request"
		slog.Error(msg,
			slog.Group("details",
				slog.String("uri", request.Params.TextDocument.Uri),
			),
		)
		panic(msg)
	}

	typeStringified, reach := tmpl.Hover(file, position)

	type HoverResult struct {
		Contents MarkupContent `json:"contents"`
		Range    Range         `json:"range,omitzero"`
	}

	response := ResponseMessage[*HoverResult]{
		JsonRpc: request.JsonRpc,
		Id:      request.Id,
		Result: &HoverResult{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: typeStringified,
			},
			Range: ConvertParserRangeToLspRange(reach),
		},
	}

	if typeStringified == "" {
		response.Result = nil
	}

	responseText, err := json.Marshal(response)
	if err != nil {
		slog.Warn("Error marshalling hover response: " + err.Error())
		return nil
	}

	return responseText
}

// DefinitionParams holds parameters for textDocument/definition.
type DefinitionParams struct {
	TextDocumentPositionParams
}

// DefinitionResults holds results for textDocument/definition.
type DefinitionResults struct {
	Location
}

// ProcessGoToDefinition handles textDocument/definition.
func ProcessGoToDefinition(
	data []byte,
	openFiles map[string]*analyzer.FileDefinition,
	rawFiles map[string][]byte,
) (response []byte, fileName string) {
	var req RequestMessage[DefinitionParams]

	err := json.Unmarshal(data, &req)
	if err != nil {
		slog.Warn("Error unmarshalling definition request: " + err.Error())
		return nil, ""
	}

	position := lexer.Position{
		Line:      uintToInt(req.Params.Position.Line),
		Character: uintToInt(req.Params.Position.Character),
	}

	currentFile := openFiles[req.Params.TextDocument.Uri]
	if currentFile == nil {
		msg := "file not found on server for go-to-definition request"
		slog.Error(msg,
			slog.Group("details",
				slog.String("uri", req.Params.TextDocument.Uri),
			),
		)
		panic(msg)
	}

	defer func() {
		if r := recover(); r != nil {
			msg := r.(string)
			slog.Error(msg,
				slog.Group("details",
					slog.String("uri", req.Params.TextDocument.Uri),
					slog.Any("position", position),
				),
			)
			panic(msg)
		}
	}()

	fileNames, reaches, errGoTo := tmpl.GoToDefinition(currentFile, position)

	var res ResponseMessage[[]DefinitionResults]
	res.Id = req.Id
	res.JsonRpc = req.JsonRpc

	for index := range fileNames {
		fileName = fileNames[index]
		targetFileNameURI := fileNames[index]
		reach := reaches[index]

		if targetFileNameURI == "" {
			msg := "found symbol definition without valid fileName"
			slog.Error(msg,
				slog.Group("details",
					slog.String("fileName", currentFile.FileName()),
				),
			)
			panic(msg)
		}

		result := DefinitionResults{}
		result.Uri = targetFileNameURI
		result.Range = ConvertParserRangeToLspRange(reach)

		res.Result = append(res.Result, result)
	}

	if errGoTo != nil {
		res.Result = nil
	}

	data, err = json.Marshal(res)
	if err != nil {
		slog.Warn("Error marshalling definition response: " + err.Error())
		return nil, fileName
	}

	return data, fileName
}

// FoldingRangeParams holds parameters for textDocument/foldingRange.
type FoldingRangeParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// FoldingRangeResult represents a folding range.
type FoldingRangeResult struct {
	StartLine      uint             `json:"startLine"`
	StartCharacter uint             `json:"startCharacter"`
	EndLine        uint             `json:"endLine"`
	EndCharacter   uint             `json:"endCharacter"`
	Kind           FoldingRangeKind `json:"kind"`
}

// FoldingRangeKind represents the kind of folding range.
type FoldingRangeKind string

const (
	FoldingRangeComment FoldingRangeKind = "comment"
	FoldingRangeImport  FoldingRangeKind = "imports"
	FoldingRangeRegion  FoldingRangeKind = "region"
)

// ProcessFoldingRangeRequest handles textDocument/foldingRange.
func ProcessFoldingRangeRequest(
	data []byte,
	parsedFiles map[string]*parser.GroupStatementNode,
	textFromClient map[string][]byte,
	muTextFromClient *sync.Mutex,
) (response []byte, fileName string) {
	req := RequestMessage[FoldingRangeParams]{}

	err := json.Unmarshal(data, &req)
	if err != nil {
		slog.Warn("Error unmarshalling folding range request: " + err.Error())
		return nil, ""
	}

	var rootNode *parser.GroupStatementNode = nil
	fileUri := req.Params.TextDocument.Uri

	muTextFromClient.Lock()
	fileContent := textFromClient[fileUri]

	if fileContent != nil {
		rootNode, _ = tmpl.ParseSingleFile(fileContent)
	}

	if rootNode == nil {
		rootNode = parsedFiles[fileUri]
	}

	muTextFromClient.Unlock()

	defer func() {
		if r := recover(); r != nil {
			msg := r.(string)
			slog.Error(msg,
				slog.Group("details",
					slog.String("file_uri", fileUri),
					slog.String("file_content", string(fileContent)),
				),
			)
			panic(msg)
		}
	}()

	if rootNode == nil {
		panic("file not found on server for folding range request: " + fileUri)
	}

	groups, comments := tmpl.FoldingRange(rootNode)

	var res ResponseMessage[[]FoldingRangeResult]
	res.Id = req.Id
	res.JsonRpc = req.JsonRpc

	for _, group := range groups {
		groupRange := group.Range()
		reach := ConvertParserRangeToLspRange(groupRange)

		if reach.Start.Line != reach.End.Line {
			reach.End.Line--
		}

		fold := FoldingRangeResult{
			StartLine:      reach.Start.Line,
			StartCharacter: reach.Start.Character,
			EndLine:        reach.End.Line,
			EndCharacter:   reach.End.Character,
			Kind:           FoldingRangeRegion,
		}

		res.Result = append(res.Result, fold)
	}

	for _, comment := range comments {
		commentRange := comment.Range()
		reach := ConvertParserRangeToLspRange(commentRange)

		fold := FoldingRangeResult{
			StartLine:      reach.Start.Line,
			StartCharacter: reach.Start.Character,
			EndLine:        reach.End.Line,
			EndCharacter:   reach.End.Character,
			Kind:           FoldingRangeComment,
		}

		if comment.GoCode != nil {
			fold.Kind = FoldingRangeImport
		}

		res.Result = append(res.Result, fold)
	}

	responseData, err := json.Marshal(res)
	if err != nil {
		slog.Warn("Error marshalling folding range response: " + err.Error())
		return nil, fileName
	}

	return responseData, fileName
}
