package pose

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// Spec pose-validate-reads-leave-the-index-alone R1, R2: the work-tree reads
// leave a stale index byte-identical, and answer as `git diff` does.
func TestWorktreeReadsLeaveTheIndexAlone(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(err, string(out))
		}
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "a.txt"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "i"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(err, string(out))
		}
	}
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(filepath.Join(root, "a.txt"), later, later); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if changes, err := WorktreeChanges(root); err != nil || len(changes) != 0 {
		t.Fatalf("a stat-only change was reported: %v %v", changes, err)
	}
	if names, err := WorktreeChangedNames(root, "HEAD"); err != nil || len(names) != 0 {
		t.Fatalf("a stat-only change was named: %v %v", names, err)
	}
	after, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("a work-tree read rewrote .git/index")
	}
	// The answers match `git diff` for real changes: a commit since the base,
	// a modified, a deleted and a staged-new file.
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...).Output()
		if err != nil {
			t.Fatal(args, err)
		}
		return string(out)
	}
	for f, body := range map[string]string{"b.txt": "b\n", "c.txt": "c\n"} {
		if err := os.WriteFile(filepath.Join(root, f), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("add", "b.txt", "c.txt")
	git("commit", "-q", "-m", "two")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "c.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "d.txt"), []byte("d\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "d.txt")
	want := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(git("diff", "HEAD", "--name-status", "--no-renames", "--")), "\n") {
		f := strings.Fields(line)
		want[f[1]] = f[0]
	}
	changes, err := WorktreeChanges(root)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, c := range changes {
		got[c.Path] = c.Status
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("WorktreeChanges %v, git diff HEAD %v", got, want)
	}
	wantNames := strings.Fields(git("diff", "--name-only", "HEAD~1", "--"))
	names, err := WorktreeChangedNames(root, "HEAD~1")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	sort.Strings(wantNames)
	if strings.Join(names, ",") != strings.Join(wantNames, ",") {
		t.Fatalf("WorktreeChangedNames %v, git diff HEAD~1 %v", names, wantNames)
	}
}
