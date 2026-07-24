package cli

import (
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

const staleEditFixture = `---
schema_version: 1
assessed_at: 2026-07-21
baseline_commit: 38a248d
---

# Capability assessment

## Mechanism: alpha
- title: Alpha
- score: 5
- target: 5
- evidence: spec:demo

Prose commentary for alpha, must survive untouched.

## Mechanism: beta
- title: Beta
- score: 3
- target: 5
- evidence: spec:demo
`

func TestAddStaleMark_InsertsAtEndOfBulletBlockPreservingProse(t *testing.T) {
	trigger := pose.StaleTrigger{Since: "2026-07-23T10:00:00Z", Trigger: "spec:closeout-demo", Hits: []string{"component:x"}}
	updated, err := addStaleMark(staleEditFixture, "alpha", trigger)
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := pose.ParseCapabilityAssessment(updated)
	if err != nil {
		t.Fatalf("edited content must still parse: %v\n%s", err, updated)
	}
	if len(assessment.Mechanisms[0].StaleTriggers) != 1 {
		t.Fatalf("alpha stale triggers = %+v", assessment.Mechanisms[0].StaleTriggers)
	}
	if len(assessment.Mechanisms[1].StaleTriggers) != 0 {
		t.Fatalf("beta must be untouched, got %+v", assessment.Mechanisms[1].StaleTriggers)
	}
	if !strings.Contains(updated, "Prose commentary for alpha, must survive untouched.") {
		t.Fatal("prose commentary was lost")
	}
}

func TestAddStaleMark_SameTriggerReplacesInPlace(t *testing.T) {
	first := pose.StaleTrigger{Since: "2026-07-23T10:00:00Z", Trigger: "spec:closeout-demo", Hits: []string{"component:x"}}
	updated, err := addStaleMark(staleEditFixture, "alpha", first)
	if err != nil {
		t.Fatal(err)
	}
	second := pose.StaleTrigger{Since: "2026-07-23T12:00:00Z", Trigger: "spec:closeout-demo", Hits: []string{"component:x", "component:y"}}
	updated, err = addStaleMark(updated, "alpha", second)
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := pose.ParseCapabilityAssessment(updated)
	if err != nil {
		t.Fatal(err)
	}
	triggers := assessment.Mechanisms[0].StaleTriggers
	if len(triggers) != 1 || triggers[0].Since != "2026-07-23T12:00:00Z" || len(triggers[0].Hits) != 2 {
		t.Fatalf("same-trigger repeat must replace in place, got %+v", triggers)
	}
}

func TestAddStaleMark_DifferentTriggerAccumulates(t *testing.T) {
	updated, err := addStaleMark(staleEditFixture, "alpha", pose.StaleTrigger{Since: "2026-07-23T10:00:00Z", Trigger: "spec:one"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err = addStaleMark(updated, "alpha", pose.StaleTrigger{Since: "2026-07-23T11:00:00Z", Trigger: "spec:two"})
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := pose.ParseCapabilityAssessment(updated)
	if err != nil {
		t.Fatal(err)
	}
	if len(assessment.Mechanisms[0].StaleTriggers) != 2 {
		t.Fatalf("different triggers must accumulate, got %+v", assessment.Mechanisms[0].StaleTriggers)
	}
}

func TestAddStaleMark_UnknownMechanismErrors(t *testing.T) {
	if _, err := addStaleMark(staleEditFixture, "ghost", pose.StaleTrigger{Since: "2026-07-23T10:00:00Z", Trigger: "spec:x"}); err == nil {
		t.Fatal("unknown mechanism must error")
	}
}

func TestClearStaleMarks_RemovesOnlyThatMechanismsMarks(t *testing.T) {
	withMarks, err := addStaleMark(staleEditFixture, "alpha", pose.StaleTrigger{Since: "2026-07-23T10:00:00Z", Trigger: "spec:x"})
	if err != nil {
		t.Fatal(err)
	}
	withMarks, err = addStaleMark(withMarks, "beta", pose.StaleTrigger{Since: "2026-07-23T10:00:00Z", Trigger: "spec:x"})
	if err != nil {
		t.Fatal(err)
	}
	cleared, didClear, err := clearStaleMarks(withMarks, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if !didClear {
		t.Fatal("expected didClear=true")
	}
	assessment, err := pose.ParseCapabilityAssessment(cleared)
	if err != nil {
		t.Fatalf("cleared content must still parse: %v\n%s", err, cleared)
	}
	if len(assessment.Mechanisms[0].StaleTriggers) != 0 {
		t.Fatalf("alpha must be cleared, got %+v", assessment.Mechanisms[0].StaleTriggers)
	}
	if len(assessment.Mechanisms[1].StaleTriggers) != 1 {
		t.Fatalf("beta must be untouched, got %+v", assessment.Mechanisms[1].StaleTriggers)
	}
	if !strings.Contains(cleared, "Prose commentary for alpha, must survive untouched.") {
		t.Fatal("prose commentary was lost during clear")
	}
}

func TestClearStaleMarks_NoMarksReportsFalse(t *testing.T) {
	_, didClear, err := clearStaleMarks(staleEditFixture, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if didClear {
		t.Fatal("expected didClear=false when there was nothing to clear")
	}
}
