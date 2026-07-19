package cli

// Amendment history behavior (spec pose-spec-amendment-history): baseline,
// material-change events, post-evidence mutation rejection at closeout and
// editorial acknowledgment.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const amendSpec = `---
slug: amended
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
- R1 [satisfied] check:test
- R2 [satisfied] check:test
## 7. Final Report
Delivered.
`

func amendFixture(t *testing.T) (root, specPath string) {
	t.Helper()
	root = t.TempDir()
	specPath = filepath.Join(root, ".pose", "specs", "amended", "spec.md")
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(specPath, []byte(amendSpec), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, specPath
}

func runAmend(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	var out, errB bytes.Buffer
	var code int
	inDir(t, root, func() {
		code = Main(append([]string{"amend"}, args...), &out, &errB)
	})
	return code, out.String() + errB.String()
}

func TestAmendBaselineAndCloseoutGate(t *testing.T) {
	root, specPath := amendFixture(t)
	if code, out := runAmend(t, root, "amended", "--baseline", "--author", "@core"); code != 0 {
		t.Fatalf("baseline exit=%d: %s", code, out)
	}
	// Acknowledged state: closeout passes.
	var o, e bytes.Buffer
	if rc := lintOneSpec(specPath, false, false, &o, &e); rc != 0 {
		t.Fatalf("acknowledged spec must pass lint: %s", o.String()+e.String())
	}
	if !strings.Contains(o.String(), "spec.amendments.events=1") {
		t.Errorf("expected amendments metric, got: %s", o.String())
	}
	// Post-evidence mutation without an event: closeout fails.
	mutated := strings.Replace(amendSpec, "- R2: keep behaving.", "- R2: do something else entirely.", 1)
	if err := os.WriteFile(specPath, []byte(mutated), 0o644); err != nil {
		t.Fatal(err)
	}
	o.Reset()
	e.Reset()
	if rc := lintOneSpec(specPath, false, false, &o, &e); rc == 0 {
		t.Fatal("unacknowledged semantic change at done must fail")
	}
	if !strings.Contains(e.String(), "R2 changed after its last acknowledged amendment") {
		t.Errorf("expected unacknowledged-change diagnostic, got: %s", e.String())
	}
	// Acknowledging the change clears the gate.
	if code, out := runAmend(t, root, "amended", "--ids", "R2", "--change", "semantic", "--rationale", "scope pivot", "--author", "@core", "--reviewer", "@lead"); code != 0 {
		t.Fatalf("amend exit=%d: %s", code, out)
	}
	o.Reset()
	e.Reset()
	if rc := lintOneSpec(specPath, false, false, &o, &e); rc != 0 {
		t.Fatalf("acknowledged change must pass: %s", o.String()+e.String())
	}
}

func TestAmendRemovalNeedsWithdrawnEvent(t *testing.T) {
	root, specPath := amendFixture(t)
	if code, _ := runAmend(t, root, "amended", "--baseline", "--author", "@core"); code != 0 {
		t.Fatal("baseline failed")
	}
	removed := strings.Replace(amendSpec, "- R2: keep behaving.\n", "", 1)
	removed = strings.Replace(removed, "- R2 [satisfied] check:test\n", "", 1)
	if err := os.WriteFile(specPath, []byte(removed), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	if rc := lintOneSpec(specPath, false, false, &o, &e); rc == 0 {
		t.Fatal("silent removal must fail closeout")
	}
	if !strings.Contains(e.String(), "R2 was removed without a withdrawn amendment event") {
		t.Errorf("expected removal diagnostic, got: %s", e.String())
	}
	if code, out := runAmend(t, root, "amended", "--ids", "R2", "--change", "withdrawn", "--rationale", "descoped", "--author", "@core"); code != 0 {
		t.Fatalf("withdrawn amend exit=%d: %s", code, out)
	}
	o.Reset()
	e.Reset()
	if rc := lintOneSpec(specPath, false, false, &o, &e); rc != 0 {
		t.Fatalf("acknowledged withdrawal must pass: %s", o.String()+e.String())
	}
}

func TestAmendValidation(t *testing.T) {
	root, _ := amendFixture(t)
	if code, out := runAmend(t, root, "amended", "--ids", "R1", "--change", "semantic", "--rationale", "x", "--author", "not-an-alias"); code == 0 {
		t.Fatalf("author without @ must be rejected: %s", out)
	}
	if code, out := runAmend(t, root, "amended", "--ids", "R1", "--change", "bogus", "--rationale", "x", "--author", "@a"); code == 0 {
		t.Fatalf("invalid change must be rejected: %s", out)
	}
	if code, out := runAmend(t, root, "amended", "--ids", "R9", "--change", "semantic", "--rationale", "x", "--author", "@a"); code == 0 {
		t.Fatalf("undeclared id for semantic must be rejected: %s", out)
	}
	if code, out := runAmend(t, root, "amended", "--ids", "R1", "--change", "withdrawn", "--rationale", "x", "--author", "@a"); code == 0 {
		t.Fatalf("withdrawn of a still-declared id must be rejected: %s", out)
	}
}

func TestAmendList(t *testing.T) {
	root, _ := amendFixture(t)
	if code, out := runAmend(t, root, "amended", "--list"); code != 0 || !strings.Contains(out, "no amendments") {
		t.Fatalf("empty list: code=%d out=%s", code, out)
	}
	runAmend(t, root, "amended", "--baseline", "--author", "@core")
	code, out := runAmend(t, root, "amended", "--list")
	if code != 0 || !strings.Contains(out, "[baseline] R1,R2 (@core)") || !strings.Contains(out, "amend.unacknowledged=0") {
		t.Fatalf("list output: code=%d out=%s", code, out)
	}
}
