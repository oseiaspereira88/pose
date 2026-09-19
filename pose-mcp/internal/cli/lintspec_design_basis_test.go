package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDesignBasisSpec(t *testing.T, root, body string) {
	t.Helper()
	dir := filepath.Join(root, ".pose", "specs", "basis")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	spec := `---
slug: basis
status: draft
created_at: 2026-09-19
components: pose-mcp
task_type: feature
---

# Spec: basis

## 1. Intent
Implement the advisory design basis projection.

## 2. Requirements
- R1: expose the decision basis without changing lifecycle.

## 3. Technical Plan
### Affected areas
- pose-mcp
### Artifacts
- modified: pose-mcp/internal/pose/design_basis.go

## 4. Tasks
- [x] implement the projection

## 5. Decisions
` + body + `

## 6. Validation
### Strategy
Run deterministic unit and CLI tests.

## 7. Final Report
### Delivered scope
Advisory projection only.
### Follow-ups
- [open] register delivery producer (owner:@pose-maintainers crit:medium review:2026-10-19)
`
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestABMDesignCheckCLIProjectsWithoutChangingLifecycle(t *testing.T) {
	root := newGitRepo(t)
	writeDesignBasisSpec(t, root, `### Assumption A1
- Claim: the local contract is idempotent.
- Status: unverified
- Affects: R1

### Decision D1
- Basis: R1, A1
- Minimal option: keep the current flow.
- Selected option: keep the current flow.
- Rationale: no durable boundary is required.
- Consequences: no new dependency.
- Falsifier: the requirement gains a durability clause.
`)
	inDir(t, root, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"lint-spec", "basis", "--design-check", "--strict"}, &out, &errB); code != 0 {
			t.Fatalf("design check exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		for _, want := range []string{
			"spec.design_basis.present=true",
			"spec.design_basis.assumptions=1",
			"spec.design_basis.decisions=1",
			"spec.design_basis.errors=0",
			"spec.design_basis.digest=sha256:",
		} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("output missing %q: %s", want, out.String())
			}
		}
		if !strings.Contains(out.String(), "spec.status=draft") {
			t.Errorf("design check lost lifecycle output: %s", out.String())
		}
		if strings.Contains(out.String(), "Resultado: FALHA") {
			t.Errorf("advisory warning unexpectedly failed: %s", out.String())
		}
	})
}

func TestABMDesignCheckCLIRejectsObjectiveBasisErrors(t *testing.T) {
	root := newGitRepo(t)
	writeDesignBasisSpec(t, root, `### Assumption A1
- Claim: external claim.
- Status: verified
- Evidence: url:https://example.invalid/claim
- Scope: production v3
- Affects: R9

### Decision D1
- Basis: R9, D1
- Selected option: current flow.
`)
	inDir(t, root, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"lint-spec", "basis", "--design-check", "--strict"}, &out, &errB); code != 1 {
			t.Fatalf("invalid design basis exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		for _, want := range []string{"design-basis/evidence-unknown-offline", "design-basis/orphan-ref", "design-basis/decision-cycle"} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("output missing diagnostic %q: %s", want, out.String())
			}
		}
	})
}

func TestABMDesignCheckFlagIsOptIn(t *testing.T) {
	root := newGitRepo(t)
	writeDesignBasisSpec(t, root, `### Decision D1
- Basis: R1
- Selected option: current flow.
`)
	inDir(t, root, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"lint-spec", "basis", "--strict"}, &out, &errB); code != 0 {
			t.Fatalf("legacy lint exit=%d out=%s err=%s", code, out.String(), errB.String())
		}
		if strings.Contains(out.String(), "spec.design_basis.") {
			t.Fatalf("design projection leaked into default lint: %s", out.String())
		}
	})
}
