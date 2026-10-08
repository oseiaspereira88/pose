package pose

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// GovernanceReplay is a counterfactual, never an authority or migration result.
// It only adds invariants to copies of sealed inputs in memory.
type GovernanceReplayReport struct {
	SchemaVersion                  int            `json:"schema_version"`
	Mode                           string         `json:"mode"`
	Complete                       bool           `json:"complete"`
	ArtifactsScanned               int            `json:"artifacts_scanned"`
	ArtifactsInvalid               int            `json:"artifacts_invalid"`
	Attestations                   int            `json:"attestations"`
	ApprovingAttestations          int            `json:"approving_attestations"`
	FrozenRejected                 int            `json:"frozen_rejected"`
	CounterfactualRejected         int            `json:"counterfactual_rejected"`
	AdditionalRejected             int            `json:"additional_rejected"`
	AdditionalRejectionDenominator int            `json:"additional_rejection_denominator"`
	CounterfactualAffectedEvidence int            `json:"counterfactual_affected_evidence"`
	RejectionReasons               map[string]int `json:"rejection_reasons"`
	LegacyAttestations             int            `json:"legacy_attestations"`
	Specs                          int            `json:"specs"`
	SpecsWithSubject               int            `json:"specs_with_subject"`
	SpecsWithDelta                 int            `json:"specs_with_delta"`
	SpecsUnknown                   int            `json:"specs_unknown"`
	StructuralFacts                int            `json:"structural_facts"`
	UntracedFacts                  int            `json:"untraced_facts"`
	// SharedCommitFacts counts material facts that came from a commit other
	// specs also claim. Only bundles sealed with path attribution can tell.
	SharedCommitFacts         int      `json:"shared_commit_facts"`
	SpecsWithUnknownStructure int      `json:"specs_with_unknown_structure"`
	UnbaselinedNodes          int      `json:"unbaselined_nodes"`
	UnknownStartOrigins       int      `json:"unknown_start_origins"`
	Limit                     int      `json:"limit"`
	Limitations               []string `json:"limitations"`
}

func replayReason(blocker string) string {
	switch {
	case strings.Contains(blocker, "no conclusion"):
		return "judgment-without-conclusion"
	case strings.Contains(blocker, "not-applicable"):
		return "inapplicable-despite-evidence"
	case strings.Contains(blocker, "finding") && strings.Contains(blocker, "risk"):
		return "unapproved-risk"
	case strings.Contains(blocker, "mapping") || strings.Contains(blocker, "structural") || strings.Contains(blocker, "basis"):
		return "untraced-structure"
	case strings.Contains(blocker, "evidence"):
		return "unsupported-evidence"
	case strings.Contains(blocker, "decision"):
		return "negative-decision"
	case strings.Contains(blocker, "signature") || strings.Contains(blocker, "trusted"):
		return "live-trust"
	default:
		return "other-invariant"
	}
}

// replayReviewBlockers retains the sealed digest and signature binding, and
// asks the existing validator about an explicitly hypothetical contract copy.
func (s Store) replayReviewBlockers(bundle ReviewBundle, att ReviewAttestation) ([]string, []string) {
	frozen := s.validateBundleAttestation(bundle, att)
	copy := bundle
	copy.Payload.GoverningContracts = uniqueSorted(append(append([]string{}, bundle.Payload.GoverningContracts...), "explicit-judgment", "structural-causality"))
	return frozen, s.validateBundleAttestation(copy, att)
}

// GovernanceReplay scans all local records up to a declared bound. Invalid
// records remain in coverage. It intentionally exports aggregates only.
func (s Store) GovernanceReplay(limit int) (GovernanceReplayReport, error) {
	if limit == 0 {
		limit = 10000
	}
	if limit < 1 || limit > 20000 {
		return GovernanceReplayReport{}, fmt.Errorf("pose: replay limit must be between 1 and 20000")
	}
	report := GovernanceReplayReport{SchemaVersion: 1, Mode: "read-only-counterfactual", Complete: true, Limit: limit, RejectionReasons: map[string]int{}, Limitations: []string{"Frozen historical approvals are not rewritten or revoked by this report.", "Record-level checks do not apply lifecycle grandfathering or determine current scope approval.", "Counterfactual rejection does not establish reviewer utility or causal benefit.", "No costs, retries, task latency or external adoption are inferred.", "Structural coverage uses the newest available sealed subject of each spec."}}
	bundles := map[string]ReviewBundle{}
	latest := map[string]ReviewBundle{}
	latestAtt := map[string]ReviewAttestation{}
	bytesLeft := int64(128 << 20)
	for _, kind := range []string{"review-bundles", "review-attestations"} {
		rel := ".pose/" + kind
		dir, err := ensureReviewArtifactDir(s.Root, rel, false)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return report, err
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return report, err
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			if report.ArtifactsScanned >= limit {
				report.Complete = false
				break
			}
			report.ArtifactsScanned++
			info, err := entry.Info()
			if err != nil || entry.Type()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > 1<<20 {
				report.ArtifactsInvalid++
				report.Complete = false
				continue
			}
			bytesLeft -= info.Size()
			if bytesLeft < 0 {
				report.Complete = false
				break
			}
			id := strings.TrimSuffix(entry.Name(), ".json")
			if kind == "review-bundles" {
				bundle, err := s.LoadReviewBundle(id)
				if err != nil {
					report.ArtifactsInvalid++
					report.Complete = false
					continue
				}
				bundles[id] = bundle
				old, ok := latest[bundle.Payload.Scope.Ref]
				if !ok || bundle.SealedAt > old.SealedAt || bundle.SealedAt == old.SealedAt && bundle.BundleID > old.BundleID {
					latest[bundle.Payload.Scope.Ref] = bundle
				}
				continue
			}
			att, err := s.LoadReviewAttestation(id)
			if err != nil {
				report.ArtifactsInvalid++
				report.Complete = false
				continue
			}
			report.Attestations++
			bundle, ok := bundles[att.BundleID]
			if !ok {
				report.ArtifactsInvalid++
				report.Complete = false
				continue
			}
			if old, ok := latestAtt[att.BundleID]; !ok || att.AttestedAt > old.AttestedAt || att.AttestedAt == old.AttestedAt && att.AttestationID > old.AttestationID {
				latestAtt[att.BundleID] = att
			}
			if governed, _ := BundleGovernedBy(bundle, "explicit-judgment"); !governed {
				report.LegacyAttestations++
			}
			if att.Decision != "approved" && att.Decision != "approved-with-reservations" {
				continue
			}
			report.ApprovingAttestations++
			frozen, hypothetical := s.replayReviewBlockers(bundle, att)
			if len(frozen) > 0 {
				report.FrozenRejected++
			} else {
				report.AdditionalRejectionDenominator++
			}
			if len(hypothetical) > 0 {
				report.CounterfactualRejected++
				report.CounterfactualAffectedEvidence += len(bundle.Payload.Evidence)
				if len(frozen) == 0 {
					report.AdditionalRejected++
				}
				seen := map[string]bool{}
				for _, blocker := range hypothetical {
					reason := replayReason(blocker)
					if !seen[reason] {
						report.RejectionReasons[reason]++
						seen[reason] = true
					}
				}
			}
		}
	}
	specs, err := s.ListSpecs("", "")
	if err != nil {
		return report, err
	}
	report.Specs = len(specs)
	for i, spec := range specs {
		if i >= limit {
			report.Complete = false
			report.SpecsUnknown += len(specs) - i
			break
		}
		status, err := s.GetStartStatus(spec.Slug)
		if err != nil {
			report.UnknownStartOrigins++
		} else {
			for _, node := range status.Nodes {
				if node.Origin == "legacy-unbaselined" {
					report.UnbaselinedNodes++
				}
			}
		}
		bundle, ok := latest["spec:"+spec.Slug]
		if !ok {
			report.SpecsUnknown++
			continue
		}
		report.SpecsWithSubject++
		delta, err := AssessDesignDelta(s.Root, bundle.Payload.Subject, bundle.Payload.Scope.Ref, DesignDeltaOptions{})
		if err != nil {
			report.SpecsWithUnknownStructure++
			continue
		}
		unknown := delta.Coverage.Truncated || delta.Status == "unknown"
		for _, detector := range delta.Coverage.Detectors {
			if detector.Unknown > 0 || detector.Unsupported > 0 || detector.State == "unknown" || detector.State == "unsupported" {
				unknown = true
			}
		}
		if unknown {
			report.SpecsWithUnknownStructure++
		}
		material := false
		mapped := map[string]bool{}
		if att, ok := latestAtt[bundle.BundleID]; ok {
			for _, criterion := range att.Criteria {
				for _, mapping := range criterion.Mappings {
					if mapping.Disposition == "mapped" {
						mapped[mapping.Delta] = true
					}
				}
			}
		}
		for _, fact := range delta.Deltas {
			if !StructuralDeltaIsMaterial(fact) {
				continue
			}
			material = true
			report.StructuralFacts++
			if len(fact.SharedWith) > 0 {
				report.SharedCommitFacts++
			}
			if !mapped[fact.DisplayID] {
				report.UntracedFacts++
			}
		}
		if material {
			report.SpecsWithDelta++
		}
	}
	// Stabilize only human-readable limitations; maps serialize deterministically.
	sort.Strings(report.Limitations)
	return report, nil
}
