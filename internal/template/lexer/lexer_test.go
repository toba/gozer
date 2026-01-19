package lexer

import (
	"testing"

	"github.com/pacer/gozer/internal/template/testutil"
)

func TestTokenize_EmptyInput(t *testing.T) {
	streams, errs := Tokenize([]byte(""))

	if len(streams) != 0 {
		t.Errorf("Expected 0 streams for empty input, got %d", len(streams))
	}

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors for empty input, got %d", len(errs))
	}
}

func TestTokenize_NoTemplates(t *testing.T) {
	streams, errs := Tokenize([]byte("just plain text without templates"))

	if len(streams) != 0 {
		t.Errorf("Expected 0 streams for text without templates, got %d", len(streams))
	}

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errs))
	}
}

func TestTokenize_SingleTemplate(t *testing.T) {
	streams, errs := Tokenize([]byte("{{ .Field }}"))

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errs), errs)
	}

	if len(streams) != 1 {
		t.Fatalf("Expected 1 stream, got %d", len(streams))
	}

	stream := streams[0]
	if stream.IsEmpty() {
		t.Error("Stream should not be empty")
	}

	// Should have tokens: .Field, EOL
	if len(stream.Tokens) < 2 {
		t.Errorf("Expected at least 2 tokens, got %d", len(stream.Tokens))
	}

	// Last token should be EOL
	lastToken := stream.Tokens[len(stream.Tokens)-1]
	if lastToken.ID != Eol {
		t.Errorf("Expected last token to be Eol, got %v", lastToken.ID)
	}
}

func TestTokenize_MultipleTemplates(t *testing.T) {
	source := "Hello {{ .Name }}, you have {{ .Count }} items"
	streams, errs := Tokenize([]byte(source))

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errs), errs)
	}

	if len(streams) != 2 {
		t.Errorf("Expected 2 streams, got %d", len(streams))
	}
}

func TestTokenize_NestedDelimiters(t *testing.T) {
	// Template with curly braces inside (not nested templates)
	source := `{{ printf "%s" "{hello}" }}`
	streams, errs := Tokenize([]byte(source))

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errs), errs)
	}

	if len(streams) != 1 {
		t.Errorf("Expected 1 stream, got %d", len(streams))
	}
}

func TestTokenizeLine_Keywords(t *testing.T) {
	keywords := []struct {
		keyword string
	}{
		{"if"},
		{"else"},
		{"end"},
		{"range"},
		{"define"},
		{"template"},
		{"block"},
		{"with"},
		{"break"},
		{"continue"},
	}

	for _, kw := range keywords {
		t.Run(kw.keyword, func(t *testing.T) {
			source := "{{ " + kw.keyword + " }}"
			streams, errs := Tokenize([]byte(source))

			// Some keywords like "if" expect a condition and will error
			// but should still produce the keyword token
			_ = errs

			if len(streams) != 1 {
				t.Fatalf("Expected 1 stream, got %d", len(streams))
			}

			// First token (before EOL) should be the keyword
			if len(streams[0].Tokens) < 2 {
				t.Fatalf("Expected at least 2 tokens, got %d", len(streams[0].Tokens))
			}

			token := streams[0].Tokens[0]
			if token.ID != Keyword {
				t.Errorf("Expected Keyword token, got %v", token.ID)
			}

			if string(token.Value) != kw.keyword {
				t.Errorf(
					"Expected keyword '%s', got '%s'",
					kw.keyword,
					string(token.Value),
				)
			}
		})
	}
}

func TestTokenizeLine_Variables(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		tokenID  Kind
		tokenVal string
	}{
		{
			name:     "dollar variable",
			source:   "{{ $var }}",
			tokenID:  DollarVariable,
			tokenVal: "$var",
		},
		{
			name:     "dot variable",
			source:   "{{ .field }}",
			tokenID:  DotVariable,
			tokenVal: ".field",
		},
		{
			name:     "root dollar",
			source:   "{{ $ }}",
			tokenID:  DollarVariable,
			tokenVal: "$",
		},
		{
			name:     "root dot",
			source:   "{{ . }}",
			tokenID:  DotVariable,
			tokenVal: ".",
		},
		{
			name:     "chained dot variable",
			source:   "{{ .Field.SubField }}",
			tokenID:  DotVariable,
			tokenVal: ".Field.SubField",
		},
		{
			name:     "dollar with chained access",
			source:   "{{ $.Field }}",
			tokenID:  DollarVariable,
			tokenVal: "$.Field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streams, errs := Tokenize([]byte(tt.source))

			if len(errs) != 0 {
				t.Errorf("Expected 0 errors, got %d: %v", len(errs), errs)
			}

			if len(streams) != 1 {
				t.Fatalf("Expected 1 stream, got %d", len(streams))
			}

			token := streams[0].Tokens[0]
			if token.ID != tt.tokenID {
				t.Errorf("Expected token ID %v, got %v", tt.tokenID, token.ID)
			}

			if string(token.Value) != tt.tokenVal {
				t.Errorf(
					"Expected token value '%s', got '%s'",
					tt.tokenVal,
					string(token.Value),
				)
			}
		})
	}
}

func TestTokenizeLine_Literals(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		tokenID  Kind
		tokenVal string
	}{
		{
			name:     "double quoted string",
			source:   `{{ "hello" }}`,
			tokenID:  StringLit,
			tokenVal: "hello",
		},
		{
			name:     "backtick string",
			source:   "{{ `hello` }}",
			tokenID:  StringLit,
			tokenVal: "hello",
		},
		{
			name:     "integer",
			source:   "{{ 42 }}",
			tokenID:  Number,
			tokenVal: "42",
		},
		{
			name:     "decimal",
			source:   "{{ 3.14 }}",
			tokenID:  Decimal,
			tokenVal: "3.14",
		},
		{
			name:     "complex number",
			source:   "{{ 1i }}",
			tokenID:  ComplexNumber,
			tokenVal: "1i",
		},
		{
			name:     "boolean true",
			source:   "{{ true }}",
			tokenID:  Boolean,
			tokenVal: "true",
		},
		{
			name:     "boolean false",
			source:   "{{ false }}",
			tokenID:  Boolean,
			tokenVal: "false",
		},
		{
			name:     "character literal",
			source:   "{{ 'a' }}",
			tokenID:  Character,
			tokenVal: "'a'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streams, errs := Tokenize([]byte(tt.source))

			if len(errs) != 0 {
				t.Errorf("Expected 0 errors, got %d: %v", len(errs), errs)
			}

			if len(streams) != 1 {
				t.Fatalf("Expected 1 stream, got %d", len(streams))
			}

			token := streams[0].Tokens[0]
			if token.ID != tt.tokenID {
				t.Errorf("Expected token ID %v, got %v", tt.tokenID, token.ID)
			}

			if string(token.Value) != tt.tokenVal {
				t.Errorf(
					"Expected token value '%s', got '%s'",
					tt.tokenVal,
					string(token.Value),
				)
			}
		})
	}
}

func TestTokenizeLine_Operators(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		tokenID  Kind
		tokenVal string
		index    int // which token to check
	}{
		{
			name:     "assignment",
			source:   "{{ $var = .Field }}",
			tokenID:  Assignment,
			tokenVal: "=",
			index:    1,
		},
		{
			name:     "declaration assignment",
			source:   "{{ $var := .Field }}",
			tokenID:  DeclarationAssignment,
			tokenVal: ":=",
			index:    1,
		},
		{
			name:     "equal comparison",
			source:   "{{ if .A == .B }}",
			tokenID:  EqualComparison,
			tokenVal: "==",
			index:    2, // if, .A, ==
		},
		{
			name:     "pipe",
			source:   "{{ .Field | upper }}",
			tokenID:  Pipe,
			tokenVal: "|",
			index:    1,
		},
		{
			name:     "comma",
			source:   "{{ $a, $b := .Field }}",
			tokenID:  Comma,
			tokenVal: ",",
			index:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streams, errs := Tokenize([]byte(tt.source))

			// Some tests may produce errors but should still have the operator
			_ = errs

			if len(streams) != 1 {
				t.Fatalf("Expected 1 stream, got %d", len(streams))
			}

			if len(streams[0].Tokens) <= tt.index {
				t.Fatalf(
					"Expected at least %d tokens, got %d",
					tt.index+1,
					len(streams[0].Tokens),
				)
			}

			token := streams[0].Tokens[tt.index]
			if token.ID != tt.tokenID {
				t.Errorf("Expected token ID %v, got %v", tt.tokenID, token.ID)
			}

			if string(token.Value) != tt.tokenVal {
				t.Errorf(
					"Expected token value '%s', got '%s'",
					tt.tokenVal,
					string(token.Value),
				)
			}
		})
	}
}

func TestTokenizeLine_Parentheses(t *testing.T) {
	source := "{{ (len .Items) }}"
	streams, errs := Tokenize([]byte(source))

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errs), errs)
	}

	if len(streams) != 1 {
		t.Fatalf("Expected 1 stream, got %d", len(streams))
	}

	// Should have: (, len, .Items, ), EOL
	tokens := streams[0].Tokens
	if len(tokens) < 5 {
		t.Fatalf("Expected at least 5 tokens, got %d", len(tokens))
	}

	if tokens[0].ID != LeftParen {
		t.Errorf("Expected first token to be LeftParen, got %v", tokens[0].ID)
	}

	if tokens[3].ID != RightParen {
		t.Errorf("Expected fourth token to be RightParen, got %v", tokens[3].ID)
	}
}

func TestTokenizeLine_Comment(t *testing.T) {
	// Comments need to have the comment syntax directly after {{ with no space
	// e.g. {{/* comment */}} not {{ /* comment */ }}
	source := "{{/* this is a comment */}}"
	streams, errs := Tokenize([]byte(source))

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errs), errs)
	}

	if len(streams) != 1 {
		t.Fatalf("Expected 1 stream, got %d", len(streams))
	}

	token := streams[0].Tokens[0]
	if token.ID != Comment {
		t.Errorf("Expected Comment token, got %v", token.ID)
	}

	// Comment value should have /* */ stripped
	if string(token.Value) != " this is a comment " {
		t.Errorf(
			"Expected comment value ' this is a comment ', got '%s'",
			string(token.Value),
		)
	}
}

func TestTokenizeLine_Errors_UnclosedParenthesis(t *testing.T) {
	source := "{{ (len .Items }}"
	streams, errs := Tokenize([]byte(source))

	if len(errs) == 0 {
		t.Error("Expected error for unclosed parenthesis")
	}

	foundError := false
	for _, err := range errs {
		if testutil.ContainsSubstring(err.GetError(), "parenthesis") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Errorf("Expected parenthesis error, got: %v", errs)
	}

	// Should still produce tokens
	if len(streams) != 1 {
		t.Errorf("Expected 1 stream even with error, got %d", len(streams))
	}
}

func TestTokenizeLine_Errors_ExtraClosingParenthesis(t *testing.T) {
	source := "{{ len .Items) }}"
	streams, errs := Tokenize([]byte(source))

	if len(errs) == 0 {
		t.Error("Expected error for extra closing parenthesis")
	}

	foundError := false
	for _, err := range errs {
		if testutil.ContainsSubstring(err.GetError(), "parenthesis") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Errorf("Expected parenthesis error, got: %v", errs)
	}

	_ = streams
}

func TestTokenizeLine_Errors_UnrecognizedCharacters(t *testing.T) {
	source := "{{ @invalid }}"
	streams, errs := Tokenize([]byte(source))

	if len(errs) == 0 {
		t.Error("Expected error for unrecognized character")
	}

	foundError := false
	for _, err := range errs {
		if testutil.ContainsSubstring(err.GetError(), "not recognized") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Errorf("Expected 'not recognized' error, got: %v", errs)
	}

	_ = streams
}

func TestTokenizeLine_Errors_EmptyTemplate(t *testing.T) {
	source := "{{ }}"
	streams, errs := Tokenize([]byte(source))

	if len(errs) == 0 {
		t.Error("Expected error for empty template")
	}

	foundError := false
	for _, err := range errs {
		if testutil.ContainsSubstring(err.GetError(), "empty") {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Errorf("Expected 'empty' error, got: %v", errs)
	}

	// Stream should exist but be empty (only EOL)
	if len(streams) != 1 {
		t.Errorf("Expected 1 stream, got %d", len(streams))
	}
}

func TestWhiteSpaceTrimmer_LeftTrim(t *testing.T) {
	source := "{{- .Field }}"
	streams, errs := Tokenize([]byte(source))

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errs), errs)
	}

	if len(streams) != 1 {
		t.Fatalf("Expected 1 stream, got %d", len(streams))
	}

	// First token should be the field, not a trim marker
	token := streams[0].Tokens[0]
	if token.ID != DotVariable {
		t.Errorf("Expected DotVariable token, got %v", token.ID)
	}
}

func TestWhiteSpaceTrimmer_RightTrim(t *testing.T) {
	source := "{{ .Field -}}"
	streams, errs := Tokenize([]byte(source))

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errs), errs)
	}

	if len(streams) != 1 {
		t.Fatalf("Expected 1 stream, got %d", len(streams))
	}

	// Should have .Field and EOL only
	if len(streams[0].Tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(streams[0].Tokens))
	}
}

func TestWhiteSpaceTrimmer_BothTrim(t *testing.T) {
	source := "{{- .Field -}}"
	streams, errs := Tokenize([]byte(source))

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errs), errs)
	}

	if len(streams) != 1 {
		t.Fatalf("Expected 1 stream, got %d", len(streams))
	}
}

func TestWhiteSpaceTrimmer_InvalidNoSpace(t *testing.T) {
	source := "{{-.Field}}"
	streams, errs := Tokenize([]byte(source))

	if len(errs) == 0 {
		t.Error("Expected error for invalid trim marker (no space)")
	}

	_ = streams
}

func TestTokenize_FunctionCall(t *testing.T) {
	source := "{{ len .Items }}"
	streams, errs := Tokenize([]byte(source))

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(errs), errs)
	}

	if len(streams) != 1 {
		t.Fatalf("Expected 1 stream, got %d", len(streams))
	}

	tokens := streams[0].Tokens
	if len(tokens) < 3 {
		t.Fatalf("Expected at least 3 tokens, got %d", len(tokens))
	}

	// First token should be function
	if tokens[0].ID != Function {
		t.Errorf("Expected Function token, got %v", tokens[0].ID)
	}

	if string(tokens[0].Value) != "len" {
		t.Errorf("Expected function name 'len', got '%s'", string(tokens[0].Value))
	}
}

func TestTokenize_StreamPosition(t *testing.T) {
	source := "text {{ .Field }} more"
	streams, _ := Tokenize([]byte(source))

	if len(streams) != 1 {
		t.Fatalf("Expected 1 stream, got %d", len(streams))
	}

	// Token should have position information
	token := streams[0].Tokens[0]
	if token.Range.Start.Line != 0 {
		t.Errorf("Expected line 0, got %d", token.Range.Start.Line)
	}

	// Character position is after "text {{ " which is 8 characters (0-indexed position 8)
	// The template content starts at character 7, and .Field starts at position 8
	if token.Range.Start.Character != 8 {
		t.Errorf("Expected character 8, got %d", token.Range.Start.Character)
	}
}

func TestTokenize_MultilineTemplate(t *testing.T) {
	source := `{{ if .Condition }}
content
{{ end }}`
	streams, errs := Tokenize([]byte(source))

	// Should have 2 streams (if and end)
	if len(errs) != 0 {
		// May have unclosed scope error from lexer perspective
		_ = errs
	}

	if len(streams) != 2 {
		t.Errorf("Expected 2 streams, got %d", len(streams))
	}
}

func TestStreamToken_String(t *testing.T) {
	source := "{{ .Field | upper }}"
	streams, _ := Tokenize([]byte(source))

	if len(streams) != 1 {
		t.Fatalf("Expected 1 stream, got %d", len(streams))
	}

	str := streams[0].String()
	if str == "" {
		t.Error("Stream String() should not be empty")
	}

	// Should contain the field and function
	if !testutil.ContainsSubstring(str, ".Field") {
		t.Errorf("Expected String() to contain '.Field', got '%s'", str)
	}
}

func TestStreamToken_IsEmpty(t *testing.T) {
	// A non-empty stream
	source := "{{ .Field }}"
	streams, _ := Tokenize([]byte(source))

	if len(streams) != 1 {
		t.Fatalf("Expected 1 stream, got %d", len(streams))
	}

	if streams[0].IsEmpty() {
		t.Error("Stream with .Field should not be empty")
	}
}

func TestRange_Contains(t *testing.T) {
	r := Range{
		Start: Position{Line: 1, Character: 5},
		End:   Position{Line: 1, Character: 10},
	}

	tests := []struct {
		name     string
		pos      Position
		expected bool
	}{
		{"before start", Position{Line: 1, Character: 4}, false},
		{"at start", Position{Line: 1, Character: 5}, true},
		{"in middle", Position{Line: 1, Character: 7}, true},
		{"at end (exclusive)", Position{Line: 1, Character: 10}, false},
		{"after end", Position{Line: 1, Character: 11}, false},
		{"different line before", Position{Line: 0, Character: 7}, false},
		{"different line after", Position{Line: 2, Character: 7}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.Contains(tt.pos)
			if result != tt.expected {
				t.Errorf("Range.Contains(%v) = %v, want %v", tt.pos, result, tt.expected)
			}
		})
	}
}

func TestRange_IsEmpty(t *testing.T) {
	emptyRange := Range{}
	if !emptyRange.IsEmpty() {
		t.Error("Zero-value Range should be empty")
	}

	nonEmptyRange := Range{
		Start: Position{Line: 1, Character: 0},
		End:   Position{Line: 1, Character: 5},
	}
	if nonEmptyRange.IsEmpty() {
		t.Error("Non-zero Range should not be empty")
	}
}

func TestPosition_Offset(t *testing.T) {
	pos := Position{Line: 5, Character: 10}
	newPos := pos.Offset(3)

	if newPos.Line != 5 {
		t.Errorf("Expected line 5, got %d", newPos.Line)
	}

	if newPos.Character != 13 {
		t.Errorf("Expected character 13, got %d", newPos.Character)
	}
}

func TestRange_AdjustStart(t *testing.T) {
	r := Range{
		Start: Position{Line: 1, Character: 5},
		End:   Position{Line: 1, Character: 10},
	}

	adjusted := r.AdjustStart(2)
	if adjusted.Start.Character != 7 {
		t.Errorf("Expected start character 7, got %d", adjusted.Start.Character)
	}

	if adjusted.End.Character != 10 {
		t.Errorf("End should remain unchanged, got %d", adjusted.End.Character)
	}
}

func TestRange_Shrink(t *testing.T) {
	r := Range{
		Start: Position{Line: 1, Character: 5},
		End:   Position{Line: 1, Character: 15},
	}

	shrunk := r.Shrink(2, 3)
	if shrunk.Start.Character != 7 {
		t.Errorf("Expected start character 7, got %d", shrunk.Start.Character)
	}

	if shrunk.End.Character != 12 {
		t.Errorf("Expected end character 12, got %d", shrunk.End.Character)
	}
}
