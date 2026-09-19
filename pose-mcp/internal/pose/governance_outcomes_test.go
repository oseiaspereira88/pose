package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeGovernanceFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGovernanceOutcomesSeparateDimensionsAndCoverage(t *testing.T) {
	root := t.TempDir()
	writeGovernanceFile(t, root, ".pose/reports/history/runs.jsonl", ""+
		`{"generated_at":"2026-09-19T00:00:00Z","task_slug":"alpha","spec":"alpha","outcome":"pass","duration_seconds":2.5,"cost_usd":0.2}`+"\n"+`{"generated_at":"2026-09-19T00:01:00Z","task_slug":"alpha","spec":"alpha","outcome":"fail"}`+"\n"+`{"generated_at":"2026-09-19T00:02:00Z","task_slug":"beta","outcome":"partial"}`+"\n"+`{"generated_at":"2026-09-19T00:03:00Z","task_slug":"gamma","outcome":"skipped"}`+"\n"+`{"generated_at":"2026-09-19T00:04:00Z","task_slug":"delta","outcome":"new-value"}`+"\n"+"not-json\n")

	report, err := (Store{Root: root}).GovernanceOutcomes(GovernanceOutcomesQuery{
		Now: time.Date(2026, 9, 19, 1, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Coverage.Available || report.Coverage.HistoryRecordsScanned != 6 || report.Coverage.HistoryInvalidRecords != 1 {
		t.Fatalf("coverage = %+v", report.Coverage)
	}
	if report.Attempts.UnitsObserved != 4 || report.Attempts.AttemptsObserved != 5 {
		t.Fatalf("attempt dimensions = %+v", report.Attempts)
	}
	if report.Attempts.Pass != 1 || report.Attempts.Fail != 1 || report.Attempts.Partial != 1 || report.Attempts.Skipped != 1 || report.Attempts.Unknown != 1 {
		t.Fatalf("outcomes = %+v", report.Attempts)
	}
	if report.Attempts.ActiveDurationObserved != 1 || report.Attempts.ActiveDurationSeconds != 2.5 || report.Attempts.WaitDurationUnknown != 5 {
		t.Fatalf("telemetry = %+v", report.Attempts)
	}
	if report.Attempts.ObservedCostCount != 1 || report.Attempts.UnknownCostCount != 4 {
		t.Fatalf("cost = %+v", report.Attempts)
	}
	if report.Remediation.Available || report.Coverage.RemediationCoverage != "unavailable" {
		t.Fatalf("remediation must remain explicitly unavailable: %+v", report.Remediation)
	}
	first := report
	second, err := (Store{Root: root}).GovernanceOutcomes(GovernanceOutcomesQuery{
		Now: time.Date(2026, 9, 19, 1, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	first.GeneratedAt, second.GeneratedAt = "", ""
	left, _ := json.Marshal(first)
	right, _ := json.Marshal(second)
	if string(left) != string(right) {
		t.Fatalf("same input must produce same projection:\n%s\n%s", left, right)
	}
}

func TestGovernanceOutcomesCountsJudgmentAndInvalidArtifacts(t *testing.T) {
	root := t.TempDir()
	writeGovernanceFile(t, root, ".pose/review-bundles/rvb-invalid.json", "{}\n")
	writeGovernanceFile(t, root, ".pose/review-attestations/rva-invalid.json", "{}\n")

	payload := ReviewBundlePayload{Scope: ReviewBundleScope{Ref: "spec:alpha"}}
	digest, err := reviewBundlePayloadDigest(payload)
	if err != nil {
		t.Fatal(err)
	}
	bundle := ReviewBundle{SchemaVersion: ReviewBundleSchemaVersion, BundleID: "rvb-" + digest[len("sha256:"):len("sha256:")+16], BundleDigest: digest, State: "sealed", SealedAt: "2026-09-19T00:00:00Z", Payload: payload}
	bundleRaw, _ := json.Marshal(bundle)
	writeGovernanceFile(t, root, ".pose/review-bundles/"+bundle.BundleID+".json", string(bundleRaw)+"\n")

	att := ReviewAttestation{SchemaVersion: ReviewBundleSchemaVersion, BundleID: bundle.BundleID, BundleDigest: bundle.BundleDigest, Reviewer: "agent:reviewer", Decision: "changes-requested", Criteria: []ReviewCriterion{{ID: "scope", Disposition: "not-applicable", Rationale: "fixture"}}, Findings: []ReviewFinding{{ID: "f1", Severity: "low", Disposition: "accepted-risk"}}, AttestedAt: "2026-09-19T00:01:00Z"}
	att.AttestationID = reviewAttestationID(att)
	attRaw, _ := json.Marshal(att)
	writeGovernanceFile(t, root, ".pose/review-attestations/"+att.AttestationID+".json", string(attRaw)+"\n")
	superseding := ReviewAttestation{SchemaVersion: ReviewBundleSchemaVersion, BundleID: bundle.BundleID, BundleDigest: bundle.BundleDigest, Reviewer: "agent:reviewer", Decision: "approved", Supersedes: att.AttestationID, AttestedAt: "2026-09-19T00:02:00Z"}
	superseding.AttestationID = reviewAttestationID(superseding)
	supersedingRaw, _ := json.Marshal(superseding)
	writeGovernanceFile(t, root, ".pose/review-attestations/"+superseding.AttestationID+".json", string(supersedingRaw)+"\n")

	report, err := (Store{Root: root}).GovernanceOutcomes(GovernanceOutcomesQuery{Now: time.Date(2026, 9, 19, 1, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if report.Coverage.ReviewBundlesInvalid != 1 || report.Coverage.AttestationsInvalid != 1 {
		t.Fatalf("invalid artifact coverage = %+v", report.Coverage)
	}
	if report.Reviews.BundlesObserved != 1 || report.Reviews.AttestationsObserved != 2 || report.Reviews.ChangesRequested != 1 || report.Reviews.Approved != 1 {
		t.Fatalf("review dimensions = %+v", report.Reviews)
	}
	if report.Reviews.NotApplicable != 1 || report.Reviews.AcceptedRisk != 1 || report.Reviews.FindingsObserved != 1 || report.Reviews.InterventionsObserved != 1 {
		t.Fatalf("intervention dimensions = %+v", report.Reviews)
	}
	if report.Attempts.JudgmentAttempts != 2 || report.Attempts.FirstApprovedUnits != 1 || report.Attempts.SupersededRelations != 1 || report.Reviews.StaleUnknown != 1 {
		t.Fatalf("judgment dimensions = %+v", report)
	}
}

func TestGovernanceOutcomesRejectsInvalidQuery(t *testing.T) {
	store := Store{Root: t.TempDir()}
	for _, query := range []GovernanceOutcomesQuery{{SinceDays: -1}, {MaturityDays: -1}, {MinSample: -1}} {
		if _, err := store.GovernanceOutcomes(query); err == nil {
			t.Fatalf("invalid query accepted: %+v", query)
		}
	}
}
