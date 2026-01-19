package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TempDir creates a temp directory with files and registers cleanup with t.Cleanup.
// Returns the path to the created directory.
func TempDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "template_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			t.Fatalf("failed to create dir for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}
	return dir
}
