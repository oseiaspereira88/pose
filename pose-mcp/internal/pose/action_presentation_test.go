package pose

import (
	"testing"
	"time"
)

// Spec pose-action-request-presentation.
func TestRelatedRequestsAreGroupedWithoutMergingAnswers(t *testing.T) {
	s := actionFixture(t)
	now := time.Now()
	mk := func(question string, phase string) ActionRequestView {
		r := decisionRequest()
		r.Question = question
		r.Effects = []ObligationEffect{{Phase: phase, Mode: EffectBlock}}
		v, err := s.OpenActionRequest(r, now)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	a := mk("Keep v1 reader?", PhaseExecution)
	b := mk("Keep v1 writer?", PhaseExecution)
	c := mk("Announce the migration?", PhaseRelease)
	answered := mk("Already settled?", PhaseExecution)
	if _, err := s.ResolveActionRequest(answer(answered, "human:maintainer", "preserve-v1", "k"), now); err != nil {
		t.Fatal(err)
	}
	views, _ := s.ListActionRequests()
	p := PresentActionRequests(views, "")
	if p.InterruptNow != 2 || p.CanWait != 1 || len(p.Groups) != 2 {
		t.Fatalf("grouping: %+v", p)
	}
	first := p.Groups[0]
	if !first.Interrupt || first.EarliestPhase != PhaseExecution || len(first.Requests) != 2 {
		t.Fatalf("execution group first: %+v", first)
	}
	ids := map[string]string{}
	for _, g := range p.Groups {
		for _, item := range g.Requests {
			ids[item.ID] = item.RequestDigest
		}
	}
	for _, v := range []ActionRequestView{a, b, c} {
		if ids[v.Request.ID] != v.Request.RequestDigest {
			t.Fatalf("request %s lost its own identity or digest in the group", v.Request.ID)
		}
	}
	if _, asked := ids[answered.Request.ID]; asked {
		t.Fatal("a satisfied request is asked again")
	}
	if p.Groups[1].Interrupt {
		t.Fatal("a release-only request demands an immediate interruption")
	}
}
