package template_test

import (
	"slices"
	"testing"

	"github.com/pacer/gozer/internal/template"
	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/parser"
	"github.com/pacer/gozer/internal/template/testutil"
)

// TestLexerParserIntegration tests the full pipeline from source to AST.
func TestLexerParserIntegration(t *testing.T) {
	tests := []struct {
		name             string
		source           string
		expectStatements int
		expectNoErrors   bool
	}{
		{
			name:             "simple expression",
			source:           "{{ .Field }}",
			expectStatements: 1,
			expectNoErrors:   true,
		},
		{
			name:             "if-else-end",
			source:           "{{ if .Cond }}true{{ else }}false{{ end }}",
			expectStatements: 3, // if, else, end
			expectNoErrors:   true,
		},
		{
			name:             "range with break",
			source:           "{{ range .Items }}{{ break }}{{ end }}",
			expectStatements: 2, // range (with break inside), end
			expectNoErrors:   true,
		},
		{
			name:           "nested structures",
			source:         "{{ if .A }}{{ range .B }}{{ with .C }}c{{ end }}b{{ end }}a{{ end }}",
			expectNoErrors: true,
		},
		{
			name:           "template definition and usage",
			source:         `{{ define "test" }}content{{ end }}{{ template "test" }}`,
			expectNoErrors: true,
		},
		{
			name:             "variable declaration and usage",
			source:           "{{ $var := .Field }}{{ $var }}",
			expectStatements: 2,
			expectNoErrors:   true,
		},
		{
			name:             "piped expression",
			source:           "{{ .Field | upper | lower }}",
			expectStatements: 1,
			expectNoErrors:   true,
		},
		{
			name:             "parenthesized expression",
			source:           "{{ (index .Items 0).Name }}",
			expectStatements: 1,
			expectNoErrors:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Tokenize
			streams, lexerErrs := lexer.Tokenize([]byte(tt.source))

			if tt.expectNoErrors && len(lexerErrs) > 0 {
				t.Errorf("Unexpected lexer errors: %v", lexerErrs)
			}

			// Step 2: Parse
			root, parseErrs := parser.Parse(streams)

			if tt.expectNoErrors && len(parseErrs) > 0 {
				t.Errorf("Unexpected parser errors: %v", parseErrs)
			}

			// Step 3: Verify AST
			if root == nil {
				t.Fatal("Expected non-nil AST root")
			}

			if !root.IsRoot() {
				t.Error("Root node should be marked as root")
			}

			if tt.expectStatements > 0 && len(root.Statements) != tt.expectStatements {
				t.Errorf(
					"Expected %d statements, got %d",
					tt.expectStatements,
					len(root.Statements),
				)
			}
		})
	}
}

// TestComplexTemplate tests a realistic complex template.
func TestComplexTemplate(t *testing.T) {
	source := `{{ define "page" }}
<!DOCTYPE html>
<html>
<head>
    <title>{{ .Title }}</title>
</head>
<body>
    {{ template "header" . }}

    <main>
        {{ if .User }}
            <p>Welcome, {{ .User.Name }}!</p>
            {{ if .User.IsAdmin }}
                <a href="/admin">Admin Panel</a>
            {{ end }}
        {{ else }}
            <p>Please <a href="/login">login</a>.</p>
        {{ end }}

        {{ with .Articles }}
            <h2>Articles</h2>
            <ul>
            {{ range . }}
                <li>
                    <a href="{{ .URL }}">{{ .Title }}</a>
                    {{ if .Tags }}
                        <span class="tags">
                        {{ range $i, $tag := .Tags }}
                            {{ if $i }}, {{ end }}
                            {{ $tag }}
                        {{ end }}
                        </span>
                    {{ end }}
                </li>
            {{ end }}
            </ul>
        {{ end }}
    </main>

    {{ template "footer" . }}
</body>
</html>
{{ end }}`

	root, errs := template.ParseSingleFile([]byte(source))

	if root == nil {
		t.Fatal("Expected non-nil root")
	}

	if len(errs) != 0 {
		t.Errorf("Expected no errors for complex template, got %d: %v", len(errs), errs)
	}

	// Verify template definition was captured
	if len(root.ShortCut.TemplateDefined) != 1 {
		t.Errorf(
			"Expected 1 template definition, got %d",
			len(root.ShortCut.TemplateDefined),
		)
	}

	if _, exists := root.ShortCut.TemplateDefined["page"]; !exists {
		t.Error("Expected 'page' template to be defined")
	}

	// Verify template calls were captured
	// Template calls inside a define block are stored in that define block's shortcut,
	// not the root. Let's check the define block.
	if defineGroup, exists := root.ShortCut.TemplateDefined["page"]; exists {
		if len(defineGroup.ShortCut.TemplateCallUsed) < 2 {
			t.Errorf(
				"Expected at least 2 template calls (header, footer) in 'page' define, got %d",
				len(defineGroup.ShortCut.TemplateCallUsed),
			)
		}
	}
}

// TestErrorCascade verifies error handling doesn't cascade incorrectly.
func TestErrorCascade(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "if with invalid operator",
			source: "{{ if -- }}content{{ end }}",
		},
		{
			name:   "range with invalid operator",
			source: "{{ range ## }}item{{ end }}",
		},
		{
			name:   "nested with error",
			source: "{{ if .x }}{{ if @@ }}inner{{ end }}{{ end }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := template.ParseSingleFile([]byte(tt.source))

			hasLexerError := false
			hasExtraneousEndError := false

			for _, err := range errs {
				errMsg := err.GetError()
				t.Logf("Error: %s at %s", errMsg, err.GetRange())
				if testutil.ContainsSubstring(errMsg, "not recognized") {
					hasLexerError = true
				}
				if testutil.ContainsSubstring(errMsg, "extraneous") &&
					testutil.ContainsSubstring(errMsg, "end") {
					hasExtraneousEndError = true
				}
			}

			if !hasLexerError {
				t.Log("Note: Expected a lexer error for invalid characters")
			}

			// The cascade issue is fixed if we don't get an extraneous end error
			if hasExtraneousEndError {
				t.Error(
					"Cascade bug: lexer error incorrectly cascades to cause 'extraneous end' error",
				)
			}
		})
	}
}

// TestMultiFileTemplate tests templates that reference each other.
func TestMultiFileTemplate(t *testing.T) {
	headerFile := `{{ define "header" }}<header>{{ .SiteName }}</header>{{ end }}`
	footerFile := `{{ define "footer" }}<footer>&copy; {{ .Year }}</footer>{{ end }}`
	mainFile := `{{ template "header" . }}
<main>{{ .Content }}</main>
{{ template "footer" . }}`

	headerRoot, headerErrs := template.ParseSingleFile([]byte(headerFile))
	footerRoot, footerErrs := template.ParseSingleFile([]byte(footerFile))
	mainRoot, mainErrs := template.ParseSingleFile([]byte(mainFile))

	if len(headerErrs) != 0 || len(footerErrs) != 0 || len(mainErrs) != 0 {
		t.Errorf("Unexpected errors: header=%v, footer=%v, main=%v",
			headerErrs, footerErrs, mainErrs)
	}

	workspace := map[string]*parser.GroupStatementNode{
		"header.html": headerRoot,
		"footer.html": footerRoot,
		"main.html":   mainRoot,
	}

	results := template.DefinitionAnalysisWithinWorkspace(workspace)

	if len(results) != 3 {
		t.Errorf("Expected 3 analysis results, got %d", len(results))
	}

	// Each file should have a valid analysis
	for _, result := range results {
		if result.File == nil {
			t.Errorf("Missing file definition for %s", result.FileName)
		}
	}
}

// TestWhitespaceTrimmingIntegration tests whitespace trimming markers.
func TestWhitespaceTrimmingIntegration(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "left trim",
			source: "text {{- .Field }} more",
		},
		{
			name:   "right trim",
			source: "text {{ .Field -}} more",
		},
		{
			name:   "both trim",
			source: "text {{- .Field -}} more",
		},
		{
			name:   "trim in control flow",
			source: "{{- if .Cond -}}content{{- end -}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := template.ParseSingleFile([]byte(tt.source))

			if root == nil {
				t.Fatal("Expected non-nil root")
			}

			if len(errs) != 0 {
				t.Errorf("Expected no errors, got: %v", errs)
			}
		})
	}
}

// TestASTWalk verifies the Walk function traverses all nodes.
func TestASTWalk(t *testing.T) {
	source := `{{ if .Cond }}
{{ $var := .Field }}
{{ $var | upper }}
{{ end }}`

	root, _ := template.ParseSingleFile([]byte(source))

	var visited []parser.Kind
	walker := &testVisitor{visited: &visited}
	parser.Walk(walker, root)

	// Should visit: GroupStatement (root), GroupStatement (if),
	// VariableDeclaration, MultiExpression (twice), Expression nodes...
	if len(visited) == 0 {
		t.Error("Expected some nodes to be visited")
	}

	// Verify we visited the if statement
	if !slices.Contains(visited, parser.KindIf) {
		t.Error("Expected to visit KindIf node")
	}
}

// testVisitor is a simple visitor for testing Walk.
type testVisitor struct {
	visited  *[]parser.Kind
	isHeader bool
}

func (v *testVisitor) Visit(node parser.AstNode) parser.Visitor {
	if node == nil {
		return nil
	}
	*v.visited = append(*v.visited, node.Kind())
	return v
}

func (v *testVisitor) SetHeaderFlag(ok bool) {
	v.isHeader = ok
}

// TestEdgeCases tests various edge cases.
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantError bool
	}{
		{
			name:   "empty template block",
			source: "before{{/* empty */}}after",
		},
		{
			name:   "multiple pipes",
			source: "{{ .A | b | c | d | e }}",
		},
		{
			name:   "deeply nested parentheses",
			source: "{{ (((len .Items))) }}",
		},
		{
			name:   "string with special chars",
			source: `{{ "hello \"world\" \n\t" }}`,
		},
		{
			name:   "backtick string multiline simulation",
			source: "{{ `line1\nline2` }}",
		},
		{
			name:   "multiple variable declaration",
			source: "{{ $a, $b := range .Items }}{{ end }}",
		},
		{
			name:   "nested comments",
			source: "{{/* outer /* not nested */ */}}",
			// This may error or not depending on lexer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := template.ParseSingleFile([]byte(tt.source))

			if root == nil {
				t.Error("Expected non-nil root even with errors")
			}

			if tt.wantError && len(errs) == 0 {
				t.Error("Expected errors but got none")
			}
		})
	}
}

// TestMethodCallsWithArguments tests that method calls with arguments
// (like .Format "2006-01-02") don't produce false positive errors at parse level.
func TestMethodCallsWithArguments(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "Format method call",
			source: `{{ .CreatedAt.Format "Jan 2, 2006" }}`,
		},
		{
			name:   "nested Format",
			source: `{{ .Staff.CreatedAt.Format "Jan 2, 2006" }}`,
		},
		{
			name:   "method with variable arg",
			source: `{{ .CloudLoggingURL $.ProjectID }}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := template.ParseSingleFile([]byte(tt.source))

			// Check for false positive errors at parse level
			for _, err := range errs {
				errMsg := err.GetError()
				// These specific error messages would indicate false positives
				if testutil.ContainsSubstring(errMsg, "only function") ||
					testutil.ContainsSubstring(errMsg, "not found") {
					t.Errorf("False positive error: %s", errMsg)
				}
			}
		})
	}
}

// TestMethodCallsWithoutTypeInfo tests that method calls on unknown types
// don't produce false positive "only functions and methods accept arguments" errors.
func TestMethodCallsWithoutTypeInfo(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "method call on unknown field",
			source: `{{ .CreatedAt.Format "Jan 2, 2006" }}`,
		},
		{
			name:   "nested method call",
			source: `{{ .Staff.CreatedAt.Format "Jan 2, 2006" }}`,
		},
		{
			name:   "method with variable argument",
			source: `{{ .CloudLoggingURL $.ProjectID }}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _ := template.ParseSingleFile([]byte(tt.source))

			workspace := map[string]*parser.GroupStatementNode{
				"test.html": root,
			}

			results := template.DefinitionAnalysisWithinWorkspace(workspace)

			for _, result := range results {
				for _, err := range result.Errs {
					errMsg := err.GetError()
					// "only functions and methods accept arguments" should NOT appear
					// when the receiver type is unknown (any)
					if testutil.ContainsSubstring(
						errMsg,
						"only functions and methods accept arguments",
					) {
						t.Errorf("False positive on unknown type: %s", errMsg)
					}
				}
			}
		})
	}
}

// TestUndefinedTemplateCall tests that calling an undefined template reports an error.
func TestUndefinedTemplateCall(t *testing.T) {
	// Template calls an undefined template
	source := `{{ template "nonexistent" . }}`

	root, _ := template.ParseSingleFile([]byte(source))

	workspace := map[string]*parser.GroupStatementNode{
		"test.html": root,
	}

	results := template.DefinitionAnalysisWithinWorkspace(workspace)

	foundError := false
	for _, result := range results {
		for _, err := range result.Errs {
			errMsg := err.GetError()
			t.Logf("Error: %s", errMsg)
			if testutil.ContainsSubstring(errMsg, "template undefined") ||
				testutil.ContainsSubstring(errMsg, "template not defined") ||
				testutil.ContainsSubstring(errMsg, "nonexistent") {
				foundError = true
			}
		}
	}

	if !foundError {
		t.Error("Expected error for undefined template 'nonexistent'")
	}
}

// TestDollarVariableInRangeBlock tests that accessing root context via $ inside
// range blocks doesn't produce false positive "invalid type" or "field or method
// not found" errors.
func TestDollarVariableInRangeBlock(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "dollar field in range",
			source: `{{range .Runs}}{{$.Location}}{{end}}`,
		},
		{
			name:   "dollar field in range inside define",
			source: `{{define "test"}}{{range .Runs}}{{$.Location}}{{end}}{{end}}`,
		},
		{
			name:   "dollar field with dot field in range",
			source: `{{range .Runs}}{{.StartedAt}} {{$.Location}}{{end}}`,
		},
		{
			name:   "nested dollar access in range",
			source: `{{range .Items}}<a href="/portfolio/{{$.PortfolioID}}">{{.Name}}</a>{{end}}`,
		},
		{
			name:   "dollar as method arg in range",
			source: `{{range .LogEntries}}<a href="{{.CloudLoggingURL $.ProjectID}}">{{end}}`,
		},
		{
			name:   "dollar in nested range",
			source: `{{range .Outer}}{{range .Inner}}{{$.RootField}}{{end}}{{end}}`,
		},
		{
			name:   "dollar in with block",
			source: `{{with .Section}}{{$.GlobalSetting}}{{end}}`,
		},
		{
			name:   "dollar in define without range",
			source: `{{define "test"}}{{.Job.Schedule}} {{$.Location}}{{end}}`,
		},
		{
			name:   "dollar as function arg in define",
			source: `{{define "test"}}{{localcron .Job.Schedule .Job.TimeZone $.Location}}{{end}}`,
		},
	}

	forbiddenErrors := []string{
		"invalid type",
		"field or method not found",
		"variable undefined",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, parseErrs := template.ParseSingleFile([]byte(tt.source))
			for _, err := range parseErrs {
				t.Logf("Parse error: %s", err.GetError())
			}

			workspace := map[string]*parser.GroupStatementNode{
				"test.html": root,
			}

			results := template.DefinitionAnalysisWithinWorkspace(workspace)

			for _, result := range results {
				for _, err := range result.Errs {
					errMsg := err.GetError()
					for _, forbidden := range forbiddenErrors {
						if testutil.ContainsSubstring(errMsg, forbidden) {
							t.Errorf("False positive error for $. access: %s", errMsg)
						}
					}
				}
			}
		})
	}
}
