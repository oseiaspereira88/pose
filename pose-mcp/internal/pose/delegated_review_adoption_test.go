package pose

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func adoptionFixture(t *testing.T) (Store, PolicyDocs, DelegationPolicy) {
	t.Helper()
	f := provenanceRunFixture(t, "anthropic", "review-model")
	p, _, _, err := f.store.LoadDelegationPolicy()
	if err != nil {
		t.Fatal(err)
	}
	p.Conclusions = &DelegationConclusions{Issuer: f.key.Name}
	raw, err := CanonicalDelegationPolicy(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(f.store.Root, ".pose/policy/reviewers.json")); err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, f.store.Root, ".pose/policy/delegation.json", string(raw))
	docs, err := LoadPolicyDocs(f.store.Root)
	if err != nil {
		t.Fatal(err)
	}
	docs.Review["delegated_review"] = false
	docs.Review["review_bundles"] = true
	docs.Review["review_bundles_adopted_at"] = "2026-10-01"
	if err := docs.Write(f.store.Root, f.store); err != nil {
		t.Fatal(err)
	}
	docs, err = LoadPolicyDocs(f.store.Root)
	if err != nil {
		t.Fatal(err)
	}
	return f.store, docs, p
}

func TestDelegatedReviewCapabilityPublicationDistinguishesUnsyncedAndConflicting(t *testing.T) {
	t.Run("published but unsynced", func(t *testing.T) {
		s, docs, _ := adoptionFixture(t)
		entry, _ := LookupCatalogEntry("delegated-review")
		entry.Adopt(docs, "2026-10-29")
		prior := PublishSyncHook
		t.Cleanup(func() { PublishSyncHook = prior })
		PublishSyncHook = func(kind, path string) error {
			if kind == "dir" && path == filepath.Join(s.Root, ".pose/policy/review.json") {
				return errors.New("injected sync failure")
			}
			return nil
		}
		err := docs.Write(s.Root, s)
		if !errors.Is(err, ErrPublishUnsynced) || !strings.Contains(err.Error(), "inspect") {
			t.Fatalf("publication state was concealed: %v", err)
		}
		p, err := s.GetReviewPolicy()
		if err != nil || !p.DelegatedReview {
			t.Fatal("published decision was misreported", p, err)
		}
	})
	t.Run("review changed since preview", func(t *testing.T) {
		s, docs, _ := adoptionFixture(t)
		entry, _ := LookupCatalogEntry("delegated-review")
		entry.Adopt(docs, "2026-10-29")
		path := filepath.Join(s.Root, ".pose/policy/review.json")
		newer, _ := os.ReadFile(path)
		newer = append(newer, '\n')
		if err := os.WriteFile(path, newer, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := docs.Write(s.Root, s); !errors.Is(err, ErrPublishConflict) {
			t.Fatal("overwrote a concurrent policy", err)
		}
		got, _ := os.ReadFile(path)
		if !bytes.Equal(got, newer) {
			t.Fatal("concurrent decision was changed")
		}
	})
}

func TestDelegatedReviewCapabilityValidatesTheExactExecutor(t *testing.T) {
	s, docs, p := adoptionFixture(t)
	if err := s.ValidateDelegatedReviewActivation(p, docs); err != nil {
		t.Fatal(err)
	}
	entry, ok := LookupCatalogEntry("delegated-review")
	if !ok {
		t.Fatal("missing catalog entry")
	}
	if missing := CatalogAdoptBlocker(s.Root, docs, entry); missing != "" {
		t.Fatal(missing)
	}
	for _, tc := range []struct {
		name   string
		mutate func(PolicyDocs, *DelegationPolicy)
	}{
		{"no read agent", func(_ PolicyDocs, p *DelegationPolicy) { p.Agents = map[string]DelegationAgent{} }},
		{"no provenance", func(_ PolicyDocs, p *DelegationPolicy) { p.Provenance = nil }},
		{"wrong issuer pin", func(d PolicyDocs, _ *DelegationPolicy) {
			d.Review["trusted_attestation_issuers"] = []string{"other#sha256:" + strings.Repeat("a", 64)}
		}},
		{"other project", func(d PolicyDocs, _ *DelegationPolicy) { d.Review["authority_project"] = "other-project" }},
		{"empty audience", func(d PolicyDocs, _ *DelegationPolicy) { d.Review["authority_audience"] = "" }},
		{"missing key", func(_ PolicyDocs, p *DelegationPolicy) { p.Conclusions = &DelegationConclusions{Issuer: "missing-key"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := LoadPolicyDocs(s.Root)
			raw, _ := json.Marshal(p)
			var copy DelegationPolicy
			if err := json.Unmarshal(raw, &copy); err != nil {
				t.Fatal(err)
			}
			tc.mutate(d, &copy)
			if err := s.ValidateDelegatedReviewActivation(copy, d); err == nil {
				t.Fatal("invalid activation admitted")
			}
		})
	}
}

func TestDelegatedReviewCapabilityKeepsTheLegacyPlanDigest(t *testing.T) {
	plan := ReviewPlan{SchemaVersion: 1, PolicySchemaVersion: 2, BaseProfile: "spec-closeout@1", Independence: "same-actor-separate-execution"}
	got, err := reviewBundlePlanDigest(plan)
	// Independent pre-B21 wire shape: neither optional addition may alter an
	// old plan's digest when no refusal or protected-proof obligation applies.
	legacy := `{"schema_version":1,"policy_schema_version":2,"base_profile":"spec-closeout@1","independence":"same-actor-separate-execution","components":[],"selected_profiles":[],"criteria":[],"tools":[]}`
	if err != nil || got != digestBytes([]byte(legacy)) {
		t.Fatal(got, err)
	}
	plan.DelegatedRunProofRequired = true
	protected, err := reviewBundlePlanDigest(plan)
	if err != nil || protected == got {
		t.Fatal("proof obligation was not bound into the plan digest", err)
	}
}

func TestDelegatedReviewCapabilityAdoptsAndExplicitlyTurnsOff(t *testing.T) {
	s, docs, _ := adoptionFixture(t)
	entry, _ := LookupCatalogEntry("delegated-review")
	if len(entry.Adopt(docs, "2026-10-29")) == 0 {
		t.Fatal("no adoption")
	}
	if err := docs.Write(s.Root, s); err != nil {
		t.Fatal(err)
	}
	on, _ := s.GetReviewPolicy()
	if !on.DelegatedReview || on.DelegatedReviewAdoptedAt != "2026-10-29" {
		t.Fatalf("bad decision %+v", on)
	}
	projection, err := s.EffectiveGovernance("")
	if err != nil {
		t.Fatal(err)
	}
	if e := governanceEntry(t, projection, "delegated-review"); !e.Supported || !e.Configured || !e.Effective || e.Value != "true" {
		t.Fatal(e)
	}
	docs, _ = LoadPolicyDocs(s.Root)
	if len(entry.Adopt(docs, "2026-11-01")) != 0 {
		t.Fatal("replay moved adoption date")
	}
	pins, _ := json.Marshal(docs.Review["trusted_attestation_issuers"])
	entry.Retire(docs)
	if err := docs.Write(s.Root, s); err != nil {
		t.Fatal(err)
	}
	off, _ := s.GetReviewPolicy()
	if off.DelegatedReview || !off.DelegatedReviewConfigured || off.DelegatedReviewDisabledAt == "" {
		t.Fatal("missing explicit off decision")
	}
	projection, err = s.EffectiveGovernance("")
	if err != nil {
		t.Fatal(err)
	}
	if e := governanceEntry(t, projection, "delegated-review"); e.Configured || e.Effective || e.Value != "false" || !strings.Contains(e.Explanation, "delegated-review-disabled") {
		t.Fatal(e)
	}
	after, _ := json.Marshal(docs.Review["trusted_attestation_issuers"])
	if string(after) != string(pins) {
		t.Fatal("off removed pins")
	}
	delete(docs.Review, "delegated_review")
	if len(entry.Retire(docs)) == 0 {
		t.Fatal("legacy omission was mistaken for explicit off")
	}
}

func TestDelegatedReviewCapabilityDisclosesOnlyAnExplicitRefusal(t *testing.T) {
	root, s := reviewBundleFixture(t)
	var p map[string]any
	raw, _ := os.ReadFile(filepath.Join(root, ".pose/policy/review.json"))
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	p["reviewer_independence"] = map[string]string{"spec": "different-actor"}
	write := func() {
		t.Helper()
		raw, _ := json.Marshal(p)
		writeReviewFixture(t, root, ".pose/policy/review.json", string(raw))
	}
	write()
	legacy, err := s.ReviewPlan("spec:backend")
	if err != nil || legacy.Independence != "different-actor" || len(legacy.Limitations) != 0 {
		t.Fatal(legacy, err)
	}
	p["delegated_review"] = false
	write()
	// Use a real repository for the sealing journey. A transient Git marker
	// in an ancestor of a non-Git fixture must never be interpreted as clean.
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(args, err, string(out))
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-qm", "disabled delegation fixture")
	head := git("rev-parse", "HEAD")
	graphPath := filepath.Join(root, ".pose/indexes/delivery-integrity.json")
	graphRaw, err := os.ReadFile(graphPath)
	if err != nil {
		t.Fatal(err)
	}
	var graph DeliveryIntegrityGraph
	if err := json.Unmarshal(graphRaw, &graph); err != nil {
		t.Fatal(err)
	}
	for i := range graph.ChangeSets {
		graph.ChangeSets[i].ResolvedBase, graph.ChangeSets[i].ResolvedHead = head, head
	}
	for i := range graph.ValidationResults {
		graph.ValidationResults[i].GitHead = head
	}
	graphRaw, _ = json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(graphRaw))
	plan, err := s.ReviewPlan("spec:backend")
	if err != nil || plan.Independence != "same-actor-separate-execution" || !containsString(plan.Limitations, ReviewLimitationDelegationDisabled) {
		t.Fatal(plan, err)
	}
	bundle, err := s.SealReviewBundle("spec:backend", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	att, err := s.RecordReviewAttestation(approvedBundleAttestation(bundle, "agent:solo"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(att.Limitations, ReviewLimitationDelegationDisabled) {
		t.Fatal("record lost the limitation")
	}
	v, err := s.VerifyReviewBundle("spec:backend")
	if err != nil || !v.Approved || v.Assurance == nil || !containsString(v.Assurance.Limitations, ReviewLimitationDelegationDisabled) {
		t.Fatal(v, err)
	}
	att.Limitations = nil
	if len(s.validateBundleAttestation(bundle, att)) == 0 {
		t.Fatal("omitted disclosure was accepted on read")
	}
	p["reviewer_independence"] = map[string]string{"spec": "mandatory-human"}
	write()
	human, err := s.ReviewPlan("spec:backend")
	if err != nil || human.Independence != "mandatory-human" || len(human.Limitations) != 0 {
		t.Fatal(human, err)
	}
}

func TestDelegatedReviewCapabilityDoesNotRelaxAProtectedBaseline(t *testing.T) {
	root, s := protectedBaselineFixture(t)
	weakenProtectedBaseline(t, root, func(p map[string]any) {
		p["delegated_review"] = false
		p["reviewer_independence"] = map[string]string{"spec": "same-actor-separate-execution"}
	})
	plan, err := s.ReviewPlan("spec:governance")
	if err != nil {
		t.Fatal(err)
	}
	if !plan.PolicyBaseline.Protected || plan.Independence != "different-actor" || len(plan.Limitations) != 0 {
		t.Fatalf("protected contract weakened: %+v", plan)
	}
}

// R1 review: delegation alone can supply the base revision's independence
// floor. Disabling it cannot make that same change review itself.
func TestDelegatedReviewCapabilityProtectsTheAdoptedFloor(t *testing.T) {
	root, s := protectedBaselineFixture(t)
	useIssuerHome(t)
	key, err := CreateIssuerKey(root, "baseline-executor", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var p map[string]any
	if err := json.Unmarshal([]byte(protectedBaselinePolicy), &p); err != nil {
		t.Fatal(err)
	}
	p["delegated_review"] = true
	p["authority_project"] = "proj.baseline"
	p["authority_audience"] = "baseline-audience"
	p["trusted_attestation_issuers"] = []string{key.Pin()}
	p["review_bundles"] = true
	p["review_bundles_adopted_at"] = "2026-09-01"
	p["reviewer_independence"] = map[string]string{"spec": "same-actor-separate-execution"}
	p["overlay_profiles"] = []string{}
	raw, _ := json.Marshal(p)
	writeReviewFixture(t, root, ".pose/policy/review.json", string(raw))
	designDeltaGit(t, root, "add", ".pose/policy/review.json")
	designDeltaGit(t, root, "commit", "-qm", "adopt delegation floor")
	head := strings.TrimSpace(string(mustDesignDeltaGitOutput(t, root, "rev-parse", "HEAD")))
	path := filepath.Join(root, ".pose/indexes/delivery-integrity.json")
	raw, _ = os.ReadFile(path)
	var graph DeliveryIntegrityGraph
	if err := json.Unmarshal(raw, &graph); err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets[0].ResolvedBase, graph.ChangeSets[0].ResolvedHead = head, head
	raw, _ = json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	p["delegated_review"] = false
	raw, _ = json.Marshal(p)
	writeReviewFixture(t, root, ".pose/policy/review.json", string(raw))
	plan, err := s.ReviewPlan("spec:governance")
	if err != nil || !plan.PolicyBaseline.Protected || plan.Independence != "different-actor" || len(plan.Limitations) != 0 || !plan.DelegatedRunProofRequired {
		t.Fatal(plan, err)
	}
	// Commit the disabling change, preserving its adopted predecessor as the
	// attributed base. Its approval needs a signed run even though live is off.
	designDeltaGit(t, root, "add", ".pose/policy/review.json")
	designDeltaGit(t, root, "commit", "-qm", "disable delegation")
	graph.ChangeSets[0].ResolvedHead = strings.TrimSpace(string(mustDesignDeltaGitOutput(t, root, "rev-parse", "HEAD")))
	raw, _ = json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	bundle, err := s.SealReviewBundle("spec:governance", time.Now())
	if err != nil || !bundle.Payload.Plan.DelegatedRunProofRequired {
		t.Fatal(bundle, err)
	}
	att := approvedBundleAttestation(bundle, "agent:independent-prefix-only")
	envelope, err := s.SignReviewAttestation(att, key, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RecordReviewAttestationEnvelope(envelope, true); err == nil || !strings.Contains(err.Error(), "signed run reference") {
		t.Fatal("signed prefix-only disabling approval was admitted", err)
	}
	att, err = s.VerifyReviewAttestationEnvelope(envelope)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(att)
	writeReviewFixture(t, root, ".pose/review-attestations/"+att.AttestationID+".json", string(raw))
	v, err := s.VerifyReviewBundle("spec:governance")
	if err != nil || v.Approved || !strings.Contains(strings.Join(v.Blockers, ";"), "signed run reference") {
		t.Fatal("read accepted the disabling prefix", v, err)
	}
	if v.Assurance == nil || v.Assurance.ReviewerIdentity != "not-verified" || !containsString(v.Assurance.Limitations, "signed-run-proof-required") {
		t.Fatal("read concealed required proof", v.Assurance)
	}
}

func TestDelegatedReviewCapabilityRefusesAConfigurationRace(t *testing.T) {
	s, docs, p := adoptionFixture(t)
	entry, _ := LookupCatalogEntry("delegated-review")
	entry.Adopt(docs, "2026-10-29")
	before := PublishFault
	t.Cleanup(func() { PublishFault = before })
	changed := false
	PublishFault = func(path string) error {
		if path == filepath.Join(s.Root, ".pose/policy/review.json") && !changed {
			changed = true
			p.Routing = &DelegationRouting{Prefer: "any"}
			raw, _ := CanonicalDelegationPolicy(p)
			return os.WriteFile(filepath.Join(s.Root, ".pose/policy/delegation.json"), raw, 0o600)
		}
		return nil
	}
	if err := docs.Write(s.Root, s); err == nil || !strings.Contains(err.Error(), "left disabled") {
		t.Fatalf("race result: %v", err)
	}
	policy, err := s.GetReviewPolicy()
	if err != nil || policy.DelegatedReview {
		t.Fatal("racing configuration remained enabled", err)
	}
}

func TestDelegatedReviewCapabilityValidatesTheCapturedConfiguration(t *testing.T) {
	s, docs, p := adoptionFixture(t)
	entry, _ := LookupCatalogEntry("delegated-review")
	entry.Adopt(docs, "2026-10-29")
	review, _, _, _, err := docs.Rendered(s)
	if err != nil {
		t.Fatal(err)
	}
	changed := false
	read := func(path string, limit int64) ([]byte, error) {
		if !changed {
			changed = true
			// LoadDelegationPolicy already validated A. Replace it with B at
			// precisely the old gap before the comparison bytes are captured.
			p.Conclusions = nil
			raw, err := CanonicalDelegationPolicy(p)
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				return nil, err
			}
		}
		return ReadReviewAdapterFile(path, limit)
	}
	if err := docs.publishDelegationActivationReading(s, review, read); err == nil || !strings.Contains(err.Error(), "conclusions.issuer") {
		t.Fatal("validated A but enabled B", err)
	}
	policy, err := s.GetReviewPolicy()
	if err != nil || policy.DelegatedReview {
		t.Fatal("captured invalid configuration was enabled", err)
	}
}

func TestDelegatedReviewCapabilitySkillAliasSubject(t *testing.T) {
	for _, tc := range []string{"valid", "external", "other-skill", "regular-file", "client-parent-link", "canonical-parent-link", "skill-file-link"} {
		t.Run(tc, func(t *testing.T) {
			root := t.TempDir()
			for _, dir := range []string{".claude/skills", ".agents/skills/pose-delegate"} {
				if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
					t.Fatal(err)
				}
			}
			skill := filepath.Join(root, ".agents/skills/pose-delegate/SKILL.md")
			if err := os.WriteFile(skill, []byte("canonical skill"), 0644); err != nil {
				t.Fatal(err)
			}
			rel := ".claude/skills/pose-delegate"
			alias := filepath.Join(root, rel)
			target := "../../.agents/skills/pose-delegate"
			switch tc {
			case "external":
				target = t.TempDir()
			case "other-skill":
				target = "../../.agents/skills/other"
			case "regular-file":
				if err := os.WriteFile(alias, []byte(target), 0644); err != nil {
					t.Fatal(err)
				}
			case "client-parent-link", "canonical-parent-link":
				parent := filepath.Join(root, ".claude/skills")
				if tc == "canonical-parent-link" {
					parent = filepath.Join(root, ".agents/skills")
				}
				moved := parent + "-real"
				if err := os.Rename(parent, moved); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(moved, parent); err != nil {
					t.Fatal(err)
				}
			case "skill-file-link":
				if err := os.Rename(skill, skill+".real"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(skill+".real", skill); err != nil {
					t.Fatal(err)
				}
			}
			if tc != "regular-file" {
				if err := os.Symlink(target, alias); err != nil {
					t.Fatal(err)
				}
			}
			class, included := reviewBundlePathClass(rel, ScopeRef{}, nil)
			if class != "governance" || !included {
				t.Fatalf("alias omitted: %q %v", class, included)
			}
			got, err := (Store{Root: root}).reviewBundleFileDigest(rel, ChangeSet{}, DeliveryIntegrityGraph{}, nil)
			if tc == "valid" {
				if err != nil || got != digestBytes([]byte(target)) {
					t.Fatalf("link bytes not sealed: %s %v", got, err)
				}
			} else if err == nil {
				t.Fatalf("unsafe alias accepted: %s", got)
			}
		})
	}
	for _, path := range []string{".claude/settings.local.json", ".claude/other", ".claude/skills/pose-delegate/private.bin"} {
		if class, _ := reviewBundlePathClass(path, ScopeRef{}, nil); class != "" {
			t.Fatalf("broad client exclusion/classification: %s => %s", path, class)
		}
	}
}
