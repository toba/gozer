package parser

import (
	"testing"

	"github.com/pacer/gozer/internal/template/lexer"
	"github.com/pacer/gozer/internal/template/testutil"
)

func TestExpressionParser_SingleVariable(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		wantError   bool
		symbolCount int
	}{
		{
			name:        "dot variable",
			source:      "{{ .Field }}",
			symbolCount: 1,
		},
		{
			name:        "dollar variable",
			source:      "{{ $var }}",
			symbolCount: 1,
		},
		{
			name:        "root dot",
			source:      "{{ $ }}",
			symbolCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := parseStatementFromSource(tt.source)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			multiExpr, ok := stmt.(*MultiExpressionNode)
			if !ok {
				t.Fatalf("Expected MultiExpressionNode, got %T", stmt)
			}

			if len(multiExpr.Expressions) != 1 {
				t.Fatalf("Expected 1 expression, got %d", len(multiExpr.Expressions))
			}

			expr := multiExpr.Expressions[0]
			if len(expr.Symbols) != tt.symbolCount {
				t.Errorf("Expected %d symbols, got %d", tt.symbolCount, len(expr.Symbols))
			}
		})
	}
}

func TestExpressionParser_ChainedAccess(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		symbolCount int
	}{
		{
			name:        "simple chain",
			source:      "{{ .Field.SubField }}",
			symbolCount: 1, // .Field.SubField is one token
		},
		{
			name:        "deep chain",
			source:      "{{ .A.B.C.D }}",
			symbolCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := parseStatementFromSource(tt.source)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			multiExpr, ok := stmt.(*MultiExpressionNode)
			if !ok {
				t.Fatalf("Expected MultiExpressionNode, got %T", stmt)
			}

			if len(multiExpr.Expressions) != 1 {
				t.Fatalf("Expected 1 expression, got %d", len(multiExpr.Expressions))
			}

			expr := multiExpr.Expressions[0]
			if len(expr.Symbols) != tt.symbolCount {
				t.Errorf("Expected %d symbols, got %d", tt.symbolCount, len(expr.Symbols))
			}
		})
	}
}

func TestExpressionParser_FunctionCall(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		symbolCount int
	}{
		{
			name:        "function with argument",
			source:      "{{ len .Items }}",
			symbolCount: 2, // len, .Items
		},
		{
			name:        "function with multiple arguments",
			source:      "{{ printf `%s` .Name }}",
			symbolCount: 3, // printf, "%s", .Name
		},
		{
			name:        "function with no arguments",
			source:      "{{ now }}",
			symbolCount: 1, // now
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := parseStatementFromSource(tt.source)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			multiExpr, ok := stmt.(*MultiExpressionNode)
			if !ok {
				t.Fatalf("Expected MultiExpressionNode, got %T", stmt)
			}

			if len(multiExpr.Expressions) != 1 {
				t.Fatalf("Expected 1 expression, got %d", len(multiExpr.Expressions))
			}

			expr := multiExpr.Expressions[0]
			if len(expr.Symbols) != tt.symbolCount {
				t.Errorf("Expected %d symbols, got %d", tt.symbolCount, len(expr.Symbols))
			}
		})
	}
}

func TestExpressionParser_Parenthesized(t *testing.T) {
	tests := []struct {
		name          string
		source        string
		symbolCount   int
		expandedCount int
		wantError     bool
		errorMatch    string
	}{
		{
			name:          "parenthesized function call",
			source:        "{{ (len .Items) }}",
			symbolCount:   1, // ExpandableGroup token
			expandedCount: 1,
		},
		{
			name:          "method chained after parenthesis",
			source:        "{{ (index .Items 0).Name }}",
			symbolCount:   1,
			expandedCount: 1,
		},
		{
			name:       "unclosed parenthesis",
			source:     "{{ (len .Items }}",
			wantError:  true,
			errorMatch: "missing closing parenthesis",
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

			if len(root.Statements) == 0 {
				t.Fatal("Expected at least one statement")
			}

			multiExpr, ok := root.Statements[0].(*MultiExpressionNode)
			if !ok {
				t.Fatalf("Expected MultiExpressionNode, got %T", root.Statements[0])
			}

			if len(multiExpr.Expressions) < 1 {
				t.Fatal("Expected at least 1 expression")
			}

			expr := multiExpr.Expressions[0]
			if len(expr.Symbols) != tt.symbolCount {
				t.Errorf("Expected %d symbols, got %d", tt.symbolCount, len(expr.Symbols))
			}
		})
	}
}

func TestExpressionParser_Pipe(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		exprCount int
	}{
		{
			name:      "single pipe",
			source:    "{{ .Field | upper }}",
			exprCount: 2, // .Field and upper
		},
		{
			name:      "double pipe",
			source:    "{{ .Field | upper | lower }}",
			exprCount: 3,
		},
		{
			name:      "pipe with function args",
			source:    "{{ .Field | printf `%s` }}",
			exprCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := parseStatementFromSource(tt.source)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			multiExpr, ok := stmt.(*MultiExpressionNode)
			if !ok {
				t.Fatalf("Expected MultiExpressionNode, got %T", stmt)
			}

			if len(multiExpr.Expressions) != tt.exprCount {
				t.Errorf("Expected %d expressions (pipe segments), got %d",
					tt.exprCount, len(multiExpr.Expressions))
			}
		})
	}
}

func TestMultiExpressionParser(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		exprCount int
	}{
		{
			name:      "piped chain",
			source:    "{{ .Field | upper | lower }}",
			exprCount: 3,
		},
		{
			name:      "complex pipeline",
			source:    "{{ .Items | len | printf `%d` }}",
			exprCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := parseStatementFromSource(tt.source)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			multiExpr, ok := stmt.(*MultiExpressionNode)
			if !ok {
				t.Fatalf("Expected MultiExpressionNode, got %T", stmt)
			}

			if len(multiExpr.Expressions) != tt.exprCount {
				t.Errorf(
					"Expected %d expressions, got %d",
					tt.exprCount,
					len(multiExpr.Expressions),
				)
			}
		})
	}
}

func TestDeclarationParser(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		varCount   int
		wantError  bool
		errorMatch string
	}{
		{
			name:     "single variable declaration",
			source:   "{{ $var := .Field }}",
			varCount: 1,
		},
		{
			name:     "two variable declaration",
			source:   "{{ $k, $v := .Field }}",
			varCount: 2,
		},
		{
			name:       "more than two variables",
			source:     "{{ $a, $b, $c := .Field }}",
			wantError:  true,
			errorMatch: "one or two variables",
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

			if len(root.Statements) == 0 {
				t.Fatal("Expected at least one statement")
			}

			varDecl, ok := root.Statements[0].(*VariableDeclarationNode)
			if !ok {
				t.Fatalf("Expected VariableDeclarationNode, got %T", root.Statements[0])
			}

			if len(varDecl.VariableNames) != tt.varCount {
				t.Errorf(
					"Expected %d variable names, got %d",
					tt.varCount,
					len(varDecl.VariableNames),
				)
			}
		})
	}
}

func TestExpressionParser_Literals(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		symbolKind  lexer.Kind
		symbolValue string
	}{
		{
			name:        "string literal",
			source:      "{{ `hello` }}",
			symbolKind:  lexer.StringLit,
			symbolValue: "hello",
		},
		{
			name:        "number literal",
			source:      "{{ 42 }}",
			symbolKind:  lexer.Number,
			symbolValue: "42",
		},
		{
			name:        "decimal literal",
			source:      "{{ 3.14 }}",
			symbolKind:  lexer.Decimal,
			symbolValue: "3.14",
		},
		{
			name:        "boolean true",
			source:      "{{ true }}",
			symbolKind:  lexer.Boolean,
			symbolValue: "true",
		},
		{
			name:        "boolean false",
			source:      "{{ false }}",
			symbolKind:  lexer.Boolean,
			symbolValue: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := parseStatementFromSource(tt.source)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			multiExpr, ok := stmt.(*MultiExpressionNode)
			if !ok {
				t.Fatalf("Expected MultiExpressionNode, got %T", stmt)
			}

			if len(multiExpr.Expressions) != 1 {
				t.Fatalf("Expected 1 expression, got %d", len(multiExpr.Expressions))
			}

			expr := multiExpr.Expressions[0]
			if len(expr.Symbols) != 1 {
				t.Fatalf("Expected 1 symbol, got %d", len(expr.Symbols))
			}

			symbol := expr.Symbols[0]
			if symbol.ID != tt.symbolKind {
				t.Errorf("Expected symbol kind %v, got %v", tt.symbolKind, symbol.ID)
			}

			if string(symbol.Value) != tt.symbolValue {
				t.Errorf(
					"Expected symbol value '%s', got '%s'",
					tt.symbolValue,
					string(symbol.Value),
				)
			}
		})
	}
}

func TestExpressionParser_EmptyExpression(t *testing.T) {
	root, errs := tokenizeAndParse("{{ }}")

	// Should get an error for empty expression
	if len(errs) == 0 {
		t.Error("Expected error for empty expression")
	}

	foundEmptyError := false
	for _, err := range errs {
		if testutil.ContainsSubstring(err.GetError(), "empty") {
			foundEmptyError = true
			break
		}
	}

	if !foundEmptyError {
		t.Errorf("Expected 'empty' error, got: %v", errs)
	}

	_ = root
}

func TestExpressionParser_NestedParentheses(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantError bool
	}{
		{
			name:   "double nested",
			source: "{{ ((len .Items)) }}",
		},
		{
			name:   "nested with chaining",
			source: "{{ ((index .Items 0)).Name }}",
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

			if len(root.Statements) == 0 {
				t.Fatal("Expected at least one statement")
			}
		})
	}
}
