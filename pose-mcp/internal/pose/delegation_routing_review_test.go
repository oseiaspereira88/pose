package pose

import (
	"sort"
	"strings"
	"testing"
	"time"
)

// B01 and B02 of the first review: a declared identity that differs from the
// governing bundle's seals a bundle that replaces it by name, so it governs
// even when both seals fall in one second and whatever their hashes say, and
// declaring a historical identity again seals anew instead of handing back the
// historical bundle.
func TestDeclaredIdentityReplacesTheGoverningBundle(t *testing.T) {
	_, s := reviewBundleFixture(t)
	prepared, err := s.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	// Order the two leads so the later seal has the smaller id: a tie broken
	// by the id would keep the earlier one governing.
	leads := []string{"anthropic/claude-opus-5-5", "openai/gpt-6.1-sol"}
	ids := map[string]string{}
	for _, lead := range leads {
		identity, _, err := s.SealingIdentity(lead)
		if err != nil {
			t.Fatal(err)
		}
		payload := prepared.Payload
		payload.Implementation = &identity
		digest, _ := reviewBundlePayloadDigest(payload)
		ids[lead] = digest
	}
	sort.Slice(leads, func(i, j int) bool { return ids[leads[i]] > ids[leads[j]] })
	now := time.Now().UTC().Truncate(time.Second)
	first, err := s.SealReviewBundleLead("spec:backend", leads[0], now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.SealReviewBundleLead("spec:backend", leads[1], now.Add(100*time.Millisecond))
	if err != nil || second.BundleID == first.BundleID {
		t.Fatalf("the second declaration did not seal: %v", err)
	}
	if current, err := s.CurrentReviewBundle("spec:backend"); err != nil || current == nil || current.BundleID != second.BundleID {
		t.Fatalf("the later declaration does not govern within one second: %+v %v", current, err)
	}
	again, err := s.SealReviewBundleLead("spec:backend", leads[0], now.Add(2*time.Second))
	if err != nil || again.BundleID == first.BundleID {
		t.Fatalf("declaring the first identity again handed back the historical bundle: %s %v", again.BundleID, err)
	}
	if current, _ := s.CurrentReviewBundle("spec:backend"); current == nil || current.BundleID != again.BundleID || !strings.Contains(current.Payload.Implementation.Principal, strings.Split(leads[0], "/")[0]) {
		t.Fatalf("the declaration again does not govern: %+v", current)
	}
}

// B07 of the first review: the last run of an agent is the one that started
// last by instant, not by the spelling of its timestamp.
func TestAvailabilityReadsTheLastRunByInstant(t *testing.T) {
	_, s := reviewBundleFixture(t)
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	policy := routingPolicy(t, `{"schema_version":2,"agents":{"x":{"vendor":"openai","model":"m","cli":"sh","commands":{"read":["sh"]}}}}`)
	for i, offset := range []time.Duration{100 * time.Millisecond, 110 * time.Millisecond} {
		run := ReviewRun{SchemaVersion: 1, BundleID: "rvb-fixture", Scope: "spec:backend", Adapter: "x", StartedAt: now.Add(offset).Format(time.RFC3339Nano), EndedAt: now.Add(time.Second).Format(time.RFC3339Nano), Status: ReviewRunCompleted}
		if i == 0 {
			run.Status, run.PartialDigest = ReviewRunInterrupted, digestBytes(nil)
			run.Interruption = &ReviewInterruption{Cause: "usage-limit", ResetAt: now.Add(time.Hour).Format(time.RFC3339)}
		}
		if _, err := s.recordReviewRun(run, nil); err != nil {
			t.Fatal(err)
		}
	}
	if a := s.DelegationAvailabilityOf(policy, now.Add(time.Second), false)["x"]; a.State != DelegationAgentAvailable {
		t.Fatalf("a limit at .1Z outranked the run that succeeded at .11Z: %+v", a)
	}
}

// F02 of the first review: a history that cannot be read whole leaves the
// agent's availability unknown, never available.
func TestAvailabilityIsUnknownOverAnUnreadableHistory(t *testing.T) {
	_, s := reviewBundleFixture(t)
	now := time.Now().UTC()
	policy := routingPolicy(t, `{"schema_version":2,"agents":{"x":{"vendor":"openai","model":"m","cli":"sh","commands":{"read":["sh"]}}}}`)
	run := ReviewRun{SchemaVersion: 1, BundleID: "rvb-fixture", Scope: "spec:backend", Adapter: "x", StartedAt: now.Format(time.RFC3339Nano), EndedAt: now.Format(time.RFC3339Nano), Status: ReviewRunInterrupted,
		PartialDigest: digestBytes(nil), Interruption: &ReviewInterruption{Cause: "usage-limit", ResetAt: "not-an-instant"}}
	if _, err := s.recordReviewRun(run, nil); err != nil {
		t.Fatal(err)
	}
	if a := s.DelegationAvailabilityOf(policy, now, false)["x"]; a.State == DelegationAgentAvailable || !strings.Contains(a.Detail, "could not be read") {
		t.Fatalf("an unreadable history read as available: %+v", a)
	}
}

// F01 of the first review: a reset with a fraction of a second is waited for
// whole.
func TestWaitKeepsTheFractionOfTheReset(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	reset := now.Add(900 * time.Millisecond)
	avail := map[string]DelegationAvailability{"x": {Agent: "x", State: DelegationAgentLimited, LimitedUntil: reset.Format(time.RFC3339Nano)}}
	_, until, ok := delegationWaitCandidate([]string{"x"}, avail, now, time.Hour)
	if at, err := time.Parse(time.RFC3339Nano, until); !ok || err != nil || at.Before(reset) {
		t.Fatalf("the wait ends at %s, before the reset %s", until, reset.Format(time.RFC3339Nano))
	}
}

// F03 of the first review: identities are compared as provenance normalizes
// them.
func TestIndependenceNormalizesBothSides(t *testing.T) {
	if got := AchievedIndependence("openai", "gpt-6.1-sol", " OpenAI ", " GPT-6.1-SOL "); got != IndependenceSameModel {
		t.Fatalf("spaces and case made a same-model run %s", got)
	}
	policy := routingPolicy(t, `{"schema_version":2,"agents":{"spaced":{"vendor":" OpenAI ","model":" GPT-6.1-SOL ","cli":"sh","commands":{"read":["sh"]}},"other":{"vendor":"anthropic","model":"m","cli":"sh","commands":{"read":["sh"]}}}}`)
	avail := map[string]DelegationAvailability{"spaced": {State: DelegationAgentAvailable}, "other": {State: DelegationAgentAvailable}}
	agent, _, err := RouteDelegation(policy, []string{"spaced", "other"}, ReviewBundleImplementation{Vendor: "openai", Model: "gpt-6.1-sol", Source: ImplementationDetected}, ImplementationFromBundle, avail, time.Now())
	if err != nil || agent != "other" {
		t.Fatalf("routing read a spaced vendor as another vendor: %s %v", agent, err)
	}
}

// B03 and B08 of the first review: a run stops governing when another bundle
// of the same content replaced its implementation identity, and a run routed
// against an identity its bundle does not record does not start.
func TestRunsFollowTheGoverningIdentity(t *testing.T) {
	declared := &ReviewBundleImplementation{Principal: "agent:anthropic/claude-opus-5-5", Vendor: "anthropic", Model: "claude-opus-5-5", ExecutionDigest: digestBytes([]byte("implementer")), Source: ImplementationDeclared}
	f := provenanceRunFixtureAs(t, declared, "openai", "implementation-model")
	recorded, err := f.store.LoadProvenanceReviewRun(f.run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	f.run = recorded
	if why := f.store.ReviewRunStillGoverns(f.run); why != "" {
		t.Fatalf("the run does not govern before the reseal: %s", why)
	}
	resealed, err := f.store.SealReviewBundleLead("spec:backend", "openai/gpt-6.1-sol", time.Now().Add(time.Second))
	if err != nil || resealed.BundleID == f.bundle.BundleID {
		t.Fatalf("reseal: %v", err)
	}
	if why := f.store.ReviewRunStillGoverns(f.run); !strings.Contains(why, "another sealed bundle of the same content now governs") {
		t.Fatalf("a run against a replaced identity still governs: %q", why)
	}
	plan, err := f.store.PlanReviewDispatch(resealed.BundleID, "review", "", "fake")
	if err != nil {
		t.Fatal(err)
	}
	stale := ExplicitDelegationRoute("fake", *declared, ImplementationFromBundle)
	plan.Routing = &stale
	if _, err := f.store.RunReviewDispatch(plan, time.Now); err == nil || !strings.Contains(err.Error(), "routed against another implementation") {
		t.Fatalf("a run routed against the replaced identity started: %v", err)
	}
}

// B05 of the first review: a run whose agent was named records that decision
// on the shared path, whatever command started it.
func TestNamedRunsRecordTheirRouting(t *testing.T) {
	declared := &ReviewBundleImplementation{Principal: "agent:openai/gpt-6.1-sol", Vendor: "openai", Model: "gpt-6.1-sol", ExecutionDigest: digestBytes([]byte("implementer")), Source: ImplementationDeclared}
	f := provenanceRunFixtureAs(t, declared, "openai", "review-model")
	if f.run.Routing == nil || f.run.Routing.Step != DelegationStepExplicit || f.run.Routing.Requested != "fake" || f.run.IndependenceAchieved != IndependenceDifferentModel {
		t.Fatalf("named run: %+v %s", f.run.Routing, f.run.IndependenceAchieved)
	}
	plan, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "", "fake", "again")
	if err != nil || !strings.Contains(plan.SameVendorWarning, "shares the implementation's vendor") {
		t.Fatalf("the shared plan does not warn: %q %v", plan.SameVendorWarning, err)
	}
}

// B09 of the second review: an agent whose vendor or model is unknown proves
// no difference, so it is never routed as the opposite vendor or another
// model ahead of an agent that is.
func TestUnknownCandidatesProveNoDifference(t *testing.T) {
	impl := ReviewBundleImplementation{Vendor: "anthropic", Model: "claude-opus-5-5", Source: ImplementationDetected}
	avail := map[string]DelegationAvailability{"a": {State: DelegationAgentAvailable}, "b": {State: DelegationAgentAvailable}}
	for name, c := range map[string]struct{ policy, agent, step string }{
		"unknown model before another model": {`{"schema_version":2,"agents":{"a":{"vendor":"anthropic","model":"unknown","cli":"sh","commands":{"read":["sh"]}},"b":{"vendor":"anthropic","model":"claude-sonnet-5-5","cli":"sh","commands":{"read":["sh"]}}}}`, "b", DelegationStepOtherModel},
		"unknown vendor before the opposite":  {`{"schema_version":2,"agents":{"a":{"vendor":"unknown","model":"m","cli":"sh","commands":{"read":["sh"]}},"b":{"vendor":"openai","model":"gpt-6.1-sol","cli":"sh","commands":{"read":["sh"]}}}}`, "b", DelegationStepPreferred},
	} {
		agent, record, err := RouteDelegation(routingPolicy(t, c.policy), []string{"a", "b"}, impl, ImplementationFromBundle, avail, time.Now())
		if err != nil || agent != c.agent || record.Step != c.step {
			t.Fatalf("%s: %s %s %v", name, agent, record.Step, err)
		}
	}
	only := `{"schema_version":2,"agents":{"a":{"vendor":"unknown","model":"unknown","cli":"sh","commands":{"read":["sh"]}}}}`
	if agent, record, err := RouteDelegation(routingPolicy(t, only), []string{"a"}, impl, ImplementationFromBundle, map[string]DelegationAvailability{"a": {State: DelegationAgentAvailable}}, time.Now()); err != nil || agent != "a" || record.Step != DelegationStepSameModel {
		t.Fatalf("an unknown agent alone is the same-model step: %s %s %v", agent, record.Step, err)
	}
}
