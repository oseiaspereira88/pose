package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-source-coverage-integrity: the cases the third independent review reproduced.

// R3-F01: the decoder matches names without regard to case; so does the attribution.
func TestCaseFoldedHeaderNamesAreRepeatedNames(t *testing.T) {
	if _, _, ambiguous := sniffJSONString([]byte(`{"bundle_id":"rvb-a","BUNDLE_ID":"rvb-b","dec`), "bundle_id"); !ambiguous {
		t.Fatal("a case-folded repeat was not ambiguous")
	}
	if v, found, _ := sniffJSONString([]byte(`{"Payload":{"Scope":{"REF":"spec:a"}}`), "payload", "scope", "ref"); !found || v != "spec:a" {
		t.Fatalf("the path did not match without regard to case: %q %v", v, found)
	}
	store, bundle, _ := retentionFixture(t)
	writeCoverageFile(t, store.Root, ".pose/review-attestations/rva-00000000000000a9.json", `{"bundle_id":"`+bundle.BundleID+`","BUNDLE_ID":"rvb-0123456789abcdef","decision":"rejected"}`)
	read, err := store.ReadReviewAttestations(bundle.BundleID)
	if err != nil || read.Coverage.Complete() {
		t.Fatalf("a case-folded repeat was filtered out as another bundle's: %+v err=%v", read.Coverage, err)
	}
}

// R3-F02: a later bundle that could not be read blocks the retention of an earlier approval.
func TestALostLaterBundleBlocksTheRetainedApproval(t *testing.T) {
	store, _, now := retentionFixture(t)
	addMatrixCheck(t, store)
	second, err := store.SealReviewBundle("spec:backend", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	rejected := approvedBundleAttestation(second, "agent:second-review")
	rejected.Decision = "rejected"
	if _, err := store.RecordReviewAttestation(rejected, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Root, ".pose/review-bundles", second.BundleID+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw[:len(raw)*3/4], 0o644); err != nil {
		t.Fatal(err)
	}
	closeBackendSpec(t, store)
	writeReviewFixture(t, store.Root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"moduleOverrides":{"api":{"checks":[{"name":"another-check","program":"true","severity":"required","evidenceClass":"unit"}]}}}`)
	v, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if v.Approved {
		t.Fatalf("an earlier approval stood over a later bundle that could not be read: %v", v.Blockers)
	}
}

// R3-F03: order is by instant, not by text.
func TestReviewTimesAreComparedAsInstants(t *testing.T) {
	// 12:00+09:00 is 03:00Z, so a record named 09:00Z is six hours LATER.
	read := ReviewAttemptsRead{Coverage: SourceCoverage{Losses: []SourceLoss{{Producer: "review-attempts", Artifact: "x", Kind: LossReadError, At: "2026-08-02T09:00:00Z", ID: "rvw-20260802T090000Z-bbbb2222"}}}}
	if got := read.HistoryUncertainAfter("spec:alpha", "2026-08-02T12:00:00+09:00", "rvw-a"); len(got) != 1 {
		t.Fatalf("a later loss was taken for an older one: %v", got)
	}
	if got := read.HistoryUncertainAfter("spec:alpha", "2026-08-02T18:00:00+09:00", "rvw-a"); len(got) != 0 {
		t.Fatalf("an earlier loss was not recognised: %v", got)
	}
	if got := read.HistoryUncertainAfter("spec:alpha", "2026-08-02T09:00:00.5Z", "rvw-a"); len(got) != 0 {
		t.Fatalf("a fraction of a second was ignored: %v", got)
	}
	if !reviewInstantLess("2026-08-02T12:00:00+09:00", "b", "2026-08-02T10:00:00Z", "a") {
		t.Fatal("attempts were not ordered by instant")
	}
}

// R3-F04: a structured line cut short is a loss, not an empty list.
func TestATruncatedFindingLineIsALoss(t *testing.T) {
	_, store := reviewFixture(t)
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260803T090000Z-ffff6666.md", "---\nschema_version: 1\nreview_id: rvw-20260803T090000Z-ffff6666\nscope: spec:alpha\ndecision: approved\nreviewed_at: 2026-08-03T09:00:00Z\n---\n\n## Findings\n\n- F1 [op")
	read, err := store.ReadReviewAttempts("")
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Attempts) != 0 || read.Coverage.Complete() || !strings.Contains(read.Coverage.Losses[0].Detail, "truncated") {
		t.Fatalf("attempts=%d coverage=%+v", len(read.Attempts), read.Coverage)
	}
}

// R3-F05: a link that stood for a spec folder and points nowhere is a lost spec.
func TestADanglingSpecFolderLinkIsALoss(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("gone", filepath.Join(root, ".pose", "specs", "2026-10-01-hidden")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if read := ReadSpecFollowups(root); read.Coverage.Complete() || read.Coverage.Losses[0].Scope != "hidden" {
		t.Fatalf("coverage = %+v", read.Coverage)
	}
}

// R3-F06: a reviews root that is there and cannot be resolved is not an empty history.
func TestADanglingReviewsRootIsALoss(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("gone", filepath.Join(root, ".pose", "reviews")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	read, err := (Store{Root: root}).ReadReviewAttempts("")
	if err != nil || read.Coverage.Complete() {
		t.Fatalf("coverage = %+v err=%v", read.Coverage, err)
	}
	if empty, _ := (Store{Root: t.TempDir()}).ReadReviewAttempts(""); !empty.Coverage.Complete() {
		t.Fatal("a repository that never had reviews is complete")
	}
}

// R3-F07: the reconciliation does not take a lost directory or an unprovable absence for none.
func TestTheDebtClassificationDoesNotTakeALostRootForAbsence(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, "pkg/sample.go", "package pkg\n// TODO: fix\n")
	if err := os.MkdirAll(filepath.Join(root, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("gone", filepath.Join(root, ".pose", "roadmaps")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	report, err := (Store{Root: root}).AnalyzeTechDebt()
	if err != nil || report.Complete {
		t.Fatalf("a dangling roadmaps root left the classification complete: %v %+v", err, report.Coverage)
	}
	if os.Geteuid() == 0 {
		return
	}
	root = t.TempDir()
	writeCoverageFile(t, root, "pkg/sample.go", "package pkg\n// TODO: fix\n")
	writeCoverageFile(t, root, ".pose/state/project-state.md", "# State\n")
	dir := filepath.Join(root, ".pose", "state")
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o755)
	report, err = (Store{Root: root}).AnalyzeTechDebt()
	if err != nil || report.Complete {
		t.Fatalf("an inaccessible state directory left the classification complete: %v %+v", err, report.Coverage)
	}
}

// R3-F09: a roadmap query inherits the losses of the milestones it contains.
func TestARoadmapQueryInheritsTheLossOfItsMilestone(t *testing.T) {
	store := obligationFixture(t)
	writeCoverageFile(t, store.Root, ".pose/roadmaps/delivery.md", "---\nslug: delivery\nstatus: active\n---\n\n# Roadmap: delivery\n\n## Milestone: m1\n- after:\n- target_start: 2026-07-01\n- target_due: 2026-07-10\n- specs: base\n")
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260101T000000Z-aaaa0000.md", "---\nschema_version: 9\nreview_id: x\nscope: milestone:delivery/m1\n---\n")
	got, err := store.ProjectObligations(ObligationQuery{Scope: "roadmap:delivery"})
	if err != nil {
		t.Fatal(err)
	}
	if cov := coverageOf(got, "findings"); cov.State != CoverageStateIncomplete {
		t.Fatalf("findings = %+v", cov)
	}
	_ = os.RemoveAll(filepath.Join(store.Root, ".pose/reviews"))
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260101T000000Z-aaaa0000.md", "---\nschema_version: 9\nreview_id: x\nscope: milestone:other/m1\n---\n")
	got, err = store.ProjectObligations(ObligationQuery{Scope: "roadmap:delivery"})
	if err != nil {
		t.Fatal(err)
	}
	if cov := coverageOf(got, "findings"); cov.State != CoverageStateCurrent {
		t.Fatalf("a milestone of another roadmap widened the query: %+v", cov)
	}
}

// R3-F10: the warning does not cut the table.
func TestTheIncompleteScanWarningDoesNotCutTheTable(t *testing.T) {
	store := Store{Root: t.TempDir()}
	state := &ComponentDiscoveryState{ComponentSlug: "svc", Path: "svc", Status: "in_progress", CompletenessScore: 1,
		Coverage: SourceCoverage{Losses: []SourceLoss{{Producer: "component-discovery", Artifact: "svc/a.go", Kind: LossReadError}}}}
	if err := store.GenerateConsolidatedAssessment([]*ComponentDiscoveryState{state}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(store.AssessmentsDir(), "consolidated.md"))
	text := string(raw)
	row, warn := strings.Index(text, "| 01 |"), strings.Index(text, "Varredura incompleta")
	if row < 0 || warn < 0 || warn < row {
		t.Fatalf("row=%d warning=%d:\n%s", row, warn, text)
	}
}

// R3-F03: the attempts of a scope come back in the order of their instants.
func TestAttemptsAreReadInInstantOrder(t *testing.T) {
	_, store := reviewFixture(t)
	_ = os.RemoveAll(filepath.Join(store.Root, ".pose/reviews"))
	doc := func(id, at string) string {
		return "---\nschema_version: 1\nreview_id: " + id + "\nscope: spec:alpha\ndecision: approved\nreviewed_at: " + at + "\n---\n"
	}
	// By text 2026-08-02T10:00:00Z < 2026-08-02T12:00:00+09:00; by instant it is the opposite.
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260802T030000Z-aaaa0001.md", doc("rvw-20260802T030000Z-aaaa0001", "2026-08-02T12:00:00+09:00"))
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260802T100000Z-aaaa0002.md", doc("rvw-20260802T100000Z-aaaa0002", "2026-08-02T10:00:00Z"))
	read, err := store.ReadReviewAttempts("")
	if err != nil || len(read.Attempts) != 2 {
		t.Fatalf("%v %v", read, err)
	}
	if read.Attempts[0].ReviewID != "rvw-20260802T030000Z-aaaa0001" {
		t.Fatalf("attempts were ordered by text: %s first", read.Attempts[0].ReviewID)
	}
}
