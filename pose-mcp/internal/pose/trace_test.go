package pose

import (
	"strings"
	"testing"
)

const traceFixture = `---
slug: fixture
status: done
---

## 2. Requirements

### Functional
- R1: parse the thing.
- R2 [high]: reject the wrong thing.
- R3: withdrawn behavior.
- R4: never traced.

## 6. Validation

### Requirement trace
- R1 [satisfied] unit cases pass; check:test report:2026-07-19-report.md
- R2 [waived: covered by upstream gate] see check:vet
- R3 [withdrawn: superseded by R1]
- R9 [satisfied] check:test

## 7. Final Report
`

func TestParseRequirementTraceBidirectional(t *testing.T) {
	trace := ParseRequirementTrace(traceFixture)
	if !trace.HasSection {
		t.Fatal("trace section not detected")
	}
	if len(trace.Requirements) != 4 {
		t.Fatalf("requirements = %d, want 4", len(trace.Requirements))
	}
	byID := map[string]TraceRequirement{}
	for _, r := range trace.Requirements {
		byID[r.ID] = r
	}
	if e := byID["R1"].Entry; e == nil || e.Disposition != "satisfied" || len(e.Refs) != 2 {
		t.Errorf("R1 entry = %+v, want satisfied with 2 refs", e)
	}
	if e := byID["R2"].Entry; e == nil || e.Disposition != "waived" || e.Reason == "" {
		t.Errorf("R2 entry = %+v, want waived with reason", e)
	}
	if byID["R2"].Criticality != "high" {
		t.Errorf("R2 criticality = %q, want high", byID["R2"].Criticality)
	}
	if e := byID["R3"].Entry; e == nil || e.Disposition != "withdrawn" {
		t.Errorf("R3 entry = %+v, want withdrawn", e)
	}
	if len(trace.Missing) != 1 || trace.Missing[0] != "R4" {
		t.Errorf("missing = %v, want [R4]", trace.Missing)
	}
	if len(trace.Orphans) != 1 || trace.Orphans[0] != "R9" {
		t.Errorf("orphans = %v, want [R9]", trace.Orphans)
	}
	ids := trace.ByEvidence["check:test"]
	if len(ids) != 2 || ids[0] != "R1" || ids[1] != "R9" {
		t.Errorf("by_evidence[check:test] = %v, want [R1 R9]", ids)
	}
}

func TestParseRequirementTraceMalformedEntries(t *testing.T) {
	body := strings.NewReplacer(
		"- R1 [satisfied] unit cases pass; check:test report:2026-07-19-report.md", "- R1 [bogus] nope",
		"- R2 [waived: covered by upstream gate] see check:vet", "- R2 [waived]",
		"- R3 [withdrawn: superseded by R1]", "- R3 [satisfied]",
	).Replace(traceFixture)
	trace := ParseRequirementTrace(body)
	if len(trace.Errors) != 3 {
		t.Fatalf("errors = %v, want 3", trace.Errors)
	}
	for _, want := range []string{"invalid trace disposition", "requires a reason", "requires evidence"} {
		found := false
		for _, e := range trace.Errors {
			if strings.Contains(e, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("expected an error containing %q, got %v", want, trace.Errors)
		}
	}
}

func TestParseRequirementTraceWithoutSection(t *testing.T) {
	body := strings.Split(traceFixture, "### Requirement trace")[0] + "\n## 7. Final Report\n"
	trace := ParseRequirementTrace(body)
	if trace.HasSection {
		t.Fatal("section should be absent")
	}
	if len(trace.Missing) != 0 {
		t.Errorf("missing should be empty without a section (additive migration), got %v", trace.Missing)
	}
	if len(trace.Requirements) != 4 {
		t.Errorf("requirements = %d, want 4", len(trace.Requirements))
	}
}
