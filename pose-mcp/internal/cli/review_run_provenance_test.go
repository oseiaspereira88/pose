package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// cliProvenanceInstance is an instance whose fake reviewer returns the file
// at draftPath, with spec:alpha sealed by a declared implementation and the
// issuer fixture-executor pinned.
func cliProvenanceInstance(t *testing.T, policies ...func(root, draftPath string)) (root, draftPath, bundleID string) {
	t.Helper()
	t.Setenv("POSE_ISSUER_HOME", filepath.Join(t.TempDir(), "issuers"))
	key, e := posemodel.CreateIssuerKey("", "fixture-executor", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	draftPath = filepath.Join(t.TempDir(), "draft.txt")
	root = dispatchInstanceWith(t, func(root string, git func(...string) string) {
		policy := map[string]any{"schema_version": 2, "enabled": true, "adopted_at": "2026-08-02", "profiles": map[string]string{"spec": "spec-closeout@1"}, "review_bundles": true, "review_bundles_adopted_at": "2026-08-14", "delegated_review": true, "component_aware": true, "component_aware_adopted_at": "2026-08-13", "unmapped_component_behavior": "warning", "reviewer_independence": map[string]string{"spec": "different-actor"}, "identity_assurance": map[string]string{"spec": "verified"}, "authority_project": "proj.fixture", "authority_audience": "fixture-audience", "trusted_attestation_issuers": []string{key.Pin()}}
		raw, _ := json.Marshal(policy)
		writeCloseoutCLIFile(t, root, ".pose/policy/review.json", string(raw))
		reviewers := posemodel.ReviewersPolicy{SchemaVersion: 1, Provenance: &posemodel.ReviewRunProvenancePolicy{Trust: "executor-attested", Implementation: posemodel.ModelIdentity{Principal: "agent:implementer", Execution: "implementation-fixture", Vendor: "openai", Model: "implementation-model"}}, Adapters: map[string]posemodel.ReviewerAdapter{"fake": {Executor: "agent:reviewer", Command: []string{"sh", "-c", "cat >/dev/null; cat '" + draftPath + "'"}, Vendor: "openai", Model: "review-model"}}}
		raw, _ = json.Marshal(reviewers)
		writeCloseoutCLIFile(t, root, ".pose/policy/reviewers.json", string(raw))
		for _, write := range policies {
			write(root, draftPath)
		}
		git("add", "-A")
	})
	// The fixture sealed in a session with no agent markers, so its bundle
	// records an unknown implementation; a declared one seals anew, and the
	// run is compared with it, not with the policy (spec pose-delegation-routing
	// R3, R4). Seal instants have a resolution of a second; the reseal waits
	// for the next one so the declared bundle is the newer.
	time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(time.Second)))
	t.Setenv("CODEX_THREAD_ID", "implementation-fixture")
	if code, out := runPose(t, root, "review", "bundle", "spec:alpha", "--seal", "--lead", "openai/implementation-model"); code != 0 || !strings.Contains(out, "review_bundle.implementation=openai/implementation-model (declared)") {
		t.Fatal(code, out)
	}
	current, e := (posemodel.Store{Root: root}).CurrentReviewBundle("spec:alpha")
	if e != nil || current == nil {
		t.Fatal(current, e)
	}
	return root, draftPath, current.BundleID
}

func cliRunProvenanceFixture(t *testing.T) (string, posemodel.Store, posemodel.ReviewRun, posemodel.ReviewRunProof) {
	t.Helper()
	root, draftPath, bundleID := cliProvenanceInstance(t)
	writeRunConclusion(t, posemodel.Store{Root: root}, bundleID, draftPath)
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "fake", "--apply"); code != 0 {
		t.Fatal(code, out)
	}
	s := posemodel.Store{Root: root}
	runs, lost, e := s.ReadReviewRuns("")
	if e != nil || len(lost) > 0 || len(runs) != 1 {
		t.Fatal(runs, lost, e)
	}
	code, out := runPose(t, root, "review", "run-sign", runs[0].RunID, "--issuer", "fixture-executor", "--apply", "--json")
	if code != 0 {
		t.Fatal(code, out)
	}
	var proof posemodel.ReviewRunProof
	if e = json.Unmarshal([]byte(out), &proof); e != nil {
		t.Fatal(e, out)
	}
	return root, s, runs[0], proof
}
func TestRunProvenanceCLIProofAndAttestation(t *testing.T) {
	root, s, run, proof := cliRunProvenanceFixture(t)
	if code, out := runPose(t, root, "review", "run-proof", run.RunID, proof.ProofID, "--json"); code != 0 || !strings.Contains(out, "machine-attested") || !strings.Contains(out, "executor-declaration-not-independent-vendor-receipt") {
		t.Fatal(code, out)
	}
	args := []string{"review", "attest", run.BundleID, "--reviewer", "agent:reviewer", "--decision", "approved", "--criterion", "correctness|passed|integration:val-alpha|fixture reviewer checked the sealed subject and found no defect", "--evidence", "integration:val-alpha", "--run", run.RunID, "--run-proof", proof.ProofID, "--sign", "fixture-executor", "--apply"}
	args = appendRunProvenanceTools(t, s, run.BundleID, "agent:reviewer", args)
	if code, out := runPose(t, root, args...); code != 0 {
		t.Fatal(code, out)
	}
	if code, out := runPose(t, root, "review", "verify", "spec:alpha"); code != 0 || !strings.Contains(out, "machine-attested executor declaration") || !strings.Contains(out, "source native-issuer-signed-executor-declaration") {
		t.Fatal(code, out)
	}
	verification, e := s.VerifyReviewBundle("spec:alpha")
	if e != nil || !verification.Approved || verification.Assurance.Run == nil {
		t.Fatal(verification.Blockers, e)
	}
	if code, out := runPose(t, root, "state", "--attention", "--scope", "spec:alpha", "--json"); code != 0 || !strings.Contains(out, `"review_assurance"`) || !strings.Contains(out, `"reviewer_identity": "machine-attested"`) {
		t.Fatal("Attention omitted current proof", code, out)
	}
	proofPath := filepath.Join(root, ".pose/review-run-proofs", run.RunID, proof.ProofID+".json")
	raw, e := os.ReadFile(proofPath)
	if e != nil {
		t.Fatal(e)
	}
	raw = []byte(strings.Replace(string(raw), proof.Claim.OutputDigest, "sha256:"+strings.Repeat("0", 64), 1))
	if e = os.WriteFile(proofPath, raw, 0644); e != nil {
		t.Fatal(e)
	}
	if code, out := runPose(t, root, "review", "verify", "spec:alpha"); code == 0 || strings.Contains(out, "machine-attested executor declaration") {
		t.Fatal("tampered CLI proof still verified", code, out)
	}
}
func TestRunProvenanceCLIPrefixCannotReplaceProof(t *testing.T) {
	root, s, run, _ := cliRunProvenanceFixture(t)
	args := []string{"review", "attest", run.BundleID, "--reviewer", "agent:independent-prefix-only", "--decision", "approved", "--criterion", "correctness|passed|integration:val-alpha|fixture reviewer checked the sealed subject and found no defect", "--evidence", "integration:val-alpha", "--apply"}
	args = appendRunProvenanceTools(t, s, run.BundleID, "agent:independent-prefix-only", args)
	if code, out := runPose(t, root, args...); code == 0 || !strings.Contains(out, "signed run reference") {
		t.Fatal("prefix bypass", code, out)
	}
	if entries, _ := os.ReadDir(filepath.Join(root, ".pose/review-attestations")); len(entries) > 0 {
		t.Fatal("refused approval wrote an attestation")
	}
	if code, _ := runPose(t, root, "review", "attest", run.BundleID, "--reviewer", "agent:reviewer", "--run", run.RunID, "--apply"); code != 2 {
		t.Fatal("partial run reference not usage error", code)
	}
}

func appendRunProvenanceTools(t *testing.T, s posemodel.Store, bundle, reviewer string, args []string) []string {
	t.Helper()
	prepared, e := s.PrepareReviewAttestation(bundle, reviewer, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	for _, tool := range prepared.Attestation.Tools {
		component := tool.Component
		if component == "" {
			component = "-"
		}
		args = append(args, "--tool", fmt.Sprintf("%s|%s|%s|%s|%s", tool.ID, component, tool.Disposition, tool.Evidence, tool.Rationale))
	}
	return args
}

// writeRunConclusion writes the typed conclusion the fixture's reviewer
// returns: the criterion and the tools the attestation will state (spec
// pose-delegated-review-conclusions D1).
func writeRunConclusion(t *testing.T, s posemodel.Store, bundleID, path string) {
	t.Helper()
	prepared, e := s.PrepareReviewAttestation(bundleID, "agent:reviewer", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	draft := posemodel.ReviewDraft{Schema: posemodel.ReviewDraftSchema, Decision: "approved", Findings: []posemodel.ReviewDraftFinding{},
		Criteria: []posemodel.ReviewDraftCriterion{{ID: "correctness", Disposition: "passed", Evidence: "integration:val-alpha", Rationale: "fixture reviewer checked the sealed subject and found no defect"}}}
	for _, tool := range prepared.Attestation.Tools {
		draft.Tools = append(draft.Tools, posemodel.ReviewDraftTool{ID: tool.ID, Component: tool.Component, Disposition: tool.Disposition, Evidence: tool.Evidence, Rationale: tool.Rationale})
	}
	raw, _ := json.Marshal(draft)
	if e := os.WriteFile(path, []byte("decision: approved\n```pose-draft\n"+string(raw)+"\n```\n"), 0o600); e != nil {
		t.Fatal(e)
	}
}
