package cli

// Closeout enforcement of the requirement trace (spec
// pose-requirement-evidence-traceability R2): satisfied/waived/withdrawn
// coverage at done, orphan rejection, and the legacy warning path.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTraceSpec(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "specs", "fixture", "spec.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const tracedDoneSpec = `---
slug: fixture
status: done
created_at: 2026-07-01
completed_at: 2026-07-02
---

## 1. Intent
Content.
## 2. Requirements
- R1: behave.
- R2: keep behaving.
## 3. Technical Plan
Content.
## 4. Tasks
- [x] done
## 6. Validation
### Requirement trace
- R1 [satisfied] unit suite; check:test
- R2 [waived: covered upstream]
## 7. Final Report
Delivered.
`

func lintFixture(t *testing.T, body string) (int, string) {
	t.Helper()
	var out, errB bytes.Buffer
	rc := lintOneSpec(writeTraceSpec(t, body), false, false, &out, &errB)
	return rc, out.String() + errB.String()
}

func TestTraceCloseoutComplete(t *testing.T) {
	rc, output := lintFixture(t, tracedDoneSpec)
	if rc != 0 {
		t.Fatalf("complete trace should pass, got rc=%d output=%s", rc, output)
	}
	if !strings.Contains(output, "spec.trace.present=true") || !strings.Contains(output, "spec.trace.entries=2") {
		t.Errorf("missing trace metrics: %s", output)
	}
}

func TestTraceCloseoutMissingRequirement(t *testing.T) {
	body := strings.Replace(tracedDoneSpec, "- R2 [waived: covered upstream]\n", "", 1)
	rc, output := lintFixture(t, body)
	if rc == 0 {
		t.Fatal("missing R2 trace at done must fail")
	}
	if !strings.Contains(output, "R2 has no trace entry") {
		t.Errorf("expected missing-entry diagnostic, got: %s", output)
	}
}

func TestTraceOrphanAlwaysFails(t *testing.T) {
	body := strings.Replace(tracedDoneSpec, "status: done", "status: in-progress", 1)
	body = strings.Replace(body, "- R1 [satisfied] unit suite; check:test", "- R1 [satisfied] unit suite; check:test\n- R9 [satisfied] check:test", 1)
	rc, output := lintFixture(t, body)
	if rc == 0 {
		t.Fatal("orphaned trace entry must fail regardless of status")
	}
	if !strings.Contains(output, "R9 is traced but not declared") {
		t.Errorf("expected orphan diagnostic, got: %s", output)
	}
}

// Was TestTraceLegacyDoneWarnsButPasses. The warning existed while eleven
// pre-contract specs still had no trace; they all have one now, so a done spec
// with requirements and no trace section is a real gap
// (spec pose-governance-gate-activation, R2).
func TestTraceDoneWithoutTraceSectionFails(t *testing.T) {
	body := strings.Replace(tracedDoneSpec, "### Requirement trace\n- R1 [satisfied] unit suite; check:test\n- R2 [waived: covered upstream]\n", "Validation prose.\n", 1)
	rc, output := lintFixture(t, body)
	if rc == 0 {
		t.Fatalf("a done spec with requirements and no trace section must fail, got rc=%d output=%s", rc, output)
	}
	if !strings.Contains(output, "Requirement trace") {
		t.Errorf("expected a diagnostic naming the trace section, got: %s", output)
	}
}

func TestLintSpecCoveredDispositionRecognizesFlatAndNestedSpecs(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, ".pose", "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".pose", "schema-version"), []byte("1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. Target spec-b in flat dated format
	specB := `---
slug: spec-b
status: in-progress
created_at: 2026-08-22
---
# Spec: B
## 1. Intent
Content.
## 2. Requirements
- R1: B works.
## 3. Technical Plan
None.
## 4. Tasks
- [ ] Task.
## 6. Validation
### Requirement trace
- R1 [satisfied] unit suite
## 7. Final Report
Pending.
`
	if err := os.WriteFile(filepath.Join(specsDir, "2026-08-22-spec-b.md"), []byte(specB), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. Target spec-c in nested folder format
	specC := `---
slug: spec-c
status: in-progress
created_at: 2026-08-22
---
# Spec: C
## 1. Intent
Content.
## 2. Requirements
- R1: C works.
## 3. Technical Plan
None.
## 4. Tasks
- [ ] Task.
## 6. Validation
### Requirement trace
- R1 [satisfied] unit suite
## 7. Final Report
Pending.
`
	if err := os.MkdirAll(filepath.Join(specsDir, "spec-c"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, "spec-c", "spec.md"), []byte(specC), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. Spec A in flat dated format, done, with follow-ups pointing to spec-b and spec-c
	specA := `---
slug: spec-a
status: done
created_at: 2026-08-01
completed_at: 2026-08-02
---
# Spec: A
## 1. Intent
Content.
## 2. Requirements
- R1: A works.
## 3. Technical Plan
Content.
## 4. Tasks
- [x] Done.
## 6. Validation
### Requirement trace
- R1 [satisfied] unit suite
## 7. Final Report
### Follow-ups
- [covered: spec-b] Covered by flat dated spec B
- [spawned: spec-c] Spawned into nested spec C
`
	specAPath := filepath.Join(specsDir, "2026-08-20-spec-a.md")
	if err := os.WriteFile(specAPath, []byte(specA), 0o644); err != nil {
		t.Fatal(err)
	}

	// Lint individual spec-a directly via lintOneSpec
	var out, errB bytes.Buffer
	rc := lintOneSpec(specAPath, false, false, &out, &errB)
	if rc != 0 {
		t.Fatalf("lintOneSpec for spec-a failed: rc=%d\nstdout=%s\nstderr=%s", rc, out.String(), errB.String())
	}
	if strings.Contains(errB.String(), "points to a missing spec") {
		t.Fatalf("falsely reported missing spec: %s", errB.String())
	}

	// Also test cmdLintSpec targeting "spec-a"
	out.Reset()
	errB.Reset()
	if code := cmdLintSpecInRoot(dir, []string{"spec-a", "--strict"}, &out, &errB); code != 0 {
		t.Fatalf("cmdLintSpec spec-a failed: code=%d\nstdout=%s\nstderr=%s", code, out.String(), errB.String())
	}

	// Also test cmdLintSpec with "--all"
	out.Reset()
	errB.Reset()
	if code := cmdLintSpecInRoot(dir, []string{"--all", "--strict"}, &out, &errB); code != 0 {
		t.Fatalf("cmdLintSpec --all failed: code=%d\nstdout=%s\nstderr=%s", code, out.String(), errB.String())
	}
}

func TestLintSpecCoveredDispositionWarnsWhenNoAnchorInTargetSpec(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, ".pose", "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Target spec with no anchor / mention of source-spec
	specTargetNoAnchor := `---
slug: target-no-anchor
status: in-progress
created_at: 2026-08-22
completed_at:
---
# Spec: Target No Anchor
## 2. Requirements
- R1: Independent requirements.
## 3. Technical Plan
Plan.
## 4. Tasks
- [ ] Task.
`
	if err := os.WriteFile(filepath.Join(specsDir, "target-no-anchor.md"), []byte(specTargetNoAnchor), 0o644); err != nil {
		t.Fatal(err)
	}

	// Target spec with depends_on anchor
	specTargetWithAnchor := `---
slug: target-with-anchor
status: in-progress
created_at: 2026-08-22
completed_at:
depends_on: source-spec
---
# Spec: Target With Anchor
## 2. Requirements
- R1: Anchored requirements.
## 3. Technical Plan
Plan.
## 4. Tasks
- [ ] Task.
`
	if err := os.WriteFile(filepath.Join(specsDir, "target-with-anchor.md"), []byte(specTargetWithAnchor), 0o644); err != nil {
		t.Fatal(err)
	}

	// Source spec referencing both
	sourceSpec := `---
slug: source-spec
status: done
created_at: 2026-08-20
completed_at: 2026-08-22
---
# Spec: Source Spec
## 1. Intent
Intent.
## 2. Requirements
- R1: Source works.
## 3. Technical Plan
Plan.
### Artifacts
- created: doc.md

### Delivery targets
Nenhum
## 4. Tasks
- [x] Done.
## 5. Validation
- Suite passed.
### Requirement trace
- R1 [satisfied] suite
## 7. Final Report
### Follow-ups
- [covered: target-no-anchor] Unanchored follow-up work
- [covered: target-with-anchor] Anchored follow-up work
`
	sourcePath := filepath.Join(specsDir, "source-spec.md")
	if err := os.WriteFile(sourcePath, []byte(sourceSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errB bytes.Buffer
	rc := lintOneSpec(sourcePath, false, false, &out, &errB)
	if rc != 0 {
		t.Fatalf("lintOneSpec failed: rc=%d err=%s", rc, errB.String())
	}

	stderrStr := errB.String()
	// Warning must be emitted for target-no-anchor
	if !strings.Contains(stderrStr, "[WARNING] source-spec: follow-up [covered: target-no-anchor] has no verifiable anchor") {
		t.Fatalf("expected warning for unanchored covered target, got: %s", stderrStr)
	}
	// Warning must NOT be emitted for target-with-anchor
	if strings.Contains(stderrStr, "target-with-anchor] has no verifiable anchor") {
		t.Fatalf("unexpected warning for anchored target: %s", stderrStr)
	}
}


