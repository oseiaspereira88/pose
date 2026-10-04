package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-governed-effect-enforcement: close refuses with the structured
// cause, and closeout-check reports the same code the domain does.
func TestCloseRefusesWhileAnAdoptedRequestRestrictsCloseout(t *testing.T) {
	root := actionCLIFixture(t)
	mustWrite(t, filepath.Join(root, ".pose/policy/review.json"), `{"schema_version":2,"enabled":false,"profiles":{"spec":"spec-closeout@1"},"agency_readiness_version":1}`)
	args := append([]string{}, openArgs...)
	for i, a := range args {
		if a == "execution:block" {
			args[i] = "closeout:block"
		}
	}
	if code, out := runPose(t, root, append(args, "--apply")...); code != 0 {
		t.Fatalf("open: %s", out)
	}
	code, out := runPose(t, root, "close", "spec:storage")
	if code == 0 || !strings.Contains(out, "restricts closeout") {
		t.Fatalf("close was not refused by the request: %d %s", code, out)
	}
	_, check := runPose(t, root, "closeout-check", "spec:storage", "--json")
	var state posemodel.CloseoutState
	if err := json.Unmarshal([]byte(check), &state); err != nil {
		t.Fatalf("%v %s", err, check)
	}
	found := false
	for _, d := range state.Diagnostics {
		found = found || d.Code == "action-request-pending"
	}
	if !found {
		t.Fatalf("closeout-check lacks the typed cause: %+v", state.Diagnostics)
	}
	_, governance := runPose(t, root, "state", "--governance")
	if !strings.Contains(governance, "governance.agency-readiness=kind=capability supported=true configured=true applicable=true effective=true") {
		t.Fatalf("effective governance does not show the capability:\n%s", governance)
	}
}
