package pose

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func attemptLedgerPlan() DispatchPlan {
	return DispatchPlan{Adapter: "fake", Config: ReviewerAdapter{MaxRunsPerBundle: 2}, Bundle: ReviewBundle{BundleID: "rvb-ledger", BundleDigest: "sha256:bundle"}, Brief: ReviewBrief{Scope: "spec:alpha"}}
}
func attemptLedgerRun(t *testing.T, s Store, a DelegatedReviewAttempt, status, output string) ReviewRun {
	t.Helper()
	run, err := s.recordReviewRun(ReviewRun{SchemaVersion: 1, AttemptID: a.AttemptID, Reason: a.Reason, Scope: "spec:alpha", BundleID: "rvb-ledger", BundleDigest: "sha256:bundle", Adapter: "fake", Vendor: "synthetic", Model: "test", StartedAt: "2026-10-25T00:00:00Z", EndedAt: "2026-10-25T00:00:01Z", Status: status, Output: output}, []byte(output))
	if err != nil {
		t.Fatal(err)
	}
	if a.AttemptID != "" {
		if err := s.finishDelegatedReviewAttempt(a, run); err != nil {
			t.Fatal(err)
		}
	}
	return run
}

func TestAttemptLedgerReasonQuotaAndReservationLoss(t *testing.T) {
	for _, status := range []string{ReviewRunCompleted, ReviewRunFailed} {
		t.Run(status, func(t *testing.T) {
			s := Store{Root: t.TempDir()}
			plan := attemptLedgerPlan()
			plan.Config.MaxRunsPerBundle = 1
			a, err := s.reserveDelegatedReviewAttempt(plan, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			attemptLedgerRun(t, s, a, status, "decision: changes-requested")
			plan.Reason = "explicit follow-up"
			if _, err := s.reserveDelegatedReviewAttempt(plan, time.Now()); err == nil || !strings.Contains(err.Error(), "budget exhausted") {
				t.Fatalf("quota: %v", err)
			}
			if err := os.Remove(reviewAttemptPath(s.Root, a)); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(attemptReceiptPath(s.Root, a)); err != nil {
				t.Fatal(err)
			}
			if _, err := s.reserveDelegatedReviewAttempt(plan, time.Now()); err == nil {
				t.Fatal("deleting reservation reset quota")
			}
		})
	}
	s := Store{Root: t.TempDir()}
	plan := attemptLedgerPlan()
	a, err := s.reserveDelegatedReviewAttempt(plan, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	attemptLedgerRun(t, s, a, ReviewRunCompleted, "decision: rejected")
	plan.Adapter = "other"
	if _, err := s.reserveDelegatedReviewAttempt(plan, time.Now()); err == nil || !strings.Contains(err.Error(), "--reason") {
		t.Fatalf("cross-adapter rerun: %v", err)
	}
	plan.Reason = "independent adjudication of recorded findings"
	next, err := s.reserveDelegatedReviewAttempt(plan, time.Now())
	if err != nil || next.Ordinal != 2 || next.Reason != plan.Reason {
		t.Fatalf("explicit reason: %+v %v", next, err)
	}
}

func TestAttemptLedgerConcurrentReservationsAndCrashStayVisible(t *testing.T) {
	s := Store{Root: t.TempDir()}
	plan := attemptLedgerPlan()
	start := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := s.reserveDelegatedReviewAttempt(plan, time.Now()); err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()
	if wins != 1 {
		t.Fatalf("winners %d", wins)
	}
	if _, err := s.reserveDelegatedReviewAttempt(plan, time.Now()); err == nil {
		t.Fatal("dangling reservation hidden")
	}
	metrics := s.delegatedReviewInsights(0)
	if metrics == nil || metrics.CoverageComplete || len(metrics.ByBundle) != 1 || metrics.ByBundle[0].Attempts != 1 || metrics.ByBundle[0].Runs != 0 {
		t.Fatalf("pending attempt omitted: %+v", metrics)
	}
}

func TestAttemptLedgerRunAndWitnessLossAndRewrite(t *testing.T) {
	for _, mutation := range []string{"delete-run", "delete-transcript", "delete-witness", "rewrite-run", "rewrite-witness", "symlink"} {
		t.Run(mutation, func(t *testing.T) {
			s := Store{Root: t.TempDir()}
			run := attemptLedgerRun(t, s, DelegatedReviewAttempt{}, ReviewRunCompleted, "decision: rejected")
			path := filepath.Join(s.Root, reviewRunsDir, run.RunID+".json")
			witness := filepath.Join(s.Root, reviewRunLedgerDir, run.RunID+".json")
			switch mutation {
			case "delete-run":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "delete-transcript":
				if err := os.Remove(filepath.Join(s.Root, run.TranscriptPath)); err != nil {
					t.Fatal(err)
				}
			case "delete-witness":
				if err := os.Remove(witness); err != nil {
					t.Fatal(err)
				}
			case "rewrite-run":
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(strings.Replace(string(raw), "decision: rejected", "decision: approved", 1)), 0644); err != nil {
					t.Fatal(err)
				}
			case "rewrite-witness":
				if err := os.WriteFile(witness, []byte("{broken\n"), 0644); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("missing", path); err != nil {
					t.Fatal(err)
				}
			}
			_, losses, err := s.ReadReviewRuns("")
			if err == nil && len(losses) == 0 {
				t.Fatal("lost history read complete")
			}
		})
	}
}

func TestAttemptLedgerPublisherFailureDoesNotEraseWitness(t *testing.T) {
	s := Store{Root: t.TempDir()}
	PublishFault = func(path string) error {
		if strings.HasSuffix(path, ".transcript") {
			return errors.New("synthetic failure")
		}
		return nil
	}
	t.Cleanup(func() { PublishFault = nil })
	_, err := s.recordReviewRun(ReviewRun{SchemaVersion: 1, Scope: "spec:alpha", BundleID: "rvb-ledger", StartedAt: "2026-10-25T00:00:00Z", EndedAt: "2026-10-25T00:00:01Z", Status: ReviewRunFailed}, nil)
	if err == nil {
		t.Fatal("fault ignored")
	}
	findings, err := s.ReviewRunLedgerFindings("")
	if err != nil || len(findings) == 0 {
		t.Fatalf("witness lost: %v %v", findings, err)
	}
}

func TestAttemptLedgerDeclaredOutcomesAndLegacyIdentity(t *testing.T) {
	s := Store{Root: t.TempDir()}
	plan := attemptLedgerPlan()
	a, err := s.reserveDelegatedReviewAttempt(plan, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	attemptLedgerRun(t, s, a, ReviewRunCompleted, "decision: changes-requested")
	plan.Reason = "address the recorded findings"
	a, err = s.reserveDelegatedReviewAttempt(plan, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	run := attemptLedgerRun(t, s, a, ReviewRunCompleted, "decision: approved")
	rows, losses := s.reviewRunSummaries("spec:alpha")
	if len(rows) != 2 || len(losses) > 0 {
		t.Fatalf("history: %+v %v", rows, losses)
	}
	metrics := s.delegatedReviewInsights(0)
	if metrics == nil || !metrics.CoverageComplete || metrics.ByBundle[0].Runs != 2 || metrics.ByBundle[0].Reruns != 1 || metrics.ByAdapter[0].ApprovalRate == nil || *metrics.ByAdapter[0].ApprovalRate != 0.5 {
		t.Fatalf("metrics %+v", metrics)
	}
	if draftReviewDecision("decision: approved\ndecision: rejected") != "unknown" {
		t.Fatal("ambiguous draft promoted")
	}
	// A schema-1 legacy record omits new fields; its preexisting canonical ID remains valid.
	run.AttemptID = ""
	run.Reason = ""
	run.LedgerVersion = 0
	run.RunID = ""
	run.TranscriptPath = ""
	run.Path = ""
	raw, _ := json.Marshal(run)
	run.RunID = "rrn-" + strings.TrimPrefix(digestBytes(raw), "sha256:")[:16]
	run.TranscriptPath = filepath.ToSlash(filepath.Join(reviewRunsDir, run.RunID+".transcript"))
	if err := validateReviewRunRecord(run.RunID+".json", run); err != nil {
		t.Fatalf("legacy changed: %v", err)
	}
}

func TestAttemptLedgerScopedWitnessAndHistoricalWindow(t *testing.T) {
	s := Store{Root: t.TempDir()}
	run := attemptLedgerRun(t, s, DelegatedReviewAttempt{}, ReviewRunCompleted, "decision: rejected")
	path := filepath.Join(s.Root, reviewRunLedgerDir, run.RunID+".json")
	var witness reviewRunWitness
	if err := readLedgerJSON(path, &witness); err != nil {
		t.Fatal(err)
	}
	witness.BundleID = "rvb-decoy"
	raw, _ := json.Marshal(witness)
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	if _, losses, err := s.ReadReviewRuns("rvb-ledger"); err == nil && len(losses) == 0 {
		t.Fatal("witness escaped scoped integrity by changing bundle")
	}
	s = Store{Root: t.TempDir()}
	now := time.Now().UTC()
	for i, at := range []time.Time{now.Add(-48 * time.Hour), now} {
		_, err := s.recordReviewRun(ReviewRun{SchemaVersion: 1, Scope: "spec:alpha", BundleID: "rvb-window", Adapter: "fake", StartedAt: at.Format(time.RFC3339Nano), EndedAt: at.Add(time.Second).Format(time.RFC3339Nano), Status: ReviewRunCompleted, Output: []string{"decision: rejected", "decision: approved"}[i]}, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	metrics := s.delegatedReviewInsights(1)
	if metrics == nil || metrics.ByBundle[0].Runs != 1 || metrics.ByBundle[0].Attempts != 1 || metrics.ByBundle[0].Reruns != 1 {
		t.Fatalf("window erased prior attempts: %+v", metrics)
	}
}

func TestAttemptLedgerWriterAndTranscriptBoundsAgree(t *testing.T) {
	s := Store{Root: t.TempDir()}
	_, err := s.recordReviewRun(ReviewRun{SchemaVersion: 1, Scope: "spec:alpha", BundleID: "rvb-ledger", StartedAt: "2026-10-25T00:00:00Z", EndedAt: "2026-10-25T00:00:01Z", Status: ReviewRunCompleted, Output: strings.Repeat("x", reviewLedgerMaxBytes+1)}, nil)
	if err == nil {
		t.Fatal("writer published a record its reader cannot read")
	}
	entries, err := ledgerEntries(s.Root, reviewRunLedgerDir)
	if err != nil || len(entries) > 0 {
		t.Fatalf("oversized write left witness: %v %d", err, len(entries))
	}
	run := attemptLedgerRun(t, s, DelegatedReviewAttempt{}, ReviewRunCompleted, "decision: rejected")
	if err := os.Truncate(filepath.Join(s.Root, run.TranscriptPath), reviewTranscriptMaxBytes+1); err != nil {
		t.Fatal(err)
	}
	if why := s.reviewRunTranscriptIntact(run); !strings.Contains(why, "bound") {
		t.Fatalf("oversized transcript %q", why)
	}
}

func TestAttemptLedgerWitnessHonorsRemainingInventoryBudget(t *testing.T) {
	s := Store{Root: t.TempDir()}
	attemptLedgerRun(t, s, DelegatedReviewAttempt{}, ReviewRunCompleted, "decision: rejected")
	remaining := int64(1)
	findings, err := s.reviewRunLedgerFindings("", &remaining)
	if err != nil || len(findings) == 0 || !strings.Contains(strings.Join(findings, ";"), "read budget") {
		t.Fatalf("witness scan ignored shared budget: %v %v", findings, err)
	}
}
