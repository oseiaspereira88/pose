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
