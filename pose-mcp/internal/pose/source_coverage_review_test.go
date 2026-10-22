package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-source-coverage-integrity: the cases the first independent review reproduced.

// F01: a closed scope keeps its approval, but not over an attestation history with a hole.
func TestAClosedScopeDoesNotRetainAnApprovalOverAnUnreadableRejection(t *testing.T) {
	store, bundle, _ := retentionFixture(t)
	closeBackendSpec(t, store)
	addMatrixCheck(t, store)
	before, err := store.VerifyReviewBundle("spec:backend")
	if err != nil || !before.Approved || before.State != "closed" {
		t.Fatalf("baseline: %+v %v", before, err)
	}
	writeCoverageFile(t, store.Root, ".pose/review-attestations/rva-00000000000000a1.json", `{"schema_version":1,"bundle_id":"`+bundle.BundleID+`","decision":"rej`)
	after, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if after.Approved || after.State == "closed" || !strings.Contains(strings.Join(after.Blockers, " "), "cannot be retained") {
		t.Fatalf("a retained approval stood over an unreadable record: approved=%v state=%s blockers=%v", after.Approved, after.State, after.Blockers)
	}
	eval, err := store.ReviewCheck("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if eval.Approved || !strings.Contains(strings.Join(eval.Blockers, " "), "cannot be retained") {
		t.Fatalf("review-check retained it: approved=%v blockers=%v", eval.Approved, eval.Blockers)
	}
}

// F02: only the structural path attributes a record; a decoy elsewhere proves nothing.
func TestSniffingAttributesOnlyTheStructuralPath(t *testing.T) {
	cases := []struct {
		name      string
		raw       string
		path      []string
		value     string
		found     bool
		ambiguous bool
	}{
		{"top-level key", `{"schema_version":1,"bundle_id":"rvb-real","decision":"rej`, []string{"bundle_id"}, "rvb-real", true, false},
		{"nested decoy first", `{"metadata":{"bundle_id":"rvb-decoy"},"bundle_id":"rvb-real","dec`, []string{"bundle_id"}, "rvb-real", true, false},
		{"only a nested decoy", `{"metadata":{"bundle_id":"rvb-decoy"},"decision":"rej`, []string{"bundle_id"}, "", false, false},
		{"duplicate key", `{"bundle_id":"rvb-a","bundle_id":"rvb-b","dec`, []string{"bundle_id"}, "", false, true},
		{"escaped key", `{"bundle_id":"rvb-real","dec`, []string{"bundle_id"}, "rvb-real", true, false},
		{"inside an array", `{"items":[{"bundle_id":"rvb-decoy"}],"dec`, []string{"bundle_id"}, "", false, false},
		{"nested path", `{"payload":{"x":1,"scope":{"ref":"spec:a","sec`, []string{"payload", "scope", "ref"}, "spec:a", true, false},
		{"wrong parent", `{"scope":{"ref":"spec:decoy"},"payload":{"scope":{"ref":"spec:real"`, []string{"payload", "scope", "ref"}, "spec:real", true, false},
		{"same depth, other parent", `{"meta":{"scope":{"ref":"spec:decoy"}},"dec`, []string{"payload", "scope", "ref"}, "", false, false},
		{"top-level scope only", `{"scope":{"ref":"spec:decoy"},"dec`, []string{"payload", "scope", "ref"}, "", false, false},
		{"truncated before the key", `{"schema_ver`, []string{"bundle_id"}, "", false, false},
	}
	for _, tc := range cases {
		value, found, ambiguous := sniffJSONString([]byte(tc.raw), tc.path...)
		if value != tc.value || found != tc.found || ambiguous != tc.ambiguous {
			t.Errorf("%s: got (%q, %v, %v), want (%q, %v, %v)", tc.name, value, found, ambiguous, tc.value, tc.found, tc.ambiguous)
		}
	}
}

func TestADecoyInATruncatedRecordDoesNotLetTheApprovalStand(t *testing.T) {
	store, bundle, _ := retentionFixture(t)
	// The decoy names another bundle inside metadata; the record's own bundle_id is this one.
	writeCoverageFile(t, store.Root, ".pose/review-attestations/rva-00000000000000b1.json", `{"metadata":{"bundle_id":"rvb-0123456789abcdef"},"bundle_id":"`+bundle.BundleID+`","decision":"rej`)
	v, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if v.Approved {
		t.Fatalf("a decoy let the earlier approval stand: %v", v.Blockers)
	}
	_ = os.Remove(filepath.Join(store.Root, ".pose/review-attestations/rva-00000000000000b1.json"))
	// With no real key at all the record is unattributed, and unattributed means any bundle.
	writeCoverageFile(t, store.Root, ".pose/review-attestations/rva-00000000000000b2.json", `{"metadata":{"bundle_id":"rvb-0123456789abcdef"},"decision":"rej`)
	v, err = store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if v.Approved {
		t.Fatal("an unattributed truncated record let the approval stand")
	}
}

// F03: valid JSON that names no bundle or scope is a record of unknown owner.
func TestValidJSONWithoutAHeaderIsALossNotAnotherBundle(t *testing.T) {
	store, bundle, _ := retentionFixture(t)
	writeCoverageFile(t, store.Root, ".pose/review-attestations/rva-00000000000000c1.json", `{"schema_version":1,"decision":"rejected"}`)
	v, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if v.Approved || !strings.Contains(strings.Join(v.Blockers, " "), "attestation history") {
		t.Fatalf("a record with no bundle_id was read as another bundle's: %v", v.Blockers)
	}
	read, err := store.ReadReviewAttestations(bundle.BundleID)
	if err != nil || read.Coverage.Complete() {
		t.Fatalf("coverage = %+v err=%v", read.Coverage, err)
	}
	writeCoverageFile(t, store.Root, ".pose/review-bundles/rvb-00000000000000c2.json", `{"schema_version":1,"payload":{"plan":{}}}`)
	bundles, err := store.ReadReviewBundles("spec:backend")
	if err != nil || bundles.Coverage.CompleteFor("spec:backend") {
		t.Fatalf("a bundle with no scope is not another scope's: %+v err=%v", bundles.Coverage, err)
	}
}

// F04: the same second proves nothing about order.
func TestAnUnreadableAttemptOfTheSameSecondIsNotProvablyOlder(t *testing.T) {
	_, store := reviewFixture(t)
	recordApprovedReview(t, store, "rvw-20260802T120000Z-aaaa1111", "spec:alpha", "spec-closeout@1", "requirements", "")
	danglingLink(t, store.Root, ".pose/reviews/rvw-20260802T120000Z-bbbb2222.md")
	eval, err := store.ReviewCheck("spec:alpha")
	if err != nil {
		t.Fatal(err)
	}
	if eval.Approved || !strings.Contains(strings.Join(eval.Blockers, " "), "review history for spec:alpha is incomplete") {
		t.Fatalf("a record of the same second was taken as older: approved=%v blockers=%v warnings=%v", eval.Approved, eval.Blockers, eval.Warnings)
	}
	// One that sorts before the selected attempt, in the same second, is provably older.
	_ = os.Remove(filepath.Join(store.Root, ".pose/reviews/rvw-20260802T120000Z-bbbb2222.md"))
	danglingLink(t, store.Root, ".pose/reviews/rvw-20260802T120000Z-0000aaaa.md")
	eval, err = store.ReviewCheck("spec:alpha")
	if err != nil || !eval.Approved {
		t.Fatalf("a record that sorts before the attempt blocked it: approved=%v blockers=%v err=%v", eval.Approved, eval.Blockers, err)
	}
}

// F05: an old malformed attempt with a readable scope is a loss, not a failure of the read.
func TestAnOldMalformedAttemptOfTheScopeOnlyWarns(t *testing.T) {
	_, store := reviewFixture(t)
	recordApprovedReview(t, store, "rvw-20260802T120000Z-aaaa1111", "spec:alpha", "spec-closeout@1", "requirements", "")
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260801T090000Z-bbbb2222.md", "---\nschema_version: 9\nreview_id: rvw-old\nscope: spec:alpha\n---\n")
	eval, err := store.ReviewCheck("spec:alpha")
	if err != nil {
		t.Fatalf("the read failed instead of reporting a loss: %v", err)
	}
	if !eval.Approved || !strings.Contains(strings.Join(eval.Warnings, " "), "predate the current attempt") {
		t.Fatalf("approved=%v blockers=%v warnings=%v", eval.Approved, eval.Blockers, eval.Warnings)
	}
	// A newer one of the scope blocks.
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260803T090000Z-cccc3333.md", "---\nschema_version: 9\nreview_id: rvw-new\nscope: spec:alpha\n---\n")
	eval, err = store.ReviewCheck("spec:alpha")
	if err != nil || eval.Approved {
		t.Fatalf("a newer malformed attempt did not block: approved=%v err=%v", eval.Approved, err)
	}
}

// F19: the global reader does not accept a record without the fields consumers select by.
func TestARecordWithoutItsRequiredFieldsIsALossInTheGlobalRead(t *testing.T) {
	_, store := reviewFixture(t)
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260803T090000Z-dddd4444.md", "---\nscope: spec:alpha\n")
	read, err := store.ReadReviewAttempts("")
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Attempts) != 0 || len(read.Coverage.Losses) != 1 || !strings.Contains(read.Coverage.Losses[0].Detail, "lacks") || read.Coverage.Losses[0].Scope != "spec:alpha" {
		t.Fatalf("attempts=%d coverage=%+v", len(read.Attempts), read.Coverage)
	}
}

// F06: findings of another scope do not make a local query incomplete.
func TestAFindingsLossInAnotherScopeDoesNotWidenALocalQuery(t *testing.T) {
	store := obligationFixture(t)
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260101T000000Z-aaaa0000.md", "---\nschema_version: 9\nreview_id: x\nscope: spec:elsewhere\n---\n")
	local, err := store.ProjectObligations(ObligationQuery{Scope: "spec:base"})
	if err != nil {
		t.Fatal(err)
	}
	if got := coverageOf(local, "findings"); got.State != CoverageStateCurrent {
		t.Fatalf("a loss attributed to another spec widened this query: %+v", got)
	}
	whole, err := store.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if got := coverageOf(whole, "findings"); got.State != CoverageStateIncomplete {
		t.Fatalf("the whole-project read lost the loss: %+v", got)
	}
	// Unattributed losses do concern a local query.
	danglingLink(t, store.Root, ".pose/reviews/rvw-20260102T000000Z-bbbb0000.md")
	local, err = store.ProjectObligations(ObligationQuery{Scope: "spec:base"})
	if err != nil {
		t.Fatal(err)
	}
	if got := coverageOf(local, "findings"); got.State != CoverageStateIncomplete {
		t.Fatalf("an unattributed loss does not concern the query: %+v", got)
	}
}

// F07: the component discovery reports what it could not read and does not certify it.
func TestComponentDiscoveryFindsAMarkerAfterALongLineAndSaysWhatItDidNotRead(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, "pkg/sample.go", "package pkg\n// "+strings.Repeat("x", 70000)+"\n// TODO: after a long line\n")
	store := Store{Root: root}
	state, err := store.DiscoverComponent(".")
	if err != nil {
		t.Fatal(err)
	}
	if state.TechnicalDebt.TODOs != 1 || !state.ScanComplete {
		t.Fatalf("todos=%d complete=%v", state.TechnicalDebt.TODOs, state.ScanComplete)
	}
	if os.Geteuid() == 0 {
		t.Skip("root reads unreadable files")
	}
	locked := filepath.Join(root, "pkg", "locked.go")
	writeCoverageFile(t, root, "pkg/locked.go", "package pkg\n// TODO: unreadable\n")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(locked, 0o644)
	state, err = store.DiscoverComponent(".")
	if err != nil {
		t.Fatal(err)
	}
	if state.ScanComplete || state.Status == "verified" || len(state.Coverage.Losses) != 1 || state.Coverage.Losses[0].Artifact != "pkg/locked.go" {
		t.Fatalf("an unreadable file left the component verified: status=%s complete=%v coverage=%+v", state.Status, state.ScanComplete, state.Coverage)
	}
}

// F08: directories and links that are not scanned are declared.
func TestSkippedDirectoriesAndLinksAreDeclaredExclusions(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, "build/sample.go", "package pkg\n// TODO: in a skipped directory\n")
	if err := os.Symlink("build/sample.go", filepath.Join(root, "linked.go")); err != nil {
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
	if report.ScannedFiles != 0 || patterns["skipped-directory"] != 1 || patterns["symlink"] != 1 {
		t.Fatalf("scanned=%d exclusions=%+v", report.ScannedFiles, report.Coverage.Exclusions)
	}
}

// F09: a folder that cannot be listed may hold a spec.
func TestAnUnlistableSpecFolderIsALossAttributedToIt(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root lists unlistable folders")
	}
	root := t.TempDir()
	writeCoverageFile(t, root, ".pose/specs/hidden/spec.md", "---\nslug: hidden\nstatus: done\n---\n\n## 7. Final Report\n\n### Follow-ups\n\n- [open] Hidden repair (owner:@core crit:low review:2099-01-01)\n")
	dir := filepath.Join(root, ".pose", "specs", "hidden")
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o755)
	read := ReadSpecFollowups(root)
	if len(read.Items) != 0 || read.Coverage.Complete() || read.Coverage.Losses[0].Scope != "hidden" || read.Coverage.Losses[0].Kind != LossReadError {
		t.Fatalf("items=%d coverage=%+v", len(read.Items), read.Coverage)
	}
}

// F10: the legacy split layout is read, not lost.
func TestTheLegacySplitLayoutIsReadForFollowups(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, ".pose/specs/old/intent.md", "# Spec: old\n")
	writeCoverageFile(t, root, ".pose/specs/old/final-report.md", "## 7. Final Report\n\n### Follow-ups\n\n- [open] Split repair (owner:@core crit:low review:2099-01-01)\n")
	read := ReadSpecFollowups(root)
	if len(read.Items) != 1 || read.Items[0].Spec != "old" || !read.Items[0].Unresolved() || !read.Coverage.Complete() {
		t.Fatalf("items=%+v coverage=%+v", read.Items, read.Coverage)
	}
}

// F11: a truncated header is a loss and an unknown disposition is not a resolution.
func TestATruncatedHeaderIsALossAndAnUnknownDispositionIsUnresolved(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, ".pose/specs/2026-10-01-cut.md", "---\nslug: cut\nstatus: done\n")
	writeCoverageFile(t, root, ".pose/specs/2026-10-02-typo.md", "---\nslug: typo\nstatus: done\n---\n\n## 7. Final Report\n\n### Follow-ups\n\n- [opne] Mistyped disposition (owner:@core crit:low review:2099-01-01)\n- [done] Fine (owner:@core crit:low review:2099-01-01)\n")
	read := ReadSpecFollowups(root)
	if read.Coverage.CompleteFor("cut") || read.Coverage.Losses[0].Kind != LossParseError {
		t.Fatalf("a truncated header certified complete: %+v", read.Coverage)
	}
	var typo *SpecFollowup
	for i := range read.Items {
		if read.Items[i].Spec == "typo" && read.Items[i].Ordinal == 1 {
			typo = &read.Items[i]
		}
	}
	if typo == nil || !typo.Unresolved() || !strings.Contains(typo.DispositionErr, "opne") {
		t.Fatalf("typo = %+v", typo)
	}
	if !FollowupUnresolved("opne") || FollowupUnresolved("done") || FollowupUnresolved("wont-do") || !FollowupUnresolved("") {
		t.Fatal("FollowupUnresolved")
	}
}

// F12: what the parser says about a bullet reaches Attention.
func TestAttentionCarriesTheFollowupDiagnostics(t *testing.T) {
	store := obligationFixture(t)
	writeCoverageFile(t, store.Root, ".pose/specs/2026-10-04-meta.md", "---\nslug: meta\nstatus: done\n---\n\n## 7. Final Report\n\n### Follow-ups\n\n- [open] Bad crit (owner:@core crit:urgent review:2099-01-01)\n- [opne] Typo (owner:@core crit:low review:2099-01-01)\n")
	report, err := store.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	var meta, typo *Obligation
	for i := range report.Obligations {
		o := &report.Obligations[i]
		if o.Source.Producer != "followups" {
			continue
		}
		if strings.Contains(o.Message, "Bad crit") {
			meta = o
		}
		if strings.Contains(o.Message, "Typo") {
			typo = o
		}
	}
	if meta == nil || !strings.Contains(strings.Join(meta.Observation.Limitations, " "), "ownership metadata is malformed") {
		t.Fatalf("metadata diagnostic lost: %+v", meta)
	}
	if typo == nil || typo.ReasonCode != "followup-invalid-disposition" || !strings.Contains(strings.Join(typo.Observation.Limitations, " "), "unknown disposition") {
		t.Fatalf("disposition diagnostic lost: %+v", typo)
	}
}

// F15: the debt classification is only as complete as the backlog it read.
func TestTheDebtClassificationSaysWhenItCouldNotReadTheBacklog(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, "pkg/sample.go", "package pkg\n// TODO: fix\n")
	danglingLink(t, root, ".pose/specs/2026-10-01-unreadable.md")
	report, err := (Store{Root: root}).AnalyzeTechDebt()
	if err != nil {
		t.Fatal(err)
	}
	if report.Complete || report.Summary.UncoveredCount != 1 {
		t.Fatalf("complete=%v uncovered=%d: the classification claimed a backlog it did not read", report.Complete, report.Summary.UncoveredCount)
	}
	found := false
	for _, l := range report.Coverage.Losses {
		found = found || l.Producer == "techdebt-reconciliation"
	}
	if !found {
		t.Fatalf("losses = %+v", report.Coverage.Losses)
	}
}

// F16: the consumers that state a history say what they could not read.
func TestGovernanceConsumersDiscloseTheRecordsTheyCouldNotRead(t *testing.T) {
	store, bundle, _ := retentionFixture(t)
	writeCoverageFile(t, store.Root, ".pose/review-attestations/rva-00000000000000d1.json", `{"bundle_id":"`+bundle.BundleID+`","decision":"rej`)
	rework, err := store.GovernanceRework()
	if err != nil {
		t.Fatal(err)
	}
	if rework.Complete || rework.Coverage.Complete() || !strings.Contains(strings.Join(rework.Limitations, " "), "incomplete") {
		t.Fatalf("rework = %+v", rework)
	}
	inventory, err := store.InventoryLegacy()
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Complete || !strings.Contains(strings.Join(inventory.Limitations, " "), "incomplete") {
		t.Fatalf("inventory complete=%v limitations=%v", inventory.Complete, inventory.Limitations)
	}
	writeCoverageFile(t, store.Root, ".pose/review-bundles/rvb-00000000000000d2.json", `{"payload":{"scope":{"ref":"spec:backend","tr`)
	effective, err := store.EffectiveGovernance("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if effective.ScopeBundle == nil || effective.ScopeBundle.HistoryIncomplete == "" {
		t.Fatalf("effective governance hides an unreadable bundle: %+v", effective.ScopeBundle)
	}
}

// F17: an unreadable ledger is not the same as no ledger.
func TestAnUnreadableLegacyLedgerIsNotAbsence(t *testing.T) {
	store := newIndexStore(t) // any empty store
	if !store.LegacyLedgerCoverage().Complete() {
		t.Fatal("no ledger directory is not a loss")
	}
	writeCoverageFile(t, store.Root, legacyLedgerDir+"/broken.json", `{"schema_version":1,"kind":"legacy-attest`)
	writeCoverageFile(t, store.Root, legacyLedgerDir+"/future.json", `{"schema_version":99,"kind":"legacy-attestations"}`)
	coverage := store.LegacyLedgerCoverage()
	kinds := map[string]bool{}
	for _, l := range coverage.Losses {
		kinds[l.Kind] = true
	}
	if coverage.Complete() || !kinds[LossParseError] || !kinds[LossUnsupported] {
		t.Fatalf("coverage = %+v", coverage)
	}
	entries, err := store.trustedLegacyEntries()
	if err != nil || len(entries) != 0 {
		t.Fatalf("an unusable ledger was trusted: %v %v", entries, err)
	}
}

// F18: an element whose producer field is empty is still a loss of the producer that returned it.
func TestAnInvalidElementWithoutAProducerIsStillThatProducersLoss(t *testing.T) {
	store := obligationFixture(t)
	defer func(previous func(Store, string, []Spec, bool, *Coverage) ([]Obligation, error)) {
		actionRequestObligationSource = previous
	}(actionRequestObligationSource)
	actionRequestObligationSource = func(s Store, project string, specs []Spec, _ bool, _ *Coverage) ([]Obligation, error) {
		o := newObligation(project, "action-requests", specNode(project, "base"), "action-request", "x")
		o.Source.Producer = ""
		return []Obligation{o}, nil
	}
	report, err := store.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if cov := coverageOf(report, "action-requests"); cov.State != CoverageStateIncomplete || report.Complete {
		t.Fatalf("coverage = %+v complete=%v", cov, report.Complete)
	}
}
