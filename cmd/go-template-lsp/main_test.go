package main

import (
	"testing"

	"github.com/yayolande/gota"
	"github.com/yayolande/gota/analyzer"
	"github.com/yayolande/gota/parser"
)

func TestBuiltinFunctions(t *testing.T) {
	// Simple template using builtin functions
	content := []byte(`{{if and true false}}yes{{end}}`)

	parseTree, parseErrs := gota.ParseSingleFile(content)
	if len(parseErrs) > 0 {
		for _, err := range parseErrs {
			t.Logf("Parse error: %s", err.GetError())
		}
	}

	if parseTree == nil {
		t.Fatal("parseTree is nil")
	}

	// Run analysis
	parsedFiles := map[string]*parser.GroupStatementNode{
		"test.html": parseTree,
	}
	analyzed := gota.DefinitionAnalisisWithinWorkspace(parsedFiles)

	for _, file := range analyzed {
		if len(file.Errs) > 0 {
			for _, err := range file.Errs {
				t.Logf("Analysis error in %s: %s", file.FileName, err.GetError())
			}
		}
	}
}

func TestBuiltinFunctionsSingleFile(t *testing.T) {
	// Test using the single file analysis path
	content := []byte(`{{if and true false}}yes{{end}}`)

	parseTree, parseErrs := gota.ParseSingleFile(content)
	if len(parseErrs) > 0 {
		for _, err := range parseErrs {
			t.Logf("Parse error: %s", err.GetError())
		}
	}

	if parseTree == nil {
		t.Fatal("parseTree is nil")
	}

	// Run single file analysis directly
	parsedFiles := map[string]*parser.GroupStatementNode{
		"test.html": parseTree,
	}
	file, errs := gota.DefinitionAnalysisSingleFile("test.html", parsedFiles)

	if len(errs) > 0 {
		for _, err := range errs {
			t.Logf("Analysis error (single file): %s", err.GetError())
		}
	}

	// Check what functions are available
	t.Logf("File: %v", file)
}

func TestDirectAnalysis(t *testing.T) {
	// Test using analyzer directly
	content := []byte(`{{if and true false}}yes{{end}}`)

	parseTree, parseErrs := gota.ParseSingleFile(content)
	if len(parseErrs) > 0 {
		for _, err := range parseErrs {
			t.Logf("Parse error: %s", err.GetError())
		}
	}

	file, errs := analyzer.DefinitionAnalysis("test.html", parseTree, nil)
	if len(errs) > 0 {
		for _, err := range errs {
			t.Logf("Analysis error (direct): %s", err.GetError())
		}
	}

	t.Logf("Direct analysis file: %v", file)
}

func TestSimpleExpression(t *testing.T) {
	// Simplest possible case - just call a function
	content := []byte(`{{and true false}}`)

	parseTree, parseErrs := gota.ParseSingleFile(content)
	if len(parseErrs) > 0 {
		for _, err := range parseErrs {
			t.Logf("Parse error: %s", err.GetError())
		}
	}

	// Debug: print the parsed tree structure
	t.Logf("ParseTree: %+v", parseTree)
	if parseTree != nil && len(parseTree.Statements) > 0 {
		for i, child := range parseTree.Statements {
			t.Logf("Statement[%d]: %T = %+v", i, child, child)
		}
	}

	parsedFiles := map[string]*parser.GroupStatementNode{
		"test.html": parseTree,
	}
	analyzed := gota.DefinitionAnalisisWithinWorkspace(parsedFiles)

	for _, file := range analyzed {
		if len(file.Errs) > 0 {
			for _, err := range file.Errs {
				t.Logf("Analysis error in %s: %s at %v", file.FileName, err.GetError(), err.GetRange())
			}
		} else {
			t.Log("No errors! Function 'and' was found.")
		}
	}
}

func TestCustomFunctions(t *testing.T) {
	// Template using custom functions like lower, upper
	content := []byte(`{{.Name | lower}}`)

	parseTree, parseErrs := gota.ParseSingleFile(content)
	if len(parseErrs) > 0 {
		for _, err := range parseErrs {
			t.Logf("Parse error: %s", err.GetError())
		}
	}

	if parseTree == nil {
		t.Fatal("parseTree is nil")
	}

	parsedFiles := map[string]*parser.GroupStatementNode{
		"test.html": parseTree,
	}
	analyzed := gota.DefinitionAnalisisWithinWorkspace(parsedFiles)

	for _, file := range analyzed {
		if len(file.Errs) > 0 {
			for _, err := range file.Errs {
				t.Logf("Analysis error in %s: %s", file.FileName, err.GetError())
			}
		}
	}
}

func TestComplexTemplate(t *testing.T) {
	// Template similar to credential-form.html
	content := []byte(`{{define "test" -}}
{{range .Platforms -}}
<option value="{{.Name | lower}}"
        {{if and $.Credential (eq ($.Credential.Platform | lower) (.Name | lower))}}selected{{end}}>
    {{.Name}} ({{.PlatformType | upper}})
</option>
{{- end -}}
{{- end -}}`)

	parseTree, parseErrs := gota.ParseSingleFile(content)
	if len(parseErrs) > 0 {
		for _, err := range parseErrs {
			t.Logf("Parse error: %s", err.GetError())
		}
	}

	if parseTree == nil {
		t.Fatal("parseTree is nil")
	}

	parsedFiles := map[string]*parser.GroupStatementNode{
		"test.html": parseTree,
	}
	analyzed := gota.DefinitionAnalisisWithinWorkspace(parsedFiles)

	for _, file := range analyzed {
		if len(file.Errs) > 0 {
			for _, err := range file.Errs {
				t.Logf("Analysis error in %s: %s at line %d:%d", file.FileName, err.GetError(), err.GetRange().Start.Line, err.GetRange().Start.Character)
			}
		}
	}
}
