package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-falsifier-reconsideration.

const falsifierSpec = `---
slug: retry
status: done
---

# Spec: retry

## 2. Requirements

- R1: retry transient failures.

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: bounded retry.
- Selected option: bounded retry.
- Rationale: transient failures are short.
- Consequences: three attempts.
- Falsifier: more than 1% of charges still fail after three attempts.
- Expected effect: transient charge failures stop reaching users.
- Falsifier check: check:payments/retry-effect

### Decision D2
- Basis: R1
- Selected option: log every retry.
- Rationale: small fix, no hypothesis.
`

func falsifierFixture(t *testing.T, outcome string) Store {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/specs/2026-10-04-retry.md", falsifierSpec)
	if outcome != "" {
		write(".pose/indexes/delivery-integrity.json", `{"schema_version":1,"input_digest":"sha256:x","nodes":[],"edges":[],"claims":[],"change_sets":[],"reverse":{},"findings":[],
"validation_results":[{"id":"payments/retry-effect","module":"payments","check":"retry-effect","evidence_class":"integration","outcome":"`+outcome+`","git_head":"abc123"}]}`)
	}
	return Store{Root: root}
}

func reconsiderations(t *testing.T, s Store) []Obligation {
	t.Helper()
	report, err := s.ProjectObligations(ObligationQuery{Scope: "spec:retry"})
	if err != nil {
		t.Fatal(err)
	}
	var out []Obligation
	for _, o := range report.Obligations {
		if o.ReasonCode == "falsifier-observed" {
			out = append(out, o)
		}
	}
	for _, c := range report.Coverage {
		if c.Producer == "falsifiers" && c.State != CoverageStateCurrent {
			t.Fatalf("falsifier producer not read: %+v", c)
		}
	}
	return out
}

func TestAFailedFalsifierCheckRaisesAReconsiderationCandidate(t *testing.T) {
	got := reconsiderations(t, falsifierFixture(t, "fail"))
	if len(got) != 1 {
		t.Fatalf("a contrary observation did not raise a candidate: %+v", got)
	}
	o := got[0]
	if o.Category != ObligationJudgment || o.Effects[0].Mode != EffectAdvisory || o.Targets[0].ID != "D1" {
		t.Fatalf("the candidate is not an advisory request for judgment on D1: %+v", o)
	}
	if !strings.Contains(o.Message, "not a verdict") || !strings.Contains(o.Message, "expected effect: transient charge failures") || o.Observation.SourceRevision != "abc123" {
		t.Fatalf("message/observation: %+v", o)
	}
}

func TestAPassingOrAbsentObservationRaisesNothingAndTheDecisionIsKept(t *testing.T) {
	for _, outcome := range []string{"pass", ""} {
		s := falsifierFixture(t, outcome)
		if outcome == "" {
			// No index yet: the producer must not fail when no result exists.
			if err := os.MkdirAll(filepath.Join(s.Root, ".pose/indexes"), 0o755); err != nil {
				t.Fatal(err)
			}
			_ = os.WriteFile(filepath.Join(s.Root, ".pose/indexes/delivery-integrity.json"), []byte(`{"schema_version":1,"input_digest":"sha256:x","nodes":[],"edges":[],"claims":[],"change_sets":[],"reverse":{},"findings":[]}`), 0o644)
		}
		if got := reconsiderations(t, s); len(got) != 0 {
			t.Fatalf("outcome %q raised a candidate: %+v", outcome, got)
		}
	}
	s := falsifierFixture(t, "fail")
	before, _ := os.ReadFile(filepath.Join(s.Root, ".pose/specs/2026-10-04-retry.md"))
	reconsiderations(t, s)
	after, _ := os.ReadFile(filepath.Join(s.Root, ".pose/specs/2026-10-04-retry.md"))
	if string(before) != string(after) {
		t.Fatal("the projection edited the decision")
	}
}

func TestFalsifierFieldsAreOptInAndValidated(t *testing.T) {
	basis := ParseDesignBasis(falsifierSpec)
	var d1, d2 DesignDecision
	for _, d := range basis.Decisions {
		if d.ID == "D1" {
			d1 = d
		} else {
			d2 = d
		}
	}
	if d1.ExpectedEffect == "" || d1.FalsifierCheck != "check:payments/retry-effect" {
		t.Fatalf("D1 fields: %+v", d1)
	}
	if d2.ExpectedEffect != "" || d2.FalsifierCheck != "" {
		t.Fatalf("a small decision received hypothesis fields: %+v", d2)
	}
	for _, diag := range basis.Diagnostics {
		if strings.HasPrefix(diag.Code, "falsifier-check") {
			t.Fatalf("a valid opt-in produced %s", diag.Code)
		}
	}
	bad := strings.Replace(falsifierSpec, "check:payments/retry-effect", "payments retry", 1)
	bad = strings.Replace(bad, "- Falsifier: more than 1% of charges still fail after three attempts.\n", "", 1)
	codes := map[string]bool{}
	for _, diag := range ParseDesignBasis(bad).Diagnostics {
		codes[diag.Code] = true
	}
	if !codes["falsifier-check-invalid"] || !codes["falsifier-check-without-falsifier"] {
		t.Fatalf("diagnostics: %v", codes)
	}
}
