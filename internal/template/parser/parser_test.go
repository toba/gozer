package parser

import (
	"testing"

	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/testutil"
)

// tokenizeAndParse is a helper that tokenizes source and parses it.
// Returns both lexer errors and parser errors combined.
func tokenizeAndParse(source string) (*GroupStatementNode, []lexer.Error) {
	streams, lexerErrs := lexer.Tokenize([]byte(source))
	root, parseErrs := Parse(streams)
	lexerErrs = append(lexerErrs, parseErrs...)
	return root, lexerErrs
}

// keywordTestCase defines a test case for keyword parsing tests.
type keywordTestCase struct {
	name       string
	source     string
	wantError  bool
	errorMatch string
}

// runKeywordTests runs a set of keyword parsing test cases with a custom validation function.
func runKeywordTests(
	t *testing.T,
	tests []keywordTestCase,
	validate func(t *testing.T, root *GroupStatementNode),
) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := tokenizeAndParse(tt.source)

			if tt.wantError {
				if len(errs) == 0 {
					t.Error("Expected error but got none")
				}
				if tt.errorMatch != "" {
					foundMatch := false
					for _, err := range errs {
						if testutil.ContainsSubstring(err.GetError(), tt.errorMatch) {
							foundMatch = true
							break
						}
					}
					if !foundMatch {
						t.Errorf(
							"Expected error containing '%s', got: %v",
							tt.errorMatch,
							errs,
						)
					}
				}
				return
			}

			if len(errs) != 0 {
				t.Errorf("Expected no errors, got: %v", errs)
			}

			if validate != nil && len(root.Statements) > 0 {
				validate(t, root)
			}
		})
	}
}

// parseStatementFromSource is a helper that tokenizes and parses a single template statement.
// Returns the first statement from the AST (skipping the root GroupStatementNode).
func parseStatementFromSource(source string) (AstNode, *ParseError) {
	streams, _ := lexer.Tokenize([]byte(source))
	if len(streams) == 0 {
		return nil, nil
	}

	parser := Parser{}
	parser.Reset(streams[0])
	return parser.ParseStatement()
}

func TestParse_EmptyInput(t *testing.T) {
	root, errs := Parse(nil)

	if root == nil {
		t.Fatal("Parse should return non-nil root even for empty input")
	}

	if root.Kind() != KindGroupStatement {
		t.Errorf("Expected root kind %v, got %v", KindGroupStatement, root.Kind())
	}

	if !root.IsRoot() {
		t.Error("Root node should be marked as root")
	}

	if len(root.Statements) != 0 {
		t.Errorf("Expected no statements for empty input, got %d", len(root.Statements))
	}

	if len(errs) != 0 {
		t.Errorf("Expected no errors for empty input, got %d", len(errs))
	}
}

func TestParse_SingleExpression(t *testing.T) {
	root, errs := tokenizeAndParse("{{ .Field }}")

	if root == nil {
		t.Fatal("Parse should return non-nil root")
	}

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
	}

	if len(root.Statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(root.Statements))
	}

	stmt := root.Statements[0]
	multiExpr, ok := stmt.(*MultiExpressionNode)
	if !ok {
		t.Fatalf("Expected MultiExpressionNode, got %T", stmt)
	}

	if multiExpr.Kind() != KindMultiExpression {
		t.Errorf("Expected kind %v, got %v", KindMultiExpression, multiExpr.Kind())
	}
}

func TestParse_MultipleStatements(t *testing.T) {
	root, errs := tokenizeAndParse("{{ .Field1 }} some text {{ .Field2 }}")

	if root == nil {
		t.Fatal("Parse should return non-nil root")
	}

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
	}

	if len(root.Statements) != 2 {
		t.Errorf("Expected 2 statements, got %d", len(root.Statements))
	}
}

func TestParse_UnclosedScope(t *testing.T) {
	root, errs := tokenizeAndParse("{{ if .Cond }}content")

	if root == nil {
		t.Fatal("Parse should return non-nil root")
	}

	if len(errs) == 0 {
		t.Error("Expected error for unclosed scope")
	}

	// Check that we got the "missing matching '{{ end }}'" error
	foundUnclosedError := false
	for _, err := range errs {
		if err.GetError() == "missing matching '{{ end }}' statement" {
			foundUnclosedError = true
			break
		}
	}

	if !foundUnclosedError {
		t.Errorf("Expected 'missing matching {{ end }}' error, got: %v", errs)
	}
}

func TestParseStatement_Expression(t *testing.T) {
	stmt, err := parseStatementFromSource("{{ .Field }}")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if stmt == nil {
		t.Fatal("Expected non-nil statement")
	}

	multiExpr, ok := stmt.(*MultiExpressionNode)
	if !ok {
		t.Fatalf("Expected MultiExpressionNode, got %T", stmt)
	}

	if len(multiExpr.Expressions) != 1 {
		t.Errorf("Expected 1 expression, got %d", len(multiExpr.Expressions))
	}
}

func TestParseStatement_VariableDeclaration(t *testing.T) {
	stmt, err := parseStatementFromSource("{{ $var := .Field }}")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if stmt == nil {
		t.Fatal("Expected non-nil statement")
	}

	varDecl, ok := stmt.(*VariableDeclarationNode)
	if !ok {
		t.Fatalf("Expected VariableDeclarationNode, got %T", stmt)
	}

	if varDecl.Kind() != KindVariableDeclaration {
		t.Errorf("Expected kind %v, got %v", KindVariableDeclaration, varDecl.Kind())
	}

	if len(varDecl.VariableNames) != 1 {
		t.Fatalf("Expected 1 variable name, got %d", len(varDecl.VariableNames))
	}

	if string(varDecl.VariableNames[0].Value) != "$var" {
		t.Errorf(
			"Expected variable name '$var', got '%s'",
			string(varDecl.VariableNames[0].Value),
		)
	}
}

func TestParseStatement_VariableAssignment(t *testing.T) {
	stmt, err := parseStatementFromSource("{{ $var = .Field }}")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if stmt == nil {
		t.Fatal("Expected non-nil statement")
	}

	varAssign, ok := stmt.(*VariableAssignationNode)
	if !ok {
		t.Fatalf("Expected VariableAssignationNode, got %T", stmt)
	}

	if varAssign.Kind() != KindVariableAssignment {
		t.Errorf("Expected kind %v, got %v", KindVariableAssignment, varAssign.Kind())
	}
}

func TestParseStatement_Comment(t *testing.T) {
	stmt, err := parseStatementFromSource("{{ /* this is a comment */ }}")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if stmt == nil {
		t.Fatal("Expected non-nil statement")
	}

	comment, ok := stmt.(*CommentNode)
	if !ok {
		t.Fatalf("Expected CommentNode, got %T", stmt)
	}

	if comment.Kind() != KindComment {
		t.Errorf("Expected kind %v, got %v", KindComment, comment.Kind())
	}
}

func TestParseKeyword_If(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantError  bool
		checkStmts int
	}{
		{
			name:       "simple if",
			source:     "{{ if .Condition }}content{{ end }}",
			wantError:  false,
			checkStmts: 2, // if group + end
		},
		{
			name:       "if with variable declaration",
			source:     "{{ if $var := .Field }}content{{ end }}",
			wantError:  false,
			checkStmts: 2,
		},
		{
			name:      "if missing condition",
			source:    "{{ if }}",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := tokenizeAndParse(tt.source)

			if tt.wantError {
				if len(errs) == 0 {
					t.Error("Expected error but got none")
				}
				return
			}

			if len(errs) != 0 {
				t.Errorf("Expected no errors, got: %v", errs)
			}

			if len(root.Statements) != tt.checkStmts {
				t.Errorf(
					"Expected %d statements, got %d",
					tt.checkStmts,
					len(root.Statements),
				)
			}

			// First statement should be a GroupStatementNode with KindIf
			if len(root.Statements) > 0 {
				firstStmt, ok := root.Statements[0].(*GroupStatementNode)
				if !ok {
					t.Fatalf("Expected GroupStatementNode, got %T", root.Statements[0])
				}
				if firstStmt.Kind() != KindIf {
					t.Errorf("Expected kind KindIf, got %v", firstStmt.Kind())
				}
			}
		})
	}
}

func TestParseKeyword_Else(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantKinds  []Kind
		wantError  bool
		errorMatch string
	}{
		{
			name:      "else standalone",
			source:    "{{ if .Cond }}a{{ else }}b{{ end }}",
			wantKinds: []Kind{KindIf, KindElse, KindEnd},
			wantError: false,
		},
		{
			name:      "else if",
			source:    "{{ if .Cond }}a{{ else if .Other }}b{{ end }}",
			wantKinds: []Kind{KindIf, KindElseIf, KindEnd},
			wantError: false,
		},
		{
			name:      "else with",
			source:    "{{ with .Field }}a{{ else with .Other }}b{{ end }}",
			wantKinds: []Kind{KindWith, KindElseWith, KindEnd},
			wantError: false,
		},
		{
			name:       "else without opener",
			source:     "{{ else }}content{{ end }}",
			wantError:  true,
			errorMatch: "extraneous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := tokenizeAndParse(tt.source)

			if tt.wantError {
				if len(errs) == 0 {
					t.Error("Expected error but got none")
				}
				if tt.errorMatch != "" {
					foundMatch := false
					for _, err := range errs {
						if testutil.ContainsSubstring(err.GetError(), tt.errorMatch) {
							foundMatch = true
							break
						}
					}
					if !foundMatch {
						t.Errorf(
							"Expected error containing '%s', got: %v",
							tt.errorMatch,
							errs,
						)
					}
				}
				return
			}

			if len(errs) != 0 {
				t.Errorf("Expected no errors, got: %v", errs)
			}

			if len(root.Statements) != len(tt.wantKinds) {
				t.Fatalf(
					"Expected %d statements, got %d",
					len(tt.wantKinds),
					len(root.Statements),
				)
			}

			for i, wantKind := range tt.wantKinds {
				group, ok := root.Statements[i].(*GroupStatementNode)
				if !ok {
					t.Errorf(
						"Statement %d: expected GroupStatementNode, got %T",
						i,
						root.Statements[i],
					)
					continue
				}
				if group.Kind() != wantKind {
					t.Errorf(
						"Statement %d: expected kind %v, got %v",
						i,
						wantKind,
						group.Kind(),
					)
				}
			}
		})
	}
}

func TestParseKeyword_Range(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantError bool
	}{
		{
			name:      "simple range",
			source:    "{{ range .Items }}item{{ end }}",
			wantError: false,
		},
		{
			name:      "range with index and value",
			source:    "{{ range $idx, $val := .Items }}item{{ end }}",
			wantError: false,
		},
		{
			name:      "range with else",
			source:    "{{ range .Items }}item{{ else }}no items{{ end }}",
			wantError: false,
		},
		{
			name:      "range missing collection",
			source:    "{{ range }}{{ end }}",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := tokenizeAndParse(tt.source)

			if tt.wantError {
				if len(errs) == 0 {
					t.Error("Expected error but got none")
				}
				return
			}

			if len(errs) != 0 {
				t.Errorf("Expected no errors, got: %v", errs)
			}

			// Find the range statement
			if len(root.Statements) > 0 {
				group, ok := root.Statements[0].(*GroupStatementNode)
				if !ok {
					t.Fatalf("Expected GroupStatementNode, got %T", root.Statements[0])
				}
				if group.Kind() != KindRangeLoop {
					t.Errorf("Expected kind KindRangeLoop, got %v", group.Kind())
				}
			}
		})
	}
}

func TestParseKeyword_With(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantError bool
	}{
		{
			name:      "simple with",
			source:    "{{ with .Field }}content{{ end }}",
			wantError: false,
		},
		{
			name:      "with variable declaration",
			source:    "{{ with $var := .Field }}content{{ end }}",
			wantError: false,
		},
		{
			name:      "with missing pipeline",
			source:    "{{ with }}{{ end }}",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := tokenizeAndParse(tt.source)

			if tt.wantError {
				if len(errs) == 0 {
					t.Error("Expected error but got none")
				}
				return
			}

			if len(errs) != 0 {
				t.Errorf("Expected no errors, got: %v", errs)
			}

			// Find the with statement
			if len(root.Statements) > 0 {
				group, ok := root.Statements[0].(*GroupStatementNode)
				if !ok {
					t.Fatalf("Expected GroupStatementNode, got %T", root.Statements[0])
				}
				if group.Kind() != KindWith {
					t.Errorf("Expected kind KindWith, got %v", group.Kind())
				}
			}
		})
	}
}

func TestParseKeyword_Define(t *testing.T) {
	tests := []keywordTestCase{
		{
			name:      "simple define",
			source:    `{{ define "mytemplate" }}content{{ end }}`,
			wantError: false,
		},
		{
			name:       "define missing name",
			source:     `{{ define }}{{ end }}`,
			wantError:  true,
			errorMatch: "expect a string",
		},
		{
			name:       "define with expression (error)",
			source:     `{{ define "name" .Data }}{{ end }}`,
			wantError:  true,
			errorMatch: "does not accept",
		},
	}

	runKeywordTests(t, tests, func(t *testing.T, root *GroupStatementNode) {
		group, ok := root.Statements[0].(*GroupStatementNode)
		if !ok {
			t.Fatalf("Expected GroupStatementNode, got %T", root.Statements[0])
		}
		if group.Kind() != KindDefineTemplate {
			t.Errorf("Expected kind KindDefineTemplate, got %v", group.Kind())
		}
	})
}

func TestParseKeyword_Block(t *testing.T) {
	tests := []keywordTestCase{
		{
			name:      "simple block",
			source:    `{{ block "myblock" . }}content{{ end }}`,
			wantError: false,
		},
		{
			name:       "block missing name",
			source:     `{{ block }}{{ end }}`,
			wantError:  true,
			errorMatch: "expect a string",
		},
	}

	runKeywordTests(t, tests, func(t *testing.T, root *GroupStatementNode) {
		group, ok := root.Statements[0].(*GroupStatementNode)
		if !ok {
			t.Fatalf("Expected GroupStatementNode, got %T", root.Statements[0])
		}
		if group.Kind() != KindBlockTemplate {
			t.Errorf("Expected kind KindBlockTemplate, got %v", group.Kind())
		}
	})
}

func TestParseKeyword_Template(t *testing.T) {
	tests := []keywordTestCase{
		{
			name:      "template without data",
			source:    `{{ template "mytemplate" }}`,
			wantError: false,
		},
		{
			name:      "template with data",
			source:    `{{ template "mytemplate" .Data }}`,
			wantError: false,
		},
		{
			name:       "template missing name",
			source:     `{{ template }}`,
			wantError:  true,
			errorMatch: "missing template name",
		},
	}

	runKeywordTests(t, tests, func(t *testing.T, root *GroupStatementNode) {
		tmpl, ok := root.Statements[0].(*TemplateStatementNode)
		if !ok {
			t.Fatalf("Expected TemplateStatementNode, got %T", root.Statements[0])
		}
		if tmpl.Kind() != KindUseTemplate {
			t.Errorf("Expected kind KindUseTemplate, got %v", tmpl.Kind())
		}
	})
}

func TestParseKeyword_End(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantError  bool
		errorMatch string
	}{
		{
			name:      "valid end",
			source:    "{{ if .Cond }}content{{ end }}",
			wantError: false,
		},
		{
			name:       "end with extra tokens",
			source:     "{{ if .Cond }}content{{ end .Extra }}",
			wantError:  true,
			errorMatch: "standalone",
		},
		{
			name:       "extraneous end",
			source:     "{{ end }}",
			wantError:  true,
			errorMatch: "extraneous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := tokenizeAndParse(tt.source)

			if tt.wantError {
				if len(errs) == 0 {
					t.Error("Expected error but got none")
				}
				if tt.errorMatch != "" {
					foundMatch := false
					for _, err := range errs {
						if testutil.ContainsSubstring(err.GetError(), tt.errorMatch) {
							foundMatch = true
							break
						}
					}
					if !foundMatch {
						t.Errorf(
							"Expected error containing '%s', got: %v",
							tt.errorMatch,
							errs,
						)
					}
				}
				return
			}

			if len(errs) != 0 {
				t.Errorf("Expected no errors, got: %v", errs)
			}

			_ = root
		})
	}
}

func TestParseKeyword_BreakContinue(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantError  bool
		errorMatch string
	}{
		{
			name:      "break in range",
			source:    "{{ range .Items }}{{ break }}{{ end }}",
			wantError: false,
		},
		{
			name:      "continue in range",
			source:    "{{ range .Items }}{{ continue }}{{ end }}",
			wantError: false,
		},
		{
			name:       "break outside range",
			source:     "{{ if .Cond }}{{ break }}{{ end }}",
			wantError:  true,
			errorMatch: "missing 'range' loop",
		},
		{
			name:       "continue outside range",
			source:     "{{ if .Cond }}{{ continue }}{{ end }}",
			wantError:  true,
			errorMatch: "missing 'range' loop",
		},
		{
			name:       "break with extra tokens",
			source:     "{{ range .Items }}{{ break .Something }}{{ end }}",
			wantError:  true,
			errorMatch: "standalone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := tokenizeAndParse(tt.source)

			if tt.wantError {
				if len(errs) == 0 {
					t.Error("Expected error but got none")
				}
				if tt.errorMatch != "" {
					foundMatch := false
					for _, err := range errs {
						if testutil.ContainsSubstring(err.GetError(), tt.errorMatch) {
							foundMatch = true
							break
						}
					}
					if !foundMatch {
						t.Errorf(
							"Expected error containing '%s', got: %v",
							tt.errorMatch,
							errs,
						)
					}
				}
				return
			}

			if len(errs) != 0 {
				t.Errorf("Expected no errors, got: %v", errs)
			}

			_ = root
		})
	}
}
