package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-review-assurance-disclosure: every review surface discloses the
// assurance of the same record the same way.
func sealAndAttestCLIBundle(t *testing.T, root, reviewer string) {
	t.Helper()
	var out, errOut bytes.Buffer
	if code := cmdReview(root, []string{"bundle", "spec:bundle", "--seal", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("seal code=%d err=%s", code, errOut.String())
	}
	var sealed posemodel.ReviewBundle
	if err := json.Unmarshal(out.Bytes(), &sealed); err != nil {
		t.Fatal(err)
	}
	evidence := sealed.Payload.Evidence[0].EvidenceClass + ":" + sealed.Payload.Evidence[0].ID
	out.Reset()
	errOut.Reset()
	attest := []string{"attest", sealed.BundleID, "--reviewer", reviewer, "--decision", "approved", "--evidence", evidence, "--tool", "artifact-check|-|passed|check:artifact|", "--tool", "validate|pose-mcp|passed|validation:module|", "--apply"}
	if code := cmdReview(root, attest, &out, &errOut); code != 0 {
		t.Fatalf("attest code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
}

func TestReviewSurfacesDiscloseDeclaredAssuranceOfAHumanLabel(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	sealAndAttestCLIBundle(t, root, "human:oseias")

	run := func(args ...string) string {
		var out, errOut bytes.Buffer
		inDir(t, root, func() { Main(args, &out, &errOut) })
		return out.String() + errOut.String()
	}
	plan := run("review-plan", "spec:bundle")
	if !strings.Contains(plan, "review_plan.identity_assurance=declared (the reviewer identity is the string the reviewer writes") {
		t.Fatalf("review-plan does not disclose declared assurance:\n%s", plan)
	}
	want := "human:oseias — declared identity; assurance declared"
	for _, tc := range []struct{ name, out, field string }{
		{"verify", run("review", "verify", "spec:bundle"), "review_verify.assurance="},
		{"review-check", run("review-check", "spec:bundle"), "review.assurance="},
		{"closeout-check", run("closeout-check", "spec:bundle"), "closeout.review_assurance="},
	} {
		if !strings.Contains(tc.out, tc.field+want) {
			t.Errorf("%s does not disclose the human label as declared:\n%s", tc.name, tc.out)
		}
		if strings.Contains(tc.out, "verified human") {
			t.Errorf("%s renders a declared human prefix as a verified person:\n%s", tc.name, tc.out)
		}
	}

	var out, errOut bytes.Buffer
	inDir(t, root, func() { Main([]string{"review", "verify", "spec:bundle", "--json"}, &out, &errOut) })
	var verification posemodel.ReviewBundleVerification
	if err := json.Unmarshal(out.Bytes(), &verification); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	if verification.Assurance == nil || verification.Assurance.ReviewerIdentity != "declared" || verification.Assurance.SeparationVerified != "not-verified" || verification.Assurance.CognitiveIndependence != "not-observable" {
		t.Fatalf("JSON assurance missing or overstated: %+v", verification.Assurance)
	}
}
