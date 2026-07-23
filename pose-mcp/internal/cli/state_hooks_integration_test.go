package cli

// Integration tests proving each of the four real event interception
// points (spec pose-project-state-refresh-contract R1/R2) actually fires
// through the exact CLI command a human/agent runs — not just the
// synthetic HookEvent unit tests above.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const closeoutDoneSpec = `---
slug: closeout-demo
status: done
created_at: 2026-07-01
completed_at: 2026-07-02
---

## 1. Intent
Content.
## 2. Requirements
- R1: behave.
## 3. Technical Plan
Content.
## 4. Tasks
- [x] done
## 6. Validation
### Requirement trace
- R1 [satisfied] check:test
## 7. Final Report
Delivered.
`

func triggerLogEntries(t *testing.T, root, trigger string) []refreshLogEntry {
	t.Helper()
	log, err := readRefreshLog(stateRefreshLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	var out []refreshLogEntry
	for _, e := range log {
		if e.Trigger == trigger {
			out = append(out, e)
		}
	}
	return out
}

func TestIntegration_SpecCloseoutFiresHook(t *testing.T) {
	root := newGitRepo(t)
	writeStateTestFile(t, root, ".pose/specs/closeout-demo/spec.md", closeoutDoneSpec)
	if code, _, errOut := runState(t, root, "init"); code != 0 {
		t.Fatalf("state init: %v", errOut)
	}

	inDir(t, root, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"lint-spec", "closeout-demo", "--strict"}, &out, &errB); code != 0 {
			t.Fatalf("lint-spec: %s", out.String()+errB.String())
		}
	})

	entries := triggerLogEntries(t, root, "spec_closeout")
	if len(entries) != 1 || entries[0].Result != "ok" || entries[0].Target != "closeout-demo" {
		t.Fatalf("spec_closeout hook entries = %+v, want one ok entry targeting closeout-demo", entries)
	}
}

func TestIntegration_LintSpecTolerantOrAllNeverFiresCloseout(t *testing.T) {
	root := newGitRepo(t)
	writeStateTestFile(t, root, ".pose/specs/closeout-demo/spec.md", closeoutDoneSpec)
	if code, _, errOut := runState(t, root, "init"); code != 0 {
		t.Fatalf("state init: %v", errOut)
	}

	inDir(t, root, func() {
		var out, errB bytes.Buffer
		Main([]string{"lint-spec", "closeout-demo", "--tolerant"}, &out, &errB)
		Main([]string{"lint-spec", "--all", "--strict"}, &out, &errB)
	})

	if entries := triggerLogEntries(t, root, "spec_closeout"); len(entries) != 0 {
		t.Fatalf("--tolerant and --all must never fire spec_closeout (it is not a real closeout gate pass on one spec): %+v", entries)
	}
}

func TestIntegration_AmendFiresHook(t *testing.T) {
	root := t.TempDir() // cmdAmend falls back to cwd when not a git repo; no git needed
	writeStateTestFile(t, root, ".pose/specs/amend-demo/spec.md", "---\nslug: amend-demo\nstatus: draft\n---\n\n## 2. Requirements\n- R1: does a thing.\n")
	if code, _, errOut := runState(t, root, "init"); code != 0 {
		t.Fatalf("state init: %v", errOut)
	}

	inDir(t, root, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"amend", "amend-demo", "--baseline", "--author", "@tester"}, &out, &errB); code != 0 {
			t.Fatalf("amend: %s", out.String()+errB.String())
		}
	})

	entries := triggerLogEntries(t, root, "spec_amend")
	if len(entries) != 1 || entries[0].Result != "ok" || entries[0].Target != "amend-demo" {
		t.Fatalf("spec_amend hook entries = %+v, want one ok entry targeting amend-demo", entries)
	}
}

func TestIntegration_ReconcileEvidenceFiresHook(t *testing.T) {
	root := newGitRepo(t)
	if code, _, errOut := runState(t, root, "init"); code != 0 {
		t.Fatalf("state init: %v", errOut)
	}

	var out, errB bytes.Buffer
	if code := cmdReconcileEvidence(root, recordEvidenceArgs("r1", "req1", "e1", "d1", "success", "harness"), &out, &errB); code != 0 {
		t.Fatalf("reconcile-evidence record: %s", out.String()+errB.String())
	}

	entries := triggerLogEntries(t, root, "evidence_reconciled")
	if len(entries) != 1 || entries[0].Result != "ok" || entries[0].Target != "req1" {
		t.Fatalf("evidence_reconciled hook entries = %+v, want one ok entry targeting req1", entries)
	}
}

func TestIntegration_AssessSnapshotFiresHook(t *testing.T) {
	root := newGitRepo(t)
	if code, _, errOut := runState(t, root, "init"); code != 0 {
		t.Fatalf("state init: %v", errOut)
	}

	var out, errB bytes.Buffer
	if code := cmdAssess(root, []string{"init"}, &out, &errB); code != 0 {
		t.Fatalf("assess init: %s", out.String()+errB.String())
	}
	out.Reset()
	errB.Reset()
	if code := cmdAssess(root, []string{"snapshot"}, &out, &errB); code != 0 {
		t.Fatalf("assess snapshot: %s", out.String()+errB.String())
	}

	entries := triggerLogEntries(t, root, "assessment_snapshot")
	if len(entries) != 1 || entries[0].Result != "ok" {
		t.Fatalf("assessment_snapshot hook entries = %+v, want one ok entry", entries)
	}
}

func TestIntegration_NoProjectStateHooksStayNoOp(t *testing.T) {
	root := newGitRepo(t) // deliberately no `pose state init`
	writeStateTestFile(t, root, ".pose/specs/amend-demo/spec.md", "---\nslug: amend-demo\nstatus: draft\n---\n\n## 2. Requirements\n- R1: does a thing.\n")

	inDir(t, root, func() {
		var out, errB bytes.Buffer
		if code := Main([]string{"amend", "amend-demo", "--baseline", "--author", "@tester"}, &out, &errB); code != 0 {
			t.Fatalf("amend: %s", out.String()+errB.String())
		}
	})
	if strings.TrimSpace(readIfExists(t, stateRefreshLogPath(root))) != "" {
		t.Fatal("no .pose/state/ means the hook must be a true no-op, not even an empty log file")
	}
}

// TestIntegration_StrictModePropagatesThroughRealCommand locks in a real
// bug found during closeout review: EmitHook originally discarded every
// consumer error unconditionally, so strict_refresh's "the triggering
// command fails" contract (R5) was silently unreachable in production even
// though the lower-level stateRefreshConsumer unit test already proved the
// function itself returns an error — the CLI command never saw it.
func TestIntegration_StrictModePropagatesThroughRealCommand(t *testing.T) {
	root := newGitRepo(t)
	if code, _, errOut := runState(t, root, "init"); code != 0 {
		t.Fatalf("state init: %v", errOut)
	}
	if err := os.MkdirAll(filepath.Join(root, ".pose", "policy"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "policy", "state.json"), []byte(`{"strict_refresh":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	blockNewEntriesInStateDir(t, root)

	var out, errB bytes.Buffer
	code := cmdReconcileEvidence(root, recordEvidenceArgs("r1", "req1", "e1", "d1", "success", "harness"), &out, &errB)
	if code == 0 {
		t.Fatal("strict_refresh must fail the triggering command when the automatic refresh fails, not just log it")
	}
	if !strings.Contains(errB.String(), "state-refresh") {
		t.Errorf("stderr should explain the state-refresh failure, got: %s", errB.String())
	}
}

func readIfExists(t *testing.T, path string) string {
	t.Helper()
	raw, err := readRefreshLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		return ""
	}
	return "non-empty"
}
