package analyzer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pacer/gozer/internal/template/analyzer"
)

func TestScanWorkspaceForFuncMap(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "funcmap_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

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
	if err := os.WriteFile(
		filepath.Join(tmpDir, "funcs1.go"),
		[]byte(testFile1),
		0600,
	); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

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
	if err := os.WriteFile(
		filepath.Join(tmpDir, "funcs2.go"),
		[]byte(testFile2),
		0600,
	); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

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
	tmpDir, err := os.MkdirTemp("", "funcmap_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test file without template import
	testFile := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	if err := os.WriteFile(
		filepath.Join(tmpDir, "main.go"),
		[]byte(testFile),
		0600,
	); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	funcs, err := analyzer.ScanWorkspaceForFuncMap(tmpDir)
	if err != nil {
		t.Fatalf("ScanWorkspaceForFuncMap failed: %v", err)
	}

	if len(funcs) != 0 {
		t.Errorf("expected no functions, got %d", len(funcs))
	}
}

func TestScanWorkspaceForFuncMap_SkipsVendor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "funcmap_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create vendor directory
	vendorDir := filepath.Join(tmpDir, "vendor")
	if err := os.Mkdir(vendorDir, 0750); err != nil {
		t.Fatalf("failed to create vendor dir: %v", err)
	}

	// Test file in vendor (should be skipped)
	testFile := `package vendor

import "text/template"

var funcs = template.FuncMap{
	"vendorFunc": func() {},
}
`
	if err := os.WriteFile(
		filepath.Join(vendorDir, "vendor.go"),
		[]byte(testFile),
		0600,
	); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	funcs, err := analyzer.ScanWorkspaceForFuncMap(tmpDir)
	if err != nil {
		t.Fatalf("ScanWorkspaceForFuncMap failed: %v", err)
	}

	if _, ok := funcs["vendorFunc"]; ok {
		t.Error("vendor function should have been skipped")
	}
}

func TestScanWorkspaceForFuncMap_SkipsTestFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "funcmap_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test file that is a test (should be skipped)
	testFile := `package main

import "text/template"

var funcs = template.FuncMap{
	"testFunc": func() {},
}
`
	if err := os.WriteFile(
		filepath.Join(tmpDir, "funcs_test.go"),
		[]byte(testFile),
		0600,
	); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	funcs, err := analyzer.ScanWorkspaceForFuncMap(tmpDir)
	if err != nil {
		t.Fatalf("ScanWorkspaceForFuncMap failed: %v", err)
	}

	if _, ok := funcs["testFunc"]; ok {
		t.Error("test file function should have been skipped")
	}
}

func TestScanWorkspaceForFuncMap_AliasedImport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "funcmap_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test file with aliased import
	testFile := `package main

import (
	tmpl "text/template"
)

var funcs = tmpl.FuncMap{
	"aliased": func() {},
}
`
	if err := os.WriteFile(
		filepath.Join(tmpDir, "aliased.go"),
		[]byte(testFile),
		0600,
	); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	funcs, err := analyzer.ScanWorkspaceForFuncMap(tmpDir)
	if err != nil {
		t.Fatalf("ScanWorkspaceForFuncMap failed: %v", err)
	}

	if _, ok := funcs["aliased"]; !ok {
		t.Error("aliased import function should have been found")
	}
}
