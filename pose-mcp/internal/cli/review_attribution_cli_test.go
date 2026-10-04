package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-review-attribution-roles.

func sealCLIBundle(t *testing.T, root string) (posemodel.ReviewBundle, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	if code := cmdReview(root, []string{"bundle", "spec:bundle", "--seal", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("seal code=%d err=%s", code, errOut.String())
	}
	var sealed posemodel.ReviewBundle
	if err := json.Unmarshal(out.Bytes(), &sealed); err != nil {
		t.Fatal(err)
	}
	return sealed, sealed.Payload.Evidence[0].EvidenceClass + ":" + sealed.Payload.Evidence[0].ID
}

func attestArgs(bundle, evidence, reviewer string, extra ...string) []string {
	args := []string{"attest", bundle, "--reviewer", reviewer, "--decision", "approved", "--evidence", evidence, "--tool", "artifact-check|-|passed|check:artifact|", "--tool", "validate|pose-mcp|passed|validation:module|"}
	return append(append(args, extra...), "--apply")
}

func onlyAttestation(t *testing.T, root string) posemodel.ReviewAttestation {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(root, ".pose", "review-attestations", "*.json"))
	if len(files) != 1 {
		t.Fatalf("attestations: %v", files)
	}
	raw, _ := os.ReadFile(files[0])
	var att posemodel.ReviewAttestation
	if err := json.Unmarshal(raw, &att); err != nil {
		t.Fatal(err)
	}
	return att
}

func TestReviewerFlagNeverFillsAttributionRoles(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	sealed, evidence := sealCLIBundle(t, root)
	var out, errOut bytes.Buffer
	if code := cmdReview(root, attestArgs(sealed.BundleID, evidence, "human:oseias"), &out, &errOut); code != 0 {
		t.Fatalf("attest: %s %s", out.String(), errOut.String())
	}
	if att := onlyAttestation(t, root); att.Attribution != nil {
		t.Fatalf("--reviewer populated attribution: %+v", att.Attribution)
	}
}

func TestReviewAttestRecordsExplicitAttributionAndVerifyRendersIt(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	sealed, evidence := sealCLIBundle(t, root)
	var out, errOut bytes.Buffer
	bad := attestArgs(sealed.BundleID, evidence, "agent:writer", "--confirmed-by", "human:oseias")
	if code := cmdReview(root, bad, &out, &errOut); code != 2 {
		t.Fatalf("confirmed-by without a mode was accepted: %d %s", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	good := attestArgs(sealed.BundleID, evidence, "agent:writer", "--prepared-by", "agent:writer", "--concluded-by", "agent:writer", "--applied-by", "agent:writer", "--confirmed-by", "human:oseias", "--confirmation-mode", "authorized-operation")
	if code := cmdReview(root, good, &out, &errOut); code != 0 {
		t.Fatalf("attest: %s %s", out.String(), errOut.String())
	}
	att := onlyAttestation(t, root)
	if att.Attribution == nil || att.Attribution.ConfirmationDigest != posemodel.ReviewConfirmationDigest(att) {
		t.Fatalf("confirmation not bound to the written content: %+v", att.Attribution)
	}
	var vout, verr bytes.Buffer
	inDir(t, root, func() { Main([]string{"review", "verify", "spec:bundle"}, &vout, &verr) })
	text := vout.String()
	if !strings.Contains(text, "confirmed by human:oseias (declared, authorized the operation, did not adopt the conclusions)") || !strings.Contains(text, "prepared by agent:writer") {
		t.Fatalf("verify does not render the attribution:\n%s", text)
	}

	var sout, serr bytes.Buffer
	inDir(t, root, func() {
		Main([]string{"review", "attribution-supplement", att.AttestationID, "--recorded-by", "agent:auditor", "--note", "clarifies the applier", "--applied-by", "agent:runner", "--apply"}, &sout, &serr)
	})
	if !strings.Contains(sout.String(), "review_attribution_supplement.id=ras-") {
		t.Fatalf("supplement not recorded: %s %s", sout.String(), serr.String())
	}
	sout.Reset()
	serr.Reset()
	inDir(t, root, func() {
		Main([]string{"review", "attribution-supplement", att.AttestationID, "--recorded-by", "agent:auditor", "--note", "n", "--confirmed-by", "human:oseias", "--apply"}, &sout, &serr)
	})
	if !strings.Contains(sout.String()+serr.String(), "cannot add a confirmation") {
		t.Fatalf("a supplement confirmation was not refused: %s %s", sout.String(), serr.String())
	}
}
