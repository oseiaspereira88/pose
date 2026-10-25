package pose

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type runProvenanceFixture struct {
	store  Store
	key    IssuerKey
	run    ReviewRun
	proof  ReviewRunProof
	bundle ReviewBundle
}

func provenanceRunFixture(t *testing.T, vendor, model string, changes ...func(*ReviewerAdapter)) runProvenanceFixture {
	t.Helper()
	useIssuerHome(t)
	key, e := CreateIssuerKey("", "executor-fixture", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	root, s := reviewBundleFixture(t)
	writeReviewFixture(t, root, ".pose/project.json", `{"schema_version":1,"project_id":"fixture-project","name":"fixture"}`)
	raw, e := os.ReadFile(filepath.Join(root, ".pose/policy/review.json"))
	if e != nil {
		t.Fatal(e)
	}
	var policy map[string]any
	if e = json.Unmarshal(raw, &policy); e != nil {
		t.Fatal(e)
	}
	policy["reviewer_independence"] = map[string]string{"spec": "different-actor"}
	policy["identity_assurance"] = map[string]string{"spec": "verified"}
	policy["delegated_review"] = true
	policy["authority_project"] = "fixture-project"
	policy["authority_audience"] = "fixture-audience"
	policy["trusted_attestation_issuers"] = []string{key.Pin()}
	raw, _ = json.Marshal(policy)
	writeReviewFixture(t, root, ".pose/policy/review.json", string(raw))
	reviewers := ReviewersPolicy{SchemaVersion: 1, Provenance: &ReviewRunProvenancePolicy{Trust: "executor-attested", Implementation: ModelIdentity{Principal: "agent:implementer", Execution: "implementation-fixture", Vendor: "openai", Model: "implementation-model"}}, Adapters: map[string]ReviewerAdapter{"fake": {Executor: "agent:reviewer", Command: []string{"sh", "-c", "cat >/dev/null; printf 'decision: approved\\n'"}, Vendor: vendor, Model: model}}}
	adapter := reviewers.Adapters["fake"]
	for _, change := range changes {
		change(&adapter)
	}
	reviewers.Adapters["fake"] = adapter
	raw, _ = json.Marshal(reviewers)
	writeReviewFixture(t, root, ".pose/policy/reviewers.json", string(raw))
	writeReviewFixture(t, root, ".gitignore", ".pose/indexes/delivery-integrity.json\n")
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false"}, args...)...)
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("git %v: %v %s", args, e, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-q", "-m", "fixture")
	head := git("rev-parse", "HEAD")
	graphPath := filepath.Join(root, ".pose/indexes/delivery-integrity.json")
	raw, e = os.ReadFile(graphPath)
	if e != nil {
		t.Fatal(e)
	}
	var graph DeliveryIntegrityGraph
	json.Unmarshal(raw, &graph)
	for i := range graph.ChangeSets {
		graph.ChangeSets[i].Base = head
		graph.ChangeSets[i].Head = head
		graph.ChangeSets[i].ResolvedBase = head
		graph.ChangeSets[i].ResolvedHead = head
		graph.ChangeSets[i].Selector = "range:" + head + ".." + head
	}
	for i := range graph.ValidationResults {
		graph.ValidationResults[i].GitHead = head
	}
	raw, _ = json.Marshal(graph)
	if e = os.WriteFile(graphPath, raw, 0644); e != nil {
		t.Fatal(e)
	}
	bundle, e := s.SealReviewBundle("spec:backend", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	plan, e := s.PlanReviewDispatch(bundle.BundleID, "review", "fixture labelled notes", "fake")
	if e != nil {
		t.Fatal(e)
	}
	run, e := s.RunReviewDispatch(plan, time.Now)
	if e != nil {
		t.Fatalf("dispatch: %+v %v", run, e)
	}
	return runProvenanceFixture{store: s, key: key, run: run, bundle: bundle}
}
func provenanceFixture(t *testing.T, vendor, model string, changes ...func(*ReviewerAdapter)) runProvenanceFixture {
	t.Helper()
	f := provenanceRunFixture(t, vendor, model, changes...)
	proof, e := f.store.SignReviewRunProof(f.run.RunID, f.key, time.Hour, time.Now(), true)
	if e != nil {
		t.Fatal(e)
	}
	f.proof = proof
	return f
}

func (f runProvenanceFixture) attestation(t *testing.T, decision ...string) ReviewAttestation {
	t.Helper()
	att := approvedBundleAttestation(f.bundle, "agent:reviewer")
	if len(decision) > 0 {
		att.Decision = decision[0]
	}
	att.Run = &ReviewRunReference{RunID: f.run.RunID, ProofID: f.proof.ProofID}
	envelope, e := f.store.SignReviewAttestation(att, f.key, nil, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	verified, e := f.store.VerifyReviewAttestationEnvelope(envelope)
	if e != nil {
		t.Fatal(e)
	}
	return verified
}
func TestRunProvenanceSignedBindingAndDisclosure(t *testing.T) {
	f := provenanceFixture(t, "openai", "review-model")
	proof, e := f.store.VerifyReviewRunProof(f.run.RunID, f.proof.ProofID)
	if e != nil || !proof.ModelDifference || proof.VendorDifference || !proof.CompleteReview {
		t.Fatal(proof, e)
	}
	att := f.attestation(t)
	if blockers := f.store.runBackedReviewBlockers(f.bundle, att); len(blockers) > 0 {
		t.Fatal(blockers)
	}
	a := f.store.DescribeReviewAssurance(f.bundle, &att)
	if a.ReviewerIdentity != ReviewRunMachineAttested || a.Run == nil || a.Run.IdentitySource != "native-issuer-signed-executor-declaration" || a.ReviewerRole != "agent" || a.SeparationVerified != "vendor-or-model" {
		t.Fatalf("disclosure %+v", a)
	}
	if blockers := f.store.PreflightReviewAttestation(att, time.Now()); len(blockers) > 0 {
		t.Fatal(blockers)
	}
	if _, e = f.store.RecordReviewAttestationEnvelope(ReviewAttestationEnvelope{SchemaVersion: 1, Issuer: att.Envelope.Issuer, Subject: att.Envelope.Subject, Algorithm: att.Envelope.Algorithm, PublicKey: att.Envelope.PublicKey, Signature: att.Envelope.Signature, Attestation: func() ReviewAttestation { x := att; x.Path = ""; x.Envelope = nil; return x }()}, true); e != nil {
		t.Fatal(e)
	}
	verification, e := f.store.VerifyReviewBundle("spec:backend")
	if e != nil || !verification.Approved || verification.Assurance.Run == nil {
		t.Fatal(verification.Blockers, e)
	}
	if !strings.Contains(RenderReviewAssurance(a), "not-observable") || !strings.Contains(RenderReviewAssurance(a), "machine-attested") {
		t.Fatal(RenderReviewAssurance(a))
	}
}
func TestRunProvenanceSameModelAndUnknownBaselineRefused(t *testing.T) {
	for _, pair := range [][2]string{{"openai", "implementation-model"}, {" OpenAI ", " IMPLEMENTATION-MODEL "}} {
		t.Run(pair[0]+pair[1], func(t *testing.T) {
			f := provenanceFixture(t, pair[0], pair[1])
			att := f.attestation(t)
			if blockers := f.store.runBackedReviewBlockers(f.bundle, att); len(blockers) == 0 {
				t.Fatal("same normalized model/vendor proved different actor")
			}
			if a := f.store.DescribeReviewAssurance(f.bundle, &att); a.ReviewerIdentity == ReviewRunMachineAttested {
				t.Fatal("same model disclosed as independent")
			}
		})
	}
	if _, e := normalizeModelIdentity(ModelIdentity{Principal: "agent:i", Vendor: "unknown", Model: "m"}); e == nil {
		t.Fatal("unknown vendor accepted")
	}
	if _, e := normalizeModelIdentity(ModelIdentity{Principal: "agent:i", Vendor: "openai", Model: "   "}); e == nil {
		t.Fatal("empty model accepted")
	}
}
func TestRunProvenanceTamperingRevocationAndExpiry(t *testing.T) {
	t.Run("proof", func(t *testing.T) {
		f := provenanceFixture(t, "different-vendor", "implementation-model")
		att := f.attestation(t)
		rel, _ := runProofPath(f.run.RunID, f.proof.ProofID)
		raw, e := os.ReadFile(filepath.Join(f.store.Root, rel))
		if e != nil {
			t.Fatal(e)
		}
		raw = []byte(strings.Replace(string(raw), f.proof.Claim.OutputDigest, "sha256:"+strings.Repeat("0", 64), 1))
		os.WriteFile(filepath.Join(f.store.Root, rel), raw, 0644)
		if _, e = f.store.VerifyReviewRunProof(f.run.RunID, f.proof.ProofID); e == nil {
			t.Fatal("rewritten proof accepted")
		}
		if a := f.store.DescribeReviewAssurance(f.bundle, &att); a.ReviewerIdentity == ReviewRunMachineAttested || a.Run != nil {
			t.Fatal("tampered proof disclosed machine assurance")
		}
	})
	t.Run("expiry", func(t *testing.T) {
		f := provenanceFixture(t, "different-vendor", "implementation-model")
		if _, e := f.store.atReviewTime(time.Now().Add(2*time.Hour)).VerifyReviewRunProof(f.run.RunID, f.proof.ProofID); e == nil {
			t.Fatal("expired proof accepted")
		}
	})
	t.Run("run", func(t *testing.T) {
		f := provenanceFixture(t, "different-vendor", "implementation-model")
		path := filepath.Join(f.store.Root, f.run.Path)
		raw, _ := os.ReadFile(path)
		raw = []byte(strings.Replace(string(raw), "decision: approved", "decision: rejected", 1))
		os.WriteFile(path, raw, 0644)
		if _, e := f.store.VerifyReviewRunProof(f.run.RunID, f.proof.ProofID); e == nil {
			t.Fatal("rewritten output accepted")
		}
	})
	t.Run("transcript", func(t *testing.T) {
		f := provenanceFixture(t, "different-vendor", "implementation-model")
		os.WriteFile(filepath.Join(f.store.Root, f.run.TranscriptPath), []byte("changed"), 0644)
		if _, e := f.store.VerifyReviewRunProof(f.run.RunID, f.proof.ProofID); e == nil {
			t.Fatal("rewritten transcript accepted")
		}
	})
	t.Run("issuer", func(t *testing.T) {
		f := provenanceFixture(t, "different-vendor", "implementation-model")
		path := filepath.Join(f.store.Root, ".pose/policy/review.json")
		raw, _ := os.ReadFile(path)
		raw = []byte(strings.Replace(string(raw), f.key.Pin(), "revoked#sha256:"+strings.Repeat("0", 64), 1))
		os.WriteFile(path, raw, 0644)
		if _, e := f.store.VerifyReviewRunProof(f.run.RunID, f.proof.ProofID); e == nil {
			t.Fatal("revoked issuer accepted")
		}
	})
}
func TestRunProvenancePrefixAndHumanNeverUpgrade(t *testing.T) {
	f := provenanceFixture(t, "different-vendor", "implementation-model")
	att := approvedBundleAttestation(f.bundle, "agent:independent-prefix-only")
	if blockers := f.store.validateBundleAttestation(f.bundle, att); len(blockers) == 0 {
		t.Fatal("prefix-only accepted after opt-in")
	}
	att = f.attestation(t)
	bundle := f.bundle
	bundle.Payload.Plan.Independence = "mandatory-human"
	if blockers := f.store.runBackedReviewBlockers(bundle, att); len(blockers) == 0 {
		t.Fatal("machine proof satisfied human requirement")
	}
	if a := f.store.DescribeReviewAssurance(bundle, &att); a.ReviewerIdentity == ReviewRunMachineAttested || a.ReviewerRole == "human" {
		t.Fatal("machine impersonated human")
	}
}

func TestRunProvenanceRejectingOrAmbiguousDraftCannotApprove(t *testing.T) {
	for _, output := range []string{"decision: rejected", "decision: approved\ndecision: rejected", "no conclusion"} {
		t.Run(output, func(t *testing.T) {
			f := provenanceFixture(t, "different-vendor", "implementation-model", func(a *ReviewerAdapter) {
				a.Command = []string{"sh", "-c", `cat >/dev/null; printf '%s\n' "$1"`, "sh", output}
			})
			att := f.attestation(t)
			if output != "decision: rejected" {
				audit, err := f.store.VerifyReviewRunProof(f.run.RunID, f.proof.ProofID)
				if err != nil || audit.CompleteReview {
					t.Fatal("unknown draft claimed complete review", audit, err)
				}
			}
			if blockers := f.store.PreflightReviewAttestation(att, time.Now()); len(blockers) == 0 {
				t.Fatal("contradictory run-backed approval accepted")
			}
			if a := f.store.DescribeReviewAssurance(f.bundle, &att); a.ReviewerIdentity == ReviewRunMachineAttested {
				t.Fatal("contradictory approval disclosed as run-backed")
			}
			if output == "decision: rejected" {
				negative := f.attestation(t, "rejected")
				if blockers := f.store.runBackedReviewBlockers(f.bundle, negative); len(blockers) > 0 {
					t.Fatal(blockers)
				}
			}
		})
	}
}
func TestRunProvenanceMissingAttemptReceiptInvalidatesAuthority(t *testing.T) {
	f := provenanceFixture(t, "different-vendor", "implementation-model")
	att := f.attestation(t)
	attempts, e := f.store.readDelegatedReviewAttempts(f.run.BundleID)
	if e != nil || len(attempts) != 1 {
		t.Fatal(attempts, e)
	}
	if e = os.Remove(attemptReceiptPath(f.store.Root, attempts[0])); e != nil {
		t.Fatal(e)
	}
	if _, e = f.store.VerifyReviewRunProof(f.run.RunID, f.proof.ProofID); e == nil {
		t.Fatal("missing initial receipt retained machine proof")
	}
	if a := f.store.DescribeReviewAssurance(f.bundle, &att); a.ReviewerIdentity == ReviewRunMachineAttested {
		t.Fatal("missing receipt retained disclosure")
	}
}
func TestRunProvenanceTruncatedOutputAndPartialAudit(t *testing.T) {
	f := provenanceRunFixture(t, "different-vendor", "implementation-model", func(a *ReviewerAdapter) {
		a.Command = []string{"sh", "-c", `cat >/dev/null; printf 'decision: approved\n'; head -c 24000 /dev/zero | tr '\000' x`}
	})
	if f.run.Status != ReviewRunCompleted || f.run.Provenance.OutputComplete {
		t.Fatal("fixture did not truncate", f.run.Status)
	}
	if _, e := f.store.SignReviewRunProof(f.run.RunID, f.key, time.Hour, time.Now(), true); e == nil {
		t.Fatal("truncated completed draft established a proof")
	}
	f = provenanceFixture(t, "different-vendor", "implementation-model", func(a *ReviewerAdapter) {
		a.Command = []string{"sh", "-c", `cat >/dev/null; printf 'session: private-fixture-ref\ndecision: rejected\n'; exit 75`}
		a.Limit = &ReviewerLimit{ExitCodes: []int{75}}
		a.Session = &ReviewerSession{Pattern: `session: (\S+)`}
		a.Resume = &ReviewerResume{Command: []string{"sh", "-c", `cat >/dev/null; printf 'decision: approved\n'`, "sh", "{session}"}}
	})
	audit, e := f.store.VerifyReviewRunProof(f.run.RunID, f.proof.ProofID)
	if e != nil || audit.State != ReviewRunInterrupted || audit.CompleteReview {
		t.Fatal("partial audit became review", audit, e)
	}
	if strings.Contains(f.run.Output, "private-fixture-ref") {
		t.Fatal("session leaked to public run")
	}
	if blockers := f.store.runBackedReviewBlockers(f.bundle, f.attestation(t)); len(blockers) == 0 {
		t.Fatal("partial audit authorized review")
	}
}

func TestRunProvenanceAttestationCannotPredateExecutionOrProof(t *testing.T) {
	f := provenanceFixture(t, "different-vendor", "implementation-model")
	att := approvedBundleAttestation(f.bundle, "agent:reviewer")
	att.Run = &ReviewRunReference{RunID: f.run.RunID, ProofID: f.proof.ProofID}
	envelope, e := f.store.SignReviewAttestation(att, f.key, nil, time.Now().Add(-time.Hour))
	if e != nil {
		return
	}
	verified, e := f.store.VerifyReviewAttestationEnvelope(envelope)
	if e != nil {
		t.Fatal(e)
	}
	blockers := f.store.PreflightReviewAttestation(verified, time.Now())
	if len(blockers) == 0 || !strings.Contains(strings.Join(blockers, ";"), "predates") {
		t.Fatal("backdated run association admitted", blockers)
	}
	if a := f.store.DescribeReviewAssurance(f.bundle, &verified); a.ReviewerIdentity == ReviewRunMachineAttested {
		t.Fatal("backdated run association disclosed machine assurance")
	}
}

func TestRunProvenancePartialStdinAndUnstartedAudit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "sh", "-c", "head -c 16 >/dev/null")
	command.WaitDelay = time.Second
	wire := strings.Repeat("x", 4<<20)
	digest, n, complete, started, e := runProvenanceCommand(command, wire)
	if e == nil || !started || complete || n < 0 || n >= int64(len(wire)) || digest != digestBytes([]byte(wire)[:int(n)]) {
		t.Fatal("partial stdin claimed complete delivery", n, complete, started, e)
	}
	f := provenanceRunFixture(t, "different-vendor", "implementation-model", func(a *ReviewerAdapter) { a.Command = []string{"/nonexistent/provenance-fixture-command"} })
	if f.run.Status != ReviewRunFailed || f.run.Provenance.ExecutionState != "not-started" || f.run.Provenance.WireComplete || f.run.Provenance.WireDigest != "" {
		t.Fatal("unstarted run claimed transmitted brief", f.run)
	}
	proof, e := f.store.SignReviewRunProof(f.run.RunID, f.key, time.Hour, time.Now(), true)
	if e != nil {
		t.Fatal(e)
	}
	audit, e := f.store.VerifyReviewRunProof(f.run.RunID, proof.ProofID)
	if e != nil || audit.CompleteReview || audit.State != ReviewRunFailed {
		t.Fatal("unstarted audit became review", audit, e)
	}
}

func TestRunProvenanceProjectConfigAndImmutablePublication(t *testing.T) {
	t.Run("project", func(t *testing.T) {
		f := provenanceFixture(t, "different-vendor", "implementation-model")
		writeReviewFixture(t, f.store.Root, ".pose/project.json", `{"schema_version":1,"project_id":"other-project"}`)
		if _, e := f.store.VerifyReviewRunProof(f.run.RunID, f.proof.ProofID); e == nil {
			t.Fatal("foreign declared project retained proof")
		}
	})
	t.Run("config", func(t *testing.T) {
		f := provenanceFixture(t, "different-vendor", "implementation-model")
		path := filepath.Join(f.store.Root, ".pose/policy/reviewers.json")
		raw, _ := os.ReadFile(path)
		raw = []byte(strings.Replace(string(raw), "implementation-model", "changed-model", 1))
		os.WriteFile(path, raw, 0644)
		if _, e := f.store.VerifyReviewRunProof(f.run.RunID, f.proof.ProofID); e == nil {
			t.Fatal("changed policy retained proof")
		}
	})
	t.Run("publication", func(t *testing.T) {
		f := provenanceFixture(t, "different-vendor", "implementation-model")
		rel, _ := runProofPath(f.run.RunID, f.proof.ProofID)
		if e := publishLedgerJSON(f.store.Root, rel, f.proof); e != nil {
			t.Fatal(e)
		}
		altered := f.proof
		altered.Claim.OutputDigest = "changed"
		if e := publishLedgerJSON(f.store.Root, rel, altered); e == nil {
			t.Fatal("immutable proof overwritten")
		}
	})
}

func TestRunProvenancePreservesArgvZeroAndResolvesWorktreeCommand(t *testing.T) {
	f := provenanceFixture(t, "different-vendor", "implementation-model", func(a *ReviewerAdapter) {
		a.Command = []string{"sh", "-c", `test "$0" = sh || exit 2; cat >/dev/null; printf 'decision: approved\n'`}
	})
	if f.run.Status != ReviewRunCompleted {
		t.Fatal("logical argv0 changed", f.run.Failure)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "reviewer")
	if e := os.WriteFile(file, []byte("#!/bin/sh\nexit 0\n"), 0755); e != nil {
		t.Fatal(e)
	}
	resolved, e := resolveProvenanceExecutable("./reviewer", dir)
	if e != nil || resolved != file {
		t.Fatal("relative executable not resolved in worktree", resolved, e)
	}
}

func TestRunProvenanceLiveOptInDoesNotGrandfatherPrefix(t *testing.T) {
	root, s := reviewBundleFixture(t)
	useIssuerHome(t)
	key, e := CreateIssuerKey("", "fixture-trust", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(root, ".pose/policy/review.json")
	raw, _ := os.ReadFile(path)
	var policy map[string]any
	json.Unmarshal(raw, &policy)
	policy["reviewer_independence"] = map[string]string{"spec": "different-actor"}
	policy["identity_assurance"] = map[string]string{"spec": "declared"}
	policy["authority_project"] = "fixture-project"
	policy["authority_audience"] = "fixture-audience"
	policy["trusted_attestation_issuers"] = []string{key.Pin()}
	raw, _ = json.Marshal(policy)
	os.WriteFile(path, raw, 0644)
	bundle, e := s.SealReviewBundle("spec:backend", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	att, _, e := s.SignableReviewAttestation(approvedBundleAttestation(bundle, "agent:independent-legacy-prefix"), time.Now())
	if e != nil {
		t.Fatal(e)
	}
	if blockers := s.validateBundleAttestation(bundle, att); len(blockers) > 0 {
		t.Fatal("non-adopting declared compatibility changed", blockers)
	}
	policy["delegated_review"] = true
	raw, _ = json.Marshal(policy)
	os.WriteFile(path, raw, 0644)
	blockers := s.validateBundleAttestation(bundle, att)
	if len(blockers) == 0 || !strings.Contains(strings.Join(blockers, ";"), "signed run reference") {
		t.Fatal("old declared gate exempted prefix after live adoption", blockers)
	}
}
