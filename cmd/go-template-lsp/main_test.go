package main

import (
	"strings"
	"testing"

	tmpl "github.com/pacer/gozer/internal/template"
	"github.com/pacer/gozer/internal/template/analyzer"
	"github.com/pacer/gozer/internal/template/parser"
)

func TestBuiltinFunctions(t *testing.T) {
	// Simple template using builtin functions
	content := []byte(`{{if and true false}}yes{{end}}`)

	parseTree, parseErrs := tmpl.ParseSingleFile(content)
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
	analyzed := tmpl.DefinitionAnalysisWithinWorkspace(parsedFiles)

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

	parseTree, parseErrs := tmpl.ParseSingleFile(content)
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
	file, errs := tmpl.DefinitionAnalysisSingleFile("test.html", parsedFiles)

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

	parseTree, parseErrs := tmpl.ParseSingleFile(content)
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

	parseTree, parseErrs := tmpl.ParseSingleFile(content)
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
	analyzed := tmpl.DefinitionAnalysisWithinWorkspace(parsedFiles)

	for _, file := range analyzed {
		if len(file.Errs) > 0 {
			for _, err := range file.Errs {
				t.Logf(
					"Analysis error in %s: %s at %v",
					file.FileName,
					err.GetError(),
					err.GetRange(),
				)
			}
		} else {
			t.Log("No errors! Function 'and' was found.")
		}
	}
}

func TestCustomFunctions(t *testing.T) {
	// Template using custom functions like lower, upper
	content := []byte(`{{.Name | lower}}`)

	parseTree, parseErrs := tmpl.ParseSingleFile(content)
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
	analyzed := tmpl.DefinitionAnalysisWithinWorkspace(parsedFiles)

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

	parseTree, parseErrs := tmpl.ParseSingleFile(content)
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
	analyzed := tmpl.DefinitionAnalysisWithinWorkspace(parsedFiles)

	for _, file := range analyzed {
		if len(file.Errs) > 0 {
			for _, err := range file.Errs {
				t.Logf(
					"Analysis error in %s: %s at line %d:%d",
					file.FileName,
					err.GetError(),
					err.GetRange().Start.Line,
					err.GetRange().Start.Character,
				)
			}
		}
	}
}

// TestCustomFunctionFromFuncMap verifies that custom template functions
// registered via SetWorkspaceCustomFunctions are recognized during analysis.
func TestCustomFunctionFromFuncMap(t *testing.T) {
	// Register custom functions like timehtml
	customFuncs := map[string]*tmpl.FunctionDefinition{}

	// Create a custom function definition with variadic signature
	// This simulates what ScanWorkspaceForFuncMap would discover
	customFuncs["timehtml"] = analyzer.NewCustomFunctionDefinition("timehtml", "test.go")
	customFuncs["formatDate"] = analyzer.NewCustomFunctionDefinition(
		"formatDate",
		"test.go",
	)

	tmpl.SetWorkspaceCustomFunctions(customFuncs)
	defer tmpl.SetWorkspaceCustomFunctions(nil) // cleanup

	// Template using custom functions
	content := []byte(`<div>{{timehtml .StartedAt .Location "short"}}</div>`)

	parseTree, parseErrs := tmpl.ParseSingleFile(content)
	if len(parseErrs) > 0 {
		for _, err := range parseErrs {
			t.Logf("Parse error: %s", err.GetError())
		}
	}

	parsedFiles := map[string]*parser.GroupStatementNode{
		"test.html": parseTree,
	}
	analyzed := tmpl.DefinitionAnalysisWithinWorkspace(parsedFiles)

	for _, file := range analyzed {
		for _, err := range file.Errs {
			errMsg := err.GetError()
			// timehtml should NOT be flagged as undefined
			if strings.Contains(errMsg, "timehtml") &&
				strings.Contains(errMsg, "undefined") {
				t.Errorf("custom function 'timehtml' incorrectly flagged: %s", errMsg)
			}
			if strings.Contains(errMsg, "timehtml") &&
				strings.Contains(errMsg, "field or method not found") {
				t.Errorf(
					"custom function 'timehtml' incorrectly flagged as field: %s",
					errMsg,
				)
			}
		}
	}
}

// TestMethodCallWithArguments verifies that method calls with arguments
// like .Format "2006-01-02" don't produce false positive errors.
func TestMethodCallWithArguments(t *testing.T) {
	tests := []struct {
		name            string
		template        string
		forbiddenErrors []string // errors that should NOT appear
	}{
		{
			name:            "Format method with string arg",
			template:        `<div>{{.CreatedAt.Format "Jan 2, 2006"}}</div>`,
			forbiddenErrors: []string{"only function and method accepts arguments"},
		},
		{
			name:            "Format in conditional",
			template:        `{{if .CreatedAt}}{{.CreatedAt.Format "2006-01-02"}}{{end}}`,
			forbiddenErrors: []string{"only function and method accepts arguments"},
		},
		{
			name:            "Nested field with Format",
			template:        `{{.Credential.ExpiresAt.Format "2006-01-02"}}`,
			forbiddenErrors: []string{"only function and method accepts arguments"},
		},
		{
			name:            "Method in attribute value",
			template:        `<input value="{{.ExpiresAt.Format "2006-01-02"}}">`,
			forbiddenErrors: []string{"only function and method accepts arguments"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parseTree, _ := tmpl.ParseSingleFile([]byte(tc.template))
			parsedFiles := map[string]*parser.GroupStatementNode{
				"test.html": parseTree,
			}
			analyzed := tmpl.DefinitionAnalysisWithinWorkspace(parsedFiles)

			for _, file := range analyzed {
				for _, err := range file.Errs {
					errMsg := err.GetError()
					for _, forbidden := range tc.forbiddenErrors {
						if strings.Contains(errMsg, forbidden) {
							t.Errorf(
								"unexpected error %q in template: %s",
								errMsg,
								tc.template,
							)
						}
					}
				}
			}
		})
	}
}

// TestCustomFunctionWithVariousArgCounts tests custom functions with
// different numbers of arguments.
func TestCustomFunctionWithVariousArgCounts(t *testing.T) {
	// Register a variadic custom function
	customFuncs := map[string]*tmpl.FunctionDefinition{
		"myFunc": analyzer.NewCustomFunctionDefinition("myFunc", "test.go"),
	}
	tmpl.SetWorkspaceCustomFunctions(customFuncs)
	defer tmpl.SetWorkspaceCustomFunctions(nil)

	tests := []struct {
		name     string
		template string
	}{
		{"no args", `{{myFunc}}`},
		{"one arg", `{{myFunc .A}}`},
		{"two args", `{{myFunc .A .B}}`},
		{"three args", `{{myFunc .A .B .C}}`},
		{"string literals", `{{myFunc "a" "b" "c"}}`},
		{"mixed args", `{{myFunc .A "literal" .B}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parseTree, _ := tmpl.ParseSingleFile([]byte(tc.template))
			parsedFiles := map[string]*parser.GroupStatementNode{
				"test.html": parseTree,
			}
			analyzed := tmpl.DefinitionAnalysisWithinWorkspace(parsedFiles)

			for _, file := range analyzed {
				for _, err := range file.Errs {
					errMsg := err.GetError()
					if strings.Contains(errMsg, "myFunc") &&
						(strings.Contains(errMsg, "undefined") || strings.Contains(errMsg, "not found")) {
						t.Errorf("custom function error: %s", errMsg)
					}
				}
			}
		})
	}
}

// TestDebugTimehtmlVariants tests different variations to isolate the issue
func TestDebugTimehtmlVariants(t *testing.T) {
	customFuncs := map[string]*tmpl.FunctionDefinition{
		"timehtml": analyzer.NewCustomFunctionDefinition("timehtml", "templates.go"),
	}
	tmpl.SetWorkspaceCustomFunctions(customFuncs)
	defer tmpl.SetWorkspaceCustomFunctions(nil)

	tests := []struct {
		name     string
		template string
	}{
		{"simple call", `{{timehtml .A}}`},
		{"two args", `{{timehtml .A .B}}`},
		{"three args mixed", `{{timehtml .A .B "c"}}`},
		{"dollar var", `{{timehtml .StartedAt $.Location "logs"}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parseTree, _ := tmpl.ParseSingleFile([]byte(tc.template))
			parsedFiles := map[string]*parser.GroupStatementNode{
				"test.html": parseTree,
			}
			analyzed := tmpl.DefinitionAnalysisWithinWorkspace(parsedFiles)

			for _, file := range analyzed {
				for _, err := range file.Errs {
					t.Logf("Error: %s at %v", err.GetError(), err.GetRange())
				}
			}
		})
	}
}

// TestRealWorldTemplatePatterns tests patterns from actual templates
// that were producing false positives.
func TestRealWorldTemplatePatterns(t *testing.T) {
	// Register custom functions similar to core/web
	customFuncs := map[string]*tmpl.FunctionDefinition{
		"timehtml": analyzer.NewCustomFunctionDefinition("timehtml", "templates.go"),
		"lower":    analyzer.NewCustomFunctionDefinition("lower", "templates.go"),
		"upper":    analyzer.NewCustomFunctionDefinition("upper", "templates.go"),
	}
	tmpl.SetWorkspaceCustomFunctions(customFuncs)
	defer tmpl.SetWorkspaceCustomFunctions(nil)

	tests := []struct {
		name            string
		template        string
		forbiddenErrors []string
	}{
		{
			name: "notification-list pattern",
			template: `<div>{{.Title}}</div>
{{if .Body}}<div class="truncate">{{.Body}}</div>{{end}}
<div>{{.CreatedAt.Format "Jan 2, 3:04 PM"}}</div>`,
			forbiddenErrors: []string{"only function and method accepts arguments"},
		},
		{
			name: "job-logs timehtml pattern",
			template: `<h4>
<span class="icon icon-sm icon-clock"></span>
{{timehtml .StartedAt $.Location "logs"}}
</h4>`,
			forbiddenErrors: []string{"field or method not found"},
		},
		{
			name:            "credential expires pattern",
			template:        `<input type="date" value="{{if and .Credential .Credential.ExpiresAt}}{{.Credential.ExpiresAt.Format "2006-01-02"}}{{end}}">`,
			forbiddenErrors: []string{"only function and method accepts arguments"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parseTree, parseErrs := tmpl.ParseSingleFile([]byte(tc.template))
			if len(parseErrs) > 0 {
				for _, err := range parseErrs {
					t.Logf("Parse error: %s", err.GetError())
				}
			}

			parsedFiles := map[string]*parser.GroupStatementNode{
				"test.html": parseTree,
			}
			analyzed := tmpl.DefinitionAnalysisWithinWorkspace(parsedFiles)

			for _, file := range analyzed {
				for _, err := range file.Errs {
					errMsg := err.GetError()
					for _, forbidden := range tc.forbiddenErrors {
						if strings.Contains(errMsg, forbidden) {
							t.Errorf(
								"unexpected error %q in template %s",
								errMsg,
								tc.name,
							)
						}
					}
				}
			}
		})
	}
}
