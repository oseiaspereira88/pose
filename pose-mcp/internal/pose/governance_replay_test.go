package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGovernanceReplayCounterfactualDoesNotRewriteFrozenApproval(t *testing.T) {
	_, store := abmMixedProfileFixture(t)
	bundle, err := store.SealReviewBundle("spec:backend", time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	att := approvedBundleAttestation(bundle, "agent:private-principal")
	att.BundleDigest = bundle.BundleDigest
	att.AttestedAt = "2026-09-28T00:01:00Z"
	for i := range att.Criteria {
		if att.Criteria[i].ID == "operability" {
			att.Criteria[i].Rationale = ""
		}
	}
	legacy := bundle
	legacy.Payload.GoverningContracts = []string{"component-aware", "review-bundles", "evidence-vocabulary"}
	before, _ := json.Marshal(legacy)
	frozen, hypothetical := store.replayReviewBlockers(legacy, att)
	if len(frozen) != 0 || !containsSubstring(hypothetical, "passed with no conclusion") {
		t.Fatalf("unexpected replay: frozen=%v hypothetical=%v", frozen, hypothetical)
	}
	after, _ := json.Marshal(legacy)
	if string(before) != string(after) {
		t.Fatal("counterfactual mutated historical bundle")
	}
	if _, err := store.RecordReviewAttestation(approvedBundleAttestation(bundle, "agent:private-principal"), time.Now()); err != nil {
		t.Fatal(err)
	}
	snapshot := reviewFixtureSnapshot(t, store.Root)
	report, err := store.GovernanceReplay(0)
	if err != nil || report.Attestations != 1 || report.ApprovingAttestations != 1 || report.CounterfactualRejected != 0 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if !reflect.DeepEqual(snapshot, reviewFixtureSnapshot(t, store.Root)) {
		t.Fatal("replay wrote project artifacts")
	}
	raw, _ := json.Marshal(report)
	for _, private := range []string{store.Root, "private-principal", "api/server.go"} {
		if strings.Contains(string(raw), private) {
			t.Fatalf("replay leaked %q", private)
		}
	}
}

func TestGovernanceReplayMalformedSymlinkAndLimitsAreCoverage(t *testing.T) {
	root, store := reviewBundleFixture(t)
	if err := os.MkdirAll(filepath.Join(root, ".pose/review-attestations"), 0755); err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, ".pose/review-attestations/rva-0000000000000000.json", "not json")
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".pose/review-attestations/rva-1111111111111111.json")); err != nil {
		t.Fatal(err)
	}
	report, err := store.GovernanceReplay(0)
	if err != nil || report.Complete || report.ArtifactsInvalid != 2 || report.Attestations != 0 {
		t.Fatalf("invalid data counted as passed: %+v %v", report, err)
	}
	report, err = store.GovernanceReplay(1)
	if err != nil || report.Complete || report.ArtifactsScanned != 1 {
		t.Fatalf("limit not visible: %+v %v", report, err)
	}
	if _, err := store.GovernanceReplay(-1); err == nil {
		t.Fatal("negative bound accepted")
	}
	if _, err := store.GovernanceReplay(20001); err == nil {
		t.Fatal("unbounded work accepted")
	}
}
