package mcpserver

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pose "github.com/harne8/pose-mcp/internal/pose"
)

func TestRunProvenanceMCPReadsCurrentMachineProof(t *testing.T) {
	t.Setenv("POSE_ISSUER_HOME", filepath.Join(t.TempDir(), "issuers"))
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.fixture")
	key, e := pose.CreateIssuerKey("", "fixture-executor", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	root := t.TempDir()
	write := func(path, data string) {
		t.Helper()
		full := filepath.Join(root, path)
		if e := os.MkdirAll(filepath.Dir(full), 0755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(full, []byte(data), 0644); e != nil {
			t.Fatal(e)
		}
	}
	write(".pose/project.json", `{"schema_version":1,"project_id":"proj.fixture","name":"fixture"}`)
	policy := map[string]any{"schema_version": 2, "enabled": true, "adopted_at": "2026-08-02", "profiles": map[string]string{"spec": "spec-closeout@1"}, "review_bundles": true, "review_bundles_adopted_at": "2026-08-14", "delegated_review": true, "reviewer_independence": map[string]string{"spec": "different-actor"}, "identity_assurance": map[string]string{"spec": "verified"}, "authority_project": "proj.fixture", "authority_audience": "fixture-audience", "trusted_attestation_issuers": []string{key.Pin()}}
	raw, _ := json.Marshal(policy)
	write(".pose/policy/review.json", string(raw))
	reviewers := pose.ReviewersPolicy{SchemaVersion: 1, Provenance: &pose.ReviewRunProvenancePolicy{Trust: "executor-attested", Implementation: pose.ModelIdentity{Principal: "agent:implementer", Execution: "implementation-fixture", Vendor: "openai", Model: "implementation-model"}}, Adapters: map[string]pose.ReviewerAdapter{"fake": {Executor: "agent:reviewer", Command: []string{"sh", "-c", "cat >/dev/null; printf 'decision: approved\\n'"}, Vendor: "different-vendor", Model: "implementation-model"}}}
	raw, _ = json.Marshal(reviewers)
	write(".pose/policy/reviewers.json", string(raw))
	write(".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
	write(".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\n---\n# Spec: alpha\n\n## 2. Requirements\n- R1: works\n")
	write("pose-mcp/lib.go", "package fixture\n")
	git := func(args ...string) string {
		t.Helper()
		command := exec.Command("git", append([]string{"-C", root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false"}, args...)...)
		out, e := command.CombinedOutput()
		if e != nil {
			t.Fatalf("git %v: %v %s", args, e, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-q", "-m", "fixture")
	head := git("rev-parse", "HEAD")
	graph := pose.DeliveryIntegrityGraph{SchemaVersion: 1, ProvenanceDigest: "sha256:" + strings.Repeat("1", 64), ChangeSets: []pose.ChangeSet{{ID: "cs-alpha", Spec: "alpha", Selector: "range:" + head + ".." + head, Base: head, Head: head, ResolvedBase: head, ResolvedHead: head, Paths: []pose.ObservedPath{{Action: "modified", Path: "pose-mcp/lib.go"}}, DiffDigest: "sha256:" + strings.Repeat("2", 64)}}, Deliveries: []pose.DeliveryTarget{{Spec: "alpha", Ref: "contract:alpha-api", Kind: "contract", ID: "alpha-api", Module: "pose-mcp", Profile: "api-contract", Entrypoint: "pose-mcp/lib.go"}}, ValidationResults: []pose.DeliveryValidationResult{{ID: "val-alpha", Module: "pose-mcp", Check: "go-test", EvidenceClass: "integration", Severity: "required", Outcome: "pass", GitHead: head, ProvenanceDigest: "sha256:" + strings.Repeat("1", 64)}}, Reverse: map[string][]string{"pose-mcp/lib.go": {"alpha"}}}
	raw, _ = json.Marshal(graph)
	write(".pose/indexes/delivery-integrity.json", string(raw))
	s := pose.Store{Root: root}
	// The implementer seals from its own session, so the bundle records it
	// and the run is compared with it (spec pose-delegation-routing R1–R3).
	t.Setenv("CODEX_THREAD_ID", "implementation-fixture")
	bundle, e := s.SealReviewBundleLead("spec:alpha", "openai/implementation-model", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	plan, e := s.PlanReviewDispatch(bundle.BundleID, "review", "", "fake")
	if e != nil {
		t.Fatal(e)
	}
	run, e := s.RunReviewDispatch(plan, time.Now)
	if e != nil || run.Status != pose.ReviewRunCompleted {
		t.Fatal(run.Failure, e)
	}
	proof, e := s.SignReviewRunProof(run.RunID, key, time.Hour, time.Now(), true)
	if e != nil {
		t.Fatal(e)
	}
	prepared, e := s.PrepareReviewAttestation(bundle.BundleID, "agent:reviewer", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	att := prepared.Attestation
	att.Decision = "approved"
	att.Criteria = []pose.ReviewCriterion{{ID: "correctness", Disposition: "passed", Evidence: "integration:val-alpha", Rationale: "fixture reviewer checked the sealed subject and found no defect"}}
	att.Run = &pose.ReviewRunReference{RunID: run.RunID, ProofID: proof.ProofID}

	envelope, e := s.SignReviewAttestation(att, key, nil, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	ts := httptest.NewServer(New(s).Handler("", ""))
	t.Cleanup(ts.Close)
	response := reviewAttestCall(t, ts, map[string]any{"envelope": envelope, "apply": true})
	if response.Error != nil || response.Result["isError"] != false {
		t.Fatal(response)
	}
	read := func() pose.ReviewBundleVerification {
		t.Helper()
		_, response := post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_review_bundle","arguments":{"scope":"spec:alpha"}}}`)
		if response.Error != nil || response.Result["isError"] != false {
			t.Fatal(response)
		}
		raw, _ := json.Marshal(response.Result["structuredContent"])
		var v pose.ReviewBundleVerification
		if e := json.Unmarshal(raw, &v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	v := read()
	if !v.Approved || v.Assurance == nil || v.Assurance.ReviewerIdentity != pose.ReviewRunMachineAttested || v.Assurance.Run == nil || !v.Assurance.Run.VendorDifference {
		t.Fatalf("MCP machine proof %+v", v)
	}
	_, attentionResponse := post(t, ts, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"pose_obligations","arguments":{"scope":"spec:alpha"}}}`)
	attentionRaw, _ := json.Marshal(attentionResponse.Result["structuredContent"])
	var attention struct {
		ReviewAssurance map[string]pose.ReviewAssurance `json:"review_assurance"`
	}
	if e = json.Unmarshal(attentionRaw, &attention); e != nil || attention.ReviewAssurance["spec:alpha"].ReviewerIdentity != pose.ReviewRunMachineAttested {
		t.Fatal("MCP Attention omitted proof", e, string(attentionRaw))
	}
	proofPath := filepath.Join(root, ".pose/review-run-proofs", run.RunID, proof.ProofID+".json")
	if e = os.Remove(proofPath); e != nil {
		t.Fatal(e)
	}
	v = read()
	if v.Approved || v.Assurance.ReviewerIdentity == pose.ReviewRunMachineAttested || v.Assurance.Run != nil {
		t.Fatal("missing proof retained MCP assurance", v.Blockers)
	}
}
