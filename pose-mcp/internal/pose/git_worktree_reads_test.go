package pose

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Spec pose-validate-reads-leave-the-index-alone R1, R2: a diff that reads the
// index read-only answers exactly as the plain diff does, over the states
// review round 1 found a status-based reading got wrong, and leaves the index
// byte-identical where the plain diff rewrites it.
func TestReadOnlyIndexDiffAnswersAsThePlainDiff(t *testing.T) {
	root := t.TempDir()
	run := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "protocol.file.allow=always"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return string(out)
	}
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A submodule whose .gitmodules says ignore=all.
	sub := t.TempDir()
	run(sub, "init", "-q")
	if err := os.WriteFile(filepath.Join(sub, "s.txt"), []byte("1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(sub, "add", "s.txt")
	run(sub, "commit", "-q", "-m", "s1")
	run(root, "init", "-q")
	for _, f := range []string{"stat.txt", "mod.txt", "del.txt", "proj/back.txt", "dir/x.txt", "conflict.txt"} {
		write(f, "x\n")
	}
	if err := os.Symlink("target-missing", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	run(root, "add", "-A")
	run(root, "submodule", "add", "-q", sub, "mod")
	run(root, "config", "-f", ".gitmodules", "submodule.mod.ignore", "all")
	run(root, "add", ".gitmodules")
	run(root, "commit", "-q", "-m", "base")
	base := run(root, "rev-parse", "HEAD")[:40]
	// A commit since the base: the gitlink moves, and a file changes and is
	// changed back in the work tree.
	if err := os.WriteFile(filepath.Join(root, "mod", "s.txt"), []byte("2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(filepath.Join(root, "mod"), "commit", "-q", "-am", "s2")
	write("proj/back.txt", "y\n")
	run(root, "add", "-A")
	run(root, "commit", "-q", "-m", "next")
	write("proj/back.txt", "x\n")
	// Work-tree states.
	write("mod.txt", "changed\n")
	if err := os.Remove(filepath.Join(root, "del.txt")); err != nil {
		t.Fatal(err)
	}
	write("staged.txt", "s\n")
	run(root, "add", "staged.txt")
	write("ita.txt", "i\n")
	run(root, "add", "-N", "ita.txt")
	write("conflict.txt", "staged then reverted\n")
	run(root, "add", "conflict.txt")
	write("conflict.txt", "x\n")
	if err := os.RemoveAll(filepath.Join(root, "dir")); err != nil {
		t.Fatal(err)
	}
	write("dir", "now a file\n")
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(filepath.Join(root, "stat.txt"), later, later); err != nil {
		t.Fatal(err)
	}
	reads := [][]string{
		{"-C", root, "-c", "diff.ignoreSubmodules=none", "diff", "--ignore-submodules=none", "--name-only", "-z", "--no-renames", base, "--"},
		{"-C", root, "diff", "HEAD", "--name-status", "--no-renames", "--diff-filter=MDT", "--"},
		{"-C", filepath.Join(root, "proj"), "diff", "--name-only", base},
	}
	for _, args := range reads {
		before, err := os.ReadFile(filepath.Join(root, ".git", "index"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := RawGitDiffReadOnly(root, args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		after, err := os.ReadFile(filepath.Join(root, ".git", "index"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) {
			t.Fatalf("%v rewrote .git/index", args)
		}
		want, err := RawGit(args...).Output()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%v answered %q, the plain diff %q", args, got, want)
		}
		// Make the stat cache stale again for the next read.
		later = later.Add(2 * time.Second)
		if err := os.Chtimes(filepath.Join(root, "stat.txt"), later, later); err != nil {
			t.Fatal(err)
		}
	}
	// The review history ledger's own diff (review round 1, N9).
	before, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reviewHistoryGit(root, "diff", "HEAD", "--name-status", "--no-renames", "--diff-filter=MDT", "--"); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(filepath.Join(root, ".git", "index")); !bytes.Equal(before, after) {
		t.Fatal("the review history diff rewrote .git/index")
	}
	entries, err := os.ReadDir(filepath.Join(root, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".lock" {
			t.Fatalf("a read left %s", e.Name())
		}
	}
}

// Review round 2, N10: an entry racily clean against the index (its file as
// new as the index) is still compared by content: the copy keeps the index's
// time, so Git does not trust the stat a fresher copy would make old.
func TestReadOnlyIndexDiffSeesARacilyCleanEntry(t *testing.T) {
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
	// An old file time, so the entry is written clean, not smudged as racy.
	stamp := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(file, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	git("add", "a.txt")
	git("commit", "-q", "-m", "i")
	// Same size, same time: only the content tells the change.
	if err := os.WriteFile(file, []byte("bbbb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{file, filepath.Join(root, ".git", "index")} {
		if err := os.Chtimes(p, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	args := []string{"-C", root, "-c", "core.trustctime=false", "diff", "--name-only", "--"}
	got, err := RawGitDiffReadOnly(root, args...)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "a.txt\n" {
		t.Fatalf("a racily clean change was hidden: %q", got)
	}
}

// Review round 4, N21: a FIFO at the index path is refused, not opened,
// which would block.
func TestReadOnlyIndexRefusesAFIFOIndex(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no FIFOs")
	}
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	index := filepath.Join(root, ".git", "index")
	_ = os.Remove(index)
	if out, err := exec.Command("mkfifo", index).CombinedOutput(); err != nil {
		t.Skip("mkfifo unavailable:", err, string(out))
	}
	done := make(chan error, 1)
	go func() {
		_, err := RawGitDiffReadOnly(root, "-C", root, "diff", "--name-only", "--")
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "not a regular file") {
			t.Fatalf("a FIFO index: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("reading a FIFO index blocked")
	}
}
