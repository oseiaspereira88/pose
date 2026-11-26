package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-review-resume-keeps-the-sealed-checkout R3: the attempt's
// directory is created exclusively; an existing path, a link included, fails
// the run instead of being shared or followed.
func TestTheAttemptRuntimeDirectoryIsCreatedExclusively(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	id := "rat-" + strings.Repeat("ab", 32)
	dir, err := reviewRunRuntimeDir(id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reviewRunRuntimeDir(id); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("an existing runtime directory was shared: %v", err)
	}
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	if err := os.Symlink(target, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := reviewRunRuntimeDir(id); err == nil {
		t.Fatal("a link at the runtime directory was followed")
	}
	if again, err := reviewRunRuntimeDir(""); err != nil || filepath.Dir(again) != os.TempDir() {
		t.Fatalf("a run without an attempt: %s %v", again, err)
	}
}
