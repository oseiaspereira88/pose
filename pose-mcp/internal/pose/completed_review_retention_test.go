package pose

// A closed scope keeps the approval it was closed with (spec
// review-verify-retains-completed-scopes). Its bundle stops matching a fresh
// preparation after routine events — a new check in the validation matrix, or
// evidence sealed after the closeout commit — and `review verify` used to
// report every such scope as superseded while `review-check` retained it.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func retentionFixture(t *testing.T) (Store, ReviewBundle, time.Time) {
	t.Helper()
	root, store := reviewBundleFixture(t)
	policyPath := filepath.Join(root, ".pose/policy/review.json")
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	policy := strings.Replace(string(raw), `"component_aware": true,`, `"component_aware": true,
  "review_bundles": true,
  "review_bundles_adopted_at": "2026-08-13",`, 1)
	if err := os.WriteFile(policyPath, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordReviewAttestation(approvedBundleAttestation(bundle, "agent:test-review"), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	return store, bundle, now
}

func closeBackendSpec(t *testing.T, store Store) {
	t.Helper()
	path := filepath.Join(store.Root, ".pose/specs/backend/spec.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	closed := strings.Replace(string(raw), "status: in-progress\n", "status: done\ncompleted_at: 2026-08-13\n", 1)
	if err := os.WriteFile(path, []byte(closed), 0o644); err != nil {
		t.Fatal(err)
	}
}

// addMatrixCheck stands for the routine change that superseded every closed
// bundle in pose-dist: a validation matrix gaining a check.
func addMatrixCheck(t *testing.T, store Store) {
	t.Helper()
	writeReviewFixture(t, store.Root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"moduleOverrides":{"api":{"checks":[{"name":"new-check","program":"true","severity":"required","evidenceClass":"unit"}]}}}`)
}

func TestCompletedReviewRetention(t *testing.T) {
	t.Run("closed scope keeps its approval after a routine change", func(t *testing.T) {
		store, bundle, _ := retentionFixture(t)
		closeBackendSpec(t, store)
		addMatrixCheck(t, store)
		verification, err := store.VerifyReviewBundle("spec:backend")
		if err != nil {
			t.Fatal(err)
		}
		if verification.State != "closed" || !verification.Fresh || !verification.Approved || verification.Bundle == nil || verification.Bundle.BundleID != bundle.BundleID || verification.Attestation == nil {
			t.Fatalf("closed scope lost its approval: %+v", verification)
		}
		if verification.Delta == nil {
			t.Fatalf("retention hid what changed since the review: %+v", verification)
		}
		if !strings.Contains(strings.Join(verification.Warnings, "\n"), retainedCompletedReviewWarning) || len(verification.Blockers) != 0 {
			t.Fatalf("retention is not reported as such: %+v", verification)
		}
		eval, err := store.ReviewCheck("spec:backend")
		if err != nil || !eval.Approved || !eval.Fresh || eval.BundleID != bundle.BundleID {
			t.Fatalf("review-check disagrees with verify: %+v %v", eval, err)
		}
	})

	t.Run("a newer rejection is not overridden by an older approval", func(t *testing.T) {
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
		closeBackendSpec(t, store)
		writeReviewFixture(t, store.Root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"moduleOverrides":{"api":{"checks":[{"name":"another-check","program":"true","severity":"required","evidenceClass":"unit"}]}}}`)
		verification, err := store.VerifyReviewBundle("spec:backend")
		if err != nil {
			t.Fatal(err)
		}
		if verification.Approved {
			t.Fatalf("an older approval overrode the newer rejection: %+v", verification)
		}
		eval, err := store.ReviewCheck("spec:backend")
		if err == nil && eval.Approved {
			t.Fatalf("review-check fell back past the newer rejection: %+v", eval)
		}
	})

	t.Run("a newer approval that no longer validates does not void an older one", func(t *testing.T) {
		store, first, now := retentionFixture(t)
		addMatrixCheck(t, store)
		second, err := store.SealReviewBundle("spec:backend", now.Add(2*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		invalid := approvedBundleAttestation(second, "agent:second-review")
		for i := range invalid.Criteria {
			if invalid.Criteria[i].Disposition == "passed" {
				invalid.Criteria[i].Evidence = "unit:absent/go/test"
				break
			}
		}
		if _, err := store.RecordReviewAttestation(invalid, now.Add(3*time.Minute)); err != nil {
			t.Fatal(err)
		}
		if blockers := store.validateBundleAttestation(second, invalid); len(blockers) == 0 {
			t.Fatal("fixture: the newer approval still validates")
		}
		closeBackendSpec(t, store)
		writeReviewFixture(t, store.Root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"moduleOverrides":{"api":{"checks":[{"name":"another-check","program":"true","severity":"required","evidenceClass":"unit"}]}}}`)
		verification, err := store.VerifyReviewBundle("spec:backend")
		if err != nil {
			t.Fatal(err)
		}
		if !verification.Approved || verification.Bundle == nil || verification.Bundle.BundleID != first.BundleID {
			t.Fatalf("an older standing approval was voided by a newer approval that no longer validates: %+v", verification)
		}
	})

	t.Run("an open scope is still superseded", func(t *testing.T) {
		store, _, _ := retentionFixture(t)
		addMatrixCheck(t, store)
		verification, err := store.VerifyReviewBundle("spec:backend")
		if err != nil {
			t.Fatal(err)
		}
		if verification.State != "superseded" || verification.Approved {
			t.Fatalf("an open scope was retained: %+v", verification)
		}
	})
}

// A closed scope whose newest review decided against it reports that review,
// not "no review attempt exists" (spec review-check-names-a-negative-verdict).
func TestReviewCheckNamesANegativeVerdictOnAClosedScope(t *testing.T) {
	store, _, now := retentionFixture(t)
	addMatrixCheck(t, store)
	second, err := store.SealReviewBundle("spec:backend", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	negative := approvedBundleAttestation(second, "agent:second-review")
	negative.Decision = "changes-requested"
	negative.Findings = []ReviewFinding{{ID: "f1", Severity: "medium", Disposition: "changes-requested", Action: "add the missing test"}}
	if _, err := store.RecordReviewAttestation(negative, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	closeBackendSpec(t, store)
	writeReviewFixture(t, store.Root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"moduleOverrides":{"api":{"checks":[{"name":"third-check","program":"true","severity":"required","evidenceClass":"unit"}]}}}`)
	eval, err := store.ReviewCheck("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if eval.Approved {
		t.Fatalf("a closed scope whose latest review requested changes was approved: %+v", eval)
	}
	joined := strings.Join(eval.Blockers, "\n")
	if strings.Contains(joined, "no review attempt exists") || !strings.Contains(joined, "decided changes-requested") || !strings.Contains(joined, "finding f1 (medium) is changes-requested: add the missing test") {
		t.Fatalf("review-check did not name the negative verdict and its findings: %v", eval.Blockers)
	}
	if eval.Current == nil || eval.Current.Decision != "changes-requested" {
		t.Fatalf("current review not reported: %+v", eval.Current)
	}
}
