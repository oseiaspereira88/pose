package pose

import (
	"reflect"
	"testing"
)

const assessmentWithPathsAndStale = `---
schema_version: 1
assessed_at: 2026-07-21
baseline_commit: 38a248d
---

# Capability assessment

## Mechanism: spec-lifecycle-closeout
- title: Spec lifecycle and closeout
- score: 5
- target: 5
- evidence: spec:demo-spec
- gaps: none named
- paths: internal/cli/lintspec.go;internal/cli/state_hooks.go
- stale: since=2026-07-23T10:00:00Z;trigger=spec:closeout-demo;hits=component:x,component:y
- stale: since=2026-07-23T11:00:00Z;trigger=spec:another-spec

## Mechanism: operational-knowledge
- title: Operational knowledge
- score: 3
- target: 5
- evidence: knowledge:demo-note
- gaps: none
`

func TestParseCapabilityAssessment_PathsAndStaleTriggers(t *testing.T) {
	assessment, err := ParseCapabilityAssessment(assessmentWithPathsAndStale)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := assessment.Mechanisms[0]
	if len(m.Paths) != 2 || m.Paths[0] != "internal/cli/lintspec.go" {
		t.Fatalf("paths = %v", m.Paths)
	}
	if len(m.StaleTriggers) != 2 {
		t.Fatalf("stale triggers = %+v, want 2", m.StaleTriggers)
	}
	first := m.StaleTriggers[0]
	if first.Since != "2026-07-23T10:00:00Z" || first.Trigger != "spec:closeout-demo" || len(first.Hits) != 2 {
		t.Fatalf("first trigger = %+v", first)
	}
	if assessment.Mechanisms[1].StaleTriggers != nil {
		t.Fatalf("second mechanism must have no stale triggers, got %+v", assessment.Mechanisms[1].StaleTriggers)
	}
}

func TestAppendUniqueStaleTrigger_DedupsBySameTrigger(t *testing.T) {
	triggers := []StaleTrigger{{Since: "2026-07-23T10:00:00Z", Trigger: "spec:a", Hits: []string{"component:x"}}}
	updated := appendUniqueStaleTrigger(triggers, StaleTrigger{Since: "2026-07-23T12:00:00Z", Trigger: "spec:a", Hits: []string{"component:y"}})
	if len(updated) != 1 || updated[0].Hits[0] != "component:y" {
		t.Fatalf("same-trigger append should replace, got %+v", updated)
	}
	updated = appendUniqueStaleTrigger(updated, StaleTrigger{Since: "2026-07-23T13:00:00Z", Trigger: "spec:b"})
	if len(updated) != 2 {
		t.Fatalf("different trigger must accumulate, got %+v", updated)
	}
}

func TestStaleTriggerBullet_FormatParseRoundTrip(t *testing.T) {
	trigger := StaleTrigger{Since: "2026-07-23T10:00:00Z", Trigger: "spec:demo", Hits: []string{"component:a", "component:b"}}
	formatted := FormatStaleTriggerBullet(trigger)
	parsed, err := ParseStaleTriggerBullet(formatted)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsed, trigger) {
		t.Fatalf("round trip = %+v, want %+v (formatted: %q)", parsed, trigger, formatted)
	}
}

func TestParseStaleTriggerBullet_RejectsMalformed(t *testing.T) {
	for _, value := range []string{
		"",
		"trigger=spec:demo", // missing since
		"since=2026-07-23T10:00:00Z", // missing trigger
		"since=not-a-date;trigger=spec:demo",
		"since=2026-07-23T10:00:00Z;trigger=spec:demo;unknown=x",
		"garbage",
	} {
		if _, err := ParseStaleTriggerBullet(value); err == nil {
			t.Errorf("ParseStaleTriggerBullet(%q) accepted malformed input", value)
		}
	}
}
