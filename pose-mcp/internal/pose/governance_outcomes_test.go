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
	for _, query := range []GovernanceOutcomesQuery{
		{SinceDays: -1}, {MaturityDays: -1}, {MinSample: -1},
		{Band: "everything"}, {ReportType: "transcript"},
	} {
		if _, err := store.GovernanceOutcomes(query); err == nil {
			t.Fatalf("invalid query accepted: %+v", query)
		}
	}
}

func TestGovernanceOutcomesDeduplicatesReplayAndPreservesNegativeSequences(t *testing.T) {
	root := t.TempDir()
	first := `{"generated_at":"2026-09-19T00:00:00Z","sequence":1,"task_slug":"alpha","report_type":"standard","spec":"alpha","stable_hash":"same","outcome":"fail"}`
	second := `{"generated_at":"2026-09-19T00:01:00Z","sequence":2,"task_slug":"alpha","report_type":"standard","spec":"alpha","stable_hash":"same","outcome":"fail"}`
	conflictPass := `{"generated_at":"2026-09-19T00:02:00Z","sequence":3,"task_slug":"alpha","report_type":"standard","spec":"alpha","stable_hash":"same","outcome":"pass"}`
	conflictFail := `{"generated_at":"2026-09-19T00:02:00Z","sequence":3,"task_slug":"alpha","report_type":"standard","spec":"alpha","stable_hash":"same","outcome":"fail"}`
	legacy := `{"generated_at":"2026-09-19T00:03:00Z","task_slug":"legacy","report_type":"standard","outcome":"fail"}`
	writeGovernanceFile(t, root, ".pose/reports/history/standard-alpha.jsonl", first+"\n"+first+"\n"+second+"\n"+conflictPass+"\n"+conflictFail+"\n"+legacy+"\n")

	report, err := (Store{Root: root}).GovernanceOutcomes(GovernanceOutcomesQuery{
		ProjectID: "proj.harne8", Now: time.Date(2026, 9, 19, 1, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != 2 || report.ProjectID != "proj.harne8" {
		t.Fatalf("versioned project scope missing: %+v", report)
	}
	if report.Attempts.AttemptsObserved != 4 || report.Attempts.Fail != 4 || report.Attempts.Pass != 0 {
		t.Fatalf("replay or conflict changed negative attempts: %+v", report.Attempts)
	}
	if report.Coverage.HistoryRecordsDuplicate != 1 || report.Coverage.HistoryIdentityConflicts != 1 || report.Coverage.HistoryIdentityUnknown != 1 {
		t.Fatalf("identity coverage = %+v", report.Coverage)
	}
	if len(report.Provenance.Sources) < 3 || report.Provenance.ReportAttemptIdentity != "report_type/task_slug/sequence" {
		t.Fatalf("provenance missing source/version identity: %+v", report.Provenance)
	}
}

func TestGovernanceOutcomesFiltersOnlyApplicableFacets(t *testing.T) {
	root := t.TempDir()
	history := `{"generated_at":"2026-09-19T00:00:00Z","sequence":1,"task_slug":"standard","report_type":"standard","outcome":"fail"}` + "\n" +
		`{"generated_at":"2026-09-19T00:01:00Z","sequence":1,"task_slug":"docs","report_type":"doc-audit","outcome":"pass"}` + "\n"
	writeGovernanceFile(t, root, ".pose/reports/history/reports.jsonl", history)
	writeBundle := func(scope, band, at string) ReviewBundle {
		t.Helper()
		payload := ReviewBundlePayload{Scope: ReviewBundleScope{Ref: scope}, Plan: ReviewBundlePlan{Band: band}}
		digest, err := reviewBundlePayloadDigest(payload)
		if err != nil {
			t.Fatal(err)
		}
		bundle := ReviewBundle{SchemaVersion: ReviewBundleSchemaVersion, BundleID: "rvb-" + digest[len("sha256:"):len("sha256:")+16], BundleDigest: digest, State: "sealed", SealedAt: at, Payload: payload}
		raw, _ := json.Marshal(bundle)
		writeGovernanceFile(t, root, ".pose/review-bundles/"+bundle.BundleID+".json", string(raw)+"\n")
		return bundle
	}
	writeAttestation := func(bundle ReviewBundle, reviewer, decision, at string) {
		t.Helper()
		att := approvedBundleAttestation(bundle, reviewer)
		att.SchemaVersion = ReviewBundleSchemaVersion
		att.Decision = decision
		att.AttestedAt = at
		att.AttestationID = reviewAttestationID(att)
		raw, _ := json.Marshal(att)
		writeGovernanceFile(t, root, ".pose/review-attestations/"+att.AttestationID+".json", string(raw)+"\n")
	}
	critical := writeBundle("spec:critical", ReviewBandCritical, "2026-09-19T00:02:00Z")
	baseline := writeBundle("spec:baseline", ReviewBandBaseline, "2026-09-19T00:03:00Z")
	writeAttestation(critical, "agent:critical", "changes-requested", "2026-09-19T00:04:00Z")
	writeAttestation(baseline, "agent:baseline", "approved", "2026-09-19T00:05:00Z")
	now := time.Date(2026, 9, 19, 1, 0, 0, 0, time.UTC)

	byType, err := (Store{Root: root}).GovernanceOutcomes(GovernanceOutcomesQuery{Now: now, ReportType: "doc-audit"})
	if err != nil {
		t.Fatal(err)
	}
	if byType.Attempts.AttemptsObserved != 1 || byType.Attempts.Pass != 1 || byType.Attempts.Fail != 0 || byType.Reviews.AttestationsObserved != 2 {
		t.Fatalf("report type filter escaped its facet: attempts=%+v reviews=%+v", byType.Attempts, byType.Reviews)
	}
	byBand, err := (Store{Root: root}).GovernanceOutcomes(GovernanceOutcomesQuery{Now: now, Band: ReviewBandCritical})
	if err != nil {
		t.Fatal(err)
	}
	if byBand.Reviews.BundlesObserved != 1 || byBand.Reviews.AttestationsObserved != 1 || byBand.Reviews.ChangesRequested != 1 || byBand.Attempts.AttemptsObserved != 2 {
		t.Fatalf("band filter escaped its facet: attempts=%+v reviews=%+v", byBand.Attempts, byBand.Reviews)
	}
}

// The projection is local by definition. It used to verify each bundle's
// freshness through whatever store it was called on; from pose-mcp that is
// the federated store, which resolves every qualified dependency again per
// bundle and turned a 30-second CLI answer into more than ten minutes, with a
// result that could differ from the CLI's. It now verifies through a local
// store on the same root, so every caller gets the same answer at the same
// cost.
func TestGovernanceOutcomesVerifiesFreshnessLocally(t *testing.T) {
	root, source, store, _ := federatedTestSpecConsumer(t)
	now := time.Now().UTC()
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	att := approvedBundleAttestation(bundle, "agent:reviewer")
	att.SchemaVersion, att.Decision = ReviewBundleSchemaVersion, "approved"
	att.AttestedAt = now.Format(time.RFC3339)
	att.AttestationID = reviewAttestationID(att)
	attRaw, _ := json.Marshal(att)
	writeGovernanceFile(t, root, ".pose/review-attestations/"+att.AttestationID+".json", string(attRaw)+"\n")
	local, err := (Store{Root: root}).GovernanceOutcomes(GovernanceOutcomesQuery{Now: now})
	if err != nil {
		t.Fatal(err)
	}
	resolver := federatedTestResolver("coordinator", "source", root, source)
	calls := 0
	authorize := resolver.Authorize
	resolver.Authorize = func(projectID string) bool {
		calls++
		if authorize == nil {
			return true
		}
		return authorize(projectID)
	}
	store.FederatedProjectID, store.FederatedResolver = "coordinator", &resolver
	federated, err := store.GovernanceOutcomes(GovernanceOutcomesQuery{Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("the projection resolved qualified dependencies %d time(s)", calls)
	}
	if local.Coverage.FreshnessChecks == 0 {
		t.Fatal("the fixture verified no bundle freshness")
	}
	a, _ := json.Marshal(local)
	b, _ := json.Marshal(federated)
	if string(a) != string(b) {
		t.Fatalf("CLI and MCP stores disagree:\n%s\n%s", a, b)
	}
}
