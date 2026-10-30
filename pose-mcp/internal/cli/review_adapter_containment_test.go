package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	pose "github.com/harne8/pose-mcp/internal/pose"
)

func TestAdapterContainmentCLIRealDispatchAndDisclosure(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]pose.ReviewerAdapter{"fake": {Command: []string{"sh", "-c", `cat >/dev/null; printf 'decision: approved\n'`}, Vendor: "fixture", Model: "fixture", Containment: &pose.ReviewerContainment{SchemaVersion: 1, Profile: "trusted-process"}}})
	if code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake", "--dry-run"); code != 0 || !strings.Contains(out, `"profile":"trusted-process"`) || !strings.Contains(out, `"filesystem-isolation":"unavailable"`) {
		t.Fatal(code, out)
	}
	if code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake"); code != 0 || !strings.Contains(out, `"outcome":"completed"`) {
		t.Fatal(code, out)
	}
	if _, out := runPose(t, root, "review", "verify", "spec:alpha"); !strings.Contains(out, "review_verify.safety.") || !strings.Contains(out, "trusted local executable required") {
		t.Fatal(out)
	}
	runs, lost, err := (pose.Store{Root: root}).ReadReviewRuns("")
	if err != nil || len(lost) > 0 || len(runs) != 1 || runs[0].Safety == nil {
		t.Fatal(runs, lost, err)
	}
}

func TestAdapterContainmentCLIDenialBeforeEffectAndBoundedNotes(t *testing.T) {
	root := dispatchInstance(t)
	marker := filepath.Join(t.TempDir(), "spawned")
	writeReviewers(t, root, map[string]pose.ReviewerAdapter{"fake": {Command: []string{"touch", marker}, Vendor: "fixture", Model: "fixture", Containment: &pose.ReviewerContainment{SchemaVersion: 1, Profile: "trusted-process", Required: []string{"network-isolation"}}}})
	if code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake"); code == 0 || !strings.Contains(out, "unsupported/unavailable") {
		t.Fatal(code, out)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("unsupported backend spawned")
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/review-attempts")); !os.IsNotExist(err) {
		t.Fatal("unsupported backend reserved quota")
	}
	note := filepath.Join(t.TempDir(), "notes")
	if err := os.WriteFile(note, []byte(strings.Repeat("x", pose.ReviewNotesMaxBytes+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake", "--note-file", note, "--dry-run"); code == 0 || !strings.Contains(out, "at most") {
		t.Fatal(code, out)
	}
}

func TestAdapterContainmentCLICleanupFailureIsRetainedAndNotApproval(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]pose.ReviewerAdapter{"fake": {Command: []string{"sh", "-c", `cat >/dev/null; i=0; while test "$i" -lt 4097; do : > "$TMPDIR/extra-$i"; i=$((i+1)); done; printf 'decision: approved\n'`}, Vendor: "fixture", Model: "fixture", Containment: &pose.ReviewerContainment{SchemaVersion: 1, Profile: "trusted-process"}}})
	code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake")
	if code != 1 || !strings.Contains(out, `"cause":"cleanup-incomplete"`) {
		t.Fatal(code, out)
	}
	s := pose.Store{Root: root}
	runs, lost, err := s.ReadReviewRuns("")
	if err != nil || len(lost) > 0 || len(runs) != 1 || runs[0].Status != pose.ReviewRunCompleted || runs[0].Safety.Operation.Outcome != pose.OutcomeFailed {
		t.Fatal(runs, lost, err)
	}
	// Remove only this deliberately oversized synthetic fixture, after proving the
	// production cleanup retained it rather than claiming removal.
	orphan := filepath.Join(os.TempDir(), runs[0].Safety.RuntimeID)
	defer os.RemoveAll(orphan)
	if stat, err := os.Stat(orphan); err != nil || !stat.IsDir() {
		t.Fatal("fixture orphan was hidden", err)
	}
	if why := s.ReviewRunStillGoverns(runs[0]); !strings.Contains(why, "cleanup") {
		t.Fatal(why)
	}
	if _, out := runPose(t, root, "review", "verify", "spec:alpha"); !strings.Contains(out, "draft=unknown") {
		t.Fatal("cleanup failure disclosed approval", out)
	}
	metrics, err := s.Insights("workflow", 0)
	if err != nil || metrics.DelegatedReview.ByBundle[0].ApprovedDrafts != 0 {
		t.Fatal(metrics, err)
	}
}

// Spec pose-delegation-parity-gate R6: with the dispatch alias gone, the run
// document of pose delegate discloses the containment the run ran under.
func TestAdapterContainmentCLIRunDocumentDisclosesIt(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]pose.ReviewerAdapter{"fake": {Command: []string{"sh", "-c", `cat >/dev/null; printf 'decision: approved\n'`}, Vendor: "fixture", Model: "fixture", Containment: &pose.ReviewerContainment{SchemaVersion: 1, Profile: "trusted-process"}}})
	if code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake", "--json"); code != 0 || !strings.Contains(out, `"safety": {`) || !strings.Contains(out, `"profile": "trusted-process"`) {
		t.Fatal(code, out)
	}
}

// safetyOrderWriter records whether the run had started — its marker
// existed — when the containment line was written.
type safetyOrderWriter struct {
	marker      string
	written     strings.Builder
	sawSafety   bool
	afterTheRun bool
}

func (w *safetyOrderWriter) Write(p []byte) (int, error) {
	if !w.sawSafety && strings.Contains(string(p), "delegate.safety=") {
		w.sawSafety = true
		_, err := os.Stat(w.marker)
		w.afterTheRun = err == nil
	}
	return w.written.Write(p)
}

// Spec pose-delegation-parity-gate R6 (review round 1, PG-R1-08): a
// foreground run discloses its containment before the agent runs, as
// pose review dispatch did, not only once it has run.
func TestAdapterContainmentCLIDisclosesItBeforeTheRun(t *testing.T) {
	root := dispatchInstance(t)
	marker := filepath.Join(t.TempDir(), "ran")
	writeReviewers(t, root, map[string]pose.ReviewerAdapter{"fake": {Command: []string{"sh", "-c", `cat >/dev/null; : > '` + marker + `'; printf 'decision: approved\n'`}, Vendor: "fixture", Model: "fixture", Containment: &pose.ReviewerContainment{SchemaVersion: 1, Profile: "trusted-process"}}})
	stdout := &safetyOrderWriter{marker: marker}
	var stderr strings.Builder
	var code int
	inDir(t, root, func() { code = Main([]string{"delegate", "review", "spec:alpha", "--agent", "fake"}, stdout, &stderr) })
	if code != 0 || !stdout.sawSafety || stdout.afterTheRun {
		t.Fatalf("containment disclosed before the run: code=%d saw=%v after=%v\n%s%s", code, stdout.sawSafety, stdout.afterTheRun, stdout.written.String(), stderr.String())
	}
}
