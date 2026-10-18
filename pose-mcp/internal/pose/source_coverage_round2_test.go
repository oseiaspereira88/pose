package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-source-coverage-integrity: the cases the second independent review reproduced.

// R2-F01: a rejection that cannot be loaded (its digest no longer matches) is not skipped.
func TestARejectionThatFailsItsDigestBlocksTheRetainedApproval(t *testing.T) {
	store, _, now := retentionFixture(t)
	addMatrixCheck(t, store)
	second, err := store.SealReviewBundle("spec:backend", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	rejected := approvedBundleAttestation(second, "agent:second-review")
	rejected.Decision = "rejected"
	recorded, err := store.RecordReviewAttestation(rejected, now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Root, ".pose/review-attestations", recorded.AttestationID+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), "agent:second-review", "agent:someone-else", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	closeBackendSpec(t, store)
	writeReviewFixture(t, store.Root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"moduleOverrides":{"api":{"checks":[{"name":"another-check","program":"true","severity":"required","evidenceClass":"unit"}]}}}`)
	v, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if v.Approved {
		t.Fatalf("an older approval stood over a rejection that could not be loaded: %v", v.Blockers)
	}
}

// R2-F02: a repeated name in valid JSON does not settle which bundle or scope a record is.
func TestARepeatedHeaderNameInValidJSONIsALossNotAnotherBundle(t *testing.T) {
	store, bundle, _ := retentionFixture(t)
	writeCoverageFile(t, store.Root, ".pose/review-attestations/rva-00000000000000e1.json", `{"bundle_id":"`+bundle.BundleID+`","bundle_id":"rvb-0123456789abcdef","decision":"rejected"}`)
	read, err := store.ReadReviewAttestations(bundle.BundleID)
	if err != nil || read.Coverage.Complete() {
		t.Fatalf("a repeated bundle_id was filtered out as another bundle's: %+v err=%v", read.Coverage, err)
	}
	writeCoverageFile(t, store.Root, ".pose/review-bundles/rvb-00000000000000e2.json", `{"payload":{"scope":{"ref":"spec:backend","ref":"spec:other"}}}`)
	bundles, err := store.ReadReviewBundles("spec:backend")
	if err != nil || bundles.Coverage.CompleteFor("spec:backend") {
		t.Fatalf("a repeated scope was filtered out: %+v err=%v", bundles.Coverage, err)
	}
}

// R2-F03: a name seen twice makes the prefix ambiguous whatever follows it.
func TestSniffingSeesARepeatedNameBeforeItsValue(t *testing.T) {
	for name, raw := range map[string]string{
		"null":      `{"bundle_id":"rvb-a","bundle_id":null,"dec`,
		"number":    `{"bundle_id":"rvb-a","bundle_id":7,"dec`,
		"cut":       `{"bundle_id":"rvb-a","bundle_id":"rvb-b`,
		"key only":  `{"bundle_id":"rvb-a","bundle_id"`,
		"ancestors": `{"payload":{"x":1},"payload":{"scope":{"ref":"spec:a"}}`,
	} {
		path := []string{"bundle_id"}
		if name == "ancestors" {
			path = []string{"payload", "scope", "ref"}
		}
		if _, _, ambiguous := sniffJSONString([]byte(raw), path...); !ambiguous {
			t.Errorf("%s: a repeated name was not ambiguous", name)
		}
	}
}

// R2-F04: an entry that is there and cannot be resolved is a lost spec, not an absent one.
func TestAnUnresolvableSpecEntryIsALossNotAbsence(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, ".pose/specs/split/intent.md", "# Spec: split\n")
	danglingLink(t, root, ".pose/specs/split/final-report.md")
	if read := ReadSpecFollowups(root); read.Coverage.CompleteFor("split") || len(read.Coverage.Losses) == 0 {
		t.Fatalf("a dangling section was taken for an absent one: %+v", read.Coverage)
	}
	root = t.TempDir()
	writeCoverageFile(t, root, ".pose/specs/folder/intent.md", "# Spec: folder\n")
	danglingLink(t, root, ".pose/specs/folder/spec.md")
	if read := ReadSpecFollowups(root); read.Coverage.CompleteFor("folder") {
		t.Fatalf("a dangling spec.md was hidden by the split fallback: %+v", read.Coverage)
	}
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("nowhere", filepath.Join(root, ".pose", "specs")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if read := ReadSpecFollowups(root); read.Coverage.Complete() {
		t.Fatalf("a dangling specs directory was taken for an absent one: %+v", read.Coverage)
	}
	if read := ReadSpecFollowups(t.TempDir()); !read.Coverage.Complete() {
		t.Fatalf("a repository with no specs directory is complete: %+v", read.Coverage)
	}
}

// R2-F05: a loss attributed to the milestone or roadmap asked about concerns that query.
func TestAFindingsLossOfTheRequestedMilestoneConcernsIt(t *testing.T) {
	store := obligationFixture(t)
	writeCoverageFile(t, store.Root, ".pose/roadmaps/delivery.md", "---\nslug: delivery\nstatus: active\n---\n\n# Roadmap: delivery\n\n## Milestone: m1\n- after:\n- target_start: 2026-07-01\n- target_due: 2026-07-10\n- specs: base\n")
	check := func(name, doc string) {
		_ = os.RemoveAll(filepath.Join(store.Root, ".pose/reviews"))
		writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260101T000000Z-aaaa0000.md", doc)
		got, err := store.ProjectObligations(ObligationQuery{Scope: "milestone:delivery/m1"})
		if err != nil {
			t.Fatal(err)
		}
		if cov := coverageOf(got, "findings"); cov.State != CoverageStateIncomplete {
			t.Fatalf("%s: a loss attributed to it left findings current: %+v", name, cov)
		}
	}
	check("the milestone", "---\nschema_version: 9\nreview_id: x\nscope: milestone:delivery/m1\n---\n")
	check("its roadmap", "---\nschema_version: 9\nreview_id: y\nscope: roadmap:delivery\n---\n")
	_ = os.RemoveAll(filepath.Join(store.Root, ".pose/reviews"))
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260102T000000Z-bbbb0000.md", "---\nschema_version: 9\nreview_id: z\nscope: milestone:delivery/other\n---\n")
	other, err := store.ProjectObligations(ObligationQuery{Scope: "spec:base"})
	if err != nil {
		t.Fatal(err)
	}
	if cov := coverageOf(other, "findings"); cov.State != CoverageStateCurrent {
		t.Fatalf("losses of other scopes widened an unrelated query: %+v", cov)
	}
}

// R2-F07: every link is declared, a directory or a module included.
func TestADirectorySymlinkIsADeclaredExclusion(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, "real/sample.go", "package pkg\n// TODO: behind a link\n")
	if err := os.Symlink("real", filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	report, err := (Store{Root: root}).AnalyzeTechDebt()
	if err != nil {
		t.Fatal(err)
	}
	patterns := map[string]int{}
	for _, e := range report.Coverage.Exclusions {
		patterns[e.Pattern] += e.Count
	}
	if patterns["symlink"] != 1 {
		t.Fatalf("exclusions = %+v", report.Coverage.Exclusions)
	}
	state, err := (Store{Root: root}).DiscoverComponent(".")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range state.Coverage.Exclusions {
		found = found || e.Pattern == "symlink"
	}
	if !found {
		t.Fatalf("discovery did not declare the link: %+v", state.Coverage)
	}
}

// R2-F09: roadmaps and the project state the reconciliation could not read are losses.
func TestTheDebtClassificationSaysWhenARoadmapOrTheProjectStateIsUnreadable(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, "pkg/sample.go", "package pkg\n// TODO: fix\n")
	danglingLink(t, root, ".pose/roadmaps/plan.md")
	report, err := (Store{Root: root}).AnalyzeTechDebt()
	if err != nil {
		t.Fatal(err)
	}
	if report.Complete {
		t.Fatalf("a roadmap that could not be read left the classification complete: %+v", report.Coverage)
	}
	root = t.TempDir()
	writeCoverageFile(t, root, "pkg/sample.go", "package pkg\n// TODO: fix\n")
	danglingLink(t, root, ".pose/state/project-state.md")
	report, err = (Store{Root: root}).AnalyzeTechDebt()
	if err != nil {
		t.Fatal(err)
	}
	if report.Complete {
		t.Fatalf("a project state that could not be read left the classification complete: %+v", report.Coverage)
	}
}

// R2-F10: a ledger directory that cannot be used is the loss, not an empty coverage.
func TestAnUnreadableLedgerDirectoryIsALoss(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads unreadable directories")
	}
	store := newIndexStore(t)
	writeCoverageFile(t, store.Root, legacyLedgerDir+"/a.json", `{}`)
	dir := filepath.Join(store.Root, filepath.FromSlash(legacyLedgerDir))
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o755)
	if store.LegacyLedgerCoverage().Complete() {
		t.Fatal("an unreadable ledger directory reported complete coverage")
	}
}

// R2-F11: a record is an attempt only when the values the history is selected by are valid.
func TestAReviewRecordWithInvalidValuesIsALoss(t *testing.T) {
	_, store := reviewFixture(t)
	for name, doc := range map[string]string{
		"decision":  "---\nschema_version: 1\nreview_id: x1\nscope: spec:alpha\ndecision: reje\nreviewed_at: 2026-08-03T09:00:00Z\n---\n",
		"timestamp": "---\nschema_version: 1\nreview_id: x2\nscope: spec:alpha\ndecision: rejected\nreviewed_at: unknown\n---\n",
		"unclosed":  "---\nschema_version: 1\nreview_id: x3\nscope: spec:alpha\ndecision: rejected\nreviewed_at: 2026-08-03T09:00:00Z\n",
	} {
		_ = os.RemoveAll(filepath.Join(store.Root, ".pose/reviews"))
		writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260803T090000Z-eeee5555.md", doc)
		read, err := store.ReadReviewAttempts("")
		if err != nil {
			t.Fatal(err)
		}
		if len(read.Attempts) != 0 || read.Coverage.Complete() {
			t.Errorf("%s: attempts=%d coverage=%+v", name, len(read.Attempts), read.Coverage)
		}
	}
}

// R2-F12: the unsigned set a ledger declares is not planned over a history with a hole.
func TestALegacyLedgerIsNotPlannedOverAnIncompleteHistory(t *testing.T) {
	f := ledgerFixtureWithHistory(t)
	writeCoverageFile(t, f.root, ".pose/review-attestations/rva-00000000000000f1.json", `{"bundle_id":"`+f.bundle.BundleID+`","decision":"rej`)
	if _, err := f.store.UnsignedAttestations(); err == nil || !strings.Contains(err.Error(), "cannot be read completely") {
		t.Fatalf("UnsignedAttestations err = %v", err)
	}
	if _, err := f.store.PlanLegacyLedger(f.key, f.now); err == nil {
		t.Fatal("a ledger was planned over an incomplete attestation history")
	}
}

// R2-F06: the aggregate report does not certify a measure taken over a lost scan.
func TestTheConsolidatedAssessmentDeclaresAnIncompleteScan(t *testing.T) {
	root := t.TempDir()
	store := Store{Root: root}
	state := &ComponentDiscoveryState{ComponentSlug: "svc", Path: "svc", Status: "in_progress", CompletenessScore: 1,
		Coverage: SourceCoverage{Losses: []SourceLoss{{Producer: "component-discovery", Artifact: "svc/a.go", Kind: LossReadError, Detail: "denied"}}}}
	if err := store.GenerateConsolidatedAssessment([]*ComponentDiscoveryState{state}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(store.AssessmentsDir(), "consolidated.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Varredura incompleta") || !strings.Contains(string(raw), "svc") {
		t.Fatalf("the consolidated report certified a lost scan:\n%s", raw)
	}
	complete := &ComponentDiscoveryState{ComponentSlug: "ok", Path: "ok", Status: "verified", CompletenessScore: 1}
	if err := store.GenerateConsolidatedAssessment([]*ComponentDiscoveryState{complete}); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(filepath.Join(store.AssessmentsDir(), "consolidated.md"))
	if strings.Contains(string(raw), "Varredura incompleta") {
		t.Fatal("a complete scan was reported incomplete")
	}
}
