package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func envOf(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

// R2, R3, D2: the vendor is detected from the session's markers, the model
// comes from --lead or routing.lead_models, and what nobody declared stays
// unknown; markers of two vendors leave the vendor unknown.
func TestSessionIdentityDetectsDeclaresAndNeverGuesses(t *testing.T) {
	cases := []struct {
		name       string
		env        map[string]string
		lead       string
		leadModels map[string]string
		vendor     string
		model      string
		source     string
		execution  bool
		warns      bool
	}{
		{"claude session", map[string]string{"CLAUDECODE": "1", "CLAUDE_CODE_SESSION_ID": "s-1"}, "", nil, "anthropic", "unknown", ImplementationDetected, true, false},
		{"codex session with lead model", map[string]string{"CODEX_THREAD_ID": "t-1"}, "", map[string]string{"openai": "gpt-6.1-sol"}, "openai", "gpt-6.1-sol", ImplementationDetected, true, false},
		{"no session", nil, "", map[string]string{"openai": "gpt-6.1-sol"}, "unknown", "unknown", ImplementationDetected, false, false},
		{"two vendors", map[string]string{"CLAUDECODE": "1", "CODEX_THREAD_ID": "t-1"}, "", nil, "unknown", "unknown", ImplementationDetected, false, true},
		{"declared over detection", map[string]string{"CLAUDECODE": "1", "CODEX_THREAD_ID": "t-1"}, "OpenAI/GPT-X", nil, "openai", "gpt-x", ImplementationDeclared, true, false},
		{"declared vendor only", nil, "anthropic", map[string]string{"anthropic": "claude-opus-5-5"}, "anthropic", "claude-opus-5-5", ImplementationDeclared, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			id, warning, err := SessionIdentity(envOf(c.env), c.lead, c.leadModels)
			if err != nil {
				t.Fatal(err)
			}
			if id.Vendor != c.vendor || id.Model != c.model || id.Source != c.source || (id.ExecutionDigest != "") != c.execution || (warning != "") != c.warns {
				t.Fatalf("%+v warning=%q", id, warning)
			}
			if strings.Contains(id.ExecutionDigest, "t-1") || strings.Contains(id.ExecutionDigest, "s-1") {
				t.Fatal("the session reference is stored, not bound")
			}
			if err := validateBundleImplementation(id); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, bad := range []string{"", "unknown", "openai/unknown", "a/b/c", "Open AI"} {
		if _, _, err := SessionIdentity(envOf(nil), bad, nil); bad != "" && err == nil {
			t.Fatalf("--lead %q accepted", bad)
		}
	}
}

// D2: an unknown vendor or model never establishes a difference.
func TestAchievedIndependence(t *testing.T) {
	for _, c := range [][5]string{
		{"anthropic", "claude-opus-5-5", "openai", "gpt-6.1-sol", IndependenceDifferentVendor},
		{"anthropic", "claude-opus-5-5", "anthropic", "claude-sonnet-5-5", IndependenceDifferentModel},
		{"anthropic", "unknown", "anthropic", "claude-sonnet-5-5", IndependenceSameModel},
		{"unknown", "unknown", "openai", "gpt-6.1-sol", IndependenceSameModel},
		{"Anthropic", "Claude-Opus-5-5", "anthropic", "claude-opus-5-5", IndependenceSameModel},
	} {
		if got := AchievedIndependence(c[0], c[1], c[2], c[3]); got != c[4] {
			t.Fatalf("%v: %s", c, got)
		}
	}
}

// R1, D1, D3: a seal records the sealing session's identity; resealing the
// same content in another session returns the bundle already sealed, so a
// reviewer closing out never becomes the implementation; a declared identity
// that differs seals anew, and both bundles match the same preparation.
func TestSealRecordsTheImplementationAndKeepsIt(t *testing.T) {
	_, s := reviewBundleFixture(t)
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CLAUDE_CODE_SESSION_ID", "implementer-session")
	first, err := s.SealReviewBundleLead("spec:backend", "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	impl := first.Payload.Implementation
	if impl == nil || impl.Vendor != "anthropic" || impl.Source != ImplementationDetected || impl.ExecutionDigest != digestBytes([]byte("implementer-session")) {
		t.Fatalf("seal recorded %+v", impl)
	}
	os.Unsetenv("CLAUDECODE")
	os.Unsetenv("CLAUDE_CODE_SESSION_ID")
	t.Setenv("CODEX_THREAD_ID", "reviewer-session")
	again, err := s.SealReviewBundleLead("spec:backend", "", time.Now())
	if err != nil || again.BundleID != first.BundleID {
		t.Fatalf("a reviewer's reseal minted %s over %s: %v", again.BundleID, first.BundleID, err)
	}
	declared, err := s.SealReviewBundleLead("spec:backend", "anthropic/claude-opus-5-5", time.Now().Add(time.Second))
	if err != nil || declared.BundleID == first.BundleID || declared.Payload.Implementation.Source != ImplementationDeclared {
		t.Fatalf("a declared identity did not seal anew: %+v %v", declared.Payload.Implementation, err)
	}
	prepared, err := s.PrepareReviewBundle("spec:backend")
	if err != nil || prepared.Payload.Implementation != nil || !sealedBundleMatches(first, prepared) || !sealedBundleMatches(declared, prepared) {
		t.Fatalf("a preparation does not match its sealed bundles: %v", err)
	}
	current, err := s.CurrentReviewBundle("spec:backend")
	if err != nil || current == nil || current.BundleID != declared.BundleID {
		t.Fatalf("the newest sealed bundle does not govern: %+v %v", current, err)
	}
	loaded, err := s.LoadReviewBundle(declared.BundleID)
	if err != nil || *loaded.Payload.Implementation != *declared.Payload.Implementation {
		t.Fatalf("the identity did not survive the record: %v", err)
	}
	raw, _ := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(declared.Path)))
	tampered := strings.Replace(string(raw), `"vendor": "anthropic"`, `"vendor": "openai"`, 1)
	os.WriteFile(filepath.Join(s.Root, filepath.FromSlash(declared.Path)), []byte(tampered), 0o644)
	if _, err := s.LoadReviewBundle(declared.BundleID); err == nil {
		t.Fatal("an edited implementation identity still loads")
	}
}

// R1, R4, R8, R11: a run is compared with the identity the bundle sealed, not
// with the policy's declaration, and discloses the independence it achieved;
// an unknown model never proves a different actor; an unknown vendor cannot
// be the implementation of an attested run.
func TestRunProvenanceReadsTheSealedImplementation(t *testing.T) {
	declared := &ReviewBundleImplementation{Principal: "agent:anthropic/claude-opus-5-5", Vendor: "anthropic", Model: "claude-opus-5-5", ExecutionDigest: digestBytes([]byte("implementer")), Source: ImplementationDeclared}
	// The policy declares openai/implementation-model; a reviewer of that
	// vendor is independent only because the sealed identity says anthropic.
	f := provenanceFixtureAs(t, declared, "openai", "implementation-model")
	att := f.attestation(t)
	if blockers := f.store.runBackedReviewBlockers(f.bundle, att); len(blockers) > 0 {
		t.Fatalf("the sealed identity was not read: %v", blockers)
	}
	a := f.store.DescribeReviewAssurance(f.bundle, &att)
	if a.Run == nil || a.Run.IndependenceAchieved != IndependenceDifferentVendor || a.Run.ImplementationSource != ImplementationDeclared || !strings.Contains(RenderReviewAssurance(a), "independence achieved different-vendor") {
		t.Fatalf("assurance: %+v", a.Run)
	}
	if f.run.IndependenceAchieved != IndependenceDifferentVendor {
		t.Fatalf("the run recorded %q", f.run.IndependenceAchieved)
	}
	replan, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "", "fake", "a second look")
	if err != nil || len(replan.Warnings) != 1 || !strings.Contains(replan.Warnings[0], "provenance.implementation in the delegation policy is ignored") {
		t.Fatalf("the ignored policy identity was not warned about: %v %v", replan.Warnings, err)
	}
	legacy := provenanceFixture(t, "openai", "review-model")
	if replan, err := legacy.store.PlanReviewDispatch(legacy.bundle.BundleID, "review", "", "fake", "a second look"); err != nil || len(replan.Warnings) != 0 || replan.Implementation().Principal != "agent:implementer" {
		t.Fatalf("an earlier bundle no longer reads the policy: %+v %v", replan.Implementation(), err)
	}

	unknownModel := &ReviewBundleImplementation{Principal: "agent:openai/unknown", Vendor: "openai", Model: "unknown", ExecutionDigest: digestBytes([]byte("implementer")), Source: ImplementationDetected}
	g := provenanceFixtureAs(t, unknownModel, "openai", "review-model")
	if blockers := g.store.runBackedReviewBlockers(g.bundle, g.attestation(t)); len(blockers) == 0 {
		t.Fatal("an unknown implementation model proved a different actor")
	}
	if g.run.IndependenceAchieved != IndependenceSameModel {
		t.Fatalf("the run recorded %q", g.run.IndependenceAchieved)
	}

	if _, err := runImplementationIdentity(ReviewBundle{Payload: ReviewBundlePayload{Implementation: &ReviewBundleImplementation{Vendor: "unknown", Model: "unknown", Source: ImplementationDetected}}}, nil); err == nil || !strings.Contains(err.Error(), "--lead") {
		t.Fatalf("an unknown vendor was accepted: %v", err)
	}
}

func routingPolicy(t *testing.T, raw string) DelegationPolicy {
	t.Helper()
	policy, err := ParseDelegationPolicy([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

const routingAgents = `{"schema_version": 2, "agents": {
  "codex": {"vendor": "openai", "model": "gpt-6.1-sol", "cli": "sh", "commands": {"read": ["sh"]}},
  "opus": {"vendor": "anthropic", "model": "claude-opus-5-5", "cli": "sh", "commands": {"read": ["sh"]}},
  "sonnet": {"vendor": "anthropic", "model": "claude-sonnet-5-5", "cli": "sh", "commands": {"read": ["sh"]}}
}%s}`

// R5, R7: auto prefers the opposite vendor, then the same vendor's other
// model, then waits for a limited agent within max_wait, then the same model;
// every skipped candidate carries its reason.
func TestRouteDelegationStepsInOrder(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	impl := ReviewBundleImplementation{Vendor: "anthropic", Model: "claude-opus-5-5", Source: ImplementationDetected}
	all := []string{"codex", "opus", "sonnet"}
	state := func(states map[string]string) map[string]DelegationAvailability {
		out := map[string]DelegationAvailability{}
		for _, name := range all {
			a := DelegationAvailability{Agent: name, State: DelegationAgentAvailable}
			switch v := states[name]; {
			case v == "missing":
				a.State, a.Detail = DelegationAgentMissing, "its CLI x is not on the path"
			case strings.HasPrefix(v, "limited "):
				a.State, a.LimitedUntil, a.Detail = DelegationAgentLimited, strings.TrimPrefix(v, "limited "), "limited until "+strings.TrimPrefix(v, "limited ")
			}
			out[name] = a
		}
		return out
	}
	policy := routingPolicy(t, strings.Replace(routingAgents, "%s", "", 1))
	cases := []struct {
		name   string
		states map[string]string
		agent  string
		step   string
	}{
		{"opposite vendor", nil, "codex", DelegationStepPreferred},
		{"other model", map[string]string{"codex": "missing"}, "sonnet", DelegationStepOtherModel},
		{"same model", map[string]string{"codex": "missing", "sonnet": "missing"}, "opus", DelegationStepSameModel},
		{"wait", map[string]string{"codex": "limited 2026-10-26T13:00:00Z", "sonnet": "limited 2026-10-26T12:30:00Z"}, "sonnet", DelegationStepWait},
		{"beyond max_wait", map[string]string{"codex": "limited 2026-10-27T12:00:00Z", "sonnet": "limited 2026-10-26T12:30:00Z"}, "opus", DelegationStepSameModel},
		{"a missing candidate is not waited for", map[string]string{"codex": "missing", "sonnet": "limited 2026-10-26T12:30:00Z"}, "opus", DelegationStepSameModel},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			agent, record, err := RouteDelegation(policy, all, impl, ImplementationFromBundle, state(c.states), now)
			if err != nil || agent != c.agent || record.Step != c.step {
				t.Fatalf("%s %+v %v", agent, record, err)
			}
			for _, cand := range record.Candidates {
				if cand.Agent != agent && cand.Skipped == "" {
					t.Fatalf("%s skipped without a reason", cand.Agent)
				}
			}
			if c.step == DelegationStepWait && record.WaitUntil != "2026-10-26T12:30:00Z" {
				t.Fatalf("wait until %s", record.WaitUntil)
			}
		})
	}
	strict := routingPolicy(t, strings.Replace(routingAgents, "%s", `, "routing": {"on_unavailable": []}`, 1))
	if _, record, err := RouteDelegation(strict, all, impl, ImplementationFromBundle, state(map[string]string{"codex": "missing"}), now); err == nil || !strings.Contains(err.Error(), "codex") || record.Step != "" {
		t.Fatalf("no fallback step still routed: %+v %v", record, err)
	}
	unknown := ReviewBundleImplementation{Vendor: "unknown", Model: "unknown", Source: ImplementationDetected}
	if agent, record, err := RouteDelegation(policy, all, unknown, ImplementationFromNone, state(nil), now); err != nil || agent != "codex" || record.Step != DelegationStepPreferred {
		t.Fatalf("an unknown implementation does not take the declared order: %s %+v %v", agent, record, err)
	}
	reordered := routingPolicy(t, `{"schema_version": 2, "agents": {
  "sonnet": {"vendor": "anthropic", "model": "claude-sonnet-5-5", "cli": "sh", "commands": {"read": ["sh"]}},
  "codex": {"vendor": "openai", "model": "gpt-6.1-sol", "cli": "sh", "commands": {"read": ["sh"]}}}}`)
	if names := reordered.DeclaredAgentNames(); strings.Join(names, ",") != "sonnet,codex" {
		t.Fatalf("declared order lost: %v", names)
	}
}

// R6: an agent is missing without its CLI, limited while its last run's
// usage limit is ahead, and unavailable when its probe fails.
func TestDelegationAvailabilityReadsPathRunsAndProbe(t *testing.T) {
	_, s := reviewBundleFixture(t)
	policy := routingPolicy(t, `{"schema_version": 2, "agents": {
  "gone": {"vendor": "a", "model": "m", "cli": "no-such-cli-for-pose-routing", "commands": {"read": ["sh"]}},
  "probed": {"vendor": "b", "model": "m", "cli": "sh", "commands": {"read": ["sh"]}, "probe": {"command": ["sh", "-c", "exit 3"], "timeout": "5s"}},
  "fine": {"vendor": "c", "model": "m", "cli": "sh", "commands": {"read": ["sh"]}, "probe": {"command": ["sh", "-c", "exit 0"]}}}}`)
	avail := s.DelegationAvailabilityOf(policy, time.Now(), true)
	if avail["gone"].State != DelegationAgentMissing || avail["probed"].State != DelegationAgentProbeFailed || avail["fine"].State != DelegationAgentAvailable {
		t.Fatalf("%+v", avail)
	}
	if skipped := s.DelegationAvailabilityOf(policy, time.Now(), false); skipped["probed"].State != DelegationAgentAvailable {
		t.Fatal("a probe ran when it was not asked to")
	}
	for _, bad := range []string{
		`"probe": {"command": []}`,
		`"probe": {"command": ["sh", "{session}"]}`,
		`"probe": {"command": ["sh"], "timeout": "2m"}`,
	} {
		raw := `{"schema_version": 2, "agents": {"x": {"vendor": "a", "model": "m", "commands": {"read": ["sh"]}, ` + bad + `}}}`
		if _, err := ParseDelegationPolicy([]byte(raw)); err == nil {
			t.Fatalf("probe accepted: %s", bad)
		}
	}
	for _, bad := range []string{
		`"routing": {"lead_models": {"openai": ""}}`,
		`"routing": {"lead_models": {"OpenAI": "gpt"}}`,
		`"routing": {"max_wait": "48h"}`,
		`"routing": {"on_unavailable": ["wait", "wait"]}`,
	} {
		raw := `{"schema_version": 2, "agents": {}, ` + bad + `}`
		if _, err := ParseDelegationPolicy([]byte(raw)); err == nil {
			t.Fatalf("routing accepted: %s", bad)
		}
	}
}
