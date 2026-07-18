package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var acceptanceIDRE = regexp.MustCompile(`(?m)^\s*-\s*R\d+\s*[:—-]`)

// hasAcceptanceCriteriaIDs detects stable acceptance-criteria IDs in the spec
// body. Section-level precision lives in pose-lint-spec.py --ready-check; here
// a body-wide match keeps the Go side dependency-free and errs permissive.
func hasAcceptanceCriteriaIDs(body string) bool {
	return acceptanceIDRE.MatchString(body)
}

// milestoneWaitingReason resolves a milestone ref ("<roadmap>/<id>"): empty
// reason = satisfied (every spec of the milestone is done).
func (s Store) milestoneWaitingReason(ref string) string {
	roadmapSlug, milestoneID, ok := strings.Cut(ref, "/")
	if !ok {
		return "malformed milestone ref (want milestone:<roadmap>/<id>)"
	}
	rm, err := s.GetRoadmap(roadmapSlug)
	if err != nil {
		return fmt.Sprintf("roadmap %q not found", roadmapSlug)
	}
	for _, ms := range rm.Milestones {
		if ms.ID != milestoneID {
			continue
		}
		var pending []string
		for _, specSlug := range ms.Specs {
			dep, depErr := s.GetSpec(specSlug)
			if depErr != nil {
				pending = append(pending, specSlug+" (not found)")
				continue
			}
			if dep.Status != "done" {
				pending = append(pending, fmt.Sprintf("%s (%s)", specSlug, dep.Status))
			}
		}
		if len(pending) == 0 {
			return ""
		}
		return "milestone specs pending: " + strings.Join(pending, ", ")
	}
	return fmt.Sprintf("milestone %q not found in roadmap %q", milestoneID, roadmapSlug)
}

// roadmapWaitingReason resolves a roadmap ref: empty reason = roadmap done.
func (s Store) roadmapWaitingReason(slug string) string {
	rm, err := s.GetRoadmap(slug)
	if err != nil {
		return fmt.Sprintf("roadmap %q not found", slug)
	}
	if rm.Status == "done" {
		return ""
	}
	return fmt.Sprintf("roadmap status is %q (needs done)", rm.Status)
}

// dorApplies reads the adoption cutoff from .pose/policy/dor.json: the DoR
// readiness condition only applies to specs created on/after adopted_at.
// Missing/invalid policy or created_at disables the condition (fail-open here
// by design — the transition gate in pose check is the enforcing layer).
func (s Store) dorApplies(createdAt string) bool {
	if createdAt == "" {
		return false
	}
	raw, err := os.ReadFile(filepath.Join(s.Root, ".pose", "policy", "dor.json"))
	if err != nil {
		return false
	}
	var policy struct {
		AdoptedAt string `json:"adopted_at"`
	}
	if json.Unmarshal(raw, &policy) != nil || policy.AdoptedAt == "" {
		return false
	}
	return createdAt >= policy.AdoptedAt
}

// WaitingRef is one unsatisfied dependency of a spec, with the reason it does
// not count as satisfied yet.
type WaitingRef struct {
	Ref    string `json:"ref"`
	Reason string `json:"reason"`
}

// Readiness answers "can this spec be worked on / executed now?" from the
// governance point of view (pose-spec-dependencies): the spec must not be in a
// terminal lifecycle status and every depends_on ref must be satisfied.
// Typed refs (milestone:/roadmap:) are fail-closed with an explicit reason
// until roadmaps are projected (pose-roadmap-artifact).
type Readiness struct {
	Slug      string       `json:"slug"`
	Status    string       `json:"status"`
	Ready     bool         `json:"ready"`
	WaitingOn []WaitingRef `json:"waiting_on"`
	Reason    string       `json:"reason,omitempty"`
}

var terminalStatuses = map[string]bool{
	"done":       true,
	"superseded": true,
	"abandoned":  true,
}

// SpecReadiness resolves the eligibility of one spec. Spec-type refs are
// satisfied when the referenced spec has status done; unknown refs and typed
// refs report an explicit reason and keep the spec not-ready (fail-closed —
// the execution gate may downgrade per policy, never silently here).
func (s Store) SpecReadiness(slug string) (*Readiness, error) {
	sp, err := s.GetSpec(slug)
	if err != nil {
		return nil, err
	}
	r := &Readiness{Slug: sp.Slug, Status: sp.Status, WaitingOn: []WaitingRef{}}

	if terminalStatuses[sp.Status] {
		r.Reason = fmt.Sprintf("spec is in terminal status %q", sp.Status)
		return r, nil
	}
	if sp.Status == "blocked" {
		r.Reason = "spec is explicitly blocked"
		return r, nil
	}

	// Definition of Ready (pose-definition-of-ready): specs criadas a partir do
	// cutoff de adoção precisam de acceptance criteria com IDs estáveis antes de
	// serem elegíveis. Specs anteriores são legadas (isentas) — o gate de
	// transição do pose check cobre o funil novo.
	if s.dorApplies(sp.CreatedAt) && !hasAcceptanceCriteriaIDs(sp.Body) {
		r.WaitingOn = append(r.WaitingOn, WaitingRef{
			Ref:    "dor:acceptance-criteria",
			Reason: "Definition of Ready: no acceptance criteria with stable IDs (- R<N>: ...) in Requirements",
		})
	}

	for _, ref := range sp.DependsOn {
		switch {
		case strings.HasPrefix(ref, "milestone:"):
			// milestone:<roadmap>/<id> — satisfeito quando todas as suas specs
			// estão done (pose-roadmap-artifact R4). Fail-closed com razão
			// quando o roadmap/milestone não resolve.
			if reason := s.milestoneWaitingReason(strings.TrimPrefix(ref, "milestone:")); reason != "" {
				r.WaitingOn = append(r.WaitingOn, WaitingRef{Ref: ref, Reason: reason})
			}
		case strings.HasPrefix(ref, "roadmap:"):
			// roadmap:<slug> — satisfeito quando o roadmap está done.
			if reason := s.roadmapWaitingReason(strings.TrimPrefix(ref, "roadmap:")); reason != "" {
				r.WaitingOn = append(r.WaitingOn, WaitingRef{Ref: ref, Reason: reason})
			}
		case strings.Contains(ref, ":"):
			r.WaitingOn = append(r.WaitingOn, WaitingRef{Ref: ref, Reason: "unknown ref type"})
		default:
			dep, depErr := s.GetSpec(ref)
			if depErr != nil {
				r.WaitingOn = append(r.WaitingOn, WaitingRef{Ref: ref, Reason: "spec not found"})
				continue
			}
			if dep.Status != "done" {
				r.WaitingOn = append(r.WaitingOn, WaitingRef{
					Ref:    ref,
					Reason: fmt.Sprintf("spec status is %q (needs done)", dep.Status),
				})
			}
		}
	}

	r.Ready = len(r.WaitingOn) == 0
	if !r.Ready && r.Reason == "" {
		r.Reason = fmt.Sprintf("%d unsatisfied dependency(ies)", len(r.WaitingOn))
	}
	return r, nil
}
