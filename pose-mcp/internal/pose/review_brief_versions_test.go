package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
)

// Spec pose-review-brief-templates-are-immutable.
//
// A run records the digest of the brief it was sent, and its proof signs it. The text a
// template version renders is therefore part of what that version means: change the text
// and every run briefed with the version can no longer be re-rendered. The text of
// review-brief/4 changed three times under one version in this repository, before this
// guard. A golden digest of what each version renders, from a fixed bundle, makes a text
// change fail the build until it takes a new version.

const reviewBriefGoldenPath = "testdata/review_brief_template_digests.json"

// fixedBriefBundle is a bundle with every kind of content a review brief renders, and
// nothing that varies between runs.
func fixedBriefBundle() ReviewBundle {
	var bundle ReviewBundle
	bundle.BundleID = "rvb-0000000000000001"
	bundle.BundleDigest = "sha256:" + strings.Repeat("a", 64)
	bundle.Payload.Scope = ReviewBundleScope{Ref: "spec:fixed", Kind: "spec", Slug: "fixed", Components: []string{"core"}, Deliveries: []string{"capability:fixed"}}
	bundle.Payload.Subject.Entries = []ReviewBundleSubjectEntry{{Action: "modified", Path: "core/a.go", Class: "source", Digest: "sha256:" + strings.Repeat("b", 64), Reason: "changed"}}
	bundle.Payload.Plan.Independence = "different-actor"
	bundle.Payload.Plan.Criteria = []ReviewPlanCriterion{
		{ID: "correctness", Description: "The behavior is correct.", Required: true, Kind: ReviewCriterionKindMechanical, EvidenceClasses: []string{"unit"}},
		{ID: "design-causality", Description: "The design is explained.", Required: true, Kind: ReviewCriterionKindJudgment},
	}
	bundle.Payload.Plan.Tools = []ReviewPlanTool{
		{ID: "artifact-check", Requiredness: "required", Args: []string{"pose", "artifact-check", "--spec", "fixed"}},
		{ID: "surface-check", Requiredness: "required", Args: []string{"pose", "surface-check"}, EvidenceClasses: []string{"integration", "reachability"}},
		{ID: "assess-design", Requiredness: "recommended", Args: []string{"pose", "assess", "design"}},
	}
	bundle.Payload.Evidence = []ReviewBundleEvidence{{ID: "core/go/test", Module: "core", Check: "test", EvidenceClass: "unit", Outcome: "pass"}}
	return bundle
}

// richBriefBundle adds what only some briefs carry: structural facts (one shared with another
// spec) and a tool scoped to a component.
func richBriefBundle() ReviewBundle {
	bundle := fixedBriefBundle()
	bundle.Payload.Plan.Structure = &ReviewPlanStructure{Material: []ReviewStructuralFact{
		{ID: "SD-aaaa1111", Kind: "governance-contract", Action: "changed", Subject: "an ADR"},
		{ID: "SD-bbbb2222", Kind: "api-contract", Action: "added", Subject: "an endpoint", SharedWith: []string{"other-spec"}},
	}}
	bundle.Payload.Plan.Tools = append(bundle.Payload.Plan.Tools, ReviewPlanTool{ID: "assess-discover", Component: "core", Requiredness: "recommended", Args: []string{"pose", "assess", "discover"}})
	bundle.Payload.Subject.Entries = append(bundle.Payload.Subject.Entries, ReviewBundleSubjectEntry{Action: "renamed", OldPath: "core/old.go", NewPath: "core/new.go", Class: "source", Reason: "moved"})
	bundle.Payload.Plan.Criteria = append(bundle.Payload.Plan.Criteria, ReviewPlanCriterion{ID: "documentation", Description: "The docs say it.", Required: false, Kind: ReviewCriterionKindMechanical})
	return bundle
}

// renderFixedBriefs renders a template every way a brief can differ: bare, with implementer
// notes and structural facts, and as the second round with an open finding of the first. A
// sentence of any section, in any of them, changes the digest.
func renderFixedBriefs(template string) string {
	conclusion := reviewBriefAsksConclusion(template)
	var out []string
	rounds := []*ReviewRound{nil}
	if template != reviewBriefTemplateFirst {
		prior := &ReviewRun{RunID: "rrn-prior0000000001", BundleID: "rvb-prior0000000001", Output: "decision: changes-requested"}
		priorBundle := fixedBriefBundle()
		priorBundle.Payload.Subject.Head = "0123456789abcdef0123456789abcdef01234567"
		rounds = []*ReviewRound{
			{Number: 1},
			{Number: 2, Prior: prior, PriorBundle: &priorBundle, Broken: "a draft of round 1 could not be read", Open: []ReviewOpenFinding{{ID: "B1", Round: 1, Severity: "medium", Disposition: "blocker", File: "core/a.go", Line: 3, Summary: "a defect", Carried: "F1"}}},
			{Number: 3},
		}
	}
	for _, round := range rounds {
		out = append(out, renderReviewBriefBody(fixedBriefBundle(), ReviewBriefKindReview, template, "", nil, round, conclusion, nil))
		out = append(out, renderReviewBriefBody(richBriefBundle(), ReviewBriefKindReview, template, "Implementer note.\n```\na fence in the notes\n```\nSecond line.", nil, round, conclusion, nil))
	}
	return strings.Join(out, "\n=====\n")
}

func briefTemplateDigest(template string) string {
	sum := sha256.Sum256([]byte(renderFixedBriefs(template)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func briefTemplatesInUse() []string {
	versions := append([]string{reviewBriefTemplateConclusions}, reviewBriefEarlierTemplates...)
	sort.Strings(versions)
	return versions
}

func TestEveryReviewBriefTemplateVersionRendersWhatItRenderedWhenItWasPublished(t *testing.T) {
	var golden map[string]string
	raw, err := os.ReadFile(reviewBriefGoldenPath)
	if os.Getenv("UPDATE_BRIEF_GOLDEN") != "" {
		golden = map[string]string{}
		if err == nil {
			if jerr := json.Unmarshal(raw, &golden); jerr != nil {
				t.Fatalf("the recorded digests cannot be read, and would be replaced by today's: %v", jerr)
			}
		} else if !os.IsNotExist(err) {
			t.Fatalf("the recorded digests cannot be read: %v", err)
		}
		for _, version := range briefTemplatesInUse() {
			if _, kept := golden[version]; !kept { // a published version is never rewritten
				golden[version] = briefTemplateDigest(version)
			}
		}
		out, _ := json.MarshalIndent(golden, "", "  ")
		if err := os.WriteFile(reviewBriefGoldenPath, append(out, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err != nil {
		t.Fatalf("no golden digests of the review brief templates: %v", err)
	}
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	inUse := map[string]bool{}
	for _, version := range briefTemplatesInUse() {
		inUse[version] = true
	}
	for version := range golden {
		if !inUse[version] {
			t.Errorf("%s has a recorded digest but is no longer a template a run may have been briefed with: runs briefed with it could not be verified or resumed; keep it in reviewBriefEarlierTemplates", version)
		}
	}
	for _, version := range briefTemplatesInUse() {
		want, ok := golden[version]
		if !ok {
			t.Errorf("%s has no golden digest: record it with UPDATE_BRIEF_GOLDEN=1 when the version is published, and never rewrite one", version)
			continue
		}
		if got := briefTemplateDigest(version); got != want {
			t.Errorf("%s renders different text than when it was published (%s, was %s): runs briefed with it can no longer be re-rendered; a change to the text takes a new template version, and the old one keeps rendering", version, got, want)
		}
	}
}

// The guard itself: every section of every variant is under the digest, two renderings agree
// (also across a rebuilt bundle), and two versions never render the same text.
func TestTheDigestSeesEverySectionOfEveryVariant(t *testing.T) {
	text := renderFixedBriefs(reviewBriefTemplateConclusions)
	for _, section := range []string{"## Structural facts to answer for", "SD-bbbb2222", "also changed by other-spec", "Implementer note.", "[core]", "round 2", "B1", "The previous round is run rrn-prior0000000001", "draft of round 1 could not be read", "none recorded", "old.go", "(recommended", "fence in the notes"} {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(section)) {
			t.Errorf("the fixed renderings carry no %q: a sentence of that section could change without the digest moving", section)
		}
	}
	if renderFixedBriefs(reviewBriefTemplateConclusions) != text || briefTemplateDigest(reviewBriefTemplateConclusions) != briefTemplateDigest(reviewBriefTemplateConclusions) {
		t.Fatal("the rendering of a version is not deterministic")
	}
	if briefTemplateDigest(reviewBriefTemplateConclusions) == briefTemplateDigest(reviewBriefTemplateTypedConclusion) {
		t.Fatal("two versions render the same text: a new version must differ from the one before it")
	}
	sum := sha256.Sum256([]byte(strings.Replace(text, "Map each fact", "Map every fact", 1)))
	if "sha256:"+hex.EncodeToString(sum[:]) == briefTemplateDigest(reviewBriefTemplateConclusions) {
		t.Fatal("an edited sentence of the structural section did not change the digest")
	}
}
