package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-material-equivalence-reuse.

func TestBookkeepingChangesKeepCriteriaEquivalentAndCodeChangesNameTheInput(t *testing.T) {
	root, store := reviewBundleFixture(t)
	sealed, err := store.SealReviewBundle("spec:backend", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	// The Validation log is bookkeeping, not a sealed semantic section.
	writeReviewFixture(t, root, ".pose/specs/backend/spec.md", strings.Replace(mustReadFile(t, root, ".pose/specs/backend/spec.md"), "Not run yet.", "Ran the suite on 2026-10-04; all green.", 1))
	bookkeeping, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ExplainCriterionReuse(sealed, bookkeeping) {
		if e.Decision != ReuseEquivalent {
			t.Fatalf("a bookkeeping edit changed %s: %v", e.Criterion, e.ChangedInputs)
		}
	}
	writeReviewFixture(t, root, "api/server.go", "package api\n\nfunc Ready() bool { return false }\n")
	changed, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	sawSubject := false
	for _, e := range ExplainCriterionReuse(sealed, changed) {
		if e.Decision == ReuseChanged {
			for _, input := range e.ChangedInputs {
				sawSubject = sawSubject || strings.Contains(input, "api/server.go") || strings.HasPrefix(input, "subject")
			}
		}
	}
	if !sawSubject {
		t.Fatalf("a code change did not name the changed subject input: %+v", ExplainCriterionReuse(sealed, changed))
	}
	verification, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if verification.Delta == nil || len(verification.Delta.CriterionReuse) == 0 {
		t.Fatalf("verify does not carry the reuse explanation: %+v", verification.Delta)
	}
}

func TestEquivalenceAgreesWithTheReuseDigest(t *testing.T) {
	root, store := reviewBundleFixture(t)
	sealed, _ := store.SealReviewBundle("spec:backend", time.Now())
	writeReviewFixture(t, root, "api/server.go", "package api\n\nfunc Ready() bool { return false }\n")
	changed, _ := store.PrepareReviewBundle("spec:backend")
	delta := ReviewBundleDiff(sealed, changed)
	delta.CriterionReuse = ExplainCriterionReuse(sealed, changed)
	reusable := map[string]bool{}
	for _, id := range delta.ReusableCriteria {
		reusable[id] = true
	}
	for _, e := range delta.CriterionReuse {
		if (e.Decision == ReuseEquivalent) != reusable[e.Criterion] {
			t.Fatalf("explanation and reuse digest disagree on %s: %+v", e.Criterion, e)
		}
	}
}

func mustReadFile(t *testing.T, root, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
