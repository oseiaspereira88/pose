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
func TestRunContinuityAttemptRuntimeDirectoryIsCreatedExclusively(t *testing.T) {
	// os.TempDir reads TMPDIR on Unix and TMP/TEMP on Windows (review round
	// 2, N10).
	tmp := t.TempDir()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, tmp)
	}
	id := "rat-" + strings.Repeat("ab", 32)
	dir, err := reviewRunRuntimeDir(id)
	if err != nil {
		t.Fatal(err)
	}
	// The adapter's TMPDIR is this directory and sandboxes create Unix
	// sockets under it: the name is no longer than a random one (review
	// rounds 1 to 3, the reviewer's sandbox failed to start).
	// os.MkdirTemp's suffix is a uint32 in decimal, at most ten digits
	// (review round 4, N16).
	if longest := len("pose-review-run-") + 10; len(filepath.Base(dir)) > longest {
		t.Fatalf("the attempt's runtime directory %s is longer than the longest random one (%d bytes)", filepath.Base(dir), longest)
	}
	if _, err := reviewRunRuntimeDir(id); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("an existing runtime directory was shared: %v", err)
	}
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	// Without the privilege to create a link (Windows), only the link case
	// is skipped (review round 4, N20).
	if err := os.Symlink(target, dir); err != nil {
		t.Logf("link case skipped: %v", err)
	} else if _, err := reviewRunRuntimeDir(id); err == nil {
		t.Fatal("a link at the runtime directory was followed")
	}
	if again, err := reviewRunRuntimeDir(""); err != nil || filepath.Dir(again) != os.TempDir() {
		t.Fatalf("a run without an attempt: %s %v", again, err)
	}
}
