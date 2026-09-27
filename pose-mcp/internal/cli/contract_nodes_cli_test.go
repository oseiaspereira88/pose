package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

const contractNodesCLISpec = `---
slug: nodes
status: in-progress
created_at: 2026-09-27
---

# Spec: nodes

## 2. Requirements

- R1: Keep legacy logs readable.

## 5. Decisions

### Assumption A1
- Claim: v1 logs hold only requirements
- Status: unverified
- Affects: R1

### Decision D1
- Basis: R1, A1
- Minimal option: keep one reader
- Selected option: dual reader
- Rationale: legacy logs stay readable
- Consequences: two schemas
- Falsifier: a v1 log misread
`

// contractNodesCLIFixture holds one folder spec and a review policy that does
// or does not adopt the contract_nodes_version capability.
func contractNodesCLIFixture(t *testing.T, adopted bool) string {
	t.Helper()
	root := t.TempDir()
	policy := `{"schema_version":1,"enabled":false}`
	if adopted {
		policy = `{"schema_version":1,"enabled":false,"contract_nodes_version":1}`
	}
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"), policy)
	mustWrite(t, filepath.Join(root, ".pose", "specs", "nodes", "spec.md"), contractNodesCLISpec)
	return root
}

func TestABMContractNodesCLIWithoutCapabilityKeepsTheRequirementGate(t *testing.T) {
	root := contractNodesCLIFixture(t, false)
	out, errOut, code := runCLI(t, root, "amend", "nodes", "--nodes", "--json")
	if code != 0 {
		t.Fatalf("--nodes failed: %s", errOut)
	}
	var view struct {
		Nodes   posepkg.ContractNodesProjection `json:"contract_nodes"`
		Adopted bool                            `json:"capability_adopted"`
	}
	if err := json.Unmarshal([]byte(out), &view); err != nil || view.Adopted || len(view.Nodes.Nodes) != 3 {
		t.Fatalf("projection without capability: %+v err=%v\n%s", view, err, out)
	}
	if _, errOut, code := runCLI(t, root, "amend", "nodes", "--ids", "A1", "--change", "semantic", "--rationale", "r", "--author", "@agent"); code == 0 || !strings.Contains(errOut, "contract_nodes_version") {
		t.Fatalf("an assumption was recorded without the capability: code=%d %s", code, errOut)
	}
	if _, _, code := runCLI(t, root, "amend", "nodes", "--baseline", "--author", "@agent"); code != 0 {
		t.Fatal("v1 baseline failed")
	}
	raw, _ := os.ReadFile(filepath.Join(root, ".pose", "specs", "nodes", "amendments.jsonl"))
	if !strings.Contains(string(raw), `"schema":1`) || strings.Contains(string(raw), `"A1"`) {
		t.Fatalf("without the capability the baseline must stay requirement-only schema 1: %s", raw)
	}
}

func TestABMContractNodesCLIAdoptedRecordsNodesAndRefusesMaskedChanges(t *testing.T) {
	root := contractNodesCLIFixture(t, true)
	specPath := filepath.Join(root, ".pose", "specs", "nodes", "spec.md")
	if _, errOut, code := runCLI(t, root, "amend", "nodes", "--baseline", "--author", "@agent"); code != 0 {
		t.Fatalf("v2 baseline failed: %s", errOut)
	}
	raw, _ := os.ReadFile(filepath.Join(root, ".pose", "specs", "nodes", "amendments.jsonl"))
	if !strings.Contains(string(raw), `"schema":2`) || !strings.Contains(string(raw), `"D1"`) || !strings.Contains(string(raw), `"assurance":"declared"`) {
		t.Fatalf("adopted baseline is not a schema-2 R/A/D snapshot: %s", raw)
	}

	// A changed basis is semantic; acknowledging it as editorial is refused.
	mustWrite(t, specPath, strings.Replace(contractNodesCLISpec, "- Basis: R1, A1", "- Basis: R1", 1))
	if _, errOut, code := runCLI(t, root, "amend", "nodes", "--ids", "D1", "--change", "editorial", "--rationale", "tidy", "--author", "@agent"); code == 0 || !strings.Contains(errOut, "cannot be acknowledged as editorial") {
		t.Fatalf("a basis change was masked as editorial: code=%d %s", code, errOut)
	}
	if _, errOut, code := runCLI(t, root, "amend", "nodes", "--ids", "D1", "--change", "semantic", "--rationale", "basis narrowed", "--author", "@agent"); code != 0 {
		t.Fatalf("semantic acknowledgement failed: %s", errOut)
	}

	// A state change needs an allowed transition event.
	mustWrite(t, specPath, strings.Replace(strings.Replace(contractNodesCLISpec, "- Basis: R1, A1", "- Basis: R1", 1), "- Status: unverified", "- Status: invalidated", 1))
	out, _, _ := runCLI(t, root, "amend", "nodes", "--list")
	if !strings.Contains(out, "A1 moved from unverified to invalidated without a transition event") {
		t.Fatalf("a silent transition was not pending:\n%s", out)
	}
	if _, errOut, code := runCLI(t, root, "amend", "nodes", "--ids", "A1", "--change", "transition", "--rationale", "contradicted by the pilot", "--author", "@agent"); code != 0 {
		t.Fatalf("transition failed: %s", errOut)
	}
	out, _, _ = runCLI(t, root, "amend", "nodes", "--list")
	if !strings.Contains(out, "amend.unacknowledged=0") || !strings.Contains(out, "unverified") || !strings.Contains(out, "→ invalidated") {
		t.Fatalf("history does not show before/after or still has pending changes:\n%s", out)
	}
}

func TestABMContractNodesCLILintRefusesSchema2WithoutCapability(t *testing.T) {
	root := contractNodesCLIFixture(t, true)
	if _, errOut, code := runCLI(t, root, "amend", "nodes", "--baseline", "--author", "@agent"); code != 0 {
		t.Fatalf("v2 baseline failed: %s", errOut)
	}
	// The capability is withdrawn: the schema-2 log must not be read as
	// requirement history by a done spec's gate.
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"), `{"schema_version":1,"enabled":false}`)
	mustWrite(t, filepath.Join(root, ".pose", "specs", "nodes", "spec.md"), strings.Replace(contractNodesCLISpec, "status: in-progress", "status: done\ncompleted_at: 2026-09-27", 1))
	out, errOut, _ := runCLI(t, root, "lint-spec", "nodes")
	if !strings.Contains(out+errOut, "schema-2 amendment events need") {
		t.Fatalf("lint read a schema-2 log without the capability:\n%s%s", out, errOut)
	}
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"), `{"schema_version":1,"enabled":false,"contract_nodes_version":2}`)
	if _, errOut, code := runCLI(t, root, "amend", "nodes", "--list"); code == 0 || !strings.Contains(errOut, "unsupported contract_nodes_version") {
		t.Fatalf("an unknown capability version was accepted: code=%d %s", code, errOut)
	}
}
