package cli

import (
	"encoding/json"
	pose "github.com/harne8/pose-mcp/internal/pose"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAttemptLedgerDispatchReasonBeforeLaunchAndDisclosure(t *testing.T) {
	root := dispatchInstance(t)
	marker := filepath.Join(t.TempDir(), "launched")
	writeReviewers(t, root, map[string]pose.ReviewerAdapter{"fake": {Command: []string{"sh", "-c", `cat >/dev/null; if [ -f "$1" ]; then echo decision: approved; else echo decision: changes-requested; fi; printf x >> "$1"`, "sh", marker}, Vendor: "synthetic", Model: "test", MaxRunsPerBundle: 2}})
	store := pose.Store{Root: root}
	stale, err := store.PlanReviewDispatch("spec:alpha", "review", "", "fake")
	if err != nil {
		t.Fatal(err)
	}
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "fake", "--apply"); code != 0 {
		t.Fatalf("first %d %s", code, out)
	}
	if _, err := store.RunReviewDispatch(stale, time.Now); err == nil {
		t.Fatal("stale preview bypassed rerun rule")
	}
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "fake", "--apply"); code == 0 || !strings.Contains(out, "--reason") {
		t.Fatalf("rerun %d %s", code, out)
	}
	raw, _ := os.ReadFile(marker)
	if string(raw) != "x" {
		t.Fatalf("refusal launched adapter: %s", raw)
	}
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "fake", "--reason", "address recorded findings", "--apply"); code != 0 {
		t.Fatalf("reasoned %d %s", code, out)
	}
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "fake", "--reason", "quota cannot be bypassed", "--apply"); code == 0 || !strings.Contains(out, "budget exhausted") {
		t.Fatalf("quota %d %s", code, out)
	}
	runs, err := store.ListReviewRuns("")
	if err != nil || len(runs) != 2 {
		t.Fatalf("runs %+v %v", runs, err)
	}
	prep, err := store.PrepareReviewAttestation(runs[0].BundleID, "agent:reviewer", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := store.LoadReviewBundle(runs[0].BundleID)
	if err != nil {
		t.Fatal(err)
	}
	evidence := ""
	for _, item := range bundle.Payload.Evidence {
		if item.EvidenceClass == "integration" {
			evidence = "integration:" + item.ID
			break
		}
	}
	if evidence == "" {
		t.Fatal("fixture has no integration evidence")
	}
	prep.Attestation.Criteria = []pose.ReviewCriterion{{ID: "correctness", Disposition: "passed", Evidence: evidence, Rationale: "synthetic separate reviewer examined the fixture's sealed subject and recorded no remaining finding"}}
	prep.Attestation.Decision = "approved"
	if _, err := store.RecordReviewAttestation(prep.Attestation, time.Now()); err != nil {
		t.Fatal(err)
	}
	code, out := runPose(t, root, "review", "verify", "spec:alpha", "--json")
	var verification pose.ReviewBundleVerification
	if err := json.Unmarshal([]byte(out), &verification); err != nil {
		t.Fatal(err)
	}
	if code != 0 || !verification.Approved || len(verification.DelegatedRuns) != 2 {
		t.Fatalf("prior rejection not next to attestation: %d %s", code, out)
	}
	seenReject := false
	for _, run := range verification.DelegatedRuns {
		if run.DraftDecision == "changes-requested" {
			seenReject = true
		}
	}
	if !seenReject {
		t.Fatal("rejecting draft hidden")
	}
	code, out = runPose(t, root, "insights", "--json")
	var metrics pose.InsightsResult
	if err := json.Unmarshal([]byte(out), &metrics); err != nil {
		t.Fatal(err)
	}
	if code != 0 || metrics.DelegatedReview == nil || metrics.DelegatedReview.ByAdapter[0].ApprovalRate == nil || *metrics.DelegatedReview.ByAdapter[0].ApprovalRate != 0.5 {
		t.Fatalf("metrics %d %s", code, out)
	}
	htmlPath := filepath.Join(root, ".pose/reports/attempt-ledger.html")
	if code, out := runPose(t, root, "insights", "--html", "--out", htmlPath); code != 0 {
		t.Fatalf("HTML %d %s", code, out)
	}
	html, err := os.ReadFile(htmlPath)
	if err != nil || !strings.Contains(string(html), "Delegated review drafts") || !strings.Contains(string(html), "0.5000") {
		t.Fatalf("HTML omitted metrics: %v %s", err, html)
	}
}

func TestAttemptLedgerHistoryCheckRejectsDeletedCommittedEvidence(t *testing.T) {
	root := dispatchInstance(t)
	writeReviewers(t, root, map[string]pose.ReviewerAdapter{"fake": {Command: []string{"sh", "-c", "cat >/dev/null; echo decision: rejected"}, Vendor: "synthetic", Model: "test"}})
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "fake", "--apply"); code != 0 {
		t.Fatal(out)
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v %s", err, out)
		}
	}
	git("add", "-A")
	git("commit", "-qm", "record immutable run")
	if code, out := runPose(t, root, "history-check", "--strict"); code != 0 {
		t.Fatalf("valid ledger-only history %d %s", code, out)
	}
	runs, err := (pose.Store{Root: root}).ListReviewRuns("")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, runs[0].Path)); err != nil {
		t.Fatal(err)
	}
	if code, out := runPose(t, root, "history-check", "--strict"); code == 0 || !strings.Contains(out, "review") {
		t.Fatalf("deleted unstaged run %d %s", code, out)
	}
	git("add", "-A")
	git("commit", "-qm", "malicious deletion")
	if code, out := runPose(t, root, "history-check", "--strict"); code == 0 || !strings.Contains(out, "immutable review evidence changed") {
		t.Fatalf("committed deletion hidden %d %s", code, out)
	}
}

func TestAttemptLedgerDispatchMissingReasonValueIsUsageError(t *testing.T) {
	root := dispatchInstance(t)
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "fake", "--reason", "--apply"); code != 2 {
		t.Fatalf("flag swallowed as reason: %d %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/review-attempts")); !os.IsNotExist(err) {
		t.Fatal("usage error reserved an attempt")
	}
}
