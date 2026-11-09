package pose

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Spec pose-review-brief-states-the-conclusion-rules R2.
//
// The messages the preflight can refuse a conclusion with are read from the source of the
// functions that raise them. Each fragment either names the sentence of the brief that states
// its rule, or is exempt with the reason a reviewer cannot break it. A fragment that is in
// neither fails the build: a blocker added to the preflight cannot ship without the brief
// saying so, or without someone writing down why it need not.

var preflightFunctions = map[string][]string{
	"review_bundle.go":   {"validateBundleAttestationWith", "reviewCriterionEvidenceBlockers"},
	"review_closeout.go": {"evaluateReviewToolCoverageIn", "reviewToolEvidenceBlocker"},
}

// stated maps a fragment to the words of the current brief that state its rule.
var stated = map[string]string{
	"unknown criterion":              "every required criterion and every tool above is answered",
	"duplicate criterion":            "every required criterion and every tool above is answered",
	"missing criterion":              "every required criterion and every tool above is answered",
	"has invalid disposition":        "`passed`, `failed`, `not-used` or `deferred`",
	"lacks not-applicable rationale": "a `not-applicable` carries a `rationale`",
	"is not-applicable while the bundle seals evidence of class":                                        "refused when the bundle seals evidence of a class the criterion asks for",
	"; missing judgment is a pendency, not inapplicability":                                             "refused when the bundle seals evidence of a class the criterion asks for",
	"is disposed as a finding and names none":                                                           "disposed `finding` carries as its `evidence` the `id` of a finding",
	", which this attestation does not record":                                                          "the `id` of a finding listed in `findings`",
	"is a judgment criterion passed with no conclusion; record what was examined and what it concluded": "carries a `rationale` saying what you examined and what you concluded",
	"is passed with no evidence":                                                                        "exactly one of the refs listed under Sealed evidence",
	"cites evidence absent from the sealed bundle:":                                                     "exactly one of the refs listed under Sealed evidence",
	"requires evidence class":                                                                           "of a class the criterion asks for",
	"but cites evidence from":                                                                           "of that component",
	"unknown review tool disposition":                                                                   "every required criterion and every tool above is answered",
	"duplicate review tool disposition":                                                                 "every required criterion and every tool above is answered",
	"has no registered producer and still needs a not-used rationale":                                   "`not-used` is accepted for a required tool only when the plan says no producer exists",
	"lacks not-used rationale":                                                                          "a recommended tool you did not run is `not-used` with a `rationale`",
	"has invalid deferred disposition":                                                                  "answer it `deferred` with a `rationale`",
	"missing required review tool disposition":                                                          "every required criterion and every tool above is answered",
	"has no disposition":                                                                                "every required criterion and every tool above is answered",
	"has no evidence":                                                                                   "`command:` followed by the command you ran",
	"lacks a required evidence class":                                                                   "when the tool lists evidence classes, a ref of one of them that the bundle sealed",
	"cites evidence from":                                                                               "of that component",
	"every required criterion is not-applicable; a scope where the whole plan is inapplicable is not reviewed by it": "a `not-applicable` carries a `rationale`",
}

// exempt names fragments no reviewer's draft can break, and why.
var exempt = map[string]string{
	"attestation does not retain the sealed review limitations": "engine-written for a run-backed attestation",
	"attestation does not reference the exact sealed bundle":    "engine-written",
	"attested_at must be RFC3339":                               "engine-written",
	"reviewer execution identity is malformed":                  "the identity of the run, not of the draft",
	"different-actor": "a policy value, not a message",
	"delegated-review requires a signed run reference; reviewer prefix does not prove independence": "the run's proof, written by the engine",
	"agent:independent-": "a prefix constant, not a message",
	"review policy requires a different-actor reviewer identity; under declared assurance that is read from the reviewer prefix, not authenticated": "the identity of the run",
	"mandatory-human":                       "a policy value, not a message",
	"review policy requires human approval": "a policy refusal of a delegated run altogether",
	"explicit-judgment":                     "a contract name, not a message",
	"structural-causality":                  "a contract name, not a message",
	"not-applicable":                        "a disposition name, not a message",
	"review-complete":                       "a precondition name, not a message",
	"delivery-target-declared":              "a precondition name, not a message",
	"required review tool":                  "the prefix of a message whose rule is stated by its other fragment",
	"recommended review tool":               "the prefix of a message whose rule is stated by its other fragment",
}

func preflightFragments(t *testing.T) []string {
	t.Helper()
	seen := map[string]bool{}
	for file, funcs := range preflightFunctions {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			named := false
			for _, name := range funcs {
				named = named || fn.Name.Name == name
			}
			if !named {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				if lit, ok := node.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if value, err := strconv.Unquote(lit.Value); err == nil && len(strings.TrimSpace(value)) >= 14 {
						seen[strings.TrimSpace(value)] = true
					}
				}
				return true
			})
		}
	}
	out := make([]string, 0, len(seen))
	for fragment := range seen {
		out = append(out, fragment)
	}
	sort.Strings(out)
	if len(out) < 30 {
		t.Fatalf("read only %d message fragments from the preflight: the extraction lost them", len(out))
	}
	return out
}

func TestEveryMessageThePreflightCanRefuseWithIsStatedByTheBriefOrExempt(t *testing.T) {
	brief := renderReviewBriefBody(fixedBriefBundle(), ReviewBriefKindReview, reviewBriefTemplates[ReviewBriefKindReview], "", nil, &ReviewRound{Number: 1}, true, nil)
	for _, fragment := range preflightFragments(t) {
		if phrase, ok := stated[fragment]; ok {
			if !strings.Contains(brief, phrase) {
				t.Errorf("the brief does not state the rule behind %q (%q)", fragment, phrase)
			}
			continue
		}
		if _, ok := exempt[fragment]; ok {
			continue
		}
		t.Errorf("the preflight can refuse a conclusion with %q and neither the brief states the rule nor is it exempt: add the sentence to the brief (a new template version) or record why a reviewer cannot break it", fragment)
	}
	for fragment := range stated {
		found := false
		for _, seen := range preflightFragments(t) {
			found = found || seen == fragment
		}
		if !found {
			t.Errorf("the table names %q, which the preflight no longer raises: remove it", fragment)
		}
	}
}
