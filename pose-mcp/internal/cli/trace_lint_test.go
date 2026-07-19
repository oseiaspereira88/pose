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

func TestTraceLegacyDoneWarnsButPasses(t *testing.T) {
	body := strings.Replace(tracedDoneSpec, "### Requirement trace\n- R1 [satisfied] unit suite; check:test\n- R2 [waived: covered upstream]\n", "Validation prose.\n", 1)
	rc, output := lintFixture(t, body)
	if rc != 0 {
		t.Fatalf("legacy done spec without trace section must pass with a warning, got rc=%d output=%s", rc, output)
	}
	if !strings.Contains(output, "Requirement trace") {
		t.Errorf("expected legacy warning mentioning the trace section, got: %s", output)
	}
}
