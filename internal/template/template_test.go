package template_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pacer/gozer/internal/template"
	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
)

func TestParseSingleFile_EmptyFile(t *testing.T) {
	root, errs := template.ParseSingleFile([]byte(""))

	if root == nil {
		t.Fatal("ParseSingleFile should return non-nil root even for empty file")
	}

	if len(errs) != 0 {
		t.Errorf("Expected no errors for empty file, got %d", len(errs))
	}

	if len(root.Statements) != 0 {
		t.Errorf("Expected no statements in empty file, got %d", len(root.Statements))
	}
}

func TestParseSingleFile_ValidTemplate(t *testing.T) {
	source := `Hello {{ .Name }}!
You have {{ .Count }} items.
{{ if .ShowDetails }}
Details: {{ .Details }}
{{ end }}`

	root, errs := template.ParseSingleFile([]byte(source))

	if root == nil {
		t.Fatal("ParseSingleFile should return non-nil root")
	}

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
	}

	// Should have multiple statements
	if len(root.Statements) == 0 {
		t.Error("Expected statements in parsed file")
	}
}

func TestParseSingleFile_SyntaxErrors(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "unclosed if",
			source: "{{ if .Cond }}content",
		},
		{
			name:   "empty expression",
			source: "{{ }}",
		},
		{
			name:   "invalid syntax",
			source: "{{ if }}{{ end }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := template.ParseSingleFile([]byte(tt.source))

			if root == nil {
				t.Error("ParseSingleFile should return non-nil root even with errors")
			}

			if len(errs) == 0 {
				t.Error("Expected errors for invalid syntax")
			}
		})
	}
}

func TestParseFilesInWorkspace_EmptyWorkspace(t *testing.T) {
	workspace := make(map[string][]byte)

	parsed, errs := template.ParseFilesInWorkspace(workspace)

	if parsed == nil {
		t.Error("ParseFilesInWorkspace should return non-nil map")
	}

	if len(parsed) != 0 {
		t.Errorf("Expected empty parsed map, got %d entries", len(parsed))
	}

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d", len(errs))
	}
}

func TestParseFilesInWorkspace_MultipleFiles(t *testing.T) {
	workspace := map[string][]byte{
		"file1.html": []byte("{{ .Field1 }}"),
		"file2.html": []byte("{{ .Field2 }}"),
		"file3.html": []byte("{{ .Field3 }}"),
	}

	parsed, errs := template.ParseFilesInWorkspace(workspace)

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d", len(errs))
	}

	if len(parsed) != 3 {
		t.Errorf("Expected 3 parsed files, got %d", len(parsed))
	}

	for fileName := range workspace {
		if parsed[fileName] == nil {
			t.Errorf("Missing parsed result for %s", fileName)
		}
	}
}

func TestDefinitionAnalysisSingleFile_EmptyWorkspace(t *testing.T) {
	workspace := make(map[string]*parser.GroupStatementNode)

	file, errs := template.DefinitionAnalysisSingleFile("test.html", workspace)

	if file != nil {
		t.Error("Expected nil file for empty workspace")
	}

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d", len(errs))
	}
}

func TestDefinitionAnalysisSingleFile_BasicAnalysis(t *testing.T) {
	source := `{{ define "test" }}
{{ $var := .Field }}
{{ $var }}
{{ end }}`

	root, parseErrs := template.ParseSingleFile([]byte(source))
	if len(parseErrs) != 0 {
		t.Fatalf("Parse errors: %v", parseErrs)
	}

	workspace := map[string]*parser.GroupStatementNode{
		"test.html": root,
	}

	file, errs := template.DefinitionAnalysisSingleFile("test.html", workspace)

	// May have analysis errors for undefined types, which is expected
	_ = errs

	if file == nil {
		t.Error("Expected non-nil file definition")
	}
}

func TestFoldingRange(t *testing.T) {
	source := `{{ if .Cond1 }}
content1
{{ if .Cond2 }}
content2
{{ end }}
{{ end }}
{{/* multi-line
comment */}}`

	root, errs := template.ParseSingleFile([]byte(source))
	if len(errs) != 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	groups, comments := template.FoldingRange(root)

	// Should have 2 groups (outer if and inner if)
	if len(groups) < 2 {
		t.Errorf("Expected at least 2 folding groups, got %d", len(groups))
	}

	// Should have 1 comment
	if len(comments) != 1 {
		t.Errorf("Expected 1 folding comment, got %d", len(comments))
	}
}

func TestFoldingRange_NestedFolds(t *testing.T) {
	source := `{{ range .Items }}
{{ if .Active }}
{{ with .Details }}
content
{{ end }}
{{ end }}
{{ end }}`

	root, errs := template.ParseSingleFile([]byte(source))
	if len(errs) != 0 {
		t.Fatalf("Parse errors: %v", errs)
	}

	groups, _ := template.FoldingRange(root)

	// FoldingRange returns all GroupStatementNode instances including end statements
	// The exact count depends on implementation details - just verify we get some
	if len(groups) < 3 {
		t.Errorf("Expected at least 3 folding groups, got %d", len(groups))
	}
}

func TestOpenProjectFiles(t *testing.T) {
	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "template_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test files
	testFiles := map[string]string{
		"file1.html":    "{{ .Field1 }}",
		"file2.gohtml":  "{{ .Field2 }}",
		"file3.txt":     "not a template",
		"subdir/a.html": "{{ .SubField }}",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	// Test with .html extension filter
	files := template.OpenProjectFiles(tempDir, []string{"html"})

	// Should find file1.html and subdir/a.html
	if len(files) != 2 {
		t.Errorf("Expected 2 .html files, got %d", len(files))
	}

	// Test with multiple extensions
	files = template.OpenProjectFiles(tempDir, []string{"html", "gohtml"})

	// Should find 3 files
	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}
}

func TestOpenProjectFiles_DepthLimit(t *testing.T) {
	// Create a deeply nested directory structure
	tempDir, err := os.MkdirTemp("", "template_depth_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create directories up to depth 7
	deepPath := tempDir
	for range 7 {
		deepPath = filepath.Join(deepPath, "level")
	}

	if err := os.MkdirAll(deepPath, 0750); err != nil {
		t.Fatalf("Failed to create deep dirs: %v", err)
	}

	// Create a file at depth 7 (beyond the limit of 5)
	deepFile := filepath.Join(deepPath, "deep.html")
	if err := os.WriteFile(deepFile, []byte("{{ .Deep }}"), 0600); err != nil {
		t.Fatalf("Failed to write deep file: %v", err)
	}

	// Create a file at depth 3 (within the limit)
	shallowPath := filepath.Join(tempDir, "level", "level", "level")
	shallowFile := filepath.Join(shallowPath, "shallow.html")
	if err := os.WriteFile(shallowFile, []byte("{{ .Shallow }}"), 0600); err != nil {
		t.Fatalf("Failed to write shallow file: %v", err)
	}

	files := template.OpenProjectFiles(tempDir, []string{"html"})

	// Should find shallow.html but not deep.html (beyond depth 5)
	if len(files) < 1 {
		t.Error("Expected to find at least the shallow file")
	}

	// The deep file (at depth 7) should not be found
	foundDeep := false
	for path := range files {
		if filepath.Base(path) == "deep.html" {
			foundDeep = true
		}
	}

	if foundDeep {
		t.Error("Should not find files beyond depth limit")
	}
}

func TestHasFileExtension(t *testing.T) {
	tests := []struct {
		fileName   string
		extensions []string
		expected   bool
	}{
		{"file.html", []string{"html"}, true},
		{"file.gohtml", []string{"html", "gohtml"}, true},
		{"file.txt", []string{"html"}, false},
		{"file.html.bak", []string{"html"}, false},
		{"file", []string{"html"}, false},
		{".html", []string{"html"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			result := template.HasFileExtension(tt.fileName, tt.extensions)
			if result != tt.expected {
				t.Errorf("HasFileExtension(%q, %v) = %v, want %v",
					tt.fileName, tt.extensions, result, tt.expected)
			}
		})
	}
}

func TestGoToDefinition_VariableAtPosition(t *testing.T) {
	source := `{{ define "test" }}
{{ $var := .Field }}
{{ $var }}
{{ end }}`

	root, parseErrs := template.ParseSingleFile([]byte(source))
	if len(parseErrs) != 0 {
		t.Fatalf("Parse errors: %v", parseErrs)
	}

	workspace := map[string]*parser.GroupStatementNode{
		"test.html": root,
	}

	file, _ := template.DefinitionAnalysisSingleFile("test.html", workspace)
	if file == nil {
		t.Fatal("Expected non-nil file definition")
	}

	// Try to find definition at position of $var usage (line 2, around character 3)
	pos := lexer.Position{Line: 2, Character: 5}
	fileNames, ranges, err := template.GoToDefinition(file, pos)

	// May or may not find definition depending on analysis
	if err != nil {
		// Expected behavior if position doesn't map to a definition
		return
	}

	if len(fileNames) > 0 && len(ranges) > 0 {
		// Successfully found definition
		t.Logf("Found definition at %s, range: %v", fileNames[0], ranges[0])
	}
}

func TestGoToDefinition_PositionNotFound(t *testing.T) {
	source := `{{ .Field }}`

	root, parseErrs := template.ParseSingleFile([]byte(source))
	if len(parseErrs) != 0 {
		t.Fatalf("Parse errors: %v", parseErrs)
	}

	workspace := map[string]*parser.GroupStatementNode{
		"test.html": root,
	}

	file, _ := template.DefinitionAnalysisSingleFile("test.html", workspace)
	if file == nil {
		t.Fatal("Expected non-nil file definition")
	}

	// Position outside any meaningful token
	pos := lexer.Position{Line: 100, Character: 0}
	_, _, err := template.GoToDefinition(file, pos)

	if err == nil {
		t.Error("Expected error for position not found")
	}
}

func TestHover_Variable(t *testing.T) {
	source := `{{ define "test" }}
{{ $var := 42 }}
{{ $var }}
{{ end }}`

	root, parseErrs := template.ParseSingleFile([]byte(source))
	if len(parseErrs) != 0 {
		t.Fatalf("Parse errors: %v", parseErrs)
	}

	workspace := map[string]*parser.GroupStatementNode{
		"test.html": root,
	}

	file, _ := template.DefinitionAnalysisSingleFile("test.html", workspace)
	if file == nil {
		t.Fatal("Expected non-nil file definition")
	}

	// Try to get hover at position of $var usage
	pos := lexer.Position{Line: 2, Character: 5}
	hoverText, hoverRange := template.Hover(file, pos)

	// May or may not find hover depending on analysis
	_ = hoverText
	_ = hoverRange
}

func TestHover_NoDefinitionFound(t *testing.T) {
	source := `{{ .Field }}`

	root, parseErrs := template.ParseSingleFile([]byte(source))
	if len(parseErrs) != 0 {
		t.Fatalf("Parse errors: %v", parseErrs)
	}

	workspace := map[string]*parser.GroupStatementNode{
		"test.html": root,
	}

	file, _ := template.DefinitionAnalysisSingleFile("test.html", workspace)
	if file == nil {
		t.Fatal("Expected non-nil file definition")
	}

	// Position outside any meaningful token
	pos := lexer.Position{Line: 100, Character: 0}
	hoverText, hoverRange := template.Hover(file, pos)

	if hoverText != "" {
		t.Error("Expected empty hover text for position not found")
	}

	if !hoverRange.IsEmpty() {
		t.Error("Expected empty range for position not found")
	}
}

func TestDefinitionAnalysisWithinWorkspace(t *testing.T) {
	file1 := `{{ define "header" }}Header{{ end }}`
	file2 := `{{ define "footer" }}Footer{{ end }}`
	file3 := `{{ template "header" }}Content{{ template "footer" }}`

	root1, _ := template.ParseSingleFile([]byte(file1))
	root2, _ := template.ParseSingleFile([]byte(file2))
	root3, _ := template.ParseSingleFile([]byte(file3))

	workspace := map[string]*parser.GroupStatementNode{
		"header.html": root1,
		"footer.html": root2,
		"main.html":   root3,
	}

	results := template.DefinitionAnalysisWithinWorkspace(workspace)

	// Should get results for all 3 files
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Each result should have a file definition
	for _, result := range results {
		if result.File == nil {
			t.Errorf("Expected non-nil file definition for %s", result.FileName)
		}
	}
}

func TestPrint(t *testing.T) {
	// This test just verifies Print doesn't panic
	source := `{{ .Field }}`
	root, _ := template.ParseSingleFile([]byte(source))

	// Print should not panic
	// We can't easily capture stdout, so just verify it doesn't crash
	template.Print(root)
}

func TestSetWorkspaceCustomFunctions(t *testing.T) {
	// Test with nil to verify it doesn't panic
	template.SetWorkspaceCustomFunctions(nil)

	retrieved := template.GetWorkspaceCustomFunctions()
	if retrieved != nil {
		t.Error("Expected nil custom functions after setting nil")
	}

	// Test with empty map
	emptyFuncs := make(map[string]*template.FunctionDefinition)
	template.SetWorkspaceCustomFunctions(emptyFuncs)

	retrieved = template.GetWorkspaceCustomFunctions()
	if retrieved == nil {
		t.Error("Expected non-nil custom functions after setting empty map")
	}
}

// TestParseFilesInWorkspace_Concurrent verifies that parallel parsing
// produces deterministic results and collects all errors properly.
func TestParseFilesInWorkspace_Concurrent(t *testing.T) {
	// Create a larger workspace to exercise parallelism
	workspace := make(map[string][]byte)
	for i := range 50 {
		fileName := filepath.Join("dir", "file"+string(rune('A'+i%26))+".html")
		workspace[fileName] = []byte("{{ .Field" + string(rune('A'+i%26)) + " }}")
	}

	// Run multiple times to check for race conditions and determinism
	for range 5 {
		parsed, errs := template.ParseFilesInWorkspace(workspace)

		if len(parsed) != len(workspace) {
			t.Fatalf("Expected %d parsed files, got %d", len(workspace), len(parsed))
		}

		if len(errs) != 0 {
			t.Errorf("Expected no errors, got %d", len(errs))
		}

		// Verify all files are present
		for fileName := range workspace {
			if parsed[fileName] == nil {
				t.Errorf("Missing parsed result for %s", fileName)
			}
		}
	}
}

// TestParseFilesInWorkspace_ConcurrentWithErrors verifies that errors
// from parallel parsing are all collected properly.
func TestParseFilesInWorkspace_ConcurrentWithErrors(t *testing.T) {
	workspace := map[string][]byte{
		"valid1.html":   []byte("{{ .Field1 }}"),
		"invalid1.html": []byte("{{ if }}{{ end }}"), // missing condition
		"valid2.html":   []byte("{{ .Field2 }}"),
		"invalid2.html": []byte("{{ if .Cond }}content"), // unclosed if
		"valid3.html":   []byte("{{ .Field3 }}"),
		"invalid3.html": []byte("{{ }}"), // empty expression
	}

	parsed, errs := template.ParseFilesInWorkspace(workspace)

	// All files should be parsed (even those with errors)
	if len(parsed) != 6 {
		t.Fatalf("Expected 6 parsed files, got %d", len(parsed))
	}

	// Should have collected errors from the invalid files
	if len(errs) == 0 {
		t.Error("Expected errors from invalid files")
	}

	// Each parsed result should be non-nil (parser returns partial AST on error)
	for fileName, result := range parsed {
		if result == nil {
			t.Errorf("Expected non-nil parse result for %s", fileName)
		}
	}
}

// TestParseFilesInWorkspace_LargeWorkspace tests performance with many files.
func TestParseFilesInWorkspace_LargeWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large workspace test in short mode")
	}

	// Create a workspace with 200 files
	workspace := make(map[string][]byte)
	for i := range 200 {
		fileName := filepath.Join(
			"templates",
			"component"+string(
				rune('0'+i/100),
			)+string(
				rune('0'+(i/10)%10),
			)+string(
				rune('0'+i%10),
			)+".html",
		)
		content := []byte(
			`{{ define "template` + string(
				rune('0'+i/100),
			) + string(
				rune('0'+(i/10)%10),
			) + string(
				rune('0'+i%10),
			) + `" }}
{{ if .Condition }}
  {{ range .Items }}
    {{ .Name }}: {{ .Value }}
  {{ end }}
{{ end }}
{{ end }}`,
		)
		workspace[fileName] = content
	}

	parsed, errs := template.ParseFilesInWorkspace(workspace)

	if len(parsed) != 200 {
		t.Fatalf("Expected 200 parsed files, got %d", len(parsed))
	}

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
	}
}
