package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFollowupCandidatesAreReachableAndWriteNothing(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-01-target.md"), "---\nslug: target-spec\nstatus: done\n---\n\n# Spec: target\n")
	mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-02-origin.md"), "---\nslug: origin\nstatus: done\n---\n\n# Spec: origin\n\n## 7. Final Report\n\n### Follow-ups\n\n- [open] Fold into `target-spec` (owner:@core crit:low review:2099-01-01)\n")
	code, out := runPose(t, root, "followups", "--candidates")
	if code != 0 || !strings.Contains(out, "none was dispositioned") || !strings.Contains(out, "spec:target-spec is done") || !strings.Contains(out, "similarity is not equivalence") {
		t.Fatalf("candidates: %d %s", code, out)
	}
}
