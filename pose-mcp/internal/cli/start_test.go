package cli

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

func startCLIFixture(t *testing.T, adopted bool) string {
	t.Helper()
	root := t.TempDir()
	policy := `{"schema_version":1,"enabled":false}`
	if adopted {
		policy = `{"schema_version":1,"enabled":false,"atomic_start_version":1}`
	}
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"), policy)
	mustWrite(t, filepath.Join(root, ".pose", "specs", "2026-09-27-work.md"), "---\nslug: work\nstatus: draft\ncreated_at: 2026-09-27\n---\n\n# Spec: work\n\n## 2. Requirements\n\n- R1: Record the baseline.\n")
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.name", "POSE start"}, {"config", "user.email", "start@example.invalid"}, {"add", "--all"}, {"commit", "-q", "-m", "fixture"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	t.Setenv("POSE_PROJECT_ROOTS", "")
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	return root
}

func TestABMAtomicStartCLIPreviewApplyStatus(t *testing.T) {
	root := startCLIFixture(t, true)
	out, errOut, code := runCLI(t, root, "start", "spec:work", "--json")
	if code != 0 {
		t.Fatalf("preview failed: %s", errOut)
	}
	var plan posepkg.StartPlan
	if err := json.Unmarshal([]byte(out), &plan); err != nil || !plan.Ready || plan.Digest == "" {
		t.Fatalf("preview plan = %+v err=%v", plan, err)
	}
	if _, errOut, code := runCLI(t, root, "start", "spec:work", "--apply"); code == 0 || !strings.Contains(errOut, "--digest") {
		t.Fatalf("apply without a digest was accepted: %s", errOut)
	}
	if out, errOut, code := runCLI(t, root, "start", "spec:work", "--apply", "--digest", plan.Digest); code != 0 || !strings.Contains(out, "start.phase=started") {
		t.Fatalf("apply failed: code=%d %s%s", code, out, errOut)
	}
	if out, errOut, code := runCLI(t, root, "start", "spec:work", "--apply", "--digest", plan.Digest); code != 0 || !strings.Contains(out, "start.phase=started") {
		t.Fatalf("repeating apply was not a no-op: code=%d %s%s", code, out, errOut)
	}
	out, _, _ = runCLI(t, root, "start", "spec:work", "--status")
	if !strings.Contains(out, "start.node.R1=recorded-before-managed-execution") {
		t.Fatalf("status did not classify the node:\n%s", out)
	}
	if _, errOut, code := runCLI(t, root, "start", "spec:work", "--cancel"); code == 0 || !strings.Contains(errOut, "start-already-transitioned") {
		t.Fatalf("a completed start was cancelled: %s", errOut)
	}
}

func TestABMAtomicStartCLIApplyNeedsCapability(t *testing.T) {
	root := startCLIFixture(t, false)
	out, _, _ := runCLI(t, root, "start", "spec:work", "--json")
	var plan posepkg.StartPlan
	_ = json.Unmarshal([]byte(out), &plan)
	if _, errOut, code := runCLI(t, root, "start", "spec:work", "--apply", "--digest", plan.Digest); code == 0 || !strings.Contains(errOut, "atomic-start-capability-not-adopted") {
		t.Fatalf("apply ran without the capability: %s", errOut)
	}
	if _, errOut, code := runCLI(t, root, "start", "work"); code != 2 || !strings.Contains(errOut, "spec:<slug>") {
		t.Fatalf("an untyped scope was accepted: %s", errOut)
	}
}
