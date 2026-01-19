package analyzer_test

import (
	"testing"

	"github.com/pacer/gozer/internal/template/analyzer"
	"github.com/pacer/gozer/internal/template/testutil"
)

func TestScanWorkspaceForFuncMap(t *testing.T) {
	// Test file with template.FuncMap composite literal
	testFile1 := `package main

import (
	"strings"
	"text/template"
)

var funcs = template.FuncMap{
	"lower": strings.ToLower,
	"upper": strings.ToUpper,
	"custom": func(s string) string { return s },
}
`
	// Test file with html/template import
	testFile2 := `package main

import (
	"html/template"
)

var htmlFuncs = template.FuncMap{
	"safe": func(s string) template.HTML { return template.HTML(s) },
	"dict": func(values ...any) map[string]any { return nil },
}
`
	tmpDir := testutil.TempDir(t, map[string]string{
		"funcs1.go": testFile1,
		"funcs2.go": testFile2,
	})

	// Scan the workspace
	funcs, err := analyzer.ScanWorkspaceForFuncMap(tmpDir)
	if err != nil {
		t.Fatalf("ScanWorkspaceForFuncMap failed: %v", err)
	}

	// Verify expected functions were found
	expectedFuncs := []string{"lower", "upper", "custom", "safe", "dict"}
	for _, name := range expectedFuncs {
		if _, ok := funcs[name]; !ok {
			t.Errorf("expected function %q not found", name)
		}
	}

	// Verify the count
	if len(funcs) != len(expectedFuncs) {
		t.Errorf("expected %d functions, got %d", len(expectedFuncs), len(funcs))
		for name := range funcs {
			t.Logf("found function: %s", name)
		}
	}
}

func TestScanWorkspaceForFuncMap_NoTemplateImport(t *testing.T) {
	// Test file without template import
	testFile := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	tmpDir := testutil.TempDir(t, map[string]string{
		"main.go": testFile,
	})

	funcs, err := analyzer.ScanWorkspaceForFuncMap(tmpDir)
	if err != nil {
		t.Fatalf("ScanWorkspaceForFuncMap failed: %v", err)
	}

	if len(funcs) != 0 {
		t.Errorf("expected no functions, got %d", len(funcs))
	}
}

func TestScanWorkspaceForFuncMap_SkipsVendor(t *testing.T) {
	// Test file in vendor (should be skipped)
	testFile := `package vendor

import "text/template"

var funcs = template.FuncMap{
	"vendorFunc": func() {},
}
`
	tmpDir := testutil.TempDir(t, map[string]string{
		"vendor/vendor.go": testFile,
	})

	funcs, err := analyzer.ScanWorkspaceForFuncMap(tmpDir)
	if err != nil {
		t.Fatalf("ScanWorkspaceForFuncMap failed: %v", err)
	}

	if _, ok := funcs["vendorFunc"]; ok {
		t.Error("vendor function should have been skipped")
	}
}

func TestScanWorkspaceForFuncMap_SkipsTestFiles(t *testing.T) {
	// Test file that is a test (should be skipped)
	testFile := `package main

import "text/template"

var funcs = template.FuncMap{
	"testFunc": func() {},
}
`
	tmpDir := testutil.TempDir(t, map[string]string{
		"funcs_test.go": testFile,
	})

	funcs, err := analyzer.ScanWorkspaceForFuncMap(tmpDir)
	if err != nil {
		t.Fatalf("ScanWorkspaceForFuncMap failed: %v", err)
	}

	if _, ok := funcs["testFunc"]; ok {
		t.Error("test file function should have been skipped")
	}
}

func TestScanWorkspaceForFuncMap_AliasedImport(t *testing.T) {
	// Test file with aliased import
	testFile := `package main

import (
	tmpl "text/template"
)

var funcs = tmpl.FuncMap{
	"aliased": func() {},
}
`
	tmpDir := testutil.TempDir(t, map[string]string{
		"aliased.go": testFile,
	})

	funcs, err := analyzer.ScanWorkspaceForFuncMap(tmpDir)
	if err != nil {
		t.Fatalf("ScanWorkspaceForFuncMap failed: %v", err)
	}

	if _, ok := funcs["aliased"]; !ok {
		t.Error("aliased import function should have been found")
	}
}
