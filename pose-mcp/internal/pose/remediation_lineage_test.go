package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func lineageSpec(t *testing.T, root, slug, links string) *Spec {
	t.Helper()
	writeReviewFixture(t, root, ".pose/specs/"+slug+".md", "---\nslug: "+slug+"\nstatus: done\nremediates: "+links+"\n---\n# Spec\n")
	sp, err := (Store{Root: root}).GetSpec(slug)
	if err != nil {
		t.Fatal(err)
	}
	return sp
}

func TestRemediationLineageReferencesAndCategories(t *testing.T) {
	for _, category := range []string{"defect-fix", "simplification", "revert", "requirement-change", "planned-evolution"} {
		t.Run(category, func(t *testing.T) {
			root := t.TempDir()
			lineageSpec(t, root, "original", "")
			sp := lineageSpec(t, root, "repair", "spec:original@"+category)
			if err := (Store{Root: root}).ValidateRemediationLineage(*sp); err != nil {
				t.Fatal(err)
			}
			if len(sp.Remediates) != 1 {
				t.Fatalf("parsed links: %+v", sp)
			}
		})
	}
	for _, tc := range []struct{ name, links, code string }{
		{"orphan", "spec:absent@defect-fix", "orphan"},
		{"category", "spec:original@accepted-risk", "category"},
		{"untyped", "original@defect-fix", "reference"},
		{"traversal", "spec:../outside@defect-fix", "reference"},
		{"self", "spec:repair@defect-fix", "cycle"},
		{"duplicate", "spec:original@defect-fix, spec:original@revert", "duplicate"},
		{"empty-item", "spec:original@defect-fix,,spec:other@revert", "syntax"},
		{"unknown-finding", "finding:rva-0000000000000000/F1@defect-fix", "orphan"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			lineageSpec(t, root, "original", "")
			sp := lineageSpec(t, root, "repair", tc.links)
			err := (Store{Root: root}).ValidateRemediationLineage(*sp)
			if err == nil || !strings.Contains(err.Error(), "remediation-lineage/"+tc.code) {
				t.Fatalf("got %v, want %s", err, tc.code)
			}
		})
	}
}

func TestRemediationLineageCycleBoundsAndConfinement(t *testing.T) {
	root := t.TempDir()
	sp := lineageSpec(t, root, "a", "spec:b@revert")
	lineageSpec(t, root, "b", "spec:c@simplification")
	lineageSpec(t, root, "c", "spec:a@defect-fix")
	store := Store{Root: root}
	if err := store.ValidateRemediationLineage(*sp); err == nil || !strings.Contains(err.Error(), "/cycle") {
		t.Fatalf("cycle: %v", err)
	}
	lineageSpec(t, root, "c", "")
	if err := store.ValidateRemediationLineage(*sp); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(root, ".pose/specs/outside")); err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateRemediationLineage(*sp); err == nil || !strings.Contains(err.Error(), "/path") {
		t.Fatalf("symlink: %v", err)
	}
	// No opt-in means no graph traversal, including unrelated broken artifacts.
	if err := store.ValidateRemediationLineage(Spec{Slug: "legacy", DependsOn: []string{"a"}}); err != nil {
		t.Fatal(err)
	}

	bounded := t.TempDir()
	for i := 0; i <= 256; i++ {
		links := ""
		if i < 256 {
			links = fmt.Sprintf("spec:n%d@defect-fix", i+1)
		}
		lineageSpec(t, bounded, fmt.Sprintf("n%d", i), links)
	}
	start, err := (Store{Root: bounded}).GetSpec("n0")
	if err != nil {
		t.Fatal(err)
	}
	if err := (Store{Root: bounded}).ValidateRemediationLineage(*start); err == nil || !strings.Contains(err.Error(), "/limit") {
		t.Fatalf("bound: %v", err)
	}
}

func TestRemediationLineageFindingUsesImmutableAttestation(t *testing.T) {
	root, store := reviewBundleFixture(t)
	bundle, err := store.SealReviewBundle("spec:backend", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	att := approvedBundleAttestation(bundle, "agent:fixture")
	att.Decision = "changes-requested"
	att.Findings = []ReviewFinding{{ID: "F1", Severity: "high", Action: "fix the defect", Disposition: "open"}}
	att, err = store.RecordReviewAttestation(att, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	sp := lineageSpec(t, root, "repair", "finding:"+att.AttestationID+"/F1@defect-fix")
	if err := store.ValidateRemediationLineage(*sp); err != nil {
		t.Fatal(err)
	}
	sp.Remediates = []string{"finding:" + att.AttestationID + "/F2@defect-fix"}
	if err := store.ValidateRemediationLineage(*sp); err == nil || !strings.Contains(err.Error(), "/orphan") {
		t.Fatalf("missing finding: %v", err)
	}
	sp.Remediates = []string{"finding:" + att.AttestationID + "/F1@defect-fix"}
	backend, err := store.GetSpec("backend")
	if err != nil {
		t.Fatal(err)
	}
	backend.Remediates = sp.Remediates
	if err := store.ValidateRemediationLineage(*backend); err == nil || !strings.Contains(err.Error(), "/cycle") {
		t.Fatalf("finding self cycle: %v", err)
	}
	att.Findings[0].Action = "tampered"
	raw, err := json.Marshal(att)
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, att.Path, string(raw))
	if err := store.ValidateRemediationLineage(*sp); err == nil || !strings.Contains(err.Error(), "/orphan") {
		t.Fatalf("tamper: %v", err)
	}
}

func TestRemediationLineageIsSemanticAndOrderIndependent(t *testing.T) {
	root, store := reviewBundleFixture(t)
	before, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	legacyDigest, err := store.ScopeDigest("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	lineageSpec(t, root, "a", "")
	lineageSpec(t, root, "b", "")
	path := filepath.Join(root, ".pose/specs/backend/spec.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	setLinks := func(links string) {
		t.Helper()
		content := strings.Replace(string(raw), "slug: backend", "slug: backend\nremediates: "+links, 1)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	setLinks("")
	empty, err := store.PrepareReviewBundle("spec:backend")
	if err != nil || before.BundleDigest != empty.BundleDigest {
		t.Fatalf("empty changed legacy bundle: %v", err)
	}
	setLinks("spec:a@defect-fix, spec:b@revert")
	linked, err := store.SealReviewBundle("spec:backend", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if linked.BundleDigest == before.BundleDigest {
		t.Fatal("lineage omitted from subject")
	}
	linkedDigest, err := store.ScopeDigest("spec:backend")
	if err != nil || linkedDigest == legacyDigest {
		t.Fatalf("lineage omitted from legacy digest: %v", err)
	}
	setLinks("spec:b@revert, spec:a@defect-fix")
	reordered, err := store.PrepareReviewBundle("spec:backend")
	if err != nil || reordered.BundleDigest != linked.BundleDigest {
		t.Fatalf("order changed bundle: %v", err)
	}
	setLinks("spec:a@simplification, spec:b@revert")
	verification, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if verification.Fresh {
		t.Fatal("changed category kept sealed review fresh")
	}
	setLinks("spec:missing@revert")
	if _, err := store.SealReviewBundle("spec:backend", time.Now()); err == nil {
		t.Fatal("sealed orphan lineage")
	}
}

// projectionFixture writes deliveries with explicit completion dates so the
// maturity window can be measured instead of waited for.
func projectionFixture(t *testing.T, root string, entries ...string) {
	t.Helper()
	for i := 0; i+2 < len(entries); i += 3 {
		slug, completed, links := entries[i], entries[i+1], entries[i+2]
		body := "---\nslug: " + slug + "\nstatus: done\ncompleted_at: " + completed + "\n"
		if links != "" {
			body += "remediates: " + links + "\n"
		}
		writeReviewFixture(t, root, ".pose/specs/"+slug+".md", body+"---\n# Spec\n")
	}
}

func projection(t *testing.T, root string, now time.Time) GovernanceRemediationDimensions {
	t.Helper()
	report, err := (Store{Root: root}).GovernanceOutcomes(GovernanceOutcomesQuery{Now: now, MinSample: 1})
	if err != nil {
		t.Fatal(err)
	}
	return report.Remediation
}

// R2: a delivery too recent to have been remediated yet is censored, not counted
// as one that came out clean. Without this the rate improves simply by shipping.
func TestRemediationProjectionCensorsImmatureDeliveries(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	projectionFixture(t, root,
		"mature-broken", "2026-07-01", "",
		"mature-quiet", "2026-07-02", "",
		"just-shipped", "2026-09-19", "",
		"fix", "2026-09-10", "spec:mature-broken@defect-fix",
	)
	got := projection(t, root, now)
	if !got.Available || got.Reason != "" {
		t.Fatalf("declared links did not make the dimension available: %+v", got)
	}
	// `fix` itself completed 10 days ago, inside the 30-day window, so it is
	// censored too: it is a delivery like any other.
	if got.MaturePopulation != 2 || got.Censored != 2 {
		t.Fatalf("window boundary wrong: mature=%d censored=%d", got.MaturePopulation, got.Censored)
	}
	if got.ObservedRemediated != 1 || got.LinkedObserved != 1 || got.UnlinkedUnknown != 1 {
		t.Fatalf("population split wrong: %+v", got)
	}
	// The unremediated mature delivery is unknown, not proven clean.
	if got.UnlinkedUnknown != got.MaturePopulation-got.LinkedObserved {
		t.Fatalf("a delivery with no link was resolved to something other than unknown: %+v", got)
	}
}

// R3: a requirement change is not a defect, and a planned evolution is the plan
// working. Both are reported, and neither enters the rate.
func TestRemediationProjectionCountsOnlyPertinentCategories(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	projectionFixture(t, root,
		"target-requirement", "2026-06-01", "",
		"target-planned", "2026-06-02", "",
		"target-defect", "2026-06-03", "",
		"target-revert", "2026-06-04", "",
		"target-simpler", "2026-06-05", "",
		"r1", "2026-07-01", "spec:target-requirement@requirement-change",
		"r2", "2026-07-02", "spec:target-planned@planned-evolution",
		"r3", "2026-07-03", "spec:target-defect@defect-fix",
		"r4", "2026-07-04", "spec:target-revert@revert",
		"r5", "2026-07-05", "spec:target-simpler@simplification",
	)
	got := projection(t, root, now)
	if got.MaturePopulation != 10 {
		t.Fatalf("mature population = %d, want 10", got.MaturePopulation)
	}
	if got.LinkedObserved != 5 {
		t.Fatalf("linked observed = %d, want 5 — every target carries a link", got.LinkedObserved)
	}
	if got.ObservedRemediated != 3 {
		t.Fatalf("observed remediated = %d, want 3 (defect-fix, revert, simplification)", got.ObservedRemediated)
	}
	for category, want := range map[string]int{
		"requirement-change": 1, "planned-evolution": 1,
		"defect-fix": 1, "revert": 1, "simplification": 1,
	} {
		if got.ByCategory[category] != want {
			t.Errorf("by_category[%s] = %d, want %d — an excluded category must still be visible", category, got.ByCategory[category], want)
		}
	}
	if !containsFold(got.CountedCategories, "defect-fix") || containsFold(got.CountedCategories, "requirement-change") {
		t.Errorf("counted_categories does not state which links become defects: %v", got.CountedCategories)
	}
}

// Absence of the contract is unavailability, not a rate of zero.
func TestRemediationProjectionWithoutLinksIsUnavailable(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	projectionFixture(t, root, "alpha", "2026-06-01", "", "beta", "2026-06-02", "")
	got := projection(t, root, now)
	if got.Available || got.Reason == "" {
		t.Fatalf("a repository with no links reported an available rate: %+v", got)
	}
	if got.ObservedRemediated != 0 || got.LinkedObserved != 0 {
		t.Fatalf("unavailable dimension reported observations: %+v", got)
	}
	report, err := (Store{Root: root}).GovernanceOutcomes(GovernanceOutcomesQuery{Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if report.Coverage.RemediationCoverage != "unavailable" {
		t.Fatalf("coverage = %q, want unavailable", report.Coverage.RemediationCoverage)
	}
}

// Coverage can only ever be partial: an absent link is unknown, so the mature
// population is never fully observed.
func TestRemediationProjectionCoverageIsNeverComplete(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	projectionFixture(t, root,
		"target", "2026-06-01", "",
		"fix", "2026-06-10", "spec:target@defect-fix",
	)
	report, err := (Store{Root: root}).GovernanceOutcomes(GovernanceOutcomesQuery{Now: now, MinSample: 1})
	if err != nil {
		t.Fatal(err)
	}
	if report.Coverage.RemediationCoverage != "partial" {
		t.Fatalf("coverage = %q, want partial even with every mature delivery linked", report.Coverage.RemediationCoverage)
	}
	if report.Remediation.InsufficientSample {
		t.Fatalf("min_sample 1 over 2 mature deliveries reported insufficient: %+v", report.Remediation)
	}
	small, err := (Store{Root: root}).GovernanceOutcomes(GovernanceOutcomesQuery{Now: now, MinSample: 50})
	if err != nil {
		t.Fatal(err)
	}
	if !small.Remediation.InsufficientSample {
		t.Fatalf("a population below min_sample was published without saying so: %+v", small.Remediation)
	}
}
