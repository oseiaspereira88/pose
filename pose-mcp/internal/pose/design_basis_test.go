package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validDesignBasisFixture = `## 2. Requirements

- R1: retry transient failures without changing the public contract.

## 5. Decisions

### Assumption A1
- Claim: The payment API accepts an idempotency key.
- Status: verified
- Evidence: contract:payments-idempotency
- Scope: API v3, charge endpoint, revision 2026-09-19
- Affects: R1

### Decision D1
- Basis: R1, A1
- Minimal option: keep the current process and add bounded retry.
- Selected option: keep the current process and add bounded retry.
- Rationale: R1 does not require durable retries and A1 makes the operation safe.
- Consequences: three attempts with jitter; no queue or worker.
- Falsifier: the API removes the idempotency guarantee.
`

func TestABMDesignBasisValidAndDigestStable(t *testing.T) {
	report := ValidateDesignBasis(validDesignBasisFixture, "")
	if !report.HasSection || len(report.Assumptions) != 1 || len(report.Decisions) != 1 {
		t.Fatalf("unexpected projection: %+v", report)
	}
	if report.Assumptions[0].ID != "A1" || report.Assumptions[0].EvidenceRefs[0].Resolution != "declared" {
		t.Fatalf("assumption projection: %+v", report.Assumptions[0])
	}
	if report.Decisions[0].Basis[0] != "R1" || report.Decisions[0].SelectedOption == "" {
		t.Fatalf("decision projection: %+v", report.Decisions[0])
	}
	if len(report.Diagnostics) != 0 {
		t.Fatalf("valid fixture diagnostics: %+v", report.Diagnostics)
	}
	if report.Digest == "" || report.Digest != ValidateDesignBasis(validDesignBasisFixture, "").Digest {
		t.Fatalf("digest is not stable: %q", report.Digest)
	}

	// A line-number-only change is not semantic, while changing a selected
	// option is. This protects the projection from becoming a prose hash.
	shifted := "\n\n" + validDesignBasisFixture
	if report.Digest != ValidateDesignBasis(shifted, "").Digest {
		t.Fatalf("line movement changed semantic digest: %q != %q", report.Digest, ValidateDesignBasis(shifted, "").Digest)
	}
	changed := strings.Replace(validDesignBasisFixture, "bounded retry.", "durable queue.", 1)
	if report.Digest == ValidateDesignBasis(changed, "").Digest {
		t.Fatal("semantic change did not change digest")
	}
}

func TestABMDesignBasisPortugueseAndFenceAreParsedSafely(t *testing.T) {
	body := "" +
		"## 2. Requisitos\n- R1: atender o pedido.\n\n" +
		"<!-- ### Assumption A98\n- Status: verified\n-->\n\n" +
		"```markdown\n### Assumption A99\n- Status: verified\n```\n\n" +
		"## 5. Decisões\n\n" +
		"### Premissa A1\n" +
		"- Afirmação: O contrato local é estável.\n" +
		"- Estado: unverified\n" +
		"- Afeta: R1\n\n" +
		"### Decisão D1\n" +
		"- Base: R1, A1\n" +
		"- Opção mínima: manter o fluxo atual.\n" +
		"- Opção selecionada: manter o fluxo atual.\n" +
		"- Justificativa: o requisito não exige outra fronteira.\n" +
		"- Consequências: nenhuma nova dependência.\n" +
		"- Falsificador: requisito passa a exigir durabilidade.\n"
	report := ValidateDesignBasis(body, "")
	if len(report.Assumptions) != 1 || report.Assumptions[0].ID != "A1" {
		t.Fatalf("fenced example leaked into assumptions: %+v", report.Assumptions)
	}
	if len(report.Decisions) != 1 || report.Decisions[0].MinimalOption == "" || report.Decisions[0].Rationale == "" {
		t.Fatalf("Portuguese fields were not projected: %+v", report.Decisions[0])
	}
	if len(report.Diagnostics) != 0 {
		t.Fatalf("Portuguese fixture diagnostics: %+v", report.Diagnostics)
	}
}

func TestABMDesignBasisNegativeReferencesAndStatuses(t *testing.T) {
	body := `## 2. Requirements
- R1: do the thing.

## 5. Decisions
### Assumption A1
- Claim: something.
- Status: verified
- Evidence: url:https://untrusted.example/claim
- Scope: production v3
- Affects: R9
### Assumption A1
- Claim: duplicate.
- Status: unknown
### Decision D1
- Basis: A1, D1
- Selected option: current flow.
`
	report := ValidateDesignBasis(body, "")
	wantCodes := []string{"duplicate-id", "invalid-status", "orphan-ref", "evidence-unknown-offline", "decision-cycle", "no-requirement-path"}
	seen := map[string]bool{}
	for _, diagnostic := range report.Diagnostics {
		seen[diagnostic.Code] = true
	}
	for _, code := range wantCodes {
		if !seen[code] {
			t.Errorf("missing diagnostic %q: %+v", code, report.Diagnostics)
		}
	}
}

func TestABMDesignBasisLocalEvidenceResolutionAndTraversal(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "knowledge", "2026-09-19-contract-note.md"), []byte("note"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := `## 2. Requirements
- R1: keep the contract.
## 5. Decisions
### Assumption A1
- Claim: local note exists.
- Status: verified
- Evidence: knowledge:contract-note
- Scope: local project revision 1
- Affects: R1
### Decision D1
- Basis: R1, A1
- Minimal option: keep current.
- Selected option: keep current.
- Rationale: no additional boundary is required.
- Consequences: no new artifact.
- Falsifier: requirement changes.
`
	resolved := ValidateDesignBasis(body, root)
	if len(resolved.Diagnostics) != 0 || resolved.Assumptions[0].EvidenceRefs[0].Resolution != "resolved" {
		t.Fatalf("local evidence should resolve: %+v", resolved)
	}
	traversal := strings.Replace(body, "knowledge:contract-note", "doc:../outside.md", 1)
	bad := ValidateDesignBasis(traversal, root)
	if !hasDesignDiagnostic(bad, "invalid-evidence-ref") {
		t.Fatalf("traversal evidence was not rejected: %+v", bad.Diagnostics)
	}
}

func TestABMDesignBasisLegacySpecIsNoop(t *testing.T) {
	report := ValidateDesignBasis("## 2. Requirements\n- R1: legacy\n", "")
	if report.HasSection || report.Digest != "" || len(report.Diagnostics) != 0 {
		t.Fatalf("legacy spec changed under advisory parser: %+v", report)
	}
}

func hasDesignDiagnostic(report DesignBasisReport, code string) bool {
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
