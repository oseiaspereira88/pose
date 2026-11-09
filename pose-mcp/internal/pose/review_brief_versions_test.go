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

func renderFixedBrief(template string) string {
	var round *ReviewRound
	if template != reviewBriefTemplateFirst {
		round = &ReviewRound{Number: 1}
	}
	return renderReviewBriefBody(fixedBriefBundle(), ReviewBriefKindReview, template, "", nil, round, reviewBriefAsksConclusion(template), nil)
}

func briefTemplateDigest(template string) string {
	sum := sha256.Sum256([]byte(renderFixedBrief(template)))
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
			_ = json.Unmarshal(raw, &golden)
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

// The guard itself: a change to the text under the same version is seen.
func TestAChangeToTheTextOfATemplateVersionChangesItsDigest(t *testing.T) {
	before := briefTemplateDigest(reviewBriefTemplateConclusions)
	text := renderFixedBrief(reviewBriefTemplateConclusions)
	sum := sha256.Sum256([]byte(text + "an added sentence"))
	if "sha256:"+hex.EncodeToString(sum[:]) == before {
		t.Fatal("a changed text has the digest of the original")
	}
	if briefTemplateDigest(reviewBriefTemplateConclusions) != before {
		t.Fatal("the rendering of a version is not deterministic")
	}
	if briefTemplateDigest(reviewBriefTemplateConclusions) == briefTemplateDigest(reviewBriefTemplateTypedConclusion) {
		t.Fatal("two versions render the same text: a new version must differ from the one before it")
	}
}
