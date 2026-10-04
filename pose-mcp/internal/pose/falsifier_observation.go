package pose

import (
	"os"
	"strings"
)

// Falsifier observation (spec pose-falsifier-reconsideration). A material
// decision may state the effect it expects (`Expected effect:`) and name an
// observable check that would contradict it (`Falsifier check:
// check:<module>/<check>`). When an indexed validation result for that check
// fails, the decision is projected as a reconsideration candidate: someone
// judges whether the basis still holds. The engine infers no cause, judges
// no architecture, and never edits the decision — its rationale and history
// stay as written.

// FalsifierCheckRef is the parsed `Falsifier check:` reference.
type FalsifierCheckRef struct {
	Module string `json:"module"`
	Check  string `json:"check"`
}

func parseFalsifierCheck(value string) (FalsifierCheckRef, bool) {
	value = strings.TrimSpace(strings.Trim(strings.TrimSpace(value), "`"))
	if value == "" {
		return FalsifierCheckRef{}, false
	}
	rest, ok := strings.CutPrefix(value, "check:")
	if !ok {
		return FalsifierCheckRef{}, false
	}
	module, check, ok := strings.Cut(rest, "/")
	module, check = strings.TrimSpace(module), strings.TrimSpace(check)
	if !ok || module == "" || check == "" || strings.ContainsAny(check, " /") {
		return FalsifierCheckRef{}, false
	}
	return FalsifierCheckRef{Module: module, Check: check}, true
}

func validateFalsifierCheck(report *DesignBasisReport, item DesignDecision) {
	if item.FalsifierCheck == "" {
		return
	}
	if _, ok := parseFalsifierCheck(item.FalsifierCheck); !ok {
		report.addDiagnostic("warning", "falsifier-check-invalid", item.ID, item.Line,
			"falsifier check "+item.FalsifierCheck+" is not check:<module>/<check>; it is not observed")
	}
	if strings.TrimSpace(item.Falsifier) == "" {
		report.addDiagnostic("warning", "falsifier-check-without-falsifier", item.ID, item.Line,
			"decision "+item.ID+" names a falsifier check without stating the falsifier it observes")
	}
}

// falsifierObligations reads the indexed validation results only when some
// in-scope decision opted in.
func (s Store) falsifierObligations(project string, specs []Spec) ([]Obligation, error) {
	type candidate struct {
		spec     Spec
		decision DesignDecision
		ref      FalsifierCheckRef
	}
	var candidates []candidate
	for _, sp := range specs {
		if sp.Status == "superseded" || sp.Status == "abandoned" {
			continue
		}
		raw, err := os.ReadFile(sp.Path)
		if err != nil {
			return nil, err
		}
		if !strings.Contains(string(raw), "alsifier check") && !strings.Contains(string(raw), "erificação do falsificador") {
			continue
		}
		for _, d := range ParseDesignBasis(string(raw)).Decisions {
			if d.Status == "withdrawn" {
				continue
			}
			if ref, ok := parseFalsifierCheck(d.FalsifierCheck); ok {
				candidates = append(candidates, candidate{sp, d, ref})
			}
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	graph, err := s.GetDeliveryIntegrity("")
	if err != nil {
		return nil, err
	}
	results := map[string]DeliveryValidationResult{}
	for _, r := range graph.ValidationResults {
		results[r.Module+"/"+r.Check] = r
	}
	var out []Obligation
	for _, c := range candidates {
		r, observed := results[c.ref.Module+"/"+c.ref.Check]
		if !observed || r.Outcome != "fail" {
			continue
		}
		artifact := ArtifactRef{Kind: "spec", Slug: c.spec.Slug}
		node := QualifyNodeRef(project, artifact, "decision", c.decision.ID)
		o := newObligation(project, "falsifiers", node, "decision-reconsideration", c.decision.ID)
		o.Category = ObligationJudgment
		o.ReasonCode = "falsifier-observed"
		o.Effects = []ObligationEffect{{Phase: PhaseReview, Mode: EffectAdvisory, Scope: []NodeRef{node}}}
		o.Waiting = WaitingActor
		o.Observation.SourceRevision = r.GitHead
		o.Condition = "someone judges whether decision " + c.decision.ID + " still holds given the observation, and either records why it does or opens a superseding decision; the original rationale is kept"
		o.Message = "decision " + c.decision.ID + " of spec " + c.spec.Slug + " is a reconsideration candidate: its falsifier check " + c.ref.Module + "/" + c.ref.Check + " failed"
		if r.GitHead != "" {
			o.Message += " at " + r.GitHead
		}
		if c.decision.ExpectedEffect != "" {
			o.Message += "; expected effect: " + c.decision.ExpectedEffect
		}
		o.Message += "; falsifier: " + c.decision.Falsifier + ". The failure is an observation, not a verdict on the decision"
		out = append(out, o)
	}
	return out, nil
}
