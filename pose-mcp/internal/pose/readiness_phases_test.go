package pose

import (
	"strings"
	"testing"
	"time"
)

// Spec pose-phase-scoped-readiness.

func phaseOf(t *testing.T, phases SpecPhases, phase string) PhaseReadiness {
	t.Helper()
	for _, p := range phases.Phases {
		if p.Phase == phase {
			return p
		}
	}
	t.Fatalf("no phase %s", phase)
	return PhaseReadiness{}
}

func TestACloseoutOnlyRequestDoesNotRestrictStartOrExecution(t *testing.T) {
	s := actionFixture(t)
	r := decisionRequest()
	r.Effects = []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}}
	if _, err := s.OpenActionRequest(r, time.Now()); err != nil {
		t.Fatal(err)
	}
	phases, err := s.SpecPhaseReadiness("storage")
	if err != nil {
		t.Fatal(err)
	}
	if p := phaseOf(t, phases, PhaseExecution); p.State != PhaseClear || len(p.Restricting) != 0 {
		t.Fatalf("execution was restricted by a closeout-only request: %+v", p)
	}
	if p := phaseOf(t, phases, PhaseStart); p.State != PhaseClear {
		t.Fatalf("start: %+v", p)
	}
	if p := phaseOf(t, phases, PhaseCloseout); p.State != PhaseRestricted && p.State != PhasePartiallyRestricted {
		t.Fatalf("closeout must carry the request: %+v", p)
	}
}

func TestALocalizedRestrictionIsNotAFullStopNorProofOfIndependence(t *testing.T) {
	s := actionFixture(t)
	if _, err := s.OpenActionRequest(decisionRequest(), time.Now()); err != nil { // execution restricted on R4 only
		t.Fatal(err)
	}
	phases, _ := s.SpecPhaseReadiness("storage")
	p := phaseOf(t, phases, PhaseExecution)
	if p.State != PhasePartiallyRestricted || len(p.Restricting) != 1 || len(p.Restricting[0].Scope) != 1 || p.Restricting[0].Scope[0].ID != "R4" {
		t.Fatalf("an R4-only restriction: %+v", p)
	}
	if !strings.Contains(p.Note, "independence") || !strings.Contains(p.Note, "not demonstrated") {
		t.Fatalf("the note must not claim independence: %q", p.Note)
	}
}

func TestAPhaseWithAnUnreadProducerIsUnknownNotClear(t *testing.T) {
	s := actionFixture(t)
	phases, _ := s.SpecPhaseReadiness("storage")
	release := phaseOf(t, phases, PhaseRelease)
	if release.State != PhaseUnknown || len(release.Unread) == 0 {
		t.Fatalf("release is judged by producers that are not integrated: %+v", release)
	}
}

func TestLegacyReadyKeepsItsMeaning(t *testing.T) {
	s := actionFixture(t)
	before, err := s.SpecReadiness("storage")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.OpenActionRequest(decisionRequest(), time.Now()); err != nil {
		t.Fatal(err)
	}
	after, _ := s.SpecReadiness("storage")
	if before.Ready != after.Ready || before.Reason != after.Reason || len(before.WaitingOn) != len(after.WaitingOn) {
		t.Fatalf("Ready changed meaning: before %+v after %+v", before, after)
	}
	// The legacy corpus keeps the values it had before phases existed.
	want := map[string]bool{"base": false, "pending": true, "ready-one": true, "waiting": false, "finished": false, "stuck": false}
	store := readinessStore(t)
	for slug, ready := range want {
		legacy, err := store.SpecReadiness(slug)
		if err != nil {
			t.Fatal(err)
		}
		if legacy.Ready != ready {
			t.Fatalf("%s: legacy Ready = %v, want %v", slug, legacy.Ready, ready)
		}
	}
}
