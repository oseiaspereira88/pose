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

// Spec pose-review-resume-runtime-dir-followups R2, R3: a continuation reuses
// the runtime name its predecessor recorded, also a random one of an earlier
// engine; a round continuing its thread reuses the previous round's; a name
// that is not a runtime name is never used as a path.
func TestRunContinuityRuntimeNameFollowsTheSession(t *testing.T) {
	tmp := t.TempDir()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, tmp)
	}
	attempt := "rat-" + strings.Repeat("cd", 32)
	stable, _ := reviewRunRuntimeDirPath(attempt)
	old := ReviewRun{RunID: "rrn-old", AttemptID: attempt, Safety: &ReviewExecutionSafety{RuntimeID: "pose-review-run-1504287996"}}
	if got := runtimeSourceOf(old); got != "pose-review-run-1504287996" {
		t.Fatalf("an earlier engine's random name was not reused: %s", got)
	}
	if dir, ok := reviewRunRuntimeDirPath(runtimeSourceOf(old)); !ok || dir != filepath.Join(os.TempDir(), "pose-review-run-1504287996") {
		t.Fatalf("the recorded name's directory: %s %v", dir, ok)
	}
	bad := ReviewRun{AttemptID: attempt, Safety: &ReviewExecutionSafety{RuntimeID: "../escape"}}
	if dir, _ := reviewRunRuntimeDirPath(runtimeSourceOf(bad)); dir != stable {
		t.Fatalf("a name that is not a runtime name was used: %s", dir)
	}
	if got := runtimeSourceAmong(DispatchPlan{predecessor: &old}, "rat-"+strings.Repeat("ef", 32), nil); got != "pose-review-run-1504287996" {
		t.Fatalf("a continuation did not follow its predecessor: %s", got)
	}
	thread := DispatchPlan{thread: &ReviewThread{Mode: ReviewThreadContinued, From: "rrn-old"}}
	if got := runtimeSourceAmong(thread, "rat-"+strings.Repeat("ef", 32), []ReviewRun{old}); got != "pose-review-run-1504287996" {
		t.Fatalf("a continued thread did not follow its previous round: %s", got)
	}
	fresh := DispatchPlan{thread: &ReviewThread{Mode: ReviewThreadNew, From: "rrn-old"}}
	if got := runtimeSourceAmong(fresh, attempt, []ReviewRun{old}); got != attempt {
		t.Fatalf("a new thread followed the previous round: %s", got)
	}
}

// Spec pose-review-resume-runtime-dir-followups R1: the continuation takes
// its runtime directory before any claim; a directory already there, with no
// claim appearing, refuses it with nothing claimed.
func TestRunContinuityTakesTheRuntimeDirectoryBeforeTheClaim(t *testing.T) {
	tmp := t.TempDir()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, tmp)
	}
	s := Store{Root: t.TempDir()}
	prior := ReviewRun{RunID: "rrn-prior", AttemptID: "rat-" + strings.Repeat("ab", 32)}
	plan := DispatchPlan{predecessor: &prior}
	dir, err := s.takeContinuationRuntimeDir(plan)
	if err != nil || dir == "" {
		t.Fatalf("a free directory was not taken: %q %v", dir, err)
	}
	if _, err := s.takeContinuationRuntimeDir(plan); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("a taken directory was shared: %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(s.Root, ".pose")); len(entries) != 0 {
		t.Fatalf("the refused continuation wrote records: %v", entries)
	}
}

// Review round 1, N2: a round does not continue a session whose checkout
// path is held, so it never fails after its attempt is reserved.
func TestRunContinuityARoundDoesNotContinueIntoAHeldPath(t *testing.T) {
	tmp := t.TempDir()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, tmp)
	}
	prior := ReviewRun{RunID: "rrn-prior", AttemptID: "rat-" + strings.Repeat("ab", 32)}
	if previousRoundCheckoutHeld(prior) {
		t.Fatal("a free path was taken for held")
	}
	dir, _ := reviewRunRuntimeDirPath(runtimeSourceOf(prior))
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if !previousRoundCheckoutHeld(prior) {
		t.Fatal("a held path was taken for free")
	}
}
