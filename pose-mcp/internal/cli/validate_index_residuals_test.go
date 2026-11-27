package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"
)

func cliStandInGit(t *testing.T, script string) {
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

// Spec pose-validate-index-reads-residuals R1: with the POSE root below the
// repository top, a modified spec is a ready-transition candidate.
func TestReadyTransitionsSeeSpecsBelowTheRepositoryTop(t *testing.T) {
	top := t.TempDir()
	root := filepath.Join(top, "proj")
	spec := filepath.Join(root, ".pose", "specs", "2026-11-27-alpha.md")
	if err := os.MkdirAll(filepath.Dir(spec), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spec, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "i"}} {
		if out, err := exec.Command("git", append([]string{"-C", top}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(args, err, string(out))
		}
	}
	if err := os.WriteFile(spec, []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := (&nativeChecker{root: root}).readyTransitionCandidates()
	if len(got) != 1 || got[0] != ".pose/specs/2026-11-27-alpha.md" {
		t.Fatalf("a spec below the repository top: %v", got)
	}
}

// R2, R5: the report removes whole absolute paths, with spaces, brackets or
// backticks, keeps relative names, and bounds its note.
func TestTheReportRemovesWholePaths(t *testing.T) {
	root := filepath.Join(t.TempDir(), "jane doe proj")
	note := reportRedactPaths(root, "open "+root+"/.git/index: denied; see [/srv/x y] and `/opt/z` for feature/index-rework")
	for _, kept := range []string{"doe", "/srv", "/opt"} {
		if strings.Contains(note, kept) {
			t.Fatalf("the note kept %q: %s", kept, note)
		}
	}
	if !strings.Contains(note, "feature/index-rework") {
		t.Fatalf("a relative name was redacted: %s", note)
	}
}

// R4: a failing listing of untracked files fails scope selection with its
// cause instead of narrowing it silently.
func TestScopeSelectionReportsAFailedUntrackedListing(t *testing.T) {
	root, _ := staleIndexRepo(t)
	cliStandInGit(t, `case "$*" in *"ls-files -z --others"*) echo "fatal: no listing here" >&2; exit 128;; esac`)
	if _, err := computeChangedScopeWith(root, "HEAD", "", nil, scopeMetadata{}); err == nil || !strings.Contains(err.Error(), "no listing here") {
		t.Fatalf("a failed untracked listing: %v", err)
	}
}

// R5: the check runs its diff in English; the report bounds its note and
// warns on standard error; an X^- base is resolved by its revision.
func TestTheRemainingIndexReadGuards(t *testing.T) {
	root, _ := staleIndexRepo(t)
	t.Setenv("LC_ALL", "pt_BR.UTF-8")
	cliStandInGit(t, `case "$*" in *"ls-files --others --exclude-standard -- .pose/specs"*) echo "fatal: locale=$LC_ALL" >&2; exit 128;; *"status --porcelain=v1"*) echo "fatal: no status here" >&2; exit 128;; esac`)
	checker := &nativeChecker{root: root, mode: "strict", locale: localeEN, stdout: &strings.Builder{}}
	checker.readyTransitionCandidates()
	if out := checker.stdout.(*strings.Builder).String(); !strings.Contains(out, "locale=C") {
		t.Fatalf("the check's listing did not run in English: %s", out)
	}
	if note := reportNote(root, errors.New(strings.Repeat("é", 2000))); len(note) > 610 || !utf8.ValidString(note) {
		t.Fatalf("the report note is not bounded: %d bytes", len(note))
	}
	files, err := reportChangedFilesNoted(root, "", localeEN)
	if err == nil || len(files) != 1 || len(files[0]) > 700 {
		t.Fatalf("the report note is not bounded: %d bytes, %v", len(strings.Join(files, "")), err)
	}
	if code, out := runPose(t, root, "report", "--task", "residual check"); code != 0 || !strings.Contains(out, "the changed files cannot be listed") {
		t.Fatalf("the report did not warn: %d %s", code, out)
	}
	index := filepath.Join(root, ".git", "index")
	if err := os.Remove(index); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(index, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := computeChangedScopeWith(root, "HEAD^-", "", nil, scopeMetadata{}); err == nil || strings.Contains(err.Error(), "revision unknown?") {
		t.Fatalf("an X^- base whose read failed was taken for an unknown revision: %v", err)
	}
}
