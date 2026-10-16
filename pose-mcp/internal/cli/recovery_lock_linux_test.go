//go:build linux

package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// deadLockOwner writes the lock a crashed start leaves: the owner this
// process would record, with the PID of a process that has finished.
func deadLockOwner(t *testing.T, root string) {
	t.Helper()
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	owner := posemodel.CurrentStartLockOwner(root, "spec:work", "start.apply")
	owner.PID, owner.ProcessStart = cmd.Process.Pid, "1"
	raw, _ := json.Marshal(owner)
	path := filepath.Join(root, ".pose", "starts", ".work.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestADeadStartLockIsRecoveredByItsDigestInBothRenderers(t *testing.T) {
	for _, jsonOutput := range []bool{true, false} {
		root := startCLIFixture(t, true)
		var plan posemodel.StartPlan
		if code, _ := runPoseJSON(t, root, &plan, "start", "spec:work", "--json"); code != 0 {
			t.Fatal("preview")
		}
		deadLockOwner(t, root)
		var blocked struct {
			posemodel.OperationResult
			Data posemodel.StartLockError `json:"data"`
		}
		code, _ := runPoseJSON(t, root, &blocked, "start", "spec:work", "--apply", "--digest", plan.Digest, "--json")
		if code != 1 || blocked.Error.Cause != "start-lock-orphaned" || blocked.Data.State != posemodel.LockOwnerDead || !blocked.Resumable ||
			!strings.Contains(blocked.NextAction, "--recover-lock "+blocked.Data.OwnerDigest) {
			t.Fatalf("a dead lock: %d %+v", code, blocked)
		}
		if jsonOutput {
			var recovered posemodel.OperationResult
			if code, _ := runPoseJSON(t, root, &recovered, "start", "spec:work", "--recover-lock", blocked.Data.OwnerDigest, "--json"); code != 0 ||
				recovered.Outcome != posemodel.OutcomeCompleted || len(recovered.Effects) != 1 || recovered.Effects[0].Effect != "lock-removed" {
				t.Fatalf("recovery: %d %+v", code, recovered)
			}
		} else if code, human := runPose(t, root, "start", "spec:work", "--recover-lock", blocked.Data.OwnerDigest); code != 0 || !strings.Contains(human, "start.lock_recovered=.pose/starts/.work.lock") {
			t.Fatalf("human recovery: %d %s", code, human)
		}
		if code, out := runPose(t, root, "start", "spec:work", "--apply", "--digest", plan.Digest); code != 0 || !strings.Contains(out, "start.phase=started") {
			t.Fatalf("the start after recovery: %d %s", code, out)
		}
	}
}
