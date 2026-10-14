package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ReviewReuseVerifiedSource       = "verified-source"
	ReviewReuseCurrentReaffirmation = "current-reaffirmation"
	reviewReuseMaxDepth             = 16
	reviewReuseMaxNodes             = 32
	reviewReuseMaxEdges             = 128
	reviewReuseMaxBytes             = 16 << 20
)

type ReviewReuseDisclosure struct {
	Criterion       string   `json:"criterion"`
	FromAttestation string   `json:"from_attestation"`
	Mode            string   `json:"mode"`
	State           string   `json:"state"`
	Blockers        []string `json:"blockers,omitempty"`
}

type reviewReuseEvaluation struct {
	blockers   []string
	disclosure []ReviewReuseDisclosure
}

type reviewReuseTraversal struct {
	evaluations         map[string]reviewReuseEvaluation
	active              map[string]bool
	memo                map[string][]string
	attestations        map[string]ReviewAttestation
	bundles             map[string]ReviewBundle
	depth, nodes, edges int
	bytes               int64
}

func newReviewReuseTraversal() *reviewReuseTraversal {
	return &reviewReuseTraversal{evaluations: map[string]reviewReuseEvaluation{}, active: map[string]bool{}, memo: map[string][]string{}, attestations: map[string]ReviewAttestation{}, bundles: map[string]ReviewBundle{}}
}

func (s Store) reserveReuseArtifact(dir, id string) error {
	path := filepath.Join(s.Root, ".pose", dir, id+".json")
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("reuse source unavailable: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxReviewBundleBytes {
		return fmt.Errorf("reuse source is not a bounded regular artifact")
	}
	if s.reviewReuse.bytes+info.Size() > reviewReuseMaxBytes {
		return fmt.Errorf("reuse traversal byte budget exceeded (inconclusive)")
	}
	s.reviewReuse.bytes += info.Size()
	return nil
}

func (s Store) loadReuseAttestation(id string) (ReviewAttestation, error) {
	state := s.reviewReuse
	state.edges++
	if state.edges > reviewReuseMaxEdges {
		return ReviewAttestation{}, fmt.Errorf("reuse traversal edge budget exceeded (inconclusive)")
	}
	if att, ok := state.attestations[id]; ok {
		return att, nil
	}
	if state.nodes >= reviewReuseMaxNodes {
		return ReviewAttestation{}, fmt.Errorf("reuse traversal node budget exceeded (inconclusive)")
	}
	// Validate ids before constructing a path or inspecting source metadata.
	if len(id) != 20 || id[:4] != "rva-" {
		return ReviewAttestation{}, fmt.Errorf("invalid reuse attestation id")
	}
	for _, c := range id[4:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return ReviewAttestation{}, fmt.Errorf("invalid reuse attestation id")
		}
	}
	if err := s.reserveReuseArtifact("review-attestations", id); err != nil {
		return ReviewAttestation{}, err
	}
	att, err := s.LoadReviewAttestation(id)
	if err != nil {
		return att, err
	}
	state.nodes++
	state.attestations[id] = att
	return att, nil
}

func (s Store) loadReuseBundle(id string) (ReviewBundle, error) {
	if b, ok := s.reviewReuse.bundles[id]; ok {
		return b, nil
	}
	if len(id) != 20 || id[:4] != "rvb-" {
		return ReviewBundle{}, fmt.Errorf("invalid reuse bundle id")
	}
	for _, c := range id[4:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return ReviewBundle{}, fmt.Errorf("invalid reuse bundle id")
		}
	}
	if err := s.reserveReuseArtifact("review-bundles", id); err != nil {
		return ReviewBundle{}, err
	}
	b, err := s.LoadReviewBundle(id)
	if err == nil {
		s.reviewReuse.bundles[id] = b
	}
	return b, err
}

func (s Store) verifyReuseSource(bundle ReviewBundle, att ReviewAttestation) []string {
	s = s.reviewOperation()
	state := s.reviewReuse
	// The state belongs to one clock and trust context only. Content, including
	// proof, is in the memo key; a name alone never memoizes a verification.
	raw, _ := json.Marshal(att)
	key := att.AttestationID + ":" + bundle.BundleDigest + ":" + digestBytes(raw) + ":" + s.reviewTrustContextDigest()
	if state.active[att.AttestationID] {
		return []string{"reuse provenance cycle (inconclusive)"}
	}
	if state.depth >= reviewReuseMaxDepth {
		return []string{"reuse traversal depth budget exceeded (inconclusive)"}
	}
	if prior, ok := state.memo[key]; ok {
		return append([]string{}, prior...)
	}
	state.active[att.AttestationID] = true
	state.depth++
	defer func() { delete(state.active, att.AttestationID); state.depth-- }()
	blockers := []string{}
	if att.Envelope == nil {
		blockers = append(blockers, "verified-source requires a signed source; legacy reference is not a verified historical pass")
	} else if err := s.verifyStoredReviewAttestationSignature(att); err != nil {
		blockers = append(blockers, "reuse source proof: "+err.Error())
	}
	blockers = append(blockers, s.reviewAuthorityTime(att).HistoricalBlockers...)
	blockers = append(blockers, s.validateBundleAttestationWith(bundle, att, false)...)
	state.memo[key] = append([]string{}, blockers...)
	return blockers
}

func reviewReaffirmedCriterion(att ReviewAttestation, id string) bool {
	for _, c := range att.Criteria {
		if c.ID == id && c.Disposition == "passed" && strings.TrimSpace(c.Rationale) != "" {
			return true
		}
	}
	return false
}

func (s Store) describeReviewReuse(bundle ReviewBundle, att ReviewAttestation) []ReviewReuseDisclosure {
	_, disclosure := s.evaluateReviewReuse(bundle, att, bundle.Payload.SealedGates().AllowCriterionReuse)
	return disclosure
}

func (s Store) evaluateReviewReuse(bundle ReviewBundle, att ReviewAttestation, allowed bool) ([]string, []ReviewReuseDisclosure) {
	if len(att.ReusedFrom) == 0 {
		return nil, nil
	}
	s = s.reviewOperation()
	raw, _ := json.Marshal(att)
	key := bundle.BundleDigest + ":" + digestBytes(raw) + ":" + s.reviewTrustContextDigest() + fmt.Sprint(allowed)
	if result, ok := s.reviewReuse.evaluations[key]; ok {
		return result.blockers, result.disclosure
	}
	blockers := []string{}
	out := []ReviewReuseDisclosure{}
	seen := map[string]bool{}
	required := map[string]ReviewPlanCriterion{}
	for _, c := range bundle.Payload.Plan.Criteria {
		if c.Required {
			required[c.ID] = c
		}
	}
	for _, reuse := range att.ReusedFrom {
		d := ReviewReuseDisclosure{Criterion: reuse.Criterion, FromAttestation: reuse.FromAttestation, Mode: reuse.Mode, State: "inconclusive"}
		add := func(message string) { d.Blockers = append(d.Blockers, message) }
		if seen[reuse.Criterion] {
			add("duplicate reused criterion " + reuse.Criterion)
		}
		seen[reuse.Criterion] = true
		if !allowed {
			add("criterion reuse is not permitted by the bundle's sealed gates")
		}
		current, ok := required[reuse.Criterion]
		if !ok {
			add("reused criterion " + reuse.Criterion + " is not required by the current bundle")
		} else if reviewCriterionInputDigest(bundle, current) != reuse.InputDigest {
			add("reused criterion " + reuse.Criterion + " input digest changed")
		}
		if len(d.Blockers) == 0 {
			prior, err := s.loadReuseAttestation(reuse.FromAttestation)
			if err != nil {
				add("references an unavailable attestation: " + err.Error())
			} else if b, err := s.loadReuseBundle(prior.BundleID); err != nil {
				add("references an unavailable bundle: " + err.Error())
			} else {
				for _, blocker := range reviewFindingDecisionBlockers(b, prior) {
					add("references a non-approving attestation: " + blocker)
				}
				contract := ""
				for _, c := range b.Payload.Plan.Criteria {
					if c.ID == reuse.Criterion && c.Required {
						contract = reviewCriterionInputDigest(b, c)
					}
				}
				passed := false
				for _, c := range prior.Criteria {
					if c.ID == reuse.Criterion && c.Disposition == "passed" {
						passed = true
					}
				}
				if contract != reuse.InputDigest || !passed {
					add("source criterion is not unchanged and passed")
				}
				switch reuse.Mode {
				case ReviewReuseVerifiedSource:
					d.Blockers = append(d.Blockers, s.verifyReuseSource(b, prior)...)
					if len(d.Blockers) == 0 {
						d.State = "source-verified-under-current-trust"
					}
				case ReviewReuseCurrentReaffirmation:
					if !reviewReaffirmedCriterion(att, reuse.Criterion) {
						add("current-reaffirmation requires a current passed judgment with a conclusion")
					}
					if len(d.Blockers) == 0 {
						d.State = "current-judgment-recorded"
					}
				case "":
					d.Mode = "legacy-reaffirmation"
					if len(d.Blockers) == 0 {
						d.State = "historical-pass-not-verified"
					}
				default:
					add("unsupported reuse mode")
				}
			}
		}
		for _, b := range d.Blockers {
			blockers = append(blockers, "reused criterion "+reuse.Criterion+": "+b)
		}
		out = append(out, d)
	}
	s.reviewReuse.evaluations[key] = reviewReuseEvaluation{blockers, out}
	return blockers, out
}

func (s Store) reviewTrustContextDigest() string {
	policy, _, err := s.loadReviewPolicy()
	if err != nil {
		return "invalid-policy:" + err.Error()
	}
	raw, _ := json.Marshal(policy)
	return digestBytes(raw)
}
