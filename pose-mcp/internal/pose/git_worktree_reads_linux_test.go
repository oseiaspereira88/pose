//go:build linux

package pose

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

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

// R5: a failing memory-file write falls back to the private copy, and the
// copy keeps the index's time, so a racily clean change is still seen.
func TestTheIndexFallbacksKeepTheAnswer(t *testing.T) {
	saveWrite, saveCreate := memfdWrite, memfdCreate
	defer func() { memfdWrite, memfdCreate = saveWrite, saveCreate }()
	memfdWrite = func(*os.File, []byte) (int, error) { return 0, errors.New("no write here") }
	top, _ := staleRepo(t, "")
	before := indexBytes(t, top)
	if out, err := RawGitDiffReadOnly(top, "-C", top, "diff", "--name-only", "--"); err != nil || len(out) != 0 {
		t.Fatalf("a failing memfd write: %q %v", out, err)
	}
	if !bytes.Equal(before, indexBytes(t, top)) {
		t.Fatal("the write fallback rewrote .git/index")
	}
	memfdWrite = saveWrite
	memfdCreate = func(string, int) (int, error) { return -1, errors.New("no memfd here") }
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(args, err, string(out))
		}
	}
	git("init", "-q")
	file := filepath.Join(root, "a.txt")
	if err := os.WriteFile(file, []byte("aaaa\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(file, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	git("add", "a.txt")
	git("commit", "-q", "-m", "i")
	if err := os.WriteFile(file, []byte("bbbb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{file, filepath.Join(root, ".git", "index")} {
		if err := os.Chtimes(p, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	got, err := RawGitDiffReadOnly(root, "-C", root, "-c", "core.trustctime=false", "diff", "--name-only", "--")
	if err != nil || string(got) != "a.txt\n" {
		t.Fatalf("the private copy hid a racily clean change: %q %v", got, err)
	}
}
