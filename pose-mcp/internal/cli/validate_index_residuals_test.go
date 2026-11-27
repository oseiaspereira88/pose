package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
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
	// Review round 1, N1: its HEAD text is read from the root, so a spec
	// already in progress at HEAD is not taken for a new transition.
	inProgress := "---\nslug: alpha\nstatus: in-progress\n---\n\n# Spec: alpha\n"
	if err := os.WriteFile(spec, []byte(inProgress), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "in progress"}} {
		if out, err := exec.Command("git", append([]string{"-C", top}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(args, err, string(out))
		}
	}
	if err := os.WriteFile(spec, []byte(inProgress+"\nmore text\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	checker := &nativeChecker{root: root, mode: "strict", locale: localeEN, stdout: &out}
	checker.checkReadyTransitions()
	if checker.errors != 0 {
		t.Fatalf("a spec already in progress was taken for a new transition: %s", out.String())
	}
	// Review round 1, N7: scope selection names changes relative to the root,
	// so the root's modules own them.
	if err := os.WriteFile(filepath.Join(root, "x.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "x"}} {
		if out, err := exec.Command("git", append([]string{"-C", top}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(args, err, string(out))
		}
	}
	sel, err := computeChangedScopeWith(root, "HEAD~1", "HEAD", nil, scopeMetadata{})
	if err != nil || len(sel.Changed) != 2 || sel.Changed[1] != "x.txt" || strings.HasPrefix(strings.Join(sel.Changed, ","), "proj/") {
		t.Fatalf("scope selection below the repository top: %v %v", sel.Changed, err)
	}
	// Review round 2, N1: a change outside the root is kept, as a change
	// outside every module, so it widens the selection instead of vanishing.
	if err := os.WriteFile(filepath.Join(top, "outside.txt"), []byte("o\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "outside"}} {
		if out, err := exec.Command("git", append([]string{"-C", top}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(args, err, string(out))
		}
	}
	if err := os.WriteFile(filepath.Join(top, "outside.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sel, err = computeChangedScopeWith(root, "HEAD", "", nil, scopeMetadata{})
	if err != nil || !strings.Contains(strings.Join(sel.Changed, ","), "../outside.txt") {
		t.Fatalf("a change outside the root vanished: %v %v", sel.Changed, err)
	}
	// Review round 3, N1: diff.relative in the configuration changes nothing.
	if out, err := exec.Command("git", "-C", top, "config", "diff.relative", "true").CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	sel, err = computeChangedScopeWith(root, "HEAD~2", "HEAD", nil, scopeMetadata{})
	if err != nil || strings.Contains(strings.Join(sel.Changed, ","), "../x.txt") || !strings.Contains(strings.Join(sel.Changed, ","), "x.txt") {
		t.Fatalf("diff.relative changed the names: %v %v", sel.Changed, err)
	}
	// Review round 3, N3: outside a root two levels down, the name is the
	// relative path from the root.
	if got := scopeRootRelative("a/b/", "a/c/f.txt"); got != "../c/f.txt" {
		t.Fatalf("a change beside a deeper root: %s", got)
	}
	if got := scopeRootRelative(" sp/", " sp/x.txt"); got != "x.txt" {
		t.Fatalf("a root whose name starts with a space: %s", got)
	}
}

// R2: the versioned report carries no message of Git, so no path of the
// checkout, whatever its spelling; the warning on standard error carries it
// (review round 1, N2–N6).
func TestTheReportStatesNoPath(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "jane doe proj")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(outside))
	files, err := reportChangedFilesNoted(outside, "HEAD", localeEN)
	if err == nil || len(files) != 1 || strings.Contains(files[0], "doe") || strings.Contains(files[0], string(filepath.Separator)+"tmp") || !strings.Contains(files[0], "could not be listed") {
		t.Fatalf("the report note: %q %v", files, err)
	}
	if !strings.Contains(err.Error(), "jane doe proj") {
		t.Fatalf("the warning lost Git's cause: %v", err)
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

// R5: the check runs its listing in English; the report warns on standard
// error with Git's message; an X^- base is resolved by its revision.
func TestTheRemainingIndexReadGuards(t *testing.T) {
	root, _ := staleIndexRepo(t)
	t.Setenv("LC_ALL", "pt_BR.UTF-8")
	cliStandInGit(t, `case "$*" in *"ls-files --others --exclude-standard -- .pose/specs"*) echo "fatal: locale=$LC_ALL" >&2; exit 128;; *"status --porcelain=v1"*) echo "fatal: no status here" >&2; exit 128;; esac`)
	checker := &nativeChecker{root: root, mode: "strict", locale: localeEN, stdout: &strings.Builder{}}
	checker.readyTransitionCandidates()
	if out := checker.stdout.(*strings.Builder).String(); !strings.Contains(out, "locale=C") {
		t.Fatalf("the check's listing did not run in English: %s", out)
	}

	// The warning on standard error carries Git's message (D1; review round
	// 2, N4).
	if code, out := runPose(t, root, "report", "--task", "residual check"); code != 0 || !strings.Contains(out, "the changed files cannot be listed") || !strings.Contains(out, "no status here") {
		t.Fatalf("the report did not warn with Git's message: %d %s", code, out)
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

// Review round 3, N2: a root below a directory whose name starts with a space
// keeps its prefix whole, so its changes are named from it.
func TestScopeSelectionBelowADirectoryWithALeadingSpace(t *testing.T) {
	top := t.TempDir()
	root := filepath.Join(top, " sp")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "x.txt")
	if err := os.WriteFile(file, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "i"}} {
		if out, err := exec.Command("git", append([]string{"-C", top}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(args, err, string(out))
		}
	}
	if err := os.WriteFile(file, []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sel, err := computeChangedScopeWith(root, "HEAD", "", nil, scopeMetadata{})
	if err != nil || len(sel.Changed) != 1 || sel.Changed[0] != "x.txt" {
		t.Fatalf("a root below a leading-space directory: %v %v", sel.Changed, err)
	}
}
