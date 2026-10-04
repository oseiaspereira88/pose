package pose

import (
	"encoding/json"
	"testing"
)

// Spec pose-legacy-contract-cutoffs: every registry entry's legacy key must be
// consumed. A policy that declares a date the engine never reads looks
// configured while producing no cutoff, which is the defect this guards.
func TestContractAdoptedAtReadsEveryRegisteredLegacyField(t *testing.T) {
	for _, contract := range ReviewContracts() {
		if contract.LegacyField == "" {
			continue
		}
		var policy ReviewPolicy
		raw := `{"schema_version":2,"` + contract.LegacyField + `":"2026-01-02"}`
		if err := json.Unmarshal([]byte(raw), &policy); err != nil {
			t.Fatal(err)
		}
		if got := policy.ContractAdoptedAt(contract.ID); got != "2026-01-02" {
			t.Errorf("%s: legacy key %s declared 2026-01-02, ContractAdoptedAt returned %q", contract.ID, contract.LegacyField, got)
		}
		if !policy.ContractAdoptionRecorded(contract.ID) {
			t.Errorf("%s: legacy key %s declared but adoption not recorded", contract.ID, contract.LegacyField)
		}
		known := map[string]bool{}
		for _, key := range ReviewPolicyKnownKeys() {
			known[key] = true
		}
		if !known[contract.LegacyField] {
			t.Errorf("%s: legacy key %s is not a key the typed policy models", contract.ID, contract.LegacyField)
		}
	}
}

func TestContractAdoptionMapPrevailsOverLegacyIncludingExplicitEmpty(t *testing.T) {
	for _, contract := range ReviewContracts() {
		if contract.LegacyField == "" {
			continue
		}
		cases := []struct {
			name, mapValue, want string
		}{
			{"map date", `"2026-03-04"`, "2026-03-04"},
			{"explicit empty", `""`, ""},
		}
		for _, tc := range cases {
			var policy ReviewPolicy
			raw := `{"schema_version":2,"` + contract.LegacyField + `":"2026-01-02","contract_adoptions":{"` + contract.ID + `":` + tc.mapValue + `}}`
			if err := json.Unmarshal([]byte(raw), &policy); err != nil {
				t.Fatal(err)
			}
			if got := policy.ContractAdoptedAt(contract.ID); got != tc.want {
				t.Errorf("%s/%s: got %q want %q", contract.ID, tc.name, got, tc.want)
			}
		}
	}
}

func TestContractAdoptionSourceDistinguishesEveryDeclaration(t *testing.T) {
	cases := []struct {
		raw, want string
	}{
		{`{}`, ContractAdoptionAbsent},
		{`{"explicit_judgment_adopted_at":"2026-01-02"}`, ContractAdoptionLegacy},
		{`{"explicit_judgment_adopted_at":""}`, ContractAdoptionLegacyEmpty},
		{`{"contract_adoptions":{"explicit-judgment":"2026-01-02"}}`, ContractAdoptionMap},
		{`{"contract_adoptions":{"explicit-judgment":""}}`, ContractAdoptionMapEmpty},
		{`{"explicit_judgment_adopted_at":"2026-01-02","contract_adoptions":{"explicit-judgment":"2026-02-03"}}`, ContractAdoptionMap},
	}
	for _, tc := range cases {
		got, err := ContractAdoptionSourceOf([]byte(tc.raw), "explicit-judgment")
		if err != nil {
			t.Fatal(err)
		}
		if got.Source != tc.want {
			t.Errorf("%s: source %q want %q", tc.raw, got.Source, tc.want)
		}
	}
	shadow, _ := ContractAdoptionSourceOf([]byte(`{"explicit_judgment_adopted_at":"2026-01-02","contract_adoptions":{"explicit-judgment":"2026-02-03"}}`), "explicit-judgment")
	if !shadow.LegacyShadowed {
		t.Error("a legacy key coexisting with a map entry must be reported as shadowed")
	}
}

func TestLoadReviewPolicyRejectsAnUnparsableRegisteredAdoptionDate(t *testing.T) {
	for _, contract := range ReviewContracts() {
		if contract.LegacyField == "" {
			continue
		}
		raw := []byte(`{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"},"` + contract.LegacyField + `":"20-26"}`)
		if _, err := (Store{}).parseReviewPolicy(raw); err == nil {
			t.Errorf("%s: an unparsable %s was accepted silently", contract.ID, contract.LegacyField)
		}
	}
}

// The two 6.0.0 contracts declared through their legacy keys now reach the
// dated rule for unstamped bundles, while a stamped bundle is still judged by
// its stamp whatever date is written afterwards.
func TestLegacyKeyCutoffsReachTheDatedRuleButNeverAStampedBundle(t *testing.T) {
	store, scope, _ := doneScopeFixture(t)
	for _, id := range []string{"explicit-judgment", "structural-causality"} {
		var policy ReviewPolicy
		raw := `{"` + LegacyContractField(id) + `":"2099-01-01"}`
		if err := json.Unmarshal([]byte(raw), &policy); err != nil {
			t.Fatal(err)
		}
		if !store.bundleContractExempt(scope, policy, ReviewBundle{}, id, "2026-09-09T12:00:00Z") {
			t.Errorf("%s: an unstamped bundle predating the legacy-declared adoption was not exempted", id)
		}
		stamped := ReviewBundle{Payload: ReviewBundlePayload{GoverningContracts: governingContractsAtSeal()}}
		if store.bundleContractExempt(scope, policy, stamped, id, "2026-09-09T12:00:00Z") {
			t.Errorf("%s: a stamped bundle was exempted by a later legacy date", id)
		}
	}
}
