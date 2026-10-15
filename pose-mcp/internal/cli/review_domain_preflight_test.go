package cli

import (
	"bytes"
	"crypto/ed25519"
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

// Spec pose-review-approval-domain-preflight: the CLI, a signed import and MCP
// consume the domain writer's refusal; none keeps an approval policy of its own.

func reviewArtifactFiles(t *testing.T, root string) []string {
	t.Helper()
	files := []string{}
	for _, dir := range []string{"review-attestations", "review-bundles", "review-attribution-supplements", "reviews"} {
		matches, _ := filepath.Glob(filepath.Join(root, ".pose", dir, "*"))
		files = append(files, matches...)
	}
	return files
}

func preflightIssuerFixture(t *testing.T) (string, ed25519.PrivateKey, ed25519.PublicKey) {
	t.Helper()
	root := reviewBundleCLIFixture(t)
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 13)
	}
	private := ed25519.NewKeyFromSeed(seed)
	public := private.Public().(ed25519.PublicKey)
	policyPath := filepath.Join(root, ".pose/policy/review.json")
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	keyHash := sha256.Sum256(public)
	policy["trusted_attestation_issuers"] = []string{fmt.Sprintf("fixture#sha256:%x", keyHash)}
	raw, _ = json.Marshal(policy)
	if err := os.WriteFile(policyPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return root, private, public
}

const riskFinding = "f1|critical|wont-fix|leave it unresolved|-"

// V1/R7: one invalid approval refused with the same blockers by the CLI
// preview, the CLI write, the CLI signed import, the Store and MCP, and none
// of them writes a review artifact.
func TestReviewDomainPreflightCLIImportAndMCPAgreeWithTheDomain(t *testing.T) {
	root, private, public := preflightIssuerFixture(t)
	bundle, evidence := sealCLIBundle(t, root)
	store := posemodel.Store{Root: root}
	args := attestArgs(bundle.BundleID, evidence, "agent:reviewer", "--criterion", "correctness|finding|f1", "--finding", riskFinding)

	now := time.Now()
	att := posemodel.ReviewAttestation{BundleID: bundle.BundleID, Reviewer: "agent:reviewer", Decision: "approved", EvidenceRefs: []string{evidence},
		Criteria: []posemodel.ReviewCriterion{{ID: "correctness", Disposition: "finding", Evidence: "f1"}},
		Findings: []posemodel.ReviewFinding{{ID: "f1", Severity: "critical", Disposition: "wont-fix", Action: "leave it unresolved", Evidence: "-"}}}
	for _, tool := range bundle.Payload.Plan.Tools {
		switch tool.ID {
		case "artifact-check":
			att.Tools = append(att.Tools, posemodel.ReviewToolDisposition{ID: tool.ID, Component: tool.Component, Disposition: "passed", Evidence: "check:artifact"})
		case "validate":
			att.Tools = append(att.Tools, posemodel.ReviewToolDisposition{ID: tool.ID, Component: tool.Component, Disposition: "passed", Evidence: "validation:module"})
		default:
			// What the CLI derives for the post-review gates it is not given.
			for _, precondition := range tool.Preconditions {
				if precondition == "review-complete" {
					att.Tools = append(att.Tools, posemodel.ReviewToolDisposition{ID: tool.ID, Component: tool.Component, Disposition: "deferred", Rationale: "post-review gate"})
					break
				}
			}
		}
	}
	domain := store.PreflightReviewAttestation(att, now)
	if !containsText(domain, "unapproved or incomplete accepted risk") {
		t.Fatalf("fixture: the domain does not refuse the approval: %v", domain)
	}
	before := reviewArtifactFiles(t, root)

	blockersOf := func(output string) []string {
		lines := []string{}
		for _, line := range strings.Split(output, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "- ") {
				lines = append(lines, strings.TrimPrefix(strings.TrimSpace(line), "- "))
			}
		}
		return lines
	}
	run := func(args []string) (int, string) {
		var out, errOut bytes.Buffer
		code := cmdReview(root, args, &out, &errOut)
		return code, out.String() + errOut.String()
	}

	// The CLI write: exit 1, the domain's blockers, nothing written.
	code, output := run(args)
	if code != 1 || !strings.Contains(output, "nothing was written") {
		t.Fatalf("CLI write code=%d: %s", code, output)
	}
	if got := blockersOf(output); !reflect.DeepEqual(got, domain) {
		t.Fatalf("CLI blockers diverged from the domain:\n%v\n%v", got, domain)
	}
	// The CLI preview reports the same refusal.
	code, output = run(args[:len(args)-1])
	if got := blockersOf(output); code != 1 || !reflect.DeepEqual(got, domain) {
		t.Fatalf("CLI preview code=%d diverged: %v", code, got)
	}

	// The signed import: a trusted signature is verified, not admitted.
	completed, canonical, err := store.SignableReviewAttestation(att, now)
	if err != nil {
		t.Fatal(err)
	}
	envelope := posemodel.ReviewAttestationEnvelope{SchemaVersion: 1, Issuer: "fixture", Subject: bundle.BundleID, Algorithm: "ed25519",
		PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, canonical)), Attestation: completed}
	raw, _ := json.Marshal(envelope)
	if err := os.WriteFile(filepath.Join(root, "envelope.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, importArgs := range [][]string{{"attest", "--envelope", "envelope.json"}, {"attest", "--envelope", "envelope.json", "--apply"}} {
		code, output = run(importArgs)
		if got := blockersOf(output); code != 1 || !reflect.DeepEqual(got, domain) {
			t.Fatalf("signed import %v code=%d diverged: %s", importArgs, code, output)
		}
	}

	// MCP: the envelope preview names the blockers; apply is a typed refusal.
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.preflight-test")
	ts := httptest.NewServer(mcpserver.New(store).Handler("", ""))
	defer ts.Close()
	call := func(apply bool) map[string]any {
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "pose_review_attest", "arguments": map[string]any{"envelope": envelope, "apply": apply}}})
		response, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var rpc struct {
			Result map[string]any `json:"result"`
		}
		if err := json.NewDecoder(response.Body).Decode(&rpc); err != nil {
			t.Fatal(err)
		}
		return rpc.Result
	}
	stringsOf := func(value any) []string {
		out := []string{}
		items, _ := value.([]any)
		for _, item := range items {
			out = append(out, fmt.Sprint(item))
		}
		return out
	}
	preview := call(false)
	structured, _ := preview["structuredContent"].(map[string]any)
	if preview["isError"] != false || !reflect.DeepEqual(stringsOf(structured["verify_blockers"]), domain) {
		t.Fatalf("MCP preview diverged: %+v", preview)
	}
	applied := call(true)
	structured, _ = applied["structuredContent"].(map[string]any)
	if applied["isError"] != true || structured["error_code"] != "review_approval_refused" || structured["bundle_id"] != bundle.BundleID || !reflect.DeepEqual(stringsOf(structured["blockers"]), domain) {
		t.Fatalf("MCP apply diverged: %+v", applied)
	}

	if after := reviewArtifactFiles(t, root); !reflect.DeepEqual(after, before) {
		t.Fatalf("a refused approval wrote review artifacts:\n%v\n%v", before, after)
	}
}

// V4: a legitimate negative recorded through the CLI does not need its
// criteria to satisfy the approval bar.
func TestReviewDomainPreflightCLIRecordsANegative(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	bundle, evidence := sealCLIBundle(t, root)
	args := attestArgs(bundle.BundleID, evidence, "agent:reviewer", "--criterion", "correctness|finding|f1", "--finding", "f1|high|open|fix it|-")
	for i, value := range args {
		if value == "approved" {
			args[i] = "changes-requested"
		}
	}
	var out, errOut bytes.Buffer
	if code := cmdReview(root, args, &out, &errOut); code != 0 {
		t.Fatalf("a negative was refused: code=%d %s %s", code, out.String(), errOut.String())
	}
	if att := onlyAttestation(t, root); att.Decision != "changes-requested" {
		t.Fatalf("recorded %q", att.Decision)
	}
}

// V6: the approval of a bundle that a newer seal superseded is refused by the
// CLI write, and the existing artifacts stay.
func TestReviewDomainPreflightCLIRefusesASupersededBundle(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	old, evidence := sealCLIBundle(t, root)
	writeCloseoutCLIFile(t, root, ".pose/indexes/validation-matrix.json", `{"defaults":{"mode":"strict"},"moduleOverrides":{"pose-mcp":{"checks":[{"name":"another-check","program":"true","severity":"required","evidenceClass":"unit"}]}}}`)
	current, _ := sealCLIBundle(t, root)
	if current.BundleID == old.BundleID {
		t.Fatal("fixture: the inputs did not change the bundle")
	}
	before := reviewArtifactFiles(t, root)
	var out, errOut bytes.Buffer
	code := cmdReview(root, attestArgs(old.BundleID, evidence, "agent:reviewer"), &out, &errOut)
	if code != 1 || !strings.Contains(out.String()+errOut.String(), "superseded") {
		t.Fatalf("a superseded bundle was approved: code=%d %s %s", code, out.String(), errOut.String())
	}
	if after := reviewArtifactFiles(t, root); !reflect.DeepEqual(after, before) {
		t.Fatalf("the refusal changed review artifacts:\n%v\n%v", before, after)
	}
}

func containsText(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
