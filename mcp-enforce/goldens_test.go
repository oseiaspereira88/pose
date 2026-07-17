package mcpenforce

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// compareGolden asserts that got matches the file at path. Run the suite with
// MCP_ENFORCE_UPDATE_GOLDEN=1 to (re)generate the golden files.
func compareGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	if os.Getenv("MCP_ENFORCE_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with MCP_ENFORCE_UPDATE_GOLDEN=1 to create)", path, err)
	}
	if !bytes.Equal(bytes.TrimSpace(want), bytes.TrimSpace(got)) {
		t.Errorf("golden mismatch %s:\n--- want ---\n%s\n--- got ---\n%s", path, want, got)
	}
}
