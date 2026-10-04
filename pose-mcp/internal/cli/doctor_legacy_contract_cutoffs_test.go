package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-legacy-contract-cutoffs: doctor names where every registered
// contract's adoption comes from, and no longer reports the two 6.0.0 legacy
// keys as unknown policy keys.
func TestDoctorReportsLegacyContractCutoffSources(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose/policy/review.json"), `{"schema_version":2,"explicit_judgment_adopted_at":"2026-09-20","structural_causality_adopted_at":"","contract_adoptions":{"component-aware":"2026-08-13"},"review_bundles_adopted_at":"2026-08-14"}`)
	findings := runDoctorJSON(t, root)
	finding, found := findDoctorFinding(findings, "review.adoption-source")
	if !found {
		t.Fatal("doctor did not report adoption sources")
	}
	for _, want := range []string{
		"explicit-judgment=legacy(2026-09-20)",
		"structural-causality=legacy-explicit-empty",
		"component-aware=map(2026-08-13)",
		"evidence-vocabulary=absent",
	} {
		if !strings.Contains(finding.Message, want) {
			t.Errorf("adoption sources %q lack %q", finding.Message, want)
		}
	}
	for _, f := range findings {
		if strings.Contains(f.Message, "explicit_judgment_adopted_at") && f.Level != "ok" {
			t.Errorf("legacy key still reported as a problem: %+v", f)
		}
	}
}

func TestDoctorReportsAShadowedLegacyCutoff(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose/policy/review.json"), `{"schema_version":2,"explicit_judgment_adopted_at":"2026-09-20","contract_adoptions":{"explicit-judgment":"2026-09-21"}}`)
	finding, _ := findDoctorFinding(runDoctorJSON(t, root), "review.adoption-source")
	if !strings.Contains(finding.Message, "explicit-judgment=map(2026-09-21)[legacy explicit_judgment_adopted_at shadowed by contract_adoptions]") {
		t.Fatalf("shadowed legacy key not named: %q", finding.Message)
	}
}
