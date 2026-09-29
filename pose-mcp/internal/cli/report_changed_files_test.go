package cli

// `pose report` lists the files a run changed. It trimmed the whole
// `git status --porcelain` output before cutting each line's three-character
// status prefix, so the first entry — whose prefix starts with a space for an
// unstaged change — lost its first character: README.md was recorded as
// EADME.md in the v5.0.7 evidence (spec pose-cli-output-machine-channel, R4).

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReportChangedFilesKeepsTheFirstPath(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "fixture@example.test"},
		{"config", "user.name", "fixture"},
		{"config", "gc.auto", "0"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	for _, name := range []string{"README.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("one\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := exec.Command("git", "-C", root, "add", "-A").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	if out, err := exec.Command("git", "-C", root, "commit", "-q", "-m", "fixture").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}
	// Unstaged modifications print as " M <path>": the leading space is part
	// of the status, not whitespace around the output.
	for _, name := range []string{"README.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("two\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := reportChangedFiles(root, "")
	want := []string{"README.md", "b.md"}
	if len(got) != len(want) {
		t.Fatalf("changed files = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("changed file %d = %q, want %q", i, got[i], want[i])
		}
	}
}
