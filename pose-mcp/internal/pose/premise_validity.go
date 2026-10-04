package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Premise validity (spec pose-assumption-validity-scope). A material
// assumption may declare the context it holds in (`Valid scope:`) and the
// material things whose change should make someone look at it again
// (`Stale trigger:`). A trigger pins the content it was judged against; when
// that content changes the premise is projected as stale. Stale asks for
// judgment: the evidence stays recorded, it is no longer presented as valid
// for the new content, and nothing about the design is invalidated
// automatically. There is no calendar expiry: validity is by context.

// DesignTrigger is one `<kind>:<value>@<pin>` entry of `Stale trigger:`.
type DesignTrigger struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
	Pin   string `json:"pin,omitempty"`
	// State is a projection detail (current, stale, unpinned, unknown,
	// unsupported) and is excluded from the semantic digest, like a
	// reference's resolution.
	State   string `json:"state,omitempty"`
	Current string `json:"current,omitempty"`
}

// Trigger states.
const (
	TriggerCurrent     = "current"
	TriggerStale       = "stale"
	TriggerUnpinned    = "unpinned"
	TriggerUnknown     = "unknown"
	TriggerUnsupported = "unsupported"
)

var triggerPinRE = regexp.MustCompile(`^[0-9a-f]{8,64}$`)

// calendarTriggerKinds are refused: an arbitrary TTL is not a material
// trigger, and trivial assumptions must not receive one.
var calendarTriggerKinds = map[string]bool{"date": true, "ttl": true, "expires": true, "expiry": true, "until": true}

func parseDesignTriggers(value, root string) []DesignTrigger {
	var out []DesignTrigger
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' }) {
		part = strings.TrimSpace(strings.Trim(strings.TrimSpace(part), "`"))
		if part == "" {
			continue
		}
		kind, rest, ok := strings.Cut(part, ":")
		kind = strings.ToLower(strings.TrimSpace(kind))
		trigger := DesignTrigger{Kind: kind}
		if !ok {
			trigger.Kind, trigger.State = "invalid", TriggerUnknown
			trigger.Value = part
			out = append(out, trigger)
			continue
		}
		trigger.Value = strings.TrimSpace(rest)
		if at := strings.LastIndex(trigger.Value, "@"); at > 0 {
			trigger.Pin = strings.ToLower(strings.TrimSpace(trigger.Value[at+1:]))
			trigger.Value = strings.TrimSpace(trigger.Value[:at])
		}
		trigger.State, trigger.Current = triggerState(root, trigger)
		out = append(out, trigger)
	}
	return out
}

// triggerState compares a local trigger's pinned digest with the content
// now. Only repository files are compared offline; a provider or remote
// contract is declared and stays unsupported in the core.
func triggerState(root string, t DesignTrigger) (string, string) {
	if calendarTriggerKinds[t.Kind] {
		return TriggerUnsupported, ""
	}
	if !designLocalReference(t.Kind) || root == "" {
		return TriggerUnsupported, ""
	}
	path := triggerPath(root, t)
	if path == "" {
		return TriggerUnknown, ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return TriggerUnknown, ""
	}
	sum := sha256.Sum256(raw)
	current := hex.EncodeToString(sum[:])
	if t.Pin == "" {
		return TriggerUnpinned, current[:12]
	}
	if !triggerPinRE.MatchString(t.Pin) {
		return TriggerUnknown, current[:12]
	}
	if strings.HasPrefix(current, t.Pin) {
		return TriggerCurrent, current[:12]
	}
	return TriggerStale, current[:12]
}

func triggerPath(root string, t DesignTrigger) string {
	if filepath.IsAbs(t.Value) || strings.Contains(t.Value, "..") {
		return ""
	}
	if t.Kind == "doc" {
		return filepath.Join(root, filepath.FromSlash(t.Value))
	}
	if !designReferenceExists(root, t.Kind, t.Value) {
		return ""
	}
	dirs := map[string]string{"knowledge": "knowledge", "report": "reports", "adr": "adr", "snapshot": "results"}
	base := filepath.Join(root, ".pose", dirs[t.Kind], filepath.FromSlash(t.Value))
	for _, candidate := range []string{base, base + ".md", base + ".json"} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	matches, _ := filepath.Glob(filepath.Join(root, ".pose", dirs[t.Kind], "*-"+t.Value+".md"))
	if len(matches) == 1 {
		return matches[0]
	}
	return ""
}

func validatePremiseValidity(report *DesignBasisReport, item DesignAssumption) {
	for _, t := range item.StaleTriggers {
		switch {
		case calendarTriggerKinds[t.Kind]:
			report.addDiagnostic("warning", "calendar-trigger", item.ID, item.Line,
				"stale trigger "+t.Kind+":"+t.Value+" is a calendar expiry; validity is by context and material change, so it is not evaluated")
		case t.State == TriggerUnpinned:
			report.addDiagnostic("warning", "stale-trigger-unpinned", item.ID, item.Line,
				"stale trigger "+t.Kind+":"+t.Value+" has no pin; append @"+t.Current+" to bind it to the content the assumption was judged against")
		case t.State == TriggerUnknown:
			report.addDiagnostic("warning", "stale-trigger-unresolved", item.ID, item.Line,
				"stale trigger "+t.Kind+":"+t.Value+" cannot be read or its pin is malformed; its staleness is unknown")
		}
	}
}

// StaleTriggerList returns the triggers whose pinned content changed.
func (a DesignAssumption) StaleTriggerList() []DesignTrigger {
	var out []DesignTrigger
	for _, t := range a.StaleTriggers {
		if t.State == TriggerStale {
			out = append(out, t)
		}
	}
	return out
}

// premiseObligations projects one `premise-stale` judgment obligation per
// assumption with a stale trigger, for specs whose design is still being
// acted on. The effect is advisory on review, scoped to the assumption and
// the decisions it affects: a person decides whether the premise still holds.
func (s Store) premiseObligations(project string, specs []Spec) ([]Obligation, error) {
	var out []Obligation
	for _, sp := range specs {
		if sp.Status != "in-progress" && sp.Status != "draft" {
			continue
		}
		raw, err := os.ReadFile(sp.Path)
		if err != nil {
			return nil, err
		}
		basis := ValidateDesignBasis(string(raw), s.Root)
		for _, a := range basis.Assumptions {
			stale := a.StaleTriggerList()
			if len(stale) == 0 {
				continue
			}
			artifact := ArtifactRef{Kind: "spec", Slug: sp.Slug}
			node := QualifyNodeRef(project, artifact, "assumption", a.ID)
			o := newObligation(project, "premises", node, "premise-stale", a.ID)
			o.Category = ObligationJudgment
			o.ReasonCode = "premise-stale"
			scope := []NodeRef{node}
			for _, d := range basis.Decisions {
				for _, b := range d.Basis {
					if b == a.ID {
						scope = append(scope, QualifyNodeRef(project, artifact, "decision", d.ID))
					}
				}
			}
			o.Targets = scope
			o.Effects = []ObligationEffect{{Phase: PhaseReview, Mode: EffectAdvisory, Scope: scope}}
			o.Waiting = WaitingActor
			var changed []string
			for _, t := range stale {
				changed = append(changed, t.Kind+":"+t.Value+" (pinned "+t.Pin+", now "+t.Current+")")
			}
			o.Condition = "someone judges whether assumption " + a.ID + " still holds for the changed content and re-pins its trigger, or revises the assumption and the decisions based on it"
			o.Message = "premise " + a.ID + " of spec " + sp.Slug + " may be stale: " + strings.Join(changed, ", ") + "; its recorded evidence was judged against the pinned content and is not presented as valid for the current one"
			if a.ValidScope != "" {
				o.Message += "; declared valid scope: " + a.ValidScope
			}
			out = append(out, o)
		}
	}
	return out, nil
}
