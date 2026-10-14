package cli

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/harne8/pose-mcp/internal/mcpserver"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReviewTrustReuseCLILegacyTimeIsNotAHistoricalReceipt(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	sealAndAttestCLIBundle(t, root, "agent:reviewer")
	domain, err := (posemodel.Store{Root: root}).VerifyReviewBundle("spec:bundle")
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	inDir(t, root, func() { Main([]string{"review", "verify", "spec:bundle", "--json"}, &out, &errOut) })
	var got posemodel.ReviewBundleVerification
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("%v: %s %s", err, out.String(), errOut.String())
	}
	if got.Assurance.Temporal == nil || domain.Assurance.Temporal == nil || got.Assurance.Temporal.TimeBasis != domain.Assurance.Temporal.TimeBasis || got.Assurance.Temporal.HistoricalValidity != "not-established" || got.Assurance.Temporal.HistoricalAuthorization != "not-established" {
		t.Fatalf("legacy history promoted: %+v", got.Assurance)
	}
	for _, args := range [][]string{{"review", "verify", "spec:bundle"}, {"review-check", "spec:bundle"}, {"closeout-check", "spec:bundle"}} {
		out.Reset()
		errOut.Reset()
		inDir(t, root, func() { Main(args, &out, &errOut) })
		if !strings.Contains(out.String(), "historical authorization not-established") || !strings.Contains(out.String(), "legacy-editable-attested-at") {
			t.Fatalf("surface %v: %s %s", args, out.String(), errOut.String())
		}
	}
}

func TestReviewTrustReuseCLISignedHistoryMatchesMCPAndDomain(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 31)
	}
	private := ed25519.NewKeyFromSeed(seed)
	public := private.Public().(ed25519.PublicKey)
	policyPath := filepath.Join(root, ".pose/policy/review.json")
	raw, _ := os.ReadFile(policyPath)
	var policy map[string]any
	json.Unmarshal(raw, &policy)
	policy["identity_assurance"] = map[string]string{"spec": "verified"}
	policy["authority_audience"] = "trust-fixture"
	policy["authority_project"] = "trust-fixture"
	keyHash := sha256.Sum256(public)
	policy["trusted_attestation_issuers"] = []string{fmt.Sprintf("fixture#sha256:%x", keyHash)}
	raw, _ = json.Marshal(policy)
	os.WriteFile(policyPath, raw, 0600)
	bundle, _ := sealCLIBundle(t, root)
	store := posemodel.Store{Root: root}
	prepared, err := store.PrepareReviewAttestation(bundle.BundleID, "agent:reviewer", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	att := prepared.Attestation
	att.AttestedAt = "2026-01-01T12:00:00Z"
	att.Authority = &posemodel.ReviewAuthorityClaim{SchemaVersion: 1, Project: "trust-fixture", Audience: "trust-fixture", BundleDigest: bundle.BundleDigest, Principal: "agent:reviewer", Role: "agent", ReviewExecution: "review-run", ImplementationPrincipal: "agent:implementer", ImplementationExecution: "implementation-run", Issuer: "fixture", IssuedAt: "2026-01-01T11:00:00Z", ExpiresAt: "2026-01-02T12:00:00Z"}
	att.AttestationID = ""
	completed, canonical, err := store.SignableReviewAttestation(att, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	completed.Envelope = &posemodel.ReviewAttestationSignature{Issuer: "fixture", Subject: bundle.BundleID, Algorithm: "ed25519", PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, canonical))}
	// Seed a historical signed record, rather than authorizing an expired write.
	dir := filepath.Join(root, ".pose/review-attestations")
	os.MkdirAll(dir, 0700)
	raw, _ = json.Marshal(completed)
	os.WriteFile(filepath.Join(dir, completed.AttestationID+".json"), raw, 0600)
	domain, err := store.VerifyReviewBundle("spec:bundle")
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	inDir(t, root, func() { Main([]string{"review", "verify", "spec:bundle", "--json"}, &out, &errOut) })
	var got posemodel.ReviewBundleVerification
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("%v: %s %s", err, out.String(), errOut.String())
	}
	if !got.Approved || got.Assurance == nil || got.Assurance.ReviewerIdentity != "verified" || got.Assurance.Temporal.HistoricalValidity != "valid-claim" || got.Assurance.Temporal.CurrentValidity != "invalid-claim" {
		t.Fatalf("history lost or current approval promoted: %+v %s", got, errOut.String())
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.trust-test")
	ts := httptest.NewServer(mcpserver.New(store).Handler("", ""))
	defer ts.Close()
	response, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_review_bundle","arguments":{"scope":"spec:bundle"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var rpc struct {
		Result struct {
			StructuredContent posemodel.ReviewBundleVerification `json:"structuredContent"`
		}
		Error any
	}
	if err := json.NewDecoder(response.Body).Decode(&rpc); err != nil {
		t.Fatal(err)
	}
	mcp := rpc.Result.StructuredContent
	if rpc.Error != nil || mcp.Assurance == nil || mcp.Assurance.Temporal == nil || mcp.Assurance.ReviewerIdentity != domain.Assurance.ReviewerIdentity || mcp.Assurance.Temporal.TimeBasis != got.Assurance.Temporal.TimeBasis || mcp.Assurance.Temporal.CurrentValidity != got.Assurance.Temporal.CurrentValidity || mcp.Approved != got.Approved {
		t.Fatalf("MCP/domain/CLI diverged: %+v", rpc)
	}
}
