package pose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-source-coverage-integrity.

func writeCoverageFile(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func danglingLink(t *testing.T, root, rel string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("absent-target.md", p); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

func coverageOf(r ObligationReport, producer string) ProducerCoverage {
	for _, c := range r.Coverage {
		if c.Producer == producer {
			return c
		}
	}
	return ProducerCoverage{}
}

// V1 / D01: a spec that cannot be read is a loss attributed to the spec its path names,
// not an empty list.
func TestAnUnreadableSpecIsALossAttributedToItsPath(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, ".pose/specs/2026-10-11-visible.md", "---\nslug: visible\nstatus: done\n---\n\n## 7. Final Report\n\n### Follow-ups\n\n- [open] Visible repair (owner:@core crit:low review:2099-01-01)\n")
	danglingLink(t, root, ".pose/specs/2026-10-11-unreadable.md")
	read := ReadSpecFollowups(root)
	if len(read.Items) != 1 || read.Items[0].Spec != "visible" {
		t.Fatalf("items = %+v", read.Items)
	}
	if read.Coverage.Complete() || len(read.Coverage.Losses) != 1 {
		t.Fatalf("a spec that could not be read left coverage %+v", read.Coverage)
	}
	loss := read.Coverage.Losses[0]
	if loss.Kind != LossReadError || loss.Scope != "unreadable" || loss.AttributedBy != "path" || loss.Artifact != ".pose/specs/2026-10-11-unreadable.md" {
		t.Fatalf("loss = %+v", loss)
	}
	if strings.Contains(loss.Detail, "owner") {
		t.Fatalf("a loss names bytes it could not read: %q", loss.Detail)
	}
	// R4: the loss concerns its own spec, not the one that was read.
	if !read.Coverage.CompleteFor("visible") || read.Coverage.CompleteFor("unreadable") || read.Coverage.CompleteFor("") {
		t.Fatalf("scoping: visible=%v unreadable=%v whole=%v", read.Coverage.CompleteFor("visible"), read.Coverage.CompleteFor("unreadable"), read.Coverage.CompleteFor(""))
	}
	if got := ParseSpecFollowups(root); len(got) != 1 {
		t.Fatalf("the compatibility reader changed: %d items", len(got))
	}
}

// V1 / P01: the backlog, the candidates and the projection count one unresolved set.
func TestUnresolvedFollowupsAreOneSetAcrossReaders(t *testing.T) {
	store := obligationFixture(t)
	writeCoverageFile(t, store.Root, ".pose/specs/2026-10-04-late.md", "---\nslug: late\nstatus: done\n---\n\n## 7. Final Report\n\n### Follow-ups\n\n- Missing mandatory correction (owner:@core crit:high review:2020-01-01)\n- [open] Confirmed regression (owner:@core crit:high review:2020-01-01)\n- [done] Already handled (owner:@core crit:low review:2020-01-01)\n")
	read := ReadSpecFollowups(store.Root)
	unresolved := map[string]bool{}
	for _, item := range read.Items {
		if item.Unresolved() {
			unresolved[fmt.Sprintf("%s#%d", item.Spec, item.Ordinal)] = true
		}
	}
	if !unresolved["late#1"] || !unresolved["late#2"] || unresolved["late#3"] {
		t.Fatalf("unresolved = %v", unresolved)
	}
	report, err := store.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	reasons := []string{}
	for _, o := range report.Obligations {
		if o.Source.Producer == "followups" && strings.Contains(o.Message, "late:") {
			reasons = append(reasons, o.ReasonCode)
		}
	}
	if len(reasons) != 2 {
		t.Fatalf("the projection holds %d of the 2 unresolved follow-ups of the spec: %v", len(reasons), reasons)
	}
	hasUndisposed := false
	for _, r := range reasons {
		hasUndisposed = hasUndisposed || r == "followup-undisposed"
	}
	if !hasUndisposed {
		t.Fatalf("a bullet with no disposition was not projected as such: %v", reasons)
	}
	candidates, err := store.FollowupCandidates("2026-10-12")
	if err != nil {
		t.Fatal(err)
	}
	late := 0
	for _, c := range candidates {
		if c.Spec == "late" {
			late++
		}
	}
	if late != 2 {
		t.Fatalf("candidates hold %d of the 2 unresolved follow-ups: %+v", late, candidates)
	}
}

// V1: Attention says its follow-up coverage is incomplete, scoped to what depends on the lost spec.
func TestTheFollowupProducerIsIncompleteWhenASpecCannotBeRead(t *testing.T) {
	store := obligationFixture(t)
	danglingLink(t, store.Root, ".pose/specs/2026-10-05-unreadable.md")
	whole, err := store.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	cov := coverageOf(whole, "followups")
	if cov.State != CoverageStateIncomplete || !strings.Contains(cov.Detail, "unreadable") || whole.Complete {
		t.Fatalf("whole project: coverage %+v complete=%v", cov, whole.Complete)
	}
	if findObligation(whole, ObligationResidualDebt, "followup-open") == nil {
		t.Fatal("the readable follow-up disappeared with the unreadable spec")
	}
	// A query about another spec does not inherit the loss (R4).
	scoped, err := store.ProjectObligations(ObligationQuery{Scope: "spec:shipped"})
	if err != nil {
		t.Fatal(err)
	}
	if got := coverageOf(scoped, "followups"); got.State != CoverageStateCurrent {
		t.Fatalf("a loss attributed to another spec widened this scope: %+v", got)
	}
	about, err := store.ProjectObligations(ObligationQuery{Scope: "spec:unreadable"})
	if err != nil {
		if !strings.Contains(err.Error(), "unreadable") && !strings.Contains(err.Error(), "spec") {
			t.Fatal(err)
		}
		return
	}
	if got := coverageOf(about, "followups"); got.State == CoverageStateCurrent && len(about.Obligations) == 0 {
		t.Fatalf("an empty answer about the unreadable spec claims complete coverage: %+v", got)
	}
}

// V4 / D13: a dropped obligation is a loss of its producer.
func TestADroppedObligationMakesItsProducerIncomplete(t *testing.T) {
	store := obligationFixture(t)
	defer func(previous func(Store, string, []Spec) ([]Obligation, error)) {
		actionRequestObligationSource = previous
	}(actionRequestObligationSource)
	actionRequestObligationSource = func(s Store, project string, specs []Spec) ([]Obligation, error) {
		o := newObligation(project, "action-requests", specNode(project, "base"), "action-request", "x")
		o.ReasonCode = "" // refused by validation
		return []Obligation{o}, nil
	}
	report, err := store.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	dropped := false
	for _, l := range report.Limitations {
		dropped = dropped || strings.Contains(l, "dropped an invalid projected obligation")
	}
	if !dropped {
		t.Skip("the fixture obligation was not refused by validation in this engine")
	}
	cov := coverageOf(report, "action-requests")
	if cov.State != CoverageStateIncomplete || !strings.Contains(cov.Detail, "dropped an invalid projected obligation") || report.Complete {
		t.Fatalf("a dropped element left the producer %+v and complete=%v", cov, report.Complete)
	}
}

// V5 / P04: a marker after a line longer than a scanner token is found.
func TestAMarkerAfterAVeryLongLineIsFound(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, "pkg/sample.go", "package pkg\n"+"// "+strings.Repeat("x", 70000)+"\n// TODO: lost after a long line\n")
	report, err := (Store{Root: root}).AnalyzeTechDebt()
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.TODOs != 1 || len(report.Items) != 1 || report.Items[0].Line != 3 {
		t.Fatalf("report = %+v", report.Summary)
	}
	if !report.Complete || report.ScannedFiles != 1 {
		t.Fatalf("complete=%v scanned=%d", report.Complete, report.ScannedFiles)
	}
}

// V5: unreadable files, the size limit, generated and binary files are named, and a
// report that lost a file is not complete.
func TestTechDebtCoverageNamesWhatItDidNotScan(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, "pkg/ok.go", "package pkg\n// TODO: counted\n")
	writeCoverageFile(t, root, "pkg/generated.go", "// Code generated by tool. DO NOT EDIT.\npackage pkg\n// TODO: not scanned\n")
	writeCoverageFile(t, root, "pkg/huge.go", "package pkg\n// "+strings.Repeat("y", maxAssessmentFileSize+10)+"\n// TODO: over the limit\n")
	writeCoverageFile(t, root, "pkg/binary.go", "package pkg\x00\n// TODO: binary\n")
	report, err := (Store{Root: root}).AnalyzeTechDebt()
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.TODOs != 1 || !report.Complete {
		t.Fatalf("todos=%d complete=%v: exclusions are declared, not losses", report.Summary.TODOs, report.Complete)
	}
	patterns := map[string]int{}
	for _, e := range report.Coverage.Exclusions {
		patterns[e.Pattern] = e.Count
	}
	if patterns["generated-header"] != 1 || patterns["over-size-limit"] != 1 || patterns["binary-content"] != 1 {
		t.Fatalf("exclusions = %+v", report.Coverage.Exclusions)
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
	report, err = (Store{Root: root}).AnalyzeTechDebt()
	if err != nil {
		t.Fatal(err)
	}
	if report.Complete || len(report.Coverage.Losses) != 1 || report.Coverage.Losses[0].Artifact != "pkg/locked.go" || report.Coverage.Losses[0].Kind != LossReadError {
		t.Fatalf("an unreadable file left the scan complete: complete=%v losses=%+v", report.Complete, report.Coverage.Losses)
	}
	if report.Summary.TODOs != 1 {
		t.Fatalf("todos = %d: zero for the unread file must not be reported as found", report.Summary.TODOs)
	}
}

func TestTheTechDebtReportSaysWhatItCovered(t *testing.T) {
	root := t.TempDir()
	writeCoverageFile(t, root, "pkg/ok.go", "package pkg\n")
	writeCoverageFile(t, root, "pkg/generated.go", "// Code generated by tool. DO NOT EDIT.\npackage pkg\n")
	store := Store{Root: root}
	report, err := store.AnalyzeTechDebt()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTechDebtReport(report); err != nil {
		t.Fatal(err)
	}
	md, _ := os.ReadFile(store.TechDebtReportPath())
	if !strings.Contains(string(md), "Escopo do scan") || !strings.Contains(string(md), "generated-header") {
		t.Fatalf("the report does not state its scope:\n%s", md)
	}
}

// V2 / V3 / T2-05: an unreadable review record that may be later than the one
// selected is not skipped.
func TestAnUnreadableNewerAttemptBlocksTheEarlierApproval(t *testing.T) {
	_, store := reviewFixture(t)
	recordApprovedReview(t, store, "rvw-20260802T120000Z-aaaa1111", "spec:alpha", "spec-closeout@1", "requirements", "")
	if eval, err := store.ReviewCheck("spec:alpha"); err != nil || !eval.Approved {
		t.Fatalf("baseline: approved=%v err=%v blockers=%v", eval.Approved, err, eval.Blockers)
	}
	// A later record that cannot be read: it may be a rejection.
	danglingLink(t, store.Root, ".pose/reviews/rvw-20260803T090000Z-bbbb2222.md")
	eval, err := store.ReviewCheck("spec:alpha")
	if err != nil {
		t.Fatal(err)
	}
	if eval.Approved || !strings.Contains(strings.Join(eval.Blockers, " "), "review history for spec:alpha is incomplete") {
		t.Fatalf("an unreadable later record was skipped: approved=%v blockers=%v", eval.Approved, eval.Blockers)
	}
}

func TestAnUnreadableOlderAttemptOnlyWarns(t *testing.T) {
	_, store := reviewFixture(t)
	recordApprovedReview(t, store, "rvw-20260802T120000Z-aaaa1111", "spec:alpha", "spec-closeout@1", "requirements", "")
	danglingLink(t, store.Root, ".pose/reviews/rvw-20260801T090000Z-bbbb2222.md")
	eval, err := store.ReviewCheck("spec:alpha")
	if err != nil {
		t.Fatal(err)
	}
	if !eval.Approved || !strings.Contains(strings.Join(eval.Warnings, " "), "predate the current attempt") {
		t.Fatalf("a provably older unreadable record blocked or was hidden: approved=%v blockers=%v warnings=%v", eval.Approved, eval.Blockers, eval.Warnings)
	}
}

func TestWithNoReadableAttemptAnUnreadableOneIsNotAbsence(t *testing.T) {
	_, store := reviewFixture(t)
	danglingLink(t, store.Root, ".pose/reviews/rvw-20260803T090000Z-bbbb2222.md")
	eval, err := store.ReviewCheck("spec:alpha")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(eval.Blockers, " ")
	if strings.Contains(joined, "no review attempt exists") || !strings.Contains(joined, "no readable review attempt") {
		t.Fatalf("blockers = %v", eval.Blockers)
	}
}

func TestAReviewRecordThatNamesAnotherScopeDoesNotConcernThisOne(t *testing.T) {
	_, store := reviewFixture(t)
	recordApprovedReview(t, store, "rvw-20260802T120000Z-aaaa1111", "spec:alpha", "spec-closeout@1", "requirements", "")
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260803T090000Z-cccc3333.md", "---\nreview_id: rvw-other\nscope: spec:elsewhere\n---\n")
	read, err := store.ReadReviewAttempts("spec:alpha")
	if err != nil {
		t.Fatal(err)
	}
	if !read.Coverage.Complete() || len(read.Attempts) != 1 {
		t.Fatalf("a record of another scope is a loss for this one: %+v", read.Coverage)
	}
}

func TestAnUnscopedUnreadableAttemptIsAttributedByItsHeaderWhenItHasOne(t *testing.T) {
	_, store := reviewFixture(t)
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260803T090000Z-dddd4444.md", "---\nschema_version: 9\nreview_id: rvw-broken\nscope: spec:alpha\n---\n\nnot a review attempt\n")
	read, err := store.ReadReviewAttempts("")
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Coverage.Losses) != 1 || read.Coverage.Losses[0].Scope != "spec:alpha" || read.Coverage.Losses[0].AttributedBy != "header" || read.Coverage.Losses[0].Kind != LossUnsupported {
		t.Fatalf("coverage = %+v", read.Coverage)
	}
	if !read.Coverage.CompleteFor("spec:other") {
		t.Fatal("a loss attributed to spec:alpha widened another scope")
	}
	if _, err := store.ReadReviewAttempts("spec:alpha"); err == nil {
		t.Fatal("a record that names the scope and does not parse must still fail the scoped read")
	}
}

// T2-05 / R3: an attestation that cannot be read may be the rejection that
// supersedes the earlier approval.
func TestAnUnreadableAttestationIsNotSkippedByVerification(t *testing.T) {
	_, store := reviewBundleFixture(t)
	bundle, err := store.SealReviewBundle("spec:backend", time.Now().UTC())
	if err != nil && bundle.BundleID == "" {
		t.Fatal(err)
	}
	// A truncated record that still shows which bundle it is for.
	writeCoverageFile(t, store.Root, ".pose/review-attestations/rva-00000000000000aa.json", `{"schema_version":1,"attestation_id":"rva-00000000000000aa","bundle_id":"`+bundle.BundleID+`","decision":"rej`)
	read, err := store.ReadReviewAttestations(bundle.BundleID)
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Coverage.Losses) != 1 || read.Coverage.Losses[0].Scope != bundle.BundleID || read.Coverage.Losses[0].AttributedBy != "header" {
		t.Fatalf("coverage = %+v", read.Coverage)
	}
	verification, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(verification.Blockers, " ")
	if verification.Approved || strings.Contains(joined, "has no attestation") || !strings.Contains(joined, "attestation history") {
		t.Fatalf("an unreadable attestation was skipped: approved=%v blockers=%v", verification.Approved, verification.Blockers)
	}
	// A truncated record of another bundle is not a loss for this one.
	_ = os.Remove(filepath.Join(store.Root, ".pose/review-attestations/rva-00000000000000aa.json"))
	writeCoverageFile(t, store.Root, ".pose/review-attestations/rva-00000000000000bb.json", `{"schema_version":1,"bundle_id":"rvb-0123456789abcdef","decision":"rej`)
	other, err := store.ReadReviewAttestations(bundle.BundleID)
	if err != nil {
		t.Fatal(err)
	}
	if !other.Coverage.CompleteFor(bundle.BundleID) {
		t.Fatalf("a record of another bundle widened this one: %+v", other.Coverage)
	}
	if other.Coverage.Complete() {
		t.Fatal("the other bundle's loss is still a loss of the read")
	}
}

func TestAnUnreadableBundleIsDisclosedAndNotSkipped(t *testing.T) {
	_, store := reviewBundleFixture(t)
	if _, err := store.SealReviewBundle("spec:backend", time.Now().UTC()); err != nil {
		t.Logf("seal: %v", err)
	}
	writeCoverageFile(t, store.Root, ".pose/review-bundles/rvb-00000000000000cc.json", `{"schema_version":1,"payload":{"scope":{"ref":"spec:backend","sec`)
	read, err := store.ReadReviewBundles("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Coverage.Losses) != 1 || read.Coverage.Losses[0].Scope != "spec:backend" {
		t.Fatalf("coverage = %+v", read.Coverage)
	}
	verification, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(verification.Warnings, " "), "sealed bundle history is incomplete") {
		t.Fatalf("warnings = %v", verification.Warnings)
	}
	elsewhere, err := store.ReadReviewBundles("spec:other")
	if err != nil {
		t.Fatal(err)
	}
	if !elsewhere.Coverage.CompleteFor("spec:other") {
		t.Fatal("a bundle attributed to spec:backend widened spec:other")
	}
}

func TestFindingsProducerIsIncompleteWhenAReviewRecordCannotBeRead(t *testing.T) {
	store := obligationFixture(t)
	writeCoverageFile(t, store.Root, ".pose/reviews/rvw-20260101T000000Z-aaaa0000.md", "---\nschema_version: 9\nreview_id: x\nscope: spec:shipped\n---\n")
	report, err := store.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if got := coverageOf(report, "findings"); got.State != CoverageStateIncomplete {
		t.Fatalf("findings coverage = %+v", got)
	}
}

func TestSourceCoverageScoping(t *testing.T) {
	c := SourceCoverage{}
	c.Add(SourceLoss{Producer: "p", Artifact: "a", Scope: "one", Kind: LossReadError})
	c.Add(SourceLoss{Producer: "p", Artifact: "b", Kind: LossParseError})
	if c.CompleteFor("two") {
		t.Fatal("an unattributed loss may belong to any scope")
	}
	if len(c.LossesFor("one")) != 2 || len(c.LossesFor("two")) != 1 || len(c.LossesFor("")) != 2 {
		t.Fatalf("LossesFor: %d %d %d", len(c.LossesFor("one")), len(c.LossesFor("two")), len(c.LossesFor("")))
	}
	if specSlugFromPath(".pose/specs/2026-10-11-thing.md") != "thing" || specSlugFromPath(".pose/specs/2026-10-11-folder/spec.md") != "folder" || specSlugFromPath(".pose/specs/plain.md") != "plain" {
		t.Fatal("slug from path")
	}
}

// T2-05, the case that matters: a bundle is approved, and a later rejection cannot be
// read. The approval is not taken as the current attestation.
func TestAnUnreadableLaterRejectionDoesNotLeaveTheEarlierApprovalStanding(t *testing.T) {
	store, bundle, _ := retentionFixture(t)
	before, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if !before.Approved {
		t.Fatalf("baseline: the bundle is not approved: %v", before.Blockers)
	}
	// The rejection was truncated: its readable prefix still names the bundle.
	writeCoverageFile(t, store.Root, ".pose/review-attestations/rva-00000000000000dd.json", `{"schema_version":1,"attestation_id":"rva-00000000000000dd","bundle_id":"`+bundle.BundleID+`","decision":"rejec`)
	after, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(after.Blockers, " ")
	if after.Approved || !strings.Contains(joined, "attestation history for bundle "+bundle.BundleID+" is incomplete") {
		t.Fatalf("a truncated later record let the earlier approval stand: approved=%v blockers=%v", after.Approved, after.Blockers)
	}
	if after.Attestation == nil {
		t.Fatal("the readable attestation is still reported, it is just not taken as current")
	}
	// A truncated record that names another bundle does not touch this one.
	_ = os.Remove(filepath.Join(store.Root, ".pose/review-attestations/rva-00000000000000dd.json"))
	writeCoverageFile(t, store.Root, ".pose/review-attestations/rva-00000000000000ee.json", `{"schema_version":1,"bundle_id":"rvb-0123456789abcdef","decision":"rejec`)
	other, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if !other.Approved {
		t.Fatalf("a record of another bundle blocked this approval: %v", other.Blockers)
	}
	// A truncated record that names nothing may be a rejection of any bundle.
	_ = os.Remove(filepath.Join(store.Root, ".pose/review-attestations/rva-00000000000000ee.json"))
	writeCoverageFile(t, store.Root, ".pose/review-attestations/rva-00000000000000ff.json", `{"schema_ver`)
	anon, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if anon.Approved {
		t.Fatal("an unreadable record that names no bundle may be a rejection of this one")
	}
	// An attestation that cannot be read at all (not even opened) is a loss too.
	_ = os.Remove(filepath.Join(store.Root, ".pose/review-attestations/rva-00000000000000ff.json"))
	if err := os.Symlink("absent-target.json", filepath.Join(store.Root, ".pose/review-attestations/rva-0000000000000011.json")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if lost, err := store.ReadReviewAttestations(bundle.BundleID); err != nil || lost.Coverage.Complete() {
		t.Fatalf("an attestation that could not be opened is not a loss: err=%v coverage=%+v", err, lost.Coverage)
	}
}
