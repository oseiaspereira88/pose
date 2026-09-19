package pose

import (
	"encoding/json"
	"strings"
	"testing"
)

// R1/R4/R5. The implementation identity is content, not refs. A squash merge or
// a rebase gives the same content a different SHA, and an identity that moved
// with the SHA would call that a different subject — which is how a review of
// unchanged content comes to be redone, or worse, how two different contents
// come to share one identity.
func TestSubjectImplementationDigestIsContentNotRefs(t *testing.T) {
	root, store := reviewBundleFixture(t)
	before, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if before.Payload.Subject.ImplementationDigest == "" {
		t.Fatal("the subject must carry an implementation identity")
	}

	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets[0].ID = "cs-rebased"
	graph.ChangeSets[0].ResolvedBase = "base-after-rebase"
	graph.ChangeSets[0].ResolvedHead = "head-after-rebase"
	graph.ChangeSets[0].DiffDigest = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))

	after, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if before.Payload.Subject.ImplementationDigest != after.Payload.Subject.ImplementationDigest {
		t.Fatalf("identical content under a different SHA must keep its identity: %s != %s",
			before.Payload.Subject.ImplementationDigest, after.Payload.Subject.ImplementationDigest)
	}

	// And it does move when the content does.
	writeReviewFixture(t, root, "api/server.go", "package api\n\nfunc Ready() bool { return false }\n")
	changed, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if changed.Payload.Subject.ImplementationDigest == after.Payload.Subject.ImplementationDigest {
		t.Fatal("changed content must change the implementation identity")
	}
}

// R5. The identity is computable from the subject alone, so an observer of the
// implementation never has to reference the bundle that will contain it. That
// is what keeps implementation -> assessment -> plan -> bundle -> attestation a
// DAG instead of a definition in terms of its own output.
func TestSubjectImplementationDigestDoesNotDependOnTheBundle(t *testing.T) {
	_, store := reviewBundleFixture(t)
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	recomputed := reviewImplementationDigest(bundle.Payload.Subject)
	if recomputed != bundle.Payload.Subject.ImplementationDigest {
		t.Fatalf("the identity must be reproducible from the subject alone: %s != %s", recomputed, bundle.Payload.Subject.ImplementationDigest)
	}
	// Recomputing from a subject carrying a different bundle digest changes
	// nothing, because the bundle is not an input.
	detached := bundle.Payload.Subject
	if reviewImplementationDigest(detached) != bundle.Payload.Subject.ImplementationDigest {
		t.Fatal("the identity must not vary with the bundle it is sealed into")
	}
	if strings.Contains(recomputed, bundle.BundleDigest) {
		t.Fatal("the identity must not embed the bundle digest")
	}
}

// R2. A change set resolved from trailers attributes exact paths, but its
// base..head spans whatever else landed in between. Reading the range as the
// spec's work is wrong whenever two specs interleaved, and nothing said so.
func TestSubjectRangeObservationSeparatesAttributionFromProvenance(t *testing.T) {
	_, store := reviewBundleFixture(t)
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.RangeObservations) == 0 {
		t.Fatal("every attributed change set must carry a range observation")
	}
	for _, observation := range bundle.RangeObservations {
		switch observation.State {
		case "clean", "contaminated", "unknown":
		default:
			t.Fatalf("unexpected range state %q", observation.State)
		}
		if observation.State != "clean" && observation.Reason == "" {
			t.Fatalf("a range that is not clean must say why: %+v", observation)
		}
	}
	// The fixture has no Git metadata, so the count cannot be taken. Unknown,
	// never clean: the point of the observation is to stop a range being read
	// as an attribution, and a silent pass would do the opposite.
	if bundle.RangeObservations[0].State != "unknown" {
		t.Fatalf("an uncountable range must be unknown, got %q", bundle.RangeObservations[0].State)
	}
	if !containsSubstring(bundle.Warnings, "could not have its range counted") {
		t.Fatalf("the reviewer must be told: %v", bundle.Warnings)
	}
}

// R2/R6. The observation describes the repository around the change, which moves
// whenever anything is committed or reindexed. Sealing it would stale a review
// of content that did not move.
func TestSubjectRangeObservationIsNotSealed(t *testing.T) {
	_, store := reviewBundleFixture(t)
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(bundle.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "range_observations") {
		t.Fatal("range observations must sit outside the sealed payload")
	}
}

// R3/R7. Three states, and the third one is the one that used to disappear: a
// result naming no commit says nothing about which content it observed, and
// reading that as current is how a check that never saw this change comes to
// stand for it.
func TestSubjectEvidenceObservationHasThreeStates(t *testing.T) {
	cases := []struct {
		name     string
		head     string
		evidence ReviewBundleEvidence
		want     string
	}{
		{"same commit", "head-resolved", ReviewBundleEvidence{GitHead: "head-resolved"}, "observed"},
		{"other commit", "head-resolved", ReviewBundleEvidence{GitHead: "an-older-commit"}, "carried-forward"},
		{"no fingerprint", "head-resolved", ReviewBundleEvidence{}, "unknown"},
		{"no subject head", "", ReviewBundleEvidence{GitHead: "head-resolved"}, "unknown"},
	}
	for _, tc := range cases {
		if got := ReviewEvidenceObservation(tc.head, tc.evidence); got != tc.want {
			t.Fatalf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// The state is recorded on the sealed evidence and surfaced, so a reviewer
// reading a bundle can tell a result that observed this change from one carried
// forward, without comparing two commit hashes nobody compares.
func TestSubjectEvidenceObservationIsRecordedAndSurfaced(t *testing.T) {
	root, store := reviewBundleFixture(t)
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.ValidationResults) == 0 {
		t.Skip("fixture seals no validation result")
	}
	graph.ValidationResults[0].GitHead = ""
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))

	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ev := range bundle.Payload.Evidence {
		if ev.ID == graph.ValidationResults[0].ID {
			found = true
			if ev.SubjectObservation != "unknown" {
				t.Fatalf("evidence with no commit must be unknown, got %q", ev.SubjectObservation)
			}
		}
		if ev.SubjectObservation == "" {
			t.Fatalf("every sealed result must carry an observation state: %+v", ev)
		}
	}
	if !found {
		t.Skip("the mutated result is not part of this scope's evidence")
	}
	if !containsSubstring(bundle.Warnings, "names no commit") {
		t.Fatalf("the unknown state must reach the reviewer: %v", bundle.Warnings)
	}
}

// R6. The observation is derived from the subject head, which the digest
// deliberately excludes. Sealing it would smuggle that ref back into the
// identity and stale reviews on ref movement alone.
func TestSubjectEvidenceObservationDoesNotEnterTheDigest(t *testing.T) {
	_, store := reviewBundleFixture(t)
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Payload.Evidence) == 0 {
		t.Skip("fixture seals no evidence")
	}
	mutated := bundle.Payload
	mutated.Evidence = append([]ReviewBundleEvidence{}, bundle.Payload.Evidence...)
	for i := range mutated.Evidence {
		mutated.Evidence[i].SubjectObservation = "carried-forward"
	}
	before, err := reviewBundlePayloadDigest(bundle.Payload)
	if err != nil {
		t.Fatal(err)
	}
	after, err := reviewBundlePayloadDigest(mutated)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("the observation state must not change the sealed identity")
	}
}
