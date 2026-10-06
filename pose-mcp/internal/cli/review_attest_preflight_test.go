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

// Spec pose-attest-refuses-what-verify-rejects.

func sealPreflightBundle(t *testing.T) (string, posemodel.ReviewBundle) {
	t.Helper()
	root := reviewBundleCLIFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdReview(root, []string{"bundle", "spec:bundle", "--seal", "--json"}, &out, &errOut); code != 0 {
		t.Fatal(errOut.String())
	}
	var bundle posemodel.ReviewBundle
	if err := json.Unmarshal(out.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	return root, bundle
}

func attestationCount(t *testing.T, root string) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, ".pose", "review-attestations"))
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

func TestAttestPreflightRefusesWhatVerifyWouldReject(t *testing.T) {
	root, bundle := sealPreflightBundle(t)
	absent := "unit:not-in-this-bundle"
	for _, apply := range []bool{false, true} {
		args := []string{"attest", bundle.BundleID, "--reviewer", "agent:preflight", "--decision", "approved", "--evidence", absent, "--tool", "artifact-check|-|passed|check:artifact|", "--tool", "validate|pose-mcp|passed|validation:module|"}
		if apply {
			args = append(args, "--apply")
		}
		var out, errOut bytes.Buffer
		code := cmdReview(root, args, &out, &errOut)
		if code == 0 || !strings.Contains(errOut.String(), "verify would reject this attestation") || !strings.Contains(errOut.String(), "cites evidence absent from the sealed bundle") {
			t.Fatalf("apply=%v: attest accepted what verify rejects: code=%d out=%s err=%s", apply, code, out.String(), errOut.String())
		}
		if strings.Contains(out.String(), "review_attestation.plan=record") {
			t.Fatalf("apply=%v: a refused attestation was still previewed as a record plan:\n%s", apply, out.String())
		}
	}
	if n := attestationCount(t, root); n != 0 {
		t.Fatalf("a refused attestation was written: %d files", n)
	}
}

func TestAttestPreflightStillRecordsNegativeDecisions(t *testing.T) {
	root, bundle := sealPreflightBundle(t)
	ref := bundle.Payload.Evidence[0].EvidenceClass + ":" + bundle.Payload.Evidence[0].ID
	args := []string{"attest", bundle.BundleID, "--reviewer", "agent:preflight", "--decision", "changes-requested", "--evidence", ref, "--tool", "artifact-check|-|passed|check:artifact|", "--tool", "validate|pose-mcp|passed|validation:module|", "--finding", "f1|high|open|fix the contract|" + ref, "--apply"}
	var out, errOut bytes.Buffer
	if code := cmdReview(root, args, &out, &errOut); code != 0 {
		t.Fatalf("a negative decision was refused: code=%d err=%s", code, errOut.String())
	}
	if n := attestationCount(t, root); n != 1 {
		t.Fatalf("the negative decision was not recorded: %d files", n)
	}
}

func TestAttestPreflightLetsAValidApprovalThrough(t *testing.T) {
	root, bundle := sealPreflightBundle(t)
	ref := bundle.Payload.Evidence[0].EvidenceClass + ":" + bundle.Payload.Evidence[0].ID
	base := []string{"attest", bundle.BundleID, "--reviewer", "agent:preflight", "--decision", "approved", "--evidence", ref, "--tool", "artifact-check|-|passed|check:artifact|", "--tool", "validate|pose-mcp|passed|validation:module|"}
	var out, errOut bytes.Buffer
	if code := cmdReview(root, base, &out, &errOut); code != 0 || !strings.Contains(out.String(), "review_attestation.plan=record") {
		t.Fatalf("a valid preview was refused: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := cmdReview(root, append(base, "--apply"), &out, &errOut); code != 0 {
		t.Fatalf("a valid approval was refused: code=%d err=%s", code, errOut.String())
	}
	if n := attestationCount(t, root); n != 1 {
		t.Fatalf("the valid approval was not recorded: %d files", n)
	}
}

func TestAttestPreflightIsDocumented(t *testing.T) {
	for _, rel := range []string{"../../../POSE.md", "../../../locales/pt-BR/POSE.md", "../scaffold/dist/POSE.md", "../scaffold/dist/locales/pt-BR/POSE.md"} {
		raw, err := os.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "verify would reject") && !strings.Contains(string(raw), "o verify reprovaria") {
			t.Fatalf("%s does not document the attestation preflight", rel)
		}
	}
}
