package pose

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// poseTreeSnapshot digests every file under .pose, so a refused write can be
// shown to have produced no record, bundle, attestation, supplement or event.
func poseTreeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	base := filepath.Join(root, ".pose")
	err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		out[filepath.ToSlash(rel)] = digestBytes(raw)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return out
}

func assertPoseTreeUnchanged(t *testing.T, root string, before map[string]string) {
	t.Helper()
	after := poseTreeSnapshot(t, root)
	changed := []string{}
	for path, digest := range after {
		if before[path] != digest {
			changed = append(changed, path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)
	if len(changed) > 0 {
		t.Fatalf("a refused approval changed .pose: %s", strings.Join(changed, ", "))
	}
}

// requireApprovalRefusal asserts that err is the domain's typed approval
// refusal, that the attempt left .pose untouched, and returns its blockers.
func requireApprovalRefusal(t *testing.T, root string, before map[string]string, err error) []string {
	t.Helper()
	var refusal *ReviewApprovalRefusal
	if !errors.As(err, &refusal) || !errors.Is(err, ErrReviewApprovalRefused) {
		t.Fatalf("want a typed approval refusal, got %v", err)
	}
	if len(refusal.Blockers) == 0 {
		t.Fatal("an approval refusal must name its blockers")
	}
	assertPoseTreeUnchanged(t, root, before)
	return refusal.Blockers
}

// recordAndVerifyOrRefusal records att through the shared writer and verifies
// its scope. When the writer refuses the approval, the refusal is what the
// verifier would have said: the helper asserts that nothing was written and
// returns an unapproved verification carrying the refusal's blockers.
func recordAndVerifyOrRefusal(t *testing.T, store Store, scope string, att ReviewAttestation, now time.Time) ReviewBundleVerification {
	t.Helper()
	before := poseTreeSnapshot(t, store.Root)
	_, err := store.RecordReviewAttestation(att, now)
	if errors.Is(err, ErrReviewApprovalRefused) {
		return ReviewBundleVerification{Scope: scope, State: "refused", Blockers: requireApprovalRefusal(t, store.Root, before, err)}
	}
	if err != nil {
		t.Fatal(err)
	}
	verification, err := store.VerifyReviewBundle(scope)
	if err != nil {
		t.Fatal(err)
	}
	return verification
}

// persistHistoricalAttestation writes an attestation as a writer from before
// the domain approval preflight could have: completed and content-addressed,
// but never admitted. It models persisted history, which the verifier keeps
// judging and the preflight must not rewrite (spec
// pose-review-approval-domain-preflight R3).
func persistHistoricalAttestation(t *testing.T, store Store, att ReviewAttestation, now time.Time) ReviewAttestation {
	t.Helper()
	bundle, err := store.LoadReviewBundle(att.BundleID)
	if err != nil {
		t.Fatal(err)
	}
	completed, err := completeReviewAttestation(bundle, att, now)
	if err != nil {
		t.Fatal(err)
	}
	completed.Path = filepath.ToSlash(filepath.Join(".pose", "review-attestations", completed.AttestationID+".json"))
	dir, err := ensureReviewArtifactDir(store.Root, ".pose/review-attestations", true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(completed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeImmutableJSON(filepath.Join(dir, completed.AttestationID+".json"), append(raw, '\n')); err != nil {
		t.Fatal(err)
	}
	return completed
}
