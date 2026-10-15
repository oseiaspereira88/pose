package cli

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/mcpserver"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// R1–R4 of pose-review-disclosure-requires-proof: compare actual CLI and
// JSON-RPC consumers against a signed positive control, then change only the
// proof or the semantic criteria. Identity and approval answer different facts.
func TestReviewAssuranceProofCLIAndMCPConsumers(t *testing.T) {
	for _, state := range []string{"valid", "damaged-signature", "missing-proof", "invalid-criteria"} {
		t.Run(state, func(t *testing.T) {
			root := reviewBundleCLIFixture(t)
			public, private, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			policyPath := filepath.Join(root, ".pose/policy/review.json")
			raw, err := os.ReadFile(policyPath)
			if err != nil {
				t.Fatal(err)
			}
			var policy map[string]any
			if err := json.Unmarshal(raw, &policy); err != nil {
				t.Fatal(err)
			}
			policy["identity_assurance"] = map[string]string{"spec": "verified"}
			policy["authority_audience"] = "proof-fixture"
			policy["authority_project"] = "proof-fixture"
			policy["trusted_attestation_issuers"] = []string{fmt.Sprintf("fixture#sha256:%x", sha256.Sum256(public))}
			raw, err = json.Marshal(policy)
			if err != nil {
				t.Fatal(err)
			}
			writeCloseoutCLIFile(t, root, ".pose/policy/review.json", string(raw))
			bundle, _ := sealCLIBundle(t, root)
			store := posemodel.Store{Root: root}
			now := time.Now().UTC()
			prepared, err := store.PrepareReviewAttestation(bundle.BundleID, "agent:reviewer", now)
			if err != nil {
				t.Fatal(err)
			}
			att := prepared.Attestation
			att.Authority = &posemodel.ReviewAuthorityClaim{
				SchemaVersion: 1, Project: "proof-fixture", Audience: "proof-fixture",
				BundleDigest: bundle.BundleDigest, Principal: "agent:reviewer", Role: "agent",
				ReviewExecution: "review-run", ImplementationPrincipal: "agent:implementer",
				ImplementationExecution: "implementation-run", Issuer: "fixture",
				IssuedAt: now.Add(-time.Minute).Format(time.RFC3339), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339),
			}
			if state == "invalid-criteria" {
				att.Criteria = nil
			}
			att.AttestationID = ""
			completed, canonical, err := store.SignableReviewAttestation(att, now)
			if err != nil {
				t.Fatal(err)
			}
			completed.Envelope = &posemodel.ReviewAttestationSignature{
				Issuer: "fixture", Subject: bundle.BundleID, Algorithm: "ed25519",
				PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, canonical)),
			}
			switch state {
			case "damaged-signature":
				completed.Envelope.Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
			case "missing-proof":
				completed.Envelope = nil
			}
			// Seed stored records directly: these read-path regressions must also
			// exercise corrupted/legacy records that an approving write can refuse.
			raw, err = json.Marshal(completed)
			if err != nil {
				t.Fatal(err)
			}
			writeCloseoutCLIFile(t, root, ".pose/review-attestations/"+completed.AttestationID+".json", string(raw))
			domain, err := store.VerifyReviewBundle("spec:bundle")
			if err != nil {
				t.Fatal(err)
			}
			identity := "not-verified"
			if state == "valid" || state == "invalid-criteria" {
				identity = "verified"
			}
			if domain.Approved != (state == "valid") || domain.Assurance == nil || domain.Assurance.ReviewerIdentity != identity {
				t.Fatalf("state=%s: %+v", state, domain)
			}
			// Match the wire representation: omitted empty slices decode as nil.
			raw, err = json.Marshal(domain.Assurance)
			if err != nil {
				t.Fatal(err)
			}
			var want posemodel.ReviewAssurance
			if err := json.Unmarshal(raw, &want); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{{"review", "verify", "spec:bundle"}, {"review-check", "spec:bundle"}, {"closeout-check", "spec:bundle"}} {
				var out, errOut bytes.Buffer
				inDir(t, root, func() { Main(append(append([]string{}, args...), "--json"), &out, &errOut) })
				var got struct {
					Assurance *posemodel.ReviewAssurance
					Review    struct{ Assurance *posemodel.ReviewAssurance }
				}
				if err := json.Unmarshal(out.Bytes(), &got); err != nil {
					t.Fatalf("%v: %s %s", err, out.String(), errOut.String())
				}
				assurance := got.Assurance
				if assurance == nil {
					assurance = got.Review.Assurance
				}
				if !reflect.DeepEqual(assurance, &want) {
					t.Fatalf("CLI %v disagrees: got=%+v want=%+v", args, assurance, domain.Assurance)
				}
				out.Reset()
				errOut.Reset()
				inDir(t, root, func() { Main(args, &out, &errOut) })
				if !strings.Contains(out.String(), posemodel.RenderReviewAssurance(*domain.Assurance)) {
					t.Fatalf("human CLI %v disagrees: %s %s", args, out.String(), errOut.String())
				}
			}
			t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.proof-test")
			ts := httptest.NewServer(mcpserver.New(store).Handler("", ""))
			defer ts.Close()
			for _, tool := range []string{"pose_review_bundle", "pose_closeout_state"} {
				request, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": tool, "arguments": map[string]string{"scope": "spec:bundle"}}})
				if err != nil {
					t.Fatal(err)
				}
				response, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader(request))
				if err != nil {
					t.Fatal(err)
				}
				var rpc struct {
					Result struct {
						IsError           bool `json:"isError"`
						StructuredContent struct {
							Assurance *posemodel.ReviewAssurance
							Review    struct{ Assurance *posemodel.ReviewAssurance }
						} `json:"structuredContent"`
					}
					Error any
				}
				err = json.NewDecoder(response.Body).Decode(&rpc)
				response.Body.Close()
				if err != nil || rpc.Error != nil || rpc.Result.IsError {
					t.Fatalf("MCP %s: err=%v rpc=%+v", tool, err, rpc)
				}
				assurance := rpc.Result.StructuredContent.Assurance
				if assurance == nil {
					assurance = rpc.Result.StructuredContent.Review.Assurance
				}
				if !reflect.DeepEqual(assurance, &want) {
					t.Fatalf("MCP %s disagrees: got=%+v want=%+v", tool, assurance, domain.Assurance)
				}
			}
		})
	}
}
