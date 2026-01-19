package parser

import (
	"testing"

	"github.com/pacer/gozer/internal/template/testutil"
)

func TestGroupMerger_IfEndNesting(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantError bool
	}{
		{
			name:   "simple if/end",
			source: "{{ if .Cond }}content{{ end }}",
		},
		{
			name:   "nested if/end",
			source: "{{ if .Outer }}{{ if .Inner }}inner{{ end }}outer{{ end }}",
		},
		{
			name:   "deeply nested if/end",
			source: "{{ if .A }}{{ if .B }}{{ if .C }}c{{ end }}b{{ end }}a{{ end }}",
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

			// Verify structure
			if root == nil {
				t.Fatal("Expected non-nil root")
			}
		})
	}
}

func TestGroupMerger_IfElseEndLinkedSiblings(t *testing.T) {
	source := "{{ if .Cond }}true{{ else }}false{{ end }}"
	root, errs := tokenizeAndParse(source)

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got: %v", errs)
	}

	// Should have: if, else, end in root.Statements
	if len(root.Statements) != 3 {
		t.Fatalf("Expected 3 statements, got %d", len(root.Statements))
	}

	ifStmt, ok := root.Statements[0].(*GroupStatementNode)
	if !ok {
		t.Fatalf("Expected GroupStatementNode for if, got %T", root.Statements[0])
	}

	if ifStmt.Kind() != KindIf {
		t.Errorf("Expected KindIf, got %v", ifStmt.Kind())
	}

	elseStmt, ok := root.Statements[1].(*GroupStatementNode)
	if !ok {
		t.Fatalf("Expected GroupStatementNode for else, got %T", root.Statements[1])
	}

	if elseStmt.Kind() != KindElse {
		t.Errorf("Expected KindElse, got %v", elseStmt.Kind())
	}

	// Check linked siblings
	if ifStmt.NextLinkedSibling == nil {
		t.Error("Expected if to have NextLinkedSibling")
	} else if ifStmt.NextLinkedSibling != elseStmt {
		t.Error("Expected if.NextLinkedSibling to be else statement")
	}
}

func TestGroupMerger_RangeElseEndWithBreak(t *testing.T) {
	source := "{{ range .Items }}{{ break }}{{ else }}no items{{ end }}"
	root, errs := tokenizeAndParse(source)

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got: %v", errs)
	}

	// Should have: range, else, end in root.Statements
	if len(root.Statements) != 3 {
		t.Fatalf("Expected 3 statements, got %d", len(root.Statements))
	}

	rangeStmt, ok := root.Statements[0].(*GroupStatementNode)
	if !ok {
		t.Fatalf("Expected GroupStatementNode for range, got %T", root.Statements[0])
	}

	if rangeStmt.Kind() != KindRangeLoop {
		t.Errorf("Expected KindRangeLoop, got %v", rangeStmt.Kind())
	}

	// Check that break is inside range
	if len(rangeStmt.Statements) != 1 {
		t.Fatalf(
			"Expected 1 statement inside range (break), got %d",
			len(rangeStmt.Statements),
		)
	}

	breakStmt, ok := rangeStmt.Statements[0].(*SpecialCommandNode)
	if !ok {
		t.Fatalf("Expected SpecialCommandNode for break, got %T", rangeStmt.Statements[0])
	}

	if breakStmt.Kind() != KindBreak {
		t.Errorf("Expected KindBreak, got %v", breakStmt.Kind())
	}

	// Break should have Target pointing to range
	if breakStmt.Target != rangeStmt {
		t.Error("Expected break.Target to point to range statement")
	}
}

func TestGroupMerger_NestedStructures(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "if inside range",
			source: "{{ range .Items }}{{ if .Active }}active{{ end }}{{ end }}",
		},
		{
			name:   "range inside if",
			source: "{{ if .HasItems }}{{ range .Items }}item{{ end }}{{ end }}",
		},
		{
			name:   "with inside range",
			source: "{{ range .Items }}{{ with .Details }}detail{{ end }}{{ end }}",
		},
		{
			name:   "complex nesting",
			source: "{{ if .A }}{{ range .B }}{{ with .C }}{{ if .D }}d{{ end }}c{{ end }}b{{ end }}a{{ end }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := tokenizeAndParse(tt.source)

			if len(errs) != 0 {
				t.Errorf("Expected no errors, got: %v", errs)
			}

			if root == nil {
				t.Fatal("Expected non-nil root")
			}
		})
	}
}

func TestGroupMerger_ExtraneousEnd(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		errorMatch string
	}{
		{
			name:       "single extraneous end",
			source:     "{{ end }}",
			errorMatch: "extraneous",
		},
		{
			name:       "extra end after if/end",
			source:     "{{ if .Cond }}content{{ end }}{{ end }}",
			errorMatch: "extraneous",
		},
		{
			name:       "double extraneous end",
			source:     "{{ end }}{{ end }}",
			errorMatch: "extraneous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := tokenizeAndParse(tt.source)

			if len(errs) == 0 {
				t.Error("Expected error but got none")
			}

			foundMatch := false
			for _, err := range errs {
				if testutil.ContainsSubstring(err.GetError(), tt.errorMatch) {
					foundMatch = true
					break
				}
			}

			if !foundMatch {
				t.Errorf("Expected error containing '%s', got: %v", tt.errorMatch, errs)
			}
		})
	}
}

func TestGroupMerger_MismatchedElse(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		errorMatch string
	}{
		{
			name:       "else if after range",
			source:     "{{ range .Items }}{{ else if .Other }}{{ end }}",
			errorMatch: "not compatible",
		},
		{
			name:       "else with after if",
			source:     "{{ if .Cond }}{{ else with .Other }}{{ end }}",
			errorMatch: "not compatible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := tokenizeAndParse(tt.source)

			if len(errs) == 0 {
				t.Error("Expected error but got none")
			}

			foundMatch := false
			for _, err := range errs {
				if testutil.ContainsSubstring(err.GetError(), tt.errorMatch) {
					foundMatch = true
					break
				}
			}

			if !foundMatch {
				t.Errorf("Expected error containing '%s', got: %v", tt.errorMatch, errs)
			}
		})
	}
}

func TestGroupMerger_ParentChildRelationships(t *testing.T) {
	source := "{{ if .Outer }}{{ if .Inner }}content{{ end }}{{ end }}"
	root, errs := tokenizeAndParse(source)

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got: %v", errs)
	}

	// Root should have the outer if and its end
	if len(root.Statements) < 1 {
		t.Fatal("Expected at least 1 statement in root")
	}

	outerIf, ok := root.Statements[0].(*GroupStatementNode)
	if !ok {
		t.Fatalf("Expected GroupStatementNode, got %T", root.Statements[0])
	}

	if outerIf.Parent() != root {
		t.Error("Outer if should have root as parent")
	}

	// Outer if should contain inner if
	if len(outerIf.Statements) < 1 {
		t.Fatal("Expected at least 1 statement in outer if")
	}

	innerIf, ok := outerIf.Statements[0].(*GroupStatementNode)
	if !ok {
		t.Fatalf("Expected GroupStatementNode, got %T", outerIf.Statements[0])
	}

	if innerIf.Parent() != outerIf {
		t.Error("Inner if should have outer if as parent")
	}
}

func TestGroupMerger_TemplateDefinition(t *testing.T) {
	source := `{{ define "mytemplate" }}content{{ end }}`
	root, errs := tokenizeAndParse(source)

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got: %v", errs)
	}

	// Should have define and end
	if len(root.Statements) != 2 {
		t.Fatalf("Expected 2 statements, got %d", len(root.Statements))
	}

	defineStmt, ok := root.Statements[0].(*GroupStatementNode)
	if !ok {
		t.Fatalf("Expected GroupStatementNode for define, got %T", root.Statements[0])
	}

	if defineStmt.Kind() != KindDefineTemplate {
		t.Errorf("Expected KindDefineTemplate, got %v", defineStmt.Kind())
	}

	// Check that template shortcut is populated
	if len(root.ShortCut.TemplateDefined) != 1 {
		t.Errorf(
			"Expected 1 template in TemplateDefined, got %d",
			len(root.ShortCut.TemplateDefined),
		)
	}

	if _, exists := root.ShortCut.TemplateDefined["mytemplate"]; !exists {
		t.Error("Expected 'mytemplate' in TemplateDefined")
	}
}

func TestGroupMerger_DuplicateTemplateDefinition(t *testing.T) {
	source := `{{ define "dup" }}first{{ end }}{{ define "dup" }}second{{ end }}`
	_, errs := tokenizeAndParse(source)

	if len(errs) == 0 {
		t.Error("Expected error for duplicate template definition")
	}

	foundDupError := false
	for _, err := range errs {
		if testutil.ContainsSubstring(err.GetError(), "already defined") {
			foundDupError = true
			break
		}
	}

	if !foundDupError {
		t.Errorf("Expected 'already defined' error, got: %v", errs)
	}
}

func TestGroupMerger_VariableDeclarationShortcut(t *testing.T) {
	source := "{{ $var := .Field }}"
	root, errs := tokenizeAndParse(source)

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got: %v", errs)
	}

	// Check that variable declaration is tracked in shortcut
	if len(root.ShortCut.VariableDeclarations) != 1 {
		t.Errorf("Expected 1 variable declaration in shortcut, got %d",
			len(root.ShortCut.VariableDeclarations))
	}

	if _, exists := root.ShortCut.VariableDeclarations["$var"]; !exists {
		t.Error("Expected '$var' in VariableDeclarations")
	}
}

func TestGroupMerger_TemplateCallShortcut(t *testing.T) {
	source := `{{ template "mytemplate" }}`
	root, errs := tokenizeAndParse(source)

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got: %v", errs)
	}

	// Check that template call is tracked in shortcut
	if len(root.ShortCut.TemplateCallUsed) != 1 {
		t.Errorf("Expected 1 template call in shortcut, got %d",
			len(root.ShortCut.TemplateCallUsed))
	}
}

func TestGroupMerger_ElseIfChain(t *testing.T) {
	source := "{{ if .A }}a{{ else if .B }}b{{ else if .C }}c{{ else }}d{{ end }}"
	root, errs := tokenizeAndParse(source)

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got: %v", errs)
	}

	// Should have: if, else if, else if, else, end
	if len(root.Statements) != 5 {
		t.Fatalf("Expected 5 statements, got %d", len(root.Statements))
	}

	expectedKinds := []Kind{KindIf, KindElseIf, KindElseIf, KindElse, KindEnd}

	for i, expectedKind := range expectedKinds {
		stmt, ok := root.Statements[i].(*GroupStatementNode)
		if !ok {
			t.Fatalf(
				"Statement %d: expected GroupStatementNode, got %T",
				i,
				root.Statements[i],
			)
		}
		if stmt.Kind() != expectedKind {
			t.Errorf(
				"Statement %d: expected kind %v, got %v",
				i,
				expectedKind,
				stmt.Kind(),
			)
		}
	}
}

func TestGroupMerger_WithElseWithChain(t *testing.T) {
	source := "{{ with .A }}a{{ else with .B }}b{{ else }}c{{ end }}"
	root, errs := tokenizeAndParse(source)

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got: %v", errs)
	}

	// Should have: with, else with, else, end
	if len(root.Statements) != 4 {
		t.Fatalf("Expected 4 statements, got %d", len(root.Statements))
	}

	expectedKinds := []Kind{KindWith, KindElseWith, KindElse, KindEnd}

	for i, expectedKind := range expectedKinds {
		stmt, ok := root.Statements[i].(*GroupStatementNode)
		if !ok {
			t.Fatalf(
				"Statement %d: expected GroupStatementNode, got %T",
				i,
				root.Statements[i],
			)
		}
		if stmt.Kind() != expectedKind {
			t.Errorf(
				"Statement %d: expected kind %v, got %v",
				i,
				expectedKind,
				stmt.Kind(),
			)
		}
	}
}

func TestGroupMerger_BlockTemplate(t *testing.T) {
	source := `{{ block "myblock" . }}default content{{ end }}`
	root, errs := tokenizeAndParse(source)

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got: %v", errs)
	}

	if len(root.Statements) != 2 {
		t.Fatalf("Expected 2 statements, got %d", len(root.Statements))
	}

	blockStmt, ok := root.Statements[0].(*GroupStatementNode)
	if !ok {
		t.Fatalf("Expected GroupStatementNode for block, got %T", root.Statements[0])
	}

	if blockStmt.Kind() != KindBlockTemplate {
		t.Errorf("Expected KindBlockTemplate, got %v", blockStmt.Kind())
	}

	// Block should have ControlFlow of type TemplateStatementNode
	tmplStmt, ok := blockStmt.ControlFlow.(*TemplateStatementNode)
	if !ok {
		t.Fatalf(
			"Expected TemplateStatementNode in ControlFlow, got %T",
			blockStmt.ControlFlow,
		)
	}

	if string(tmplStmt.TemplateName.Value) != "myblock" {
		t.Errorf(
			"Expected template name 'myblock', got '%s'",
			string(tmplStmt.TemplateName.Value),
		)
	}
}

func TestGroupMerger_ContinueInNestedRange(t *testing.T) {
	source := "{{ range .Outer }}{{ range .Inner }}{{ continue }}{{ end }}{{ end }}"
	root, errs := tokenizeAndParse(source)

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got: %v", errs)
	}

	// Navigate to the continue statement
	outerRange, ok := root.Statements[0].(*GroupStatementNode)
	if !ok || outerRange.Kind() != KindRangeLoop {
		t.Fatal("Expected outer range")
	}

	innerRange, ok := outerRange.Statements[0].(*GroupStatementNode)
	if !ok || innerRange.Kind() != KindRangeLoop {
		t.Fatal("Expected inner range")
	}

	continueStmt, ok := innerRange.Statements[0].(*SpecialCommandNode)
	if !ok {
		t.Fatalf("Expected SpecialCommandNode, got %T", innerRange.Statements[0])
	}

	if continueStmt.Kind() != KindContinue {
		t.Errorf("Expected KindContinue, got %v", continueStmt.Kind())
	}

	// Continue should target the inner range (nearest enclosing range)
	if continueStmt.Target != innerRange {
		t.Error("Continue should target the innermost range")
	}
}

func TestGroupMerger_UnclosedScopes(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		errorCount int
	}{
		{
			name:       "single unclosed if",
			source:     "{{ if .Cond }}content",
			errorCount: 1,
		},
		{
			name:       "nested unclosed",
			source:     "{{ if .A }}{{ if .B }}",
			errorCount: 2,
		},
		{
			name:       "unclosed range",
			source:     "{{ range .Items }}item",
			errorCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := tokenizeAndParse(tt.source)

			if len(errs) < tt.errorCount {
				t.Errorf("Expected at least %d errors, got %d: %v",
					tt.errorCount, len(errs), errs)
			}
		})
	}
}
