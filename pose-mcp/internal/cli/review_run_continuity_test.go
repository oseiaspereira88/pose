package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func runContinuityFixture(t *testing.T, changes ...func(*posemodel.ReviewerAdapter)) (string, posemodel.Store, posemodel.ReviewRun) {
	t.Helper()
	root := dispatchInstance(t)
	a := posemodel.ReviewerAdapter{Command: []string{"sh", "-c", "printf 'session: fixture-private-session\\nreset: 2030-01-01T00:00:00Z\\ndecision: rejected\\n'; exit 75"}, Vendor: "fixture", Model: "fixture", MaxRunsPerBundle: 1, Limit: &posemodel.ReviewerLimit{ExitCodes: []int{75}, ResetPattern: `reset: (\S+)`}, Session: &posemodel.ReviewerSession{Pattern: `session: (\S+)`}, Resume: &posemodel.ReviewerResume{Command: []string{"sh", "-c", `test "$1" = fixture-private-session || exit 2; printf 'decision: approved\n'`, "sh", "{session}"}}}
	for _, change := range changes {
		change(&a)
	}
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"fake": a, "other": {Command: []string{"sh", "-c", "printf 'decision: approved\\n'"}, Vendor: "other", Model: "fixture"}})
	code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake")
	if code != 1 || !strings.Contains(out, "status=interrupted") {
		t.Fatalf("interrupt: %d %s", code, out)
	}
	s := posemodel.Store{Root: root}
	runs, lost, e := s.ReadReviewRuns("")
	if e != nil || len(lost) > 0 || len(runs) != 1 {
		t.Fatal(runs, lost, e)
	}
	return root, s, runs[0]
}
func TestRunContinuityInterruptedResumeAndAccounting(t *testing.T) {
	root, s, prior := runContinuityFixture(t)
	if prior.PartialDigest == "" || prior.Interruption.Cause != "usage-limit" || !prior.Interruption.Resumable {
		t.Fatalf("partial: %+v", prior)
	}
	for _, path := range []string{prior.Path, prior.TranscriptPath} {
		raw, e := os.ReadFile(filepath.Join(root, path))
		if e != nil || strings.Contains(string(raw), "fixture-private-session") {
			t.Fatal("session leaked", path, e)
		}
	}
	if code, out := runPose(t, root, "delegate", "ls"); code != 0 || !strings.Contains(out, "cause=usage-limit") || !strings.Contains(out, "partial=true") || strings.Contains(out, "fixture-private-session") {
		t.Fatal(code, out)
	}
	metrics, e := s.Insights("workflow", 0)
	if e != nil || metrics.DelegatedReview.ByBundle[0].RejectingDrafts != 0 {
		t.Fatal("partial counted as verdict", e)
	}
	if _, e = s.PlanReviewResume(prior.RunID); e != nil {
		t.Fatal(e)
	}
	code, out := runPose(t, root, "delegate", "resume", prior.RunID)
	if code != 0 || !strings.Contains(out, "status=completed") {
		t.Fatal(code, out)
	}
	runs, lost, e := s.ReadReviewRuns("")
	if e != nil || len(lost) > 0 || len(runs) != 2 {
		t.Fatal(len(runs), lost, e)
	}
	var completed posemodel.ReviewRun
	for _, r := range runs {
		if r.Status == posemodel.ReviewRunCompleted {
			completed = r
		}
	}
	if completed.Resumes != prior.RunID || completed.AttemptID != prior.AttemptID || completed.BriefDigest != prior.BriefDigest || completed.TreeDigest != prior.TreeDigest {
		t.Fatalf("chain: %+v", completed)
	}
	if code, out := runPose(t, root, "delegate", "resume", prior.RunID); code != 0 || !strings.Contains(out, completed.RunID) {
		t.Fatal("idempotent", code, out)
	}
	runs, _ = s.ListReviewRuns("")
	if len(runs) != 2 {
		t.Fatal("duplicate process")
	}
	metrics, e = s.Insights("workflow", 0)
	m := metrics.DelegatedReview.ByBundle[0]
	if e != nil || m.Attempts != 1 || m.Resumes != 1 || m.Reruns != 0 || m.ApprovedDrafts != 1 || m.RejectingDrafts != 0 {
		t.Fatal(m, e)
	}
	if _, e = s.PlanReviewResume(completed.RunID); e == nil {
		t.Fatal("completed run resumed")
	}
	if _, e = s.PlanReviewDispatch(prior.BundleID, "review", "", "fake", "reason"); e == nil || !strings.Contains(e.Error(), "budget") {
		t.Fatal("resume reset attempt quota", e)
	}
	if rows, lost := s.PendingReviewInterruptions(); len(rows) != 0 || len(lost) != 0 {
		t.Fatal(rows, lost)
	}
}
func TestRunContinuityRefusalAndExplicitFallback(t *testing.T) {
	root, s, prior := runContinuityFixture(t)
	if code, out := runPose(t, root, "delegate", "resume", prior.RunID, "--wait-until-reset=1s"); code != 1 || !strings.Contains(out, "maximum wait") {
		t.Fatal(code, out)
	}
	plan, e := s.PlanReviewDispatch(prior.BundleID, "review", "", "other")
	if e != nil {
		t.Fatal(e)
	}
	next, e := s.RunReviewDispatch(plan, time.Now)
	if e != nil || next.LimitOf != prior.RunID || next.Reason != "limit-of:"+prior.RunID || next.AttemptID == prior.AttemptID {
		t.Fatal(next, e)
	}
	raw, e := os.ReadFile(filepath.Join(root, prior.Path))
	if e != nil {
		t.Fatal(e)
	}
	var edited map[string]any
	json.Unmarshal(raw, &edited)
	edited["output"] = "rewritten"
	changed, _ := json.Marshal(edited)
	os.WriteFile(filepath.Join(root, prior.Path), changed, 0644)
	if _, e = s.PlanReviewResume(prior.RunID); e == nil {
		t.Fatal("rewritten predecessor resumed")
	}
}

func TestRunContinuityOutputCaptureAndConflicts(t *testing.T) {
	root, s, r := runContinuityFixture(t, func(a *posemodel.ReviewerAdapter) {
		a.Command = []string{"sh", "-c", `printf 'session: fixture-private-session\nreset: 2030-01-01T00:00:00Z\nLIMIT\n' > "$1"`, "sh", "{output}"}
		a.Limit.OutputPatterns = []string{"LIMIT"}
	})
	if !r.Interruption.Resumable || r.SessionDigest == "" || r.Interruption.ResetAt == "" || strings.Contains(r.Output, "fixture-private-session") {
		t.Fatal("output channel not captured/redacted", r)
	}
	if code, out := runPose(t, root, "delegate", "resume", r.RunID); code != 0 {
		t.Fatal(code, out)
	}
	runs, lost, e := s.ReadReviewRuns("")
	if e != nil || len(lost) > 0 || len(runs) != 2 {
		t.Fatal(lost, e)
	}
	root = dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"fake": {
		Command: []string{"sh", "-c", `printf 'session: one\n'; printf 'session: two\n' > "$1"; exit 75`, "sh", "{output}"}, Vendor: "fixture", Model: "fixture",
		Limit: &posemodel.ReviewerLimit{ExitCodes: []int{75}}, Session: &posemodel.ReviewerSession{Pattern: `session: (\S+)`}, Resume: &posemodel.ReviewerResume{Command: []string{"fake", "{session}"}},
	}})
	if code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake"); code != 1 || !strings.Contains(out, "unredacted output withheld") {
		t.Fatal(code, out)
	}
	runs, lost, e = (posemodel.Store{Root: root}).ReadReviewRuns("")
	if e != nil || len(lost) > 0 || len(runs) != 1 || runs[0].Output != "" {
		t.Fatal("conflict not withheld", runs, lost, e)
	}
}
func TestRunContinuityBudgetsAndChangedInputs(t *testing.T) {
	_, s, r := runContinuityFixture(t, func(a *posemodel.ReviewerAdapter) { a.Resume.MaxWallTime = "1ns" })
	if _, e := s.PlanReviewResume(r.RunID); e == nil || !strings.Contains(e.Error(), "budget exhausted") {
		t.Fatal("wall budget accepted", e)
	}
	root, s, r := runContinuityFixture(t, func(a *posemodel.ReviewerAdapter) {
		a.Resume.MaxResumes = 1
		a.Resume.Command = a.Command
		a.Resume.Command = append(a.Resume.Command, "{session}")
	})
	if code, out := runPose(t, root, "delegate", "resume", r.RunID); code != 1 || !strings.Contains(out, "status=interrupted") {
		t.Fatal(code, out)
	}
	runs, lost, e := s.ReadReviewRuns("")
	if e != nil || len(lost) > 0 {
		t.Fatal(lost, e)
	}
	for _, next := range runs {
		if next.Resumes == r.RunID {
			if _, e = s.PlanReviewResume(next.RunID); e == nil || !strings.Contains(e.Error(), "budget exhausted") {
				t.Fatal("resume budget accepted", e)
			}
		}
	}
	root, s, r = runContinuityFixture(t)
	if e = os.WriteFile(filepath.Join(root, "pose-mcp/lib.go"), []byte("package changed\n"), 0644); e != nil {
		t.Fatal(e)
	}
	if _, e = s.PlanReviewResume(r.RunID); e == nil {
		t.Fatal("changed sealed source resumed")
	}
	root, s, r = runContinuityFixture(t)
	bundlePath := filepath.Join(root, ".pose/review-bundles", r.BundleID+".json")
	raw, e := os.ReadFile(bundlePath)
	if e != nil {
		t.Fatal(e)
	}
	raw = []byte(strings.Replace(string(raw), r.BundleDigest, "sha256:changed", 1))
	if e = os.WriteFile(bundlePath, raw, 0644); e != nil {
		t.Fatal(e)
	}
	if _, e = s.PlanReviewResume(r.RunID); e == nil {
		t.Fatal("changed bundle resumed")
	}
}

func TestRunContinuityLaunchFailureAndSessionRotation(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]posemodel.ReviewerAdapter{"missing": {Command: []string{"/nonexistent/pose-fixture-command"}, Vendor: "fixture", Model: "fixture"}})
	if code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "missing"); code != 1 || !strings.Contains(out, "status=failed") {
		t.Fatal(code, out)
	}
	runs, lost, e := (posemodel.Store{Root: root}).ReadReviewRuns("")
	if e != nil || len(lost) > 0 || len(runs) != 1 || runs[0].Interruption != nil {
		t.Fatal(runs, lost, e)
	}
	root, s, r := runContinuityFixture(t, func(a *posemodel.ReviewerAdapter) {
		a.Resume.Command = []string{"sh", "-c", `printf 'session: different-reference\nold reference: %s\n' "$1"; exit 75`, "sh", "{session}"}
	})
	if code, out := runPose(t, root, "delegate", "resume", r.RunID); code != 1 || !strings.Contains(out, "status=failed") {
		t.Fatal(code, out)
	}
	runs, lost, e = s.ReadReviewRuns("")
	if e != nil || len(lost) > 0 || len(runs) != 2 {
		t.Fatal(runs, lost, e)
	}
	for _, next := range runs {
		if next.Resumes != "" {
			raw, _ := os.ReadFile(filepath.Join(root, next.TranscriptPath))
			if next.Output != "" || len(raw) != 0 || next.Interruption != nil {
				t.Fatal("rotated session leaked", next)
			}
		}
	}
}
func TestRunContinuityDoctorStatusAndPending(t *testing.T) {
	root, s, r := runContinuityFixture(t)
	warns := 0
	diagnoseReviewContinuity(root, func(check, level, message, hint string) {
		if check == "review.interrupted" && level == "warn" && strings.Contains(message, r.RunID) {
			warns++
		}
	})
	if warns != 1 {
		t.Fatal("doctor missed interruption")
	}
	if code, out := runPose(t, root, "delegate", "ls", "--json"); code != 0 || !strings.Contains(out, `"time_left"`) || !strings.Contains(out, `"partial": true`) {
		t.Fatal(code, out)
	}
	raw, _ := json.Marshal(r)
	digest := sha256.Sum256(raw)
	claim := map[string]any{"schema_version": 1, "predecessor": r.RunID, "attempt_id": r.AttemptID, "predecessor_digest": "sha256:" + hex.EncodeToString(digest[:]), "claimed_at": time.Now().UTC().Format(time.RFC3339Nano), "nonce": strings.Repeat("a", 32)}
	claimRaw, _ := json.Marshal(claim)
	dir := filepath.Join(root, ".pose/review-continuations")
	if e := os.MkdirAll(dir, 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(dir, r.RunID+".json"), claimRaw, 0644); e != nil {
		t.Fatal(e)
	}
	if code, out := runPose(t, root, "delegate", "resume", r.RunID); code != 0 || !strings.Contains(out, "delegate.status=pending") || !strings.Contains(out, "resume-of:"+r.RunID) {
		t.Fatal("pending not returned", code, out)
	}
	runs, _ := s.ListReviewRuns("")
	if len(runs) != 1 {
		t.Fatal("pending claim launched another process")
	}
}

func TestRunContinuityConcurrentResumeLaunchesOnce(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "launches")
	_, s, r := runContinuityFixture(t, func(a *posemodel.ReviewerAdapter) {
		a.Resume.Command = []string{"sh", "-c", `printf 'start\n' >> "$2"; sleep 0.05; printf 'decision: approved\n'`, "sh", "{session}", marker}
	})
	plan, e := s.PlanReviewResume(r.RunID)
	if e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = s.RunReviewDispatch(plan, time.Now) }()
	}
	wg.Wait()
	raw, e := os.ReadFile(marker)
	if e != nil || string(raw) != "start\n" {
		t.Fatalf("duplicate launches %q %v", raw, e)
	}
	runs, lost, e := s.ReadReviewRuns("")
	if e != nil || len(lost) > 0 || len(runs) != 2 {
		t.Fatal(len(runs), lost, e)
	}
}
func TestRunContinuityObservedProcessLoss(t *testing.T) {
	_, s, r := runContinuityFixture(t, func(a *posemodel.ReviewerAdapter) {
		a.Command = []string{"sh", "-c", `printf 'session: fixture-private-session\n'; kill -KILL $$`}
	})
	if r.Interruption.Cause != "process-lost" || !r.Interruption.Resumable {
		t.Fatal(r.Interruption)
	}
	if _, e := s.PlanReviewResume(r.RunID); e != nil {
		t.Fatal(e)
	}
}
