package pose

import (
	"encoding/json"
	"sort"
)

// CriterionReuseExplanation says why a criterion's answer can or cannot be
// carried from one bundle to the next (spec pose-material-equivalence-reuse).
// Reuse is decided by the criterion's input digest; this names which inputs
// differ, so a reviewer sees the material change rather than a verdict.
type CriterionReuseExplanation struct {
	Criterion     string   `json:"criterion"`
	Decision      string   `json:"decision"`
	ChangedInputs []string `json:"changed_inputs,omitempty"`
	ComparedOn    []string `json:"compared_on"`
}

// Reuse decisions.
const (
	ReuseEquivalent = "equivalent"
	ReuseChanged    = "changed"
	ReuseNew        = "new"
)

var criterionInputKinds = []string{"criterion", "independence", "scope", "subject", "patch", "tree", "evidence", "inputs", "tools"}

// ExplainCriterionReuse compares every criterion of to with from.
func ExplainCriterionReuse(from, to ReviewBundle) []CriterionReuseExplanation {
	previous := map[string]criterionInputContract{}
	for _, criterion := range from.Payload.Plan.Criteria {
		previous[criterion.ID] = reviewCriterionInputContract(from, criterion)
	}
	var out []CriterionReuseExplanation
	for _, criterion := range to.Payload.Plan.Criteria {
		current := reviewCriterionInputContract(to, criterion)
		explanation := CriterionReuseExplanation{Criterion: criterion.ID, ComparedOn: criterionInputKinds}
		prior, existed := previous[criterion.ID]
		switch {
		case !existed:
			explanation.Decision = ReuseNew
		default:
			explanation.ChangedInputs = changedCriterionInputs(prior, current)
			explanation.Decision = ReuseEquivalent
			if len(explanation.ChangedInputs) > 0 {
				explanation.Decision = ReuseChanged
			}
		}
		out = append(out, explanation)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Criterion < out[j].Criterion })
	return out
}

func changedCriterionInputs(a, b criterionInputContract) []string {
	var changed []string
	same := func(x, y any) bool {
		left, _ := json.Marshal(x)
		right, _ := json.Marshal(y)
		return string(left) == string(right)
	}
	if !same(a.Criterion, b.Criterion) {
		changed = append(changed, "criterion definition")
	}
	if a.Independence != b.Independence {
		changed = append(changed, "independence")
	}
	changed = append(changed, keyedChanges("scope", inputsByPath(a.Scope), inputsByPath(b.Scope))...)
	subjectA, subjectB := map[string]string{}, map[string]string{}
	for _, e := range a.Subject {
		subjectA[reviewBundleEntryPath(e)] = e.Digest
	}
	for _, e := range b.Subject {
		subjectB[reviewBundleEntryPath(e)] = e.Digest
	}
	changed = append(changed, keyedChanges("subject", subjectA, subjectB)...)
	if a.PatchDigest != b.PatchDigest {
		changed = append(changed, "subject patch")
	}
	if a.TreeDigest != b.TreeDigest {
		changed = append(changed, "subject tree")
	}
	evidenceA, evidenceB := map[string]string{}, map[string]string{}
	for _, e := range a.Evidence {
		raw, _ := json.Marshal(e)
		evidenceA[e.EvidenceClass+":"+e.ID] = digestBytes(raw)
	}
	for _, e := range b.Evidence {
		raw, _ := json.Marshal(e)
		evidenceB[e.EvidenceClass+":"+e.ID] = digestBytes(raw)
	}
	changed = append(changed, keyedChanges("evidence", evidenceA, evidenceB)...)
	changed = append(changed, keyedChanges("input", inputsByPath(a.Inputs), inputsByPath(b.Inputs))...)
	toolsA, toolsB := map[string]string{}, map[string]string{}
	for _, t := range a.Tools {
		raw, _ := json.Marshal(t)
		toolsA[t.ID+"@"+t.Component] = digestBytes(raw)
	}
	for _, t := range b.Tools {
		raw, _ := json.Marshal(t)
		toolsB[t.ID+"@"+t.Component] = digestBytes(raw)
	}
	changed = append(changed, keyedChanges("tool", toolsA, toolsB)...)
	return changed
}

func inputsByPath(inputs []ReviewBundleInput) map[string]string {
	out := map[string]string{}
	for _, input := range inputs {
		out[input.Kind+":"+input.Path] = input.Digest
	}
	return out
}

func keyedChanges(label string, a, b map[string]string) []string {
	var out []string
	for _, key := range changedReviewBundleKeys(a, b) {
		out = append(out, label+" "+key)
	}
	return out
}
