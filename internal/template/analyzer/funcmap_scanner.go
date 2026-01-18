package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

// ScanWorkspaceForFuncMap finds all template.FuncMap definitions in Go files
// and returns a map of function names to FunctionDefinitions.
// This allows the analyzer to recognize custom template functions and avoid
// false "function undefined" errors.
func ScanWorkspaceForFuncMap(rootPath string) (map[string]*FunctionDefinition, error) {
	customFunctions := make(map[string]*FunctionDefinition)

	walkErr := filepath.Walk(
		rootPath,
		func(path string, info os.FileInfo, err error) error {
			// Skip files we can't access
			if err != nil || info == nil {
				return err
			}

			// Skip directories we don't want to traverse
			if info.IsDir() {
				name := info.Name()
				// Skip hidden directories, vendor, node_modules, etc.
				if strings.HasPrefix(name, ".") || name == "vendor" ||
					name == "node_modules" ||
					name == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}

			// Only process .go files, skip test files
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			funcs, _ := scanFileForFuncMap(path)
			maps.Copy(customFunctions, funcs)

			return nil
		},
	)

	if walkErr != nil {
		return customFunctions, walkErr
	}

	return customFunctions, nil
}

// scanFileForFuncMap parses a single Go file and extracts template.FuncMap function names.
func scanFileForFuncMap(filePath string) (map[string]*FunctionDefinition, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	customFunctions := make(map[string]*FunctionDefinition)

	// Track imports to identify template.FuncMap
	templateImportAlias := findTemplateImportAlias(file)
	if templateImportAlias == "" {
		// No template import found, nothing to do
		return customFunctions, nil
	}

	// Walk the AST looking for FuncMap patterns
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CompositeLit:
			// Pattern 1: template.FuncMap{ "name": func... }
			if isFuncMapType(node.Type, templateImportAlias) {
				extractFuncMapKeys(node, customFunctions, filePath)
			}

		case *ast.AssignStmt:
			// Pattern 2: funcs["name"] = func...
			// where funcs is of type template.FuncMap
			for i, lhs := range node.Lhs {
				if indexExpr, ok := lhs.(*ast.IndexExpr); ok {
					if key := extractStringLiteral(indexExpr.Index); key != "" {
						// We can't easily determine if this is a FuncMap assignment
						// without type information, but if the key is a string literal
						// in an index expression, it's likely a FuncMap pattern.
						// We'll be conservative and only add it if we see the pattern.
						if i < len(node.Rhs) {
							// Check if rhs is a function
							if isFunctionValue(node.Rhs[i]) {
								addCustomFunction(customFunctions, key, filePath)
							}
						}
					}
				}
			}

		case *ast.KeyValueExpr:
			// This is handled within CompositeLit, but we include it for completeness
			// when walking nested structures
		}
		return true
	})

	return customFunctions, nil
}

// findTemplateImportAlias returns the alias used for text/template or html/template.
// Returns empty string if neither is imported.
func findTemplateImportAlias(file *ast.File) string {
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if path == "text/template" || path == "html/template" {
			if imp.Name != nil {
				return imp.Name.Name
			}
			// Default alias is "template"
			return "template"
		}
	}
	return ""
}

// isFuncMapType checks if the given expression represents template.FuncMap.
func isFuncMapType(expr ast.Expr, templateAlias string) bool {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		// template.FuncMap
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == templateAlias && t.Sel.Name == "FuncMap"
		}
	case *ast.Ident:
		// FuncMap (if imported with dot import or type aliased)
		return t.Name == "FuncMap"
	case *ast.MapType:
		// map[string]any or map[string]interface{}
		// This is the underlying type of FuncMap
		if key, ok := t.Key.(*ast.Ident); ok && key.Name == "string" {
			return true
		}
	}
	return false
}

// extractFuncMapKeys extracts function names from a FuncMap composite literal.
func extractFuncMapKeys(
	lit *ast.CompositeLit,
	funcs map[string]*FunctionDefinition,
	filePath string,
) {
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if key := extractStringLiteral(kv.Key); key != "" {
				addCustomFunction(funcs, key, filePath)
			}
		}
	}
}

// extractStringLiteral extracts the string value from a basic literal.
func extractStringLiteral(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		// Remove quotes
		return strings.Trim(lit.Value, `"'`+"`")
	}
	return ""
}

// isFunctionValue checks if the expression represents a function value.
func isFunctionValue(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.FuncLit:
		// Anonymous function
		return true
	case *ast.SelectorExpr:
		// Method or function reference like strings.ToLower
		return true
	case *ast.Ident:
		// Function name reference
		return true
	}
	return false
}

// addCustomFunction adds a custom function definition with a generic signature.
func addCustomFunction(
	funcs map[string]*FunctionDefinition,
	name string,
	filePath string,
) {
	funcs[name] = NewCustomFunctionDefinition(name, filePath)
}

// NewCustomFunctionDefinition creates a custom function definition with a generic
// variadic signature: func(args ...any) any. This allows the function to accept
// any number of arguments and return any type, which is appropriate for custom
// template functions whose exact signatures are not known at analysis time.
func NewCustomFunctionDefinition(name, filePath string) *FunctionDefinition {
	// Create a generic variadic signature: func(args ...any) any
	// This allows any number of arguments and any return type
	anyType := typeAny.Type()
	anySlice := types.NewSlice(anyType)
	params := types.NewTuple(types.NewVar(token.NoPos, nil, "args", anySlice))
	results := types.NewTuple(types.NewVar(token.NoPos, nil, "", anyType))
	sig := types.NewSignatureType(nil, nil, nil, params, results, true)

	return &FunctionDefinition{
		name:     name,
		fileName: filePath,
		typ:      sig,
		node:     nil,
	}
}
