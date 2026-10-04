package pose

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Spec pose-governed-effect-enforcement.

func adoptAgencyReadiness(t *testing.T, s Store, extra string) {
	t.Helper()
	policy := `{"schema_version":2,"enabled":false,"profiles":{"spec":"spec-closeout@1"},"agency_readiness_version":1` + extra + `}`
	if err := os.WriteFile(filepath.Join(s.Root, ".pose/policy/review.json"), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
}

func closeoutRequest() ActionRequest {
	r := decisionRequest()
	r.Effects = []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}, {Phase: PhaseStart, Mode: EffectBlock}, {Phase: PhaseRelease, Mode: EffectBlock}}
	return r
}

func hasDiagnostic(state CloseoutState, code string) bool {
	for _, d := range state.Diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}

func TestWithoutAdoptionARequestRestrictsNoTransition(t *testing.T) {
	s := actionFixture(t)
	if _, err := s.OpenActionRequest(closeoutRequest(), time.Now()); err != nil {
		t.Fatal(err)
	}
	state, err := s.GetCloseoutState("spec:storage")
	if err != nil {
		t.Fatal(err)
	}
	if hasDiagnostic(state, "action-request-pending") {
		t.Fatal("an instance that did not adopt received the gate")
	}
	if refusals, _ := s.ReleaseGovernedRefusal([]string{"storage"}); len(refusals) != 0 {
		t.Fatalf("release refused without adoption: %+v", refusals)
	}
}

func TestAdoptedEffectsRefuseCloseoutUntilTheRequestIsSatisfied(t *testing.T) {
	s := actionFixture(t)
	adoptAgencyReadiness(t, s, "")
	view, err := s.OpenActionRequest(closeoutRequest(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	state, _ := s.GetCloseoutState("spec:storage")
	if !hasDiagnostic(state, "action-request-pending") || state.Terminal {
		t.Fatalf("closeout is not restricted: %+v", state.Diagnostics)
	}
	if refusals, _ := s.ReleaseGovernedRefusal([]string{"storage"}); len(refusals) != 1 {
		t.Fatalf("release is not refused: %+v", refusals)
	}
	if _, err := s.ResolveActionRequest(answer(view, "human:maintainer", "preserve-v1", "k1"), time.Now()); err != nil {
		t.Fatal(err)
	}
	state, _ = s.GetCloseoutState("spec:storage")
	if hasDiagnostic(state, "action-request-pending") {
		t.Fatal("a satisfied request still restricts closeout: the gate did not revalidate")
	}
}

func TestADeclinedRequestKeepsRefusing(t *testing.T) {
	s := actionFixture(t)
	adoptAgencyReadiness(t, s, "")
	r := closeoutRequest()
	r.Kind, r.Options, r.Recommend = ActionApproval, nil, ""
	view, _ := s.OpenActionRequest(r, time.Now())
	if _, err := s.ResolveActionRequest(answer(view, "human:maintainer", "decline", "k1"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if state, _ := s.GetCloseoutState("spec:storage"); !hasDiagnostic(state, "action-request-pending") {
		t.Fatal("a declined approval released closeout")
	}
}

func TestAStartPreviewTakenBeforeARequestIsStaleAtApply(t *testing.T) {
	s := actionFixture(t)
	spec := filepath.Join(s.Root, ".pose/specs/2026-10-04-storage.md")
	raw, _ := os.ReadFile(spec)
	_ = os.WriteFile(spec, []byte(string(raw[:0])+replaceOnce(string(raw), "status: in-progress", "status: draft")), 0o644)
	adoptAgencyReadiness(t, s, `,"atomic_start_version":1`)
	plan, err := s.PreviewStart("storage")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.OpenActionRequest(closeoutRequest(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ApplyStart(plan, plan.Digest, nil); err == nil {
		t.Fatal("a start previewed before the request was applied after it")
	}
	fresh, _ := s.PreviewStart("storage")
	found := false
	for _, w := range fresh.WaitingOn {
		found = found || w.Code == "action-request-pending"
	}
	if fresh.Ready || !found {
		t.Fatalf("the fresh preview does not report the restriction: %+v", fresh)
	}
}

func replaceOnce(s, old, new string) string {
	for i := 0; i+len(old) <= len(s); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}
