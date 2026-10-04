package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-assumption-validity-scope.

func premiseFixture(t *testing.T, trigger string) (string, Store) {
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
	write("api/payments.proto", "syntax = \"proto3\";\nmessage Charge { string idempotency_key = 1; }\n")
	write(".pose/specs/2026-10-04-payments.md", `---
slug: payments
status: in-progress
---

# Spec: payments

## 2. Requirements

- R1: retry transient failures without changing the public contract.

## 5. Decisions

### Assumption A1
- Claim: The payment API accepts an idempotency key.
- Status: verified
- Evidence: contract:payments-idempotency
- Scope: API v3, charge endpoint
- Valid scope: payments API v3 as described by api/payments.proto
- Stale trigger: `+trigger+`
- Affects: R1

### Assumption A2
- Claim: Retries are rare.
- Status: assumed
- Affects: R1

### Decision D1
- Basis: R1, A1
- Minimal option: bounded retry.
- Selected option: bounded retry.
- Rationale: A1 makes the retry safe.
- Consequences: three attempts.
- Falsifier: the provider drops idempotency keys.
`)
	return root, Store{Root: root}
}

func digestPrefix(t *testing.T, root, rel string) string {
	raw, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])[:12]
}

func premiseObligationsFor(t *testing.T, s Store) []Obligation {
	t.Helper()
	report, err := s.ProjectObligations(ObligationQuery{Scope: "spec:payments"})
	if err != nil {
		t.Fatal(err)
	}
	var out []Obligation
	for _, o := range report.Obligations {
		if o.ReasonCode == "premise-stale" {
			out = append(out, o)
		}
	}
	return out
}

func TestAContractChangeAsksForJudgmentOnTheBoundPremise(t *testing.T) {
	probe, _ := premiseFixture(t, "doc:api/payments.proto")
	pin := digestPrefix(t, probe, "api/payments.proto")
	_, s := premiseFixture(t, "doc:api/payments.proto@"+pin)
	if got := premiseObligationsFor(t, s); len(got) != 0 {
		t.Fatalf("an unchanged contract signalled a stale premise: %+v", got)
	}
	if err := os.WriteFile(filepath.Join(s.Root, "api/payments.proto"), []byte("syntax = \"proto3\";\nmessage Charge {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := premiseObligationsFor(t, s)
	if len(got) != 1 {
		t.Fatalf("a changed contract did not signal its premise: %+v", got)
	}
	o := got[0]
	if o.Category != ObligationJudgment || o.Waiting != WaitingActor {
		t.Fatalf("the signal is not a request for judgment: %+v", o)
	}
	// R4: advisory on review, scoped to the premise and the decisions on it.
	if len(o.Effects) != 1 || o.Effects[0].Mode != EffectAdvisory || o.Effects[0].Phase != PhaseReview {
		t.Fatalf("the stale premise invalidates more than it should: %+v", o.Effects)
	}
	ids := []string{}
	for _, n := range o.Effects[0].Scope {
		ids = append(ids, n.Kind+":"+n.ID)
	}
	if strings.Join(ids, ",") != "assumption:A1,decision:D1" {
		t.Fatalf("scope: %v", ids)
	}
	// R2: the evidence is retained and named as judged against the old content.
	if !strings.Contains(o.Message, "not presented as valid for the current one") || !strings.Contains(o.Message, "pinned "+pin) {
		t.Fatalf("message: %s", o.Message)
	}
	basis := ValidateDesignBasis(mustReadSpec(t, s, "payments"), s.Root)
	if basis.Assumptions[0].Evidence != "contract:payments-idempotency" {
		t.Fatal("the old evidence was dropped")
	}
}

func TestTrivialAssumptionsGetNoTTLAndCalendarTriggersAreNotEvaluated(t *testing.T) {
	_, s := premiseFixture(t, "date:2026-12-31")
	basis := ValidateDesignBasis(mustReadSpec(t, s, "payments"), s.Root)
	var a2 DesignAssumption
	sawCalendar := false
	for _, a := range basis.Assumptions {
		if a.ID == "A2" {
			a2 = a
		}
	}
	for _, d := range basis.Diagnostics {
		sawCalendar = sawCalendar || d.Code == "calendar-trigger"
		if d.ID == "A2" && strings.Contains(d.Code, "trigger") {
			t.Fatalf("a trivial assumption was asked for a trigger: %+v", d)
		}
	}
	if len(a2.StaleTriggers) != 0 || a2.ValidScope != "" {
		t.Fatalf("A2 received validity fields it did not declare: %+v", a2)
	}
	if !sawCalendar {
		t.Fatal("a calendar expiry was accepted as a material trigger")
	}
	if got := premiseObligationsFor(t, s); len(got) != 0 {
		t.Fatalf("a calendar trigger produced a stale premise: %+v", got)
	}
}

func TestValidityFieldsAreParsedAndDoNotChangeExistingDigests(t *testing.T) {
	// An assumption without the optional fields serializes exactly as
	// before, so existing design digests are unchanged.
	legacy := ValidateDesignBasis(validDesignBasisFixture, "")
	if raw, _ := json.Marshal(legacy.Assumptions[0]); strings.Contains(string(raw), "valid_scope") || strings.Contains(string(raw), "stale_triggers") {
		t.Fatalf("absent validity fields reach the digest input: %s", raw)
	}
	root, s := premiseFixture(t, "doc:api/payments.proto")
	basis := ValidateDesignBasis(mustReadSpec(t, s, "payments"), root)
	a1 := basis.Assumptions[0]
	if a1.ValidScope == "" || len(a1.StaleTriggers) != 1 || a1.StaleTriggers[0].State != TriggerUnpinned {
		t.Fatalf("fields: %+v", a1)
	}
	unpinned := false
	for _, d := range basis.Diagnostics {
		unpinned = unpinned || (d.Code == "stale-trigger-unpinned" && strings.Contains(d.Message, "@"+a1.StaleTriggers[0].Current))
	}
	if !unpinned {
		t.Fatalf("an unpinned trigger is not reported with the pin to use: %+v", basis.Diagnostics)
	}
	// Trigger state is projection, not semantics: editing the file does not
	// change the design digest.
	before := basis.Digest
	_ = os.WriteFile(filepath.Join(root, "api/payments.proto"), []byte("changed\n"), 0o644)
	if after := ValidateDesignBasis(mustReadSpec(t, s, "payments"), root).Digest; after != before {
		t.Fatal("content of a trigger target leaked into the design digest")
	}
}

func mustReadSpec(t *testing.T, s Store, slug string) string {
	t.Helper()
	sp, err := s.GetSpec(slug)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(sp.Path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestPortugueseValidityLabelsAreParsed(t *testing.T) {
	body := "## 2. Requirements\n\n- R1: algo.\n\n## 5. Decisions\n\n### Premissa A1\n- Afirmação: a API aceita chave de idempotência.\n- Estado: verified\n- Escopo de validade: API v3\n- Gatilho de obsolescência: doc:api/x.proto@abcdef12\n- Afeta: R1\n"
	a := ParseDesignBasis(body).Assumptions[0]
	if a.ValidScope != "API v3" || len(a.StaleTriggers) != 1 || a.StaleTriggers[0].Pin != "abcdef12" {
		t.Fatalf("pt-BR labels: %+v", a)
	}
}
