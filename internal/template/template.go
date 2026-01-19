package template

import (
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	checker "github.com/pacer/gozer/internal/template/analyzer"
	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

// FileAnalysisAndError pairs analysis results with any errors for a single file.
type FileAnalysisAndError struct {
	FileName string
	File     *checker.FileDefinition
	Errs     []lexer.Error
}

type Error = lexer.Error

// FunctionDefinition is an alias to allow external packages to work with custom functions.
type FunctionDefinition = checker.FunctionDefinition

// workspaceCustomFunctions holds custom template functions discovered by scanning Go source files.
// These are merged with builtin functions during analysis.
var workspaceCustomFunctions map[string]*checker.FunctionDefinition

// SetWorkspaceCustomFunctions sets the custom template functions for the workspace.
// These functions are discovered by scanning Go source files for template.FuncMap definitions.
// Call this once when initializing the workspace.
func SetWorkspaceCustomFunctions(funcs map[string]*checker.FunctionDefinition) {
	workspaceCustomFunctions = funcs
	checker.SetCustomFunctions(funcs)
}

// GetWorkspaceCustomFunctions returns the currently set custom template functions.
func GetWorkspaceCustomFunctions() map[string]*checker.FunctionDefinition {
	return workspaceCustomFunctions
}

// OpenProjectFiles recursively opens files from 'rootDir'.
// There is a depth limit for the recursion (current MAX_DEPTH = 5).
func OpenProjectFiles(rootDir string, withFileExtensions []string) map[string][]byte {
	return openProjectFilesSafely(
		rootDir, withFileExtensions, 0, checker.MaxProjectFileDepth,
	)
}

func openProjectFilesSafely(
	rootDir string,
	withFileExtensions []string,
	currentDepth, maxDepth int,
) map[string][]byte {
	if currentDepth > maxDepth {
		return nil
	}

	list, err := os.ReadDir(rootDir)
	if err != nil {
		panic("error while reading directory content: " + err.Error())
	}

	fileNamesToContent := make(map[string][]byte)

	for _, entry := range list {
		fileName := filepath.Join(rootDir, entry.Name())

		if entry.IsDir() {
			subDir := fileName
			subFiles := openProjectFilesSafely(
				subDir,
				withFileExtensions,
				currentDepth+1,
				maxDepth,
			)

			maps.Copy(fileNamesToContent, subFiles)
			continue
		}

		if !HasFileExtension(fileName, withFileExtensions) {
			continue
		}

		//nolint:gosec // fileName comes from trusted caller
		file, err := os.Open(fileName)
		if err != nil {
			log.Println("unable to open file, ", err.Error())
			continue
		}

		fileContent, _ := io.ReadAll(file)
		fileNamesToContent[fileName] = fileContent
	}

	return fileNamesToContent
}

// ParseSingleFile parses file content (buffer) and returns an AST node and error list.
// Returned parse tree is never 'nil', even when empty.
func ParseSingleFile(source []byte) (*parser.GroupStatementNode, []Error) {
	streamsOfToken, tokenErrs := lexer.Tokenize(source)

	parseTree, parseErrs := parser.Parse(streamsOfToken)
	if parseTree == nil {
		panic(
			"root parse tree should never be <nil>, even when empty. source = " + string(
				source,
			),
		)
	}

	parseErrs = append(parseErrs, tokenErrs...)
	return parseTree, parseErrs
}

// parseResult holds the result of parsing a single file.
type parseResult struct {
	fileName  string
	parseTree *parser.GroupStatementNode
	errs      []Error
}

// ParseFilesInWorkspace parses all files within a workspace using parallel goroutines.
// Returns AST nodes and error list. Never returns nil, always an empty 'map' if nothing found.
// Files are parsed concurrently for improved performance on multi-core systems.
func ParseFilesInWorkspace(
	workspaceFiles map[string][]byte,
) (map[string]*parser.GroupStatementNode, []Error) {
	if len(workspaceFiles) == 0 {
		return make(map[string]*parser.GroupStatementNode), nil
	}

	numWorkers := min(runtime.GOMAXPROCS(0), len(workspaceFiles))

	results := make(chan parseResult, len(workspaceFiles))

	// Use a semaphore to limit concurrency
	sem := make(chan struct{}, numWorkers)

	var wg sync.WaitGroup
	for fileName, content := range workspaceFiles {
		wg.Add(1)
		go func(fileName string, content []byte) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			streamsOfToken, tokenErrs := lexer.Tokenize(content)
			parseTree, parseErrs := parser.Parse(streamsOfToken)

			errs := make([]Error, 0, len(tokenErrs)+len(parseErrs))
			errs = append(errs, tokenErrs...)
			errs = append(errs, parseErrs...)

			results <- parseResult{fileName, parseTree, errs}
		}(fileName, content)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	parsedFilesInWorkspace := make(
		map[string]*parser.GroupStatementNode,
		len(workspaceFiles),
	)
	allErrs := make([]Error, 0, len(workspaceFiles)*2)

	for result := range results {
		parsedFilesInWorkspace[result.fileName] = result.parseTree
		allErrs = append(allErrs, result.errs...)
	}

	if len(workspaceFiles) != len(parsedFilesInWorkspace) {
		panic("number of parsed files do not match the amount present in the workspace")
	}

	return parsedFilesInWorkspace, allErrs
}

// analyzeAffectedFiles performs definition analysis on a set of affected files.
// This is the common core loop extracted from the analysis chain functions.
func analyzeAffectedFiles(
	affectedFiles map[string]bool,
	templateManager *checker.WorkspaceTemplateManager,
	workspaceTemplateDefinition map[*parser.GroupStatementNode]*checker.TemplateDefinition,
) []FileAnalysisAndError {
	chainAnalysis := make([]FileAnalysisAndError, 0, len(affectedFiles))

	for fileNameAffected := range affectedFiles {
		partialFile := templateManager.AnalyzedDefinedTemplatesWithinFile[fileNameAffected].PartialFile

		file, errs := checker.DefinitionAnalysisFromPartialFile(
			partialFile,
			workspaceTemplateDefinition,
		)

		// Append template and cycle errors
		errs = appendAnalysisErrors(
			errs,
			templateManager,
			fileNameAffected,
		)

		chainAnalysis = append(chainAnalysis, FileAnalysisAndError{
			FileName: fileNameAffected,
			File:     file,
			Errs:     errs,
		})
	}

	return chainAnalysis
}

// appendAnalysisErrors appends template and cycle errors from the template manager.
func appendAnalysisErrors(
	errs []lexer.Error,
	templateManager *checker.WorkspaceTemplateManager,
	fileName string,
) []lexer.Error {
	templateErrs := templateManager.AnalyzedDefinedTemplatesWithinFile[fileName].GetTemplateErrs()
	cycleErrs := templateManager.AnalyzedDefinedTemplatesWithinFile[fileName].CycleTemplateErrs

	errs = append(errs, templateErrs...)
	errs = append(errs, cycleErrs...)

	return errs
}

// validateFileInWorkspace checks that a file exists in the workspace and panics if not.
func validateFileInWorkspace(
	fileName string,
	parsedFilesInWorkspace map[string]*parser.GroupStatementNode,
) {
	if parsedFilesInWorkspace[fileName] == nil {
		log.Printf(
			"file '%s' is unavailable in the current workspace\n parsedFilesInWorkspace = %#v\n",
			fileName,
			parsedFilesInWorkspace,
		)
		panic("file '" + fileName + "' is unavailable in the current workspace")
	}
}

// DefinitionAnalysisSingleFile performs semantic analysis on a single file.
// Use DefinitionAnalysisChainTriggeredBySingleFileChange instead for better performance.
func DefinitionAnalysisSingleFile(
	fileName string,
	parsedFilesInWorkspace map[string]*parser.GroupStatementNode,
) (*checker.FileDefinition, []Error) {
	if len(parsedFilesInWorkspace) == 0 {
		return nil, nil
	}

	validateFileInWorkspace(fileName, parsedFilesInWorkspace)

	templateManager := checker.TemplateManager
	templateManager.RemoveTemplateScopeAssociatedToFileName(fileName)

	_ = templateManager.BuildWorkspaceTemplateDefinition(parsedFilesInWorkspace)

	workspaceTemplateDefinition := templateManager.TemplateScopeToDefinition
	partialFile := templateManager.AnalyzedDefinedTemplatesWithinFile[fileName].PartialFile

	file, errs := checker.DefinitionAnalysisFromPartialFile(
		partialFile,
		workspaceTemplateDefinition,
	)

	errs = appendAnalysisErrors(errs, templateManager, fileName)

	return file, errs
}

// DefinitionAnalysisChainTriggeredBySingleFileChange computes semantic analysis for a file and all affected files.
func DefinitionAnalysisChainTriggeredBySingleFileChange(
	parsedFilesInWorkspace map[string]*parser.GroupStatementNode,
	fileName string,
) []FileAnalysisAndError {
	if len(parsedFilesInWorkspace) == 0 {
		return nil
	}

	validateFileInWorkspace(fileName, parsedFilesInWorkspace)

	templateManager := checker.TemplateManager
	templateManager.RemoveTemplateScopeAssociatedToFileName(fileName)

	affectedFiles := templateManager.BuildWorkspaceTemplateDefinition(
		parsedFilesInWorkspace,
	)
	affectedFiles[fileName] = true

	return analyzeAffectedFiles(
		affectedFiles,
		templateManager,
		templateManager.TemplateScopeToDefinition,
	)
}

// DefinitionAnalysisChainTriggeredByBatchFileChange computes semantic analysis for multiple file changes.
func DefinitionAnalysisChainTriggeredByBatchFileChange(
	parsedFilesInWorkspace map[string]*parser.GroupStatementNode,
	fileNames ...string,
) []FileAnalysisAndError {
	if len(parsedFilesInWorkspace) == 0 {
		return nil
	}

	templateManager := checker.TemplateManager
	nameOfFileChanged := make(map[string]bool)

	for _, fileName := range fileNames {
		validateFileInWorkspace(fileName, parsedFilesInWorkspace)
		templateManager.RemoveTemplateScopeAssociatedToFileName(fileName)
		nameOfFileChanged[fileName] = true
	}

	affectedFiles := templateManager.BuildWorkspaceTemplateDefinition(
		parsedFilesInWorkspace,
	)
	maps.Copy(affectedFiles, nameOfFileChanged)

	return analyzeAffectedFiles(
		affectedFiles,
		templateManager,
		templateManager.TemplateScopeToDefinition,
	)
}

// DefinitionAnalysisWithinWorkspace performs definition analysis for all files in a workspace.
func DefinitionAnalysisWithinWorkspace(
	parsedFilesInWorkspace map[string]*parser.GroupStatementNode,
) []FileAnalysisAndError {
	if len(parsedFilesInWorkspace) == 0 {
		return nil
	}

	checker.TemplateManager = checker.NewWorkspaceTemplateManager()
	templateManager := checker.TemplateManager

	affectedFiles := templateManager.BuildWorkspaceTemplateDefinition(
		parsedFilesInWorkspace,
	)
	for fileName := range parsedFilesInWorkspace {
		affectedFiles[fileName] = true
	}

	if len(affectedFiles) != len(parsedFilesInWorkspace) {
		log.Printf(
			"count of files in workspace do not match number of files found during template analysis"+
				"\n len(affectedFiles) = %d ::: len(parsedFilesInWorkspace) = %d\n",
			len(affectedFiles),
			len(parsedFilesInWorkspace),
		)
		panic(
			"count of files in workspace do not match number of files found during template analysis",
		)
	}

	return analyzeAffectedFiles(
		affectedFiles,
		templateManager,
		templateManager.TemplateScopeToDefinition,
	)
}

func GoToDefinition(
	file *checker.FileDefinition,
	position lexer.Position,
) (fileNames []string, ranges []lexer.Range, err error) {
	definitions := checker.FindSourceDefinitionFromPosition(file, position)

	if len(definitions) == 0 {
		log.Println("token not found for definition")
		return nil, nil, errors.New("meaningful token not found for go-to definition")
	}

	fileNames = make([]string, 0, len(definitions))
	ranges = make([]lexer.Range, 0, len(definitions))

	for _, definition := range definitions {
		fileNames = append(fileNames, definition.FileName())
		ranges = append(ranges, definition.Range())
	}

	return fileNames, ranges, nil
}

func Hover(file *checker.FileDefinition, position lexer.Position) (string, lexer.Range) {
	definitions := checker.FindSourceDefinitionFromPosition(file, position)
	if len(definitions) == 0 {
		log.Println("definition not found for token at position, ", position)
		return "", lexer.EmptyRange()
	}

	if len(definitions) > 1 {
		typeStringified := "Multiple Source Found [lsp]"

		return typeStringified, lexer.EmptyRange()
	}

	definition := definitions[0]
	typeStringified, reach := checker.Hover(definition)

	if typeStringified == "" {
		log.Printf(
			"definition exist, but type was not found\n definition = %#v\n",
			definition,
		)
		panic("definition exist, but type was not found")
	}

	if file.FileName() != definition.FileName() {
		return typeStringified, lexer.EmptyRange()
	}

	return typeStringified, reach
}

func FoldingRange(
	rootNode *parser.GroupStatementNode,
) ([]*parser.GroupStatementNode, []*parser.CommentNode) {
	foldingGroups := make([]*parser.GroupStatementNode, 0, 10)
	foldingComments := make([]*parser.CommentNode, 0, 10)
	queue := make([]*parser.GroupStatementNode, 0, 10)

	queue = append(queue, rootNode)
	index := 0
	counter := 0

	for {
		if counter++; counter > 10_000 {
			panic("infinite loop while computing 'FoldingRange()'")
		}

		if index >= len(queue) { // index out of bound
			break
		}

		scope := queue[index]
		index++

		for _, statement := range scope.Statements {
			switch node := statement.(type) {
			case *parser.CommentNode:
				foldingComments = append(foldingComments, node)

			case *parser.GroupStatementNode:
				foldingGroups = append(foldingGroups, node)
				queue = append(queue, node)

			default: // do nothing
			}
		}
	}

	return foldingGroups, foldingComments
}

// DocumentHighlight returns the keyword ranges of all linked control flow keywords
// when the cursor is on one of them. For example, if the cursor is on {{if}}, {{else}},
// or {{end}}, all related keywords in the same control flow block are highlighted.
func DocumentHighlight(
	rootNode *parser.GroupStatementNode,
	position lexer.Position,
) []lexer.Range {
	// Find the GroupStatementNode whose KeywordRange contains the position
	var foundNode *parser.GroupStatementNode
	queue := make([]*parser.GroupStatementNode, 0, 10)
	queue = append(queue, rootNode)
	counter := 0

	for len(queue) > 0 {
		if counter++; counter > 10_000 {
			panic("infinite loop while computing 'DocumentHighlight()'")
		}

		node := queue[0]
		queue = queue[1:]

		// Check if position is within this node's keyword range
		if !node.KeywordRange.IsEmpty() && node.KeywordRange.Contains(position) {
			foundNode = node
			break
		}

		// Add children to queue
		for _, statement := range node.Statements {
			if groupNode, ok := statement.(*parser.GroupStatementNode); ok {
				queue = append(queue, groupNode)
			}
		}
	}

	if foundNode == nil {
		return nil
	}

	// Collect all keyword ranges by walking the NextLinkedSibling circular list
	ranges := make([]lexer.Range, 0, 4)
	startNode := foundNode
	current := foundNode

	for {
		if !current.KeywordRange.IsEmpty() {
			ranges = append(ranges, current.KeywordRange)
		}

		current = current.NextLinkedSibling
		if current == nil || current == startNode {
			break
		}

		if len(ranges) > 100 {
			panic("too many linked siblings while computing 'DocumentHighlight()'")
		}
	}

	return ranges
}

// HasFileExtension reports whether fileName's extension is found within extensions.
func HasFileExtension(fileName string, extensions []string) bool {
	for _, ext := range extensions {
		if strings.HasSuffix(fileName, "."+ext) {
			return true
		}
	}

	return false
}

// Print outputs AST nodes as JSON to stdout (use jq for pretty formatting).
func Print(node ...parser.AstNode) {
	str := parser.PrettyAstNodeFormater(node)
	fmt.Println(str)
}
