// Command go-template-lsp provides a Language Server Protocol server for Go templates.
//
// Based on https://github.com/yayolande/go-template-lsp (MIT License)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/pacer/gozer/cmd/go-template-lsp/lsp"
	tmpl "github.com/pacer/gozer/internal/template"
	"github.com/pacer/gozer/internal/template/analyzer"
	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

// version is set by goreleaser at build time.
var version = "dev"

// workspaceStore holds the state for a workspace.
type workspaceStore struct {
	RootPath            string
	RawFiles            map[string][]byte
	ParsedFiles         map[string]*parser.GroupStatementNode
	ErrorsParsedFiles   map[string][]lexer.Error
	OpenedFilesAnalyzed map[string]*analyzer.FileDefinition
	ErrorsAnalyzedFiles map[string][]lexer.Error
}

// requestCounter tracks the number of each request type.
type requestCounter struct {
	Initialize   int
	Initialized  int
	Shutdown     int
	TextDocument struct {
		DidClose  int
		DidOpen   int
		DidChange int
	}
	FoldingRange int
	Definition   int
	Hover        int
	Other        int
}

// TargetFileExtensions lists the file extensions this LSP supports.
var TargetFileExtensions = []string{
	"go.html", "go.tmpl", "go.txt",
	"gohtml", "gotmpl", "tmpl", "tpl",
	"html",
}

const (
	serverName = "Go Template LSP"
)

var serverCounter requestCounter

func main() {
	versionFlag := flag.Bool("version", false, "print the LSP version")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("%s -- version %s\n", serverName, version)
		os.Exit(0)
	}

	configureLogging()
	scanner := lsp.ReceiveInput(os.Stdin)

	storage := &workspaceStore{}

	rootPathNotification := make(chan string, 2)
	textChangedNotification := make(chan bool, 2)
	textFromClient := make(map[string][]byte)
	muTextFromClient := new(sync.Mutex)

	go processDiagnosticNotification(
		storage,
		rootPathNotification,
		textChangedNotification,
		textFromClient,
		muTextFromClient,
	)

	var request lsp.RequestMessage[any]
	var response []byte
	var isRequestResponse bool
	var isExiting bool
	var fileURI string
	var fileContent []byte

	slog.Info("starting lsp server",
		slog.String("server_name", serverName),
		slog.String("server_version", version),
	)
	defer slog.Info(
		"shutting down lsp server",
		getServerGroupLogging(storage, &serverCounter, request, textFromClient),
	)

	for scanner.Scan() {
		data := scanner.Bytes()
		_ = json.Unmarshal(data, &request)

		if isExiting {
			if request.Method == "exit" {
				break
			} else {
				response = lsp.ProcessIllegalRequestAfterShutdown(
					request.JsonRpc,
					request.Id,
				)
				lsp.SendToLspClient(os.Stdout, response)
			}
			continue
		}

		slog.Info(
			"request "+request.Method,
			getServerGroupLogging(storage, &serverCounter, request, textFromClient),
		)

		switch request.Method {
		case "initialize":
			serverCounter.Initialize++
			var rootURI string
			response, rootURI = lsp.ProcessInitializeRequest(data, serverName, version)
			notifyTheRootPath(rootPathNotification, rootURI)
			rootPathNotification = nil
			isRequestResponse = true

		case "initialized":
			serverCounter.Initialized++
			isRequestResponse = false
			lsp.ProcessInitializedNotification(data)

		case "shutdown":
			serverCounter.Shutdown++
			isExiting = true
			isRequestResponse = true
			response = lsp.ProcessShutdownRequest(request.JsonRpc, request.Id)

		case "textDocument/didOpen":
			serverCounter.TextDocument.DidOpen++
			isRequestResponse = false
			fileURI, fileContent = lsp.ProcessDidOpenTextDocumentNotification(data)
			insertTextDocumentToDiagnostic(
				fileURI,
				fileContent,
				textChangedNotification,
				textFromClient,
				muTextFromClient,
			)

		case "textDocument/didChange":
			serverCounter.TextDocument.DidChange++
			isRequestResponse = false
			fileURI, fileContent = lsp.ProcessDidChangeTextDocumentNotification(data)
			insertTextDocumentToDiagnostic(
				fileURI,
				fileContent,
				textChangedNotification,
				textFromClient,
				muTextFromClient,
			)

		case "textDocument/didClose":
			serverCounter.TextDocument.DidClose++

		case "textDocument/hover":
			serverCounter.Hover++
			isRequestResponse = true
			response = lsp.ProcessHoverRequest(data, storage.OpenedFilesAnalyzed)

		case "textDocument/definition":
			serverCounter.Definition++
			isRequestResponse = true
			response, _ = lsp.ProcessGoToDefinition(
				data,
				storage.OpenedFilesAnalyzed,
				storage.RawFiles,
			)

		case "textDocument/foldingRange":
			serverCounter.FoldingRange++
			isRequestResponse = true
			response, _ = lsp.ProcessFoldingRangeRequest(
				data,
				storage.ParsedFiles,
				textFromClient,
				muTextFromClient,
			)

		default:
			serverCounter.Other++
		}

		if isRequestResponse {
			lsp.SendToLspClient(os.Stdout, response)

			res := lsp.ResponseMessage[any]{}
			_ = json.Unmarshal(response, &res)
			slog.Info("response "+request.Method,
				slog.Group("server",
					slog.String("name", serverName),
					slog.String("version", version),
					slog.String("root_path", storage.RootPath),
					slog.Any("request_counter", serverCounter),
					slog.Any("open_files", mapToKeys(storage.RawFiles)),
					slog.Any("files_waiting_processing", mapToKeys(textFromClient)),
					slog.Any("last_response", res),
				),
			)
		}

		response = nil
		isRequestResponse = false
	}

	if scanner.Err() != nil {
		msg := "error while closing LSP: " + scanner.Err().Error()
		slog.Error(msg)
		panic(msg)
	}
}

// insertTextDocumentToDiagnostic queues a document for diagnostic processing.
func insertTextDocumentToDiagnostic(
	uri string,
	content []byte,
	textChangedNotification chan bool,
	textFromClient map[string][]byte,
	muTextFromClient *sync.Mutex,
) {
	if uri == "" {
		return
	}

	muTextFromClient.Lock()
	textFromClient[uri] = content

	if len(textChangedNotification) == 0 {
		textChangedNotification <- true
	}

	muTextFromClient.Unlock()

	if len(textChangedNotification) >= 2 {
		msg := "'textChangedNotification' channel size should never exceed 1"
		slog.Error(msg,
			slog.Group("error_details",
				slog.String("uri_file_to_diagnostic", uri),
				slog.Any("files_waiting_processing", mapToKeys(textFromClient)),
			),
		)
		panic(msg)
	}
}

// notifyTheRootPath sends the root path to the diagnostic goroutine.
func notifyTheRootPath(rootPathNotification chan string, rootURI string) {
	if rootPathNotification == nil {
		return
	}

	if len(rootPathNotification) > 0 {
		msg := "'rootPathNotification' channel should be empty at this point"
		slog.Error(msg)
		panic(msg)
	} else if cap(rootPathNotification) < 2 {
		msg := "'rootPathNotification' channel should have a buffer capacity of at least 2"
		slog.Error(msg)
		panic(msg)
	}

	rootPathNotification <- rootURI
	close(rootPathNotification)
}

// processDiagnosticNotification runs diagnostics and sends notifications to the client.
func processDiagnosticNotification(
	storage *workspaceStore,
	rootPathNotification chan string,
	textChangedNotification chan bool,
	textFromClient map[string][]byte,
	muTextFromClient *sync.Mutex,
) {
	if rootPathNotification == nil || textChangedNotification == nil {
		msg := "channels for 'processDiagnosticNotification()' not properly initialized"
		slog.Error(msg)
		panic(msg)
	}

	if textFromClient == nil {
		msg := "empty reference to 'textFromClient'"
		slog.Error(msg)
		panic(msg)
	}

	rootPath, ok := <-rootPathNotification
	// Note: rootPathNotification is closed after first use, no need to nil it
	if !ok {
		msg := "rootPathNotification is closed or nil"
		slog.Error(msg)
		panic(msg)
	}

	rootPath = uriToFilePath(rootPath)

	// Scan for custom template functions defined in Go source files
	customFuncs, err := tmpl.ScanWorkspaceForFuncMap(rootPath)
	if err != nil {
		slog.Warn(
			"failed to scan for custom template functions",
			slog.String("error", err.Error()),
		)
	} else if len(customFuncs) > 0 {
		tmpl.SetWorkspaceCustomFunctions(customFuncs)
		funcNames := make([]string, 0, len(customFuncs))
		for name := range customFuncs {
			funcNames = append(funcNames, name)
		}
		slog.Info(
			"discovered custom template functions",
			slog.Any("functions", funcNames),
		)
	}

	storage.RootPath = rootPath
	storage.RawFiles = tmpl.OpenProjectFiles(rootPath, TargetFileExtensions)
	storage.RawFiles = convertKeysFromFilePathToUri(storage.RawFiles)

	muTextFromClient.Lock()
	{
		temporaryClone := maps.Clone(textFromClient)
		maps.Copy(textFromClient, storage.RawFiles)
		maps.Copy(textFromClient, temporaryClone)
	}

	if len(textFromClient) > 0 && len(textChangedNotification) == 0 {
		textChangedNotification <- true
	}

	muTextFromClient.Unlock()

	storage.ParsedFiles = make(map[string]*parser.GroupStatementNode)
	storage.OpenedFilesAnalyzed = make(map[string]*analyzer.FileDefinition)
	storage.ErrorsAnalyzedFiles = make(map[string][]lexer.Error)
	storage.ErrorsParsedFiles = make(map[string][]lexer.Error)

	notification := &lsp.NotificationMessage[lsp.PublishDiagnosticsParams]{
		JsonRpc: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params: lsp.PublishDiagnosticsParams{
			Uri:         "placeholder",
			Diagnostics: []lsp.Diagnostic{},
		},
	}

	defer func() {
		slog.Info("notification details",
			slog.Any("len_textChangedNotification", len(textChangedNotification)),
			slog.Any("open_files", mapToKeys(storage.RawFiles)),
			slog.Any("files_waiting_processing", mapToKeys(textFromClient)),
		)
	}()

	var chainedFiles []tmpl.FileAnalysisAndError
	cloneTextFromClient := make(map[string][]byte)

	for range textChangedNotification {
		if len(textFromClient) == 0 {
			msg := "got a change notification but textFromClient was empty"
			slog.Error(msg)
			panic(msg)
		}

		muTextFromClient.Lock()

		clear(cloneTextFromClient)
		namesOfFileChanged := make([]string, 0, len(textFromClient))

		for uri, fileContent := range textFromClient {
			if !isFileInsideWorkspace(uri, rootPath, TargetFileExtensions) {
				slog.Warn("skipped file", slog.String("file_uri", uri))
				continue
			}

			storage.RawFiles[uri] = fileContent
			cloneTextFromClient[uri] = fileContent

			parseTree, localErrs := tmpl.ParseSingleFile(fileContent)

			storage.ParsedFiles[uri] = parseTree
			storage.ErrorsParsedFiles[uri] = localErrs
			namesOfFileChanged = append(namesOfFileChanged, uri)
		}

		clear(textFromClient)
		for range len(textChangedNotification) {
			<-textChangedNotification
		}

		muTextFromClient.Unlock()

		if len(cloneTextFromClient) == 0 {
			continue
		}

		chainedFiles = nil

		if len(cloneTextFromClient) == len(storage.ParsedFiles) {
			chainedFiles = tmpl.DefinitionAnalysisWithinWorkspace(storage.ParsedFiles)
		} else if len(cloneTextFromClient) > 0 {
			chainedFiles = tmpl.DefinitionAnalysisChainTriggeredByBatchFileChange(
				storage.ParsedFiles,
				namesOfFileChanged...)
		}

		for _, fileAnalyzed := range chainedFiles {
			localUri := fileAnalyzed.FileName
			storage.OpenedFilesAnalyzed[localUri] = fileAnalyzed.File
			storage.ErrorsAnalyzedFiles[localUri] = fileAnalyzed.Errs
		}

		for uri := range storage.OpenedFilesAnalyzed {
			errs := make(
				[]tmpl.Error,
				0,
				len(storage.ErrorsParsedFiles[uri])+len(storage.ErrorsAnalyzedFiles[uri]),
			)
			errs = append(errs, storage.ErrorsParsedFiles[uri]...)
			errs = append(errs, storage.ErrorsAnalyzedFiles[uri]...)

			notification = clearPushDiagnosticNotification(notification)
			notification = setParseErrorsToDiagnosticsNotification(errs, notification)
			notification.Params.Uri = uri

			response, err := json.Marshal(notification)
			if err != nil {
				msg := "unable to marshal notification response: " + err.Error()
				slog.Error(msg,
					slog.Group("error",
						slog.String("file_uri", uri),
						slog.Any("file_parse_error", errs),
					),
				)
				panic(msg)
			}

			lsp.SendToLspClient(os.Stdout, response)
		}

		storageSanityCheck(storage)
	}
}

// storageSanityCheck verifies that storage state is consistent.
func storageSanityCheck(storage *workspaceStore) {
	switch {
	case len(storage.OpenedFilesAnalyzed) != len(storage.ParsedFiles):
		msg := "size mismatch between 'semantic analysed files' and 'parsed files'"
		slog.Error(msg,
			slog.Group("error_details",
				slog.Int("len_openFilesAnalyzed", len(storage.OpenedFilesAnalyzed)),
				slog.Int("len_parsedFiles", len(storage.ParsedFiles)),
			),
		)
		panic(msg)
	case len(storage.OpenedFilesAnalyzed) != len(storage.RawFiles):
		msg := "found more 'semantic analysed files' than 'raw files'"
		slog.Error(msg,
			slog.Group("error_details",
				slog.Int("len_openFilesAnalyzed", len(storage.OpenedFilesAnalyzed)),
				slog.Int("len_rawFiles", len(storage.RawFiles)),
			),
		)
		panic(msg)
	case len(storage.ErrorsAnalyzedFiles) != len(storage.ErrorsParsedFiles):
		msg := "size mismatch between 'errors semantic analysed files' and 'errors parsed files'"
		slog.Error(msg,
			slog.Group("error_details",
				slog.Int("len_errorsAnalyzedFiles", len(storage.ErrorsAnalyzedFiles)),
				slog.Int("len_errorsParsedFiles", len(storage.ErrorsParsedFiles)),
			),
		)
		panic(msg)
	case len(storage.ErrorsAnalyzedFiles) != len(storage.OpenedFilesAnalyzed):
		msg := "size mismatch between errors associated to files and opened files"
		slog.Error(msg,
			slog.Group("error_details",
				slog.Int("len_errorsAnalyzedFiles", len(storage.ErrorsAnalyzedFiles)),
				slog.Int("len_openedFilesAnalyzed", len(storage.OpenedFilesAnalyzed)),
			),
		)
		panic(msg)
	}
}

// isFileInsideWorkspace checks if a file is inside the workspace and has allowed extension.
func isFileInsideWorkspace(uri, rootPath string, allowedFileExtensions []string) bool {
	path := uri
	rootPath = filePathToUri(rootPath)

	if !strings.HasPrefix(path, rootPath) {
		return false
	}

	return tmpl.HasFileExtension(path, allowedFileExtensions)
}

// clearPushDiagnosticNotification clears the diagnostics in a notification.
func clearPushDiagnosticNotification(
	notification *lsp.NotificationMessage[lsp.PublishDiagnosticsParams],
) *lsp.NotificationMessage[lsp.PublishDiagnosticsParams] {
	notification.Params.Diagnostics = []lsp.Diagnostic{}
	notification.Params.Uri = ""
	return notification
}

// setParseErrorsToDiagnosticsNotification adds parse errors to a notification.
func setParseErrorsToDiagnosticsNotification(
	errs []tmpl.Error,
	response *lsp.NotificationMessage[lsp.PublishDiagnosticsParams],
) *lsp.NotificationMessage[lsp.PublishDiagnosticsParams] {
	if response == nil {
		msg := "diagnostics errors cannot be appended on nil response"
		slog.Error(msg)
		panic(msg)
	}

	response.Params.Diagnostics = []lsp.Diagnostic{}

	for _, err := range errs {
		if err == nil {
			msg := "nil should not be in the error list"
			slog.Error(msg)
			panic(msg)
		}

		diagnostic := lsp.Diagnostic{
			Message:  err.GetError(),
			Range:    lsp.ConvertParserRangeToLspRange(err.GetRange()),
			Severity: 1, // 1 = Error, 2 = Warning, 3 = Info, 4 = Hint
		}

		response.Params.Diagnostics = append(response.Params.Diagnostics, diagnostic)
	}

	return response
}

// uriToFilePath converts a file URI to an OS path.
func uriToFilePath(uri string) string {
	if uri == "" {
		msg := "URI to a file cannot be empty"
		slog.Error(msg)
		panic(msg)
	}

	defer func() {
		if err := recover(); err != nil {
			msg, _ := err.(string)
			slog.Error(msg, slog.String("uri_to_convert", uri))
			panic(msg)
		}
	}()

	u, err := url.Parse(uri)
	if err != nil {
		panic("unable to convert from URI to OS path: " + err.Error())
	}

	if u.Scheme == "" {
		panic("expected a scheme for the file URI: " + uri)
	}

	if u.Scheme != "file" {
		panic("can only handle 'file' scheme: " + uri)
	}

	if u.RawQuery != "" {
		panic("'?' character is not permitted in file URI: " + uri)
	}

	if u.Fragment != "" {
		panic("'#' character is not permitted in file URI: " + uri)
	}

	if u.Path == "" {
		panic("path to a file cannot be empty")
	}

	path := u.Path
	if runtime.GOOS == "windows" {
		if path[0] == '/' && len(path) >= 3 && path[2] == ':' {
			path = path[1:]
		}
	}

	path = filepath.FromSlash(path)

	return path
}

// filePathToUri converts an OS path to a file URI.
func filePathToUri(path string) string {
	if path == "" {
		msg := "path to a file cannot be empty"
		slog.Error(msg)
		panic(msg)
	}

	defer func() {
		if err := recover(); err != nil {
			msg, _ := err.(string)
			slog.Error(msg, slog.String("path_to_convert", path))
			panic(msg)
		}
	}()

	absPath, err := filepath.Abs(path)
	if err != nil {
		panic("malformed file path: " + err.Error())
	}

	slashPath := filepath.ToSlash(absPath)

	if runtime.GOOS == "windows" && slashPath[0] != '/' {
		slashPath = "/" + slashPath
	}

	u := url.URL{
		Scheme: "file",
		Path:   slashPath,
	}

	return u.String()
}

// convertKeysFromFilePathToUri converts map keys from file paths to URIs.
func convertKeysFromFilePathToUri(files map[string][]byte) map[string][]byte {
	if len(files) == 0 {
		return files
	}

	filesWithUriKeys := make(map[string][]byte)

	for path, fileContent := range files {
		uri := filePathToUri(path)
		filesWithUriKeys[uri] = fileContent
	}

	return filesWithUriKeys
}

// mapToKeys returns the keys of a map as a slice.
func mapToKeys[K comparable, V any](dict map[K]V) []K {
	list := make([]K, 0, len(dict))
	for key := range dict {
		list = append(list, key)
	}
	return list
}

// createLogFile creates or opens the log file.
func createLogFile() *os.File {
	userCachePath, err := os.UserCacheDir()
	if err != nil {
		return os.Stdout
	}

	appCachePath := filepath.Join(userCachePath, "go-template-lsp")
	logFilePath := filepath.Join(appCachePath, "go-template-lsp.log")

	_ = os.Mkdir(appCachePath, 0750)

	fileInfo, err := os.Stat(logFilePath)
	if err == nil && fileInfo.Size() >= 5_000_000 {
		//nolint:gosec // safe log file path
		file, err := os.OpenFile(logFilePath, os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			return os.Stdout
		}
		return file
	}

	//nolint:gosec // safe log file path
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return os.Stdout
	}

	return file
}

// configureLogging sets up structured logging.
func configureLogging() {
	file := createLogFile()
	if file == nil {
		file = os.Stdout
	}

	logger := slog.New(slog.NewJSONHandler(file, nil))
	slog.SetDefault(logger)
}

// getServerGroupLogging returns a structured logging group with server state.
func getServerGroupLogging[T any](
	storage *workspaceStore,
	counter *requestCounter,
	request lsp.RequestMessage[T],
	textFromClient map[string][]byte,
) slog.Attr {
	return slog.Group("server",
		slog.String("root_path", storage.RootPath),
		slog.Any("last_request", request),
		slog.Any("open_files", mapToKeys(storage.RawFiles)),
		slog.Any("files_waiting_processing", mapToKeys(textFromClient)),
		slog.Any("request_counter", counter),
	)
}
