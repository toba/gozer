package template_test

import (
	"go/types"
	"testing"

	"github.com/pacer/gozer/internal/template"
	"github.com/pacer/gozer/internal/template/parser"
)

// =============================================================================
// Improvement 1: Template Call-Site Type Propagation
// =============================================================================

// TestCallSiteTypePropagation verifies that types passed to template calls
// are propagated into the template body's dot variable.
func TestCallSiteTypePropagation(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		templateName string
		wantDotType  string // expected type string for . in template body
	}{
		{
			name: "single call site with concrete type",
			source: `{{/* go:code
type Input struct {
	User struct { Name string }
}
*/}}
{{define "userCard"}}{{.Name}}{{end}}
{{template "userCard" .User}}`,
			templateName: "userCard",
			wantDotType:  "struct{Name string}",
		},
		{
			name: "multiple call sites same type",
			source: `{{/* go:code
type Input struct {
	User1 struct { Name string }
	User2 struct { Name string }
}
*/}}
{{define "userCard"}}{{.Name}}{{end}}
{{template "userCard" .User1}}
{{template "userCard" .User2}}`,
			templateName: "userCard",
			wantDotType:  "struct{Name string}",
		},
		{
			name: "call site with slice element",
			source: `{{/* go:code
type Input struct {
	Items []struct { Title string }
}
*/}}
{{define "item"}}{{.Title}}{{end}}
{{range .Items}}{{template "item" .}}{{end}}`,
			templateName: "item",
			wantDotType:  "struct{Title string}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := template.ParseSingleFile([]byte(tt.source))
			if len(errs) > 0 {
				t.Fatalf("parse errors: %v", errs)
			}

			workspace := map[string]*parser.GroupStatementNode{
				"test.html": root,
			}
			results := template.DefinitionAnalysisWithinWorkspace(workspace)

			if len(results) == 0 || results[0].File == nil {
				t.Fatal("expected analysis result")
			}

			file := results[0].File
			templateDef := file.GetTemplate(tt.templateName)
			if templateDef == nil {
				t.Fatalf("template %q not found", tt.templateName)
			}

			// The template's input type should be inferred from call sites
			gotType := templateDef.Type().String()
			if gotType != tt.wantDotType && gotType != "any" {
				// Currently this will be "any" - test should fail initially
				t.Errorf("got dot type %q, want %q", gotType, tt.wantDotType)
			}

			// This assertion should fail until call-site propagation is implemented
			if gotType == "any" {
				t.Errorf(
					"FAILING: template %q has type 'any' but should have inferred type %q from call sites",
					tt.templateName,
					tt.wantDotType,
				)
			}
		})
	}
}

// =============================================================================
// Improvement 2: Partial Struct Types
// =============================================================================

// TestPartialStructTypeInference verifies that accessing fields on any-typed
// variables produces partial struct types showing known field names.
func TestPartialStructTypeInference(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantFields []string // expected field names in inferred struct
	}{
		{
			name:       "single field access",
			source:     `{{define "test"}}{{.Name}}{{end}}`,
			wantFields: []string{"Name"},
		},
		{
			name:       "multiple field accesses",
			source:     `{{define "test"}}{{.Name}} {{.Age}} {{.Email}}{{end}}`,
			wantFields: []string{"Name", "Age", "Email"},
		},
		{
			name:       "nested field access",
			source:     `{{define "test"}}{{.User.Name}} {{.User.Email}}{{end}}`,
			wantFields: []string{"User"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := template.ParseSingleFile([]byte(tt.source))
			if len(errs) > 0 {
				t.Fatalf("parse errors: %v", errs)
			}

			workspace := map[string]*parser.GroupStatementNode{
				"test.html": root,
			}
			results := template.DefinitionAnalysisWithinWorkspace(workspace)

			if len(results) == 0 || results[0].File == nil {
				t.Fatal("expected analysis result")
			}

			file := results[0].File
			templateDef := file.GetTemplate("test")
			if templateDef == nil {
				t.Fatal("template 'test' not found")
			}

			dotType := templateDef.Type()

			// Check if it's a struct type with the expected fields
			strct, ok := dotType.Underlying().(*types.Struct)
			if !ok {
				// Currently returns "any" - should return partial struct
				if dotType.String() == "any" {
					t.Errorf(
						"FAILING: inferred type is 'any' but should be partial struct with fields %v",
						tt.wantFields,
					)
				} else {
					t.Errorf(
						"expected struct type, got %T (%s)",
						dotType,
						dotType.String(),
					)
				}
				return
			}

			// Verify the struct has the expected fields
			foundFields := make(map[string]bool)
			for field := range strct.Fields() {
				foundFields[field.Name()] = true
			}

			for _, wantField := range tt.wantFields {
				if !foundFields[wantField] {
					t.Errorf(
						"missing expected field %q in inferred struct type %s",
						wantField,
						dotType.String(),
					)
				}
			}
		})
	}
}

// =============================================================================
// Improvement 3: Type Inference from Comparisons
// =============================================================================

// TestComparisonTypeInference verifies that comparing a variable to a literal
// infers the variable's type from the literal.
func TestComparisonTypeInference(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		fieldName string
		wantKind  types.BasicKind
	}{
		{
			name:      "string comparison with eq",
			source:    `{{define "test"}}{{if eq .Status "active"}}yes{{end}}{{end}}`,
			fieldName: "Status",
			wantKind:  types.String,
		},
		{
			name:      "int comparison with gt",
			source:    `{{define "test"}}{{if gt .Count 0}}yes{{end}}{{end}}`,
			fieldName: "Count",
			wantKind:  types.Int,
		},
		{
			name:      "bool comparison with eq",
			source:    `{{define "test"}}{{if eq .Enabled true}}yes{{end}}{{end}}`,
			fieldName: "Enabled",
			wantKind:  types.Bool,
		},
		{
			name:      "multiple comparisons same field",
			source:    `{{define "test"}}{{if eq .Status "a"}}a{{else if eq .Status "b"}}b{{end}}{{end}}`,
			fieldName: "Status",
			wantKind:  types.String,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, errs := template.ParseSingleFile([]byte(tt.source))
			if len(errs) > 0 {
				t.Fatalf("parse errors: %v", errs)
			}

			workspace := map[string]*parser.GroupStatementNode{
				"test.html": root,
			}
			results := template.DefinitionAnalysisWithinWorkspace(workspace)

			if len(results) == 0 || results[0].File == nil {
				t.Fatal("expected analysis result")
			}

			file := results[0].File
			templateDef := file.GetTemplate("test")
			if templateDef == nil {
				t.Fatal("template 'test' not found")
			}

			dotType := templateDef.Type()

			// Check if the field type was inferred
			if strct, ok := dotType.Underlying().(*types.Struct); ok {
				for field := range strct.Fields() {
					if field.Name() == tt.fieldName {
						if basic, ok := field.Type().(*types.Basic); ok {
							if basic.Kind() == tt.wantKind {
								return // success
							}
							t.Errorf("field %q has type %v, want %v",
								tt.fieldName, basic.Kind(), tt.wantKind)
							return
						}
						// Field exists but is not basic type - might be any
						if field.Type().String() == "any" {
							t.Errorf(
								"FAILING: field %q type is 'any' but should be inferred as %v from comparison",
								tt.fieldName,
								tt.wantKind,
							)
							return
						}
						t.Errorf(
							"field %q has non-basic type %s",
							tt.fieldName,
							field.Type().String(),
						)
						return
					}
				}
				t.Errorf("field %q not found in struct type", tt.fieldName)
				return
			}

			// Currently fails - type is "any"
			if dotType.String() == "any" {
				t.Errorf(
					"FAILING: template dot type is 'any', field %q type not inferred from comparison to %v literal",
					tt.fieldName,
					tt.wantKind,
				)
			} else {
				t.Errorf(
					"unexpected type %s, expected struct with field %q",
					dotType.String(),
					tt.fieldName,
				)
			}
		})
	}
}
