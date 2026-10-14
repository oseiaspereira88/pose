package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func reuseSourceFixture(t *testing.T) (authorityFixture, ReviewAttestation, ReviewAttestation, string) {
	t.Helper()
	f := verifiedAuthorityFixture(t, "different-actor")
	changeAuthorityPolicy(t, f, func(p map[string]any) { p["allow_criterion_reuse"] = true })
	resealAfterPolicyChange(t, &f)
	source := f.attestation(t, "agent:source", "agent:implementer", "source-run", "implementation-run")
	source = resignAuthorityAttestation(t, f, source)
	raw, _ := json.Marshal(source)
	p := filepath.Join(f.root, ".pose/review-attestations", source.AttestationID+".json")
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, raw, 0600); err != nil {
		t.Fatal(err)
	}
	var criterion ReviewPlanCriterion
	for _, c := range f.bundle.Payload.Plan.Criteria {
		if c.Required {
			criterion = c
			break
		}
	}
	if criterion.ID == "" {
		t.Fatal("missing fixture criterion")
	}
	current := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run", "implementation-run")
	current.ReusedFrom = []ReviewAttestationReuse{{Criterion: criterion.ID, FromAttestation: source.AttestationID, InputDigest: reviewCriterionInputDigest(f.bundle, criterion), Mode: ReviewReuseVerifiedSource}}
	current = resignAuthorityAttestation(t, f, current)
	return f, source, current, p
}

func TestReviewReuseProvenanceAuthenticatesSourceAndBindings(t *testing.T) {
	for _, name := range []string{"valid", "signature", "authority", "negative", "criterion", "input", "missing", "unsigned", "grant"} {
		t.Run(name, func(t *testing.T) {
			f, source, current, path := reuseSourceFixture(t)
			switch name {
			case "signature":
				source.Envelope.Signature = "invalid"
			case "authority":
				source.Authority.ImplementationPrincipal = source.Reviewer
				source = resignAuthorityAttestation(t, f, source)
				current.ReusedFrom[0].FromAttestation = source.AttestationID
			case "negative":
				source.Decision = "rejected"
				source = resignAuthorityAttestation(t, f, source)
				current.ReusedFrom[0].FromAttestation = source.AttestationID
			case "criterion":
				source.Criteria[0].Disposition = "not-applicable"
				source = resignAuthorityAttestation(t, f, source)
				current.ReusedFrom[0].FromAttestation = source.AttestationID
			case "input":
				current.ReusedFrom[0].InputDigest = "sha256:changed"
			case "unsigned":
				source.Envelope = nil
			case "grant":
				changeAuthorityPolicy(t, f, func(p map[string]any) { p["trusted_attestation_issuers"] = []string{} })
			}
			if name == "missing" {
				os.Remove(path)
			} else {
				raw, _ := json.Marshal(source)
				path = filepath.Join(filepath.Dir(path), source.AttestationID+".json")
				os.WriteFile(path, raw, 0600)
			}
			current = resignAuthorityAttestation(t, f, current)
			s := f.store.atReviewTime(f.now)
			blockers, disclosure := s.evaluateReviewReuse(f.bundle, current, true)
			if (len(blockers) == 0) != (name == "valid") {
				t.Fatalf("%s: %v", name, blockers)
			}
			if len(disclosure) != 1 || (disclosure[0].State == "source-verified-under-current-trust") != (name == "valid") {
				t.Fatalf("disclosure diverged: %+v", disclosure)
			}
		})
	}
}

func TestReviewReuseProvenanceReaffirmationIsCurrentJudgment(t *testing.T) {
	f, source, current, path := reuseSourceFixture(t)
	source.Envelope = nil
	raw, _ := json.Marshal(source)
	os.WriteFile(path, raw, 0600)
	current.ReusedFrom[0].Mode = ""
	blockers, d := f.store.evaluateReviewReuse(f.bundle, current, true)
	if len(blockers) > 0 || d[0].Mode != "legacy-reaffirmation" || d[0].State != "historical-pass-not-verified" {
		t.Fatalf("legacy promoted or blocked: %v %+v", blockers, d)
	}
	current.ReusedFrom[0].Mode = ReviewReuseCurrentReaffirmation
	for i := range current.Criteria {
		if current.Criteria[i].ID == current.ReusedFrom[0].Criterion {
			current.Criteria[i].Rationale = "I re-examined this criterion in the current review."
		}
	}
	if b, _ := f.store.evaluateReviewReuse(f.bundle, current, true); len(b) != 0 {
		t.Fatal(b)
	}
	for i := range current.Criteria {
		current.Criteria[i].Rationale = ""
	}
	if b, _ := f.store.evaluateReviewReuse(f.bundle, current, true); len(b) == 0 {
		t.Fatal("reference dispensed with current judgment")
	}
}

func TestReviewReuseProvenanceBudgetsAreInconclusiveAndOperationLocal(t *testing.T) {
	f, source, current, _ := reuseSourceFixture(t)
	for _, name := range []string{"cycle", "depth", "nodes", "edges", "bytes"} {
		t.Run(name, func(t *testing.T) {
			s := f.store.reviewOperation()
			switch name {
			case "cycle":
				s.reviewReuse.active[source.AttestationID] = true
			case "depth":
				s.reviewReuse.depth = reviewReuseMaxDepth
			case "nodes":
				s.reviewReuse.nodes = reviewReuseMaxNodes
			case "edges":
				s.reviewReuse.edges = reviewReuseMaxEdges
			case "bytes":
				s.reviewReuse.bytes = reviewReuseMaxBytes
			}
			b, _ := s.evaluateReviewReuse(f.bundle, current, true)
			if len(b) == 0 || !(strings.Contains(strings.Join(b, " "), "inconclusive")) {
				t.Fatalf("budget returned success: %v", b)
			}
		})
	}
	s := f.store.reviewOperation()
	if _, err := s.loadReuseAttestation(source.AttestationID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.loadReuseAttestation(source.AttestationID); err != nil || s.reviewReuse.nodes != 1 {
		t.Fatalf("memo nodes=%d err=%v", s.reviewReuse.nodes, err)
	}
	if s.reviewReuse == f.store.reviewOperation().reviewReuse {
		t.Fatal("shared cache between operations")
	}
}

func TestReviewReuseProvenanceRejectsLargeSourceBeforeAllocation(t *testing.T) {
	f, source, current, path := reuseSourceFixture(t)
	file, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxReviewBundleBytes + 1); err != nil {
		t.Fatal(err)
	}
	file.Close()
	b, _ := f.store.evaluateReviewReuse(f.bundle, current, true)
	if len(b) == 0 {
		t.Fatal("oversized source accepted")
	}
	_ = source
}

func TestReviewReuseProvenanceRealChainStopsAtDepthBudget(t *testing.T) {
	f, source, current, _ := reuseSourceFixture(t)
	for i := 0; i < reviewReuseMaxDepth+2; i++ {
		next := current
		next.Authority = new(ReviewAuthorityClaim)
		*next.Authority = *current.Authority
		next.Authority.ReviewExecution = fmt.Sprintf("review-chain-%d", i)
		next.ReusedFrom = append([]ReviewAttestationReuse{}, current.ReusedFrom...)
		next.ReusedFrom[0].FromAttestation = source.AttestationID
		next = resignAuthorityAttestation(t, f, next)
		raw, err := json.Marshal(next)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(f.root, ".pose/review-attestations", next.AttestationID+".json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		source = next
	}
	current.ReusedFrom[0].FromAttestation = source.AttestationID
	current = resignAuthorityAttestation(t, f, current)
	blockers, _ := f.store.atReviewTime(f.now).evaluateReviewReuse(f.bundle, current, true)
	if !strings.Contains(strings.Join(blockers, " "), "depth budget exceeded") {
		t.Fatalf("deep chain not stopped: %v", blockers)
	}
}
