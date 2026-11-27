//go:build linux

package pose

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// Spec pose-validate-read-only-index-followups R4 (review round 1, N4): when
// no memory file can be made, or it cannot carry the index's time, the read
// falls back to the private copy, answers and leaves the index.
func TestReadOnlyIndexFallsBackWithoutAMemoryFile(t *testing.T) {
	for _, broken := range []string{"memfd", "utimes"} {
		t.Run(broken, func(t *testing.T) {
			saveMemfd, saveUtimes := memfdCreate, utimesNanoAt
			defer func() { memfdCreate, utimesNanoAt = saveMemfd, saveUtimes }()
			if broken == "memfd" {
				memfdCreate = func(string, int) (int, error) { return -1, errors.New("no memfd here") }
			} else {
				utimesNanoAt = func(int, string, []unix.Timespec, int) error { return errors.New("no /proc here") }
			}
			top, _ := staleRepo(t, "")
			before, err := os.ReadFile(filepath.Join(top, ".git", "index"))
			if err != nil {
				t.Fatal(err)
			}
			out, err := RawGitDiffReadOnly(top, "-C", top, "diff", "--name-only", "--")
			if err != nil || len(out) != 0 {
				t.Fatalf("the fallback answered %q %v", out, err)
			}
			if after, _ := os.ReadFile(filepath.Join(top, ".git", "index")); !bytes.Equal(before, after) {
				t.Fatal("the fallback read rewrote .git/index")
			}
		})
	}
}

// Review round 2, N16: a failure handing the index to git, here with neither
// a memory file nor a usable temporary directory, is an index failure.
func TestAFailedIndexHandOverIsAnIndexFailure(t *testing.T) {
	top, _ := staleRepo(t, "")
	save := memfdCreate
	defer func() { memfdCreate = save }()
	memfdCreate = func(string, int) (int, error) { return -1, errors.New("no memfd here") }
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))
	if _, err := RawGitDiffReadOnly(top, "-C", top, "diff", "--name-only", "--"); !errors.Is(err, ErrIndexUnreadable) {
		t.Fatalf("a failed hand-over: %v", err)
	}
}
