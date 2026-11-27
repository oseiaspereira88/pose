package pose

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// standInGit puts a git first on PATH that runs script before the real git:
// the script may answer and exit, or fall through to the real one.
func standInGit(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell stand-in")
	}
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	body := "#!/bin/sh\n" + script + "\nexec " + real + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func indexBytes(t *testing.T, top string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(top, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// Spec pose-validate-index-reads-residuals R5: a Git that echoes an unknown
// --path-format back (before 2.31) does not leave the read unprotected.
func TestTheIndexLookupNeedsNoPathFormat(t *testing.T) {
	top, _ := staleRepo(t, "")
	standInGit(t, `case "$*" in *--path-format*) printf '%s\n' --path-format=absolute .git/index; exit 0;; esac`)
	before := indexBytes(t, top)
	if _, err := RawGitDiffReadOnly(top, "-C", top, "diff", "--name-only", "--"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, indexBytes(t, top)) {
		t.Fatal("a Git without --path-format left the read unprotected")
	}
}

// R5: the lookup and the hand-over run Git in English, whatever the locale
// and whatever translations are installed: the stand-in says which it got.
func TestIndexReadsRunGitInEnglish(t *testing.T) {
	top, _ := staleRepo(t, "")
	t.Setenv("LC_ALL", "pt_BR.UTF-8")
	standInGit(t, `case "$*" in *"diff --name-only"*) echo "fatal: locale=$LC_ALL" >&2; exit 128;; esac`)
	if _, err := RawGitDiffReadOnly(top, "-C", top, "diff", "--name-only", "--"); err == nil || !strings.Contains(err.Error(), "locale=C") {
		t.Fatalf("the hand-over did not run in English: %v", err)
	}
	standInGit(t, `case "$*" in *"--git-path index"*) echo "fatal: locale=$LC_ALL" >&2; exit 128;; esac`)
	if _, err := ReadOnlyIndex(RawGit("-C", top, "diff"), top); err == nil || !strings.Contains(err.Error(), "locale=C") {
		t.Fatalf("the lookup did not run in English: %v", err)
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

// R3: a bound never cuts inside a character.
func TestBoundsCutAtACharacter(t *testing.T) {
	s := strings.Repeat("a", 599) + "é" + "tail"
	if got := TruncateUTF8(s, 600); !utf8.ValidString(got) || !strings.HasSuffix(got, "…") {
		t.Fatalf("a cut inside a character: %q", got[590:])
	}
	err := GitReadError("git diff", errors.New("exit status 128"), []byte("fatal: "+strings.Repeat("é", 400)))
	if !utf8.ValidString(err.Error()) {
		t.Fatal("Git's message was cut inside a character")
	}
}
