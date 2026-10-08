package pose

import (
	"slices"
	"testing"
)

// Spec pose-attention-lists-only-open-obligations: what waits on a person or
// on a gate is what is still open. A satisfied, waived or cancelled obligation
// stays in the report and leaves the attention lists, as it already leaves
// Blocking; an invalidated one is open again.
func TestAttentionListsOnlyOpenObligations(t *testing.T) {
	block := []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}}
	ob := func(id, category, satisfaction string, recipient ObligationActor) Obligation {
		return Obligation{ID: id, Category: category, Satisfaction: satisfaction, Recipient: recipient, Effects: block, Waiting: WaitingActor}
	}
	maintainer := ObligationActor{Role: "maintainer"}
	report := ObligationReport{Snapshot: ObligationSnapshot{Coherent: true}, Obligations: []Obligation{
		ob("obl-answered", "actor-action", SatisfactionSatisfied, maintainer),
		ob("obl-waived", "actor-action", SatisfactionWaived, maintainer),
		ob("obl-cancelled", "actor-action", SatisfactionCancelled, maintainer),
		ob("obl-asked", "actor-action", SatisfactionPending, maintainer),
		ob("obl-reasked", "actor-action", SatisfactionInvalidated, maintainer),
		{ID: "obl-gate-open", Category: "dependency", Satisfaction: SatisfactionPending, Effects: block, Waiting: WaitingArtifact},
		{ID: "obl-gate-met", Category: "dependency", Satisfaction: SatisfactionSatisfied, Effects: block, Waiting: WaitingArtifact},
	}}
	for _, actor := range []string{"", "maintainer"} {
		a := BuildAttention(report, actor)
		if !slices.Equal(a.ForActor, []string{"obl-asked", "obl-reasked"}) {
			t.Errorf("actor %q: for_actor = %v, want only the open requests", actor, a.ForActor)
		}
		if !slices.Equal(a.Gates, []string{"obl-gate-open"}) {
			t.Errorf("actor %q: gates = %v, want only the open gate", actor, a.Gates)
		}
		if !slices.Equal(a.Blocking[PhaseCloseout], []string{"obl-asked", "obl-gate-open", "obl-reasked"}) {
			t.Errorf("actor %q: blocking = %v", actor, a.Blocking[PhaseCloseout])
		}
	}
}
