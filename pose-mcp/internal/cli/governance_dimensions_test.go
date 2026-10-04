package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-governance-wait-rework-observability: --waits selects its
// dimension without recomputing the outcomes.
func TestWaitsAloneDoesNotComputeOutcomes(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-04-x.md"), "---\nslug: x\nstatus: in-progress\n---\n\n# Spec: x\n")
	code, out := runPose(t, root, "stats", "governance", "--waits")
	if code != 0 || !strings.Contains(out, "waits.requests=") {
		t.Fatalf("waits: %d %s", code, out)
	}
	if strings.Contains(out, "coverage.history") || strings.Contains(out, "judgment=") {
		t.Fatalf("--waits printed the outcomes it should not compute: %s", out)
	}
	code, out = runPose(t, root, "stats", "governance", "--waits", "--outcomes")
	if code != 0 || !strings.Contains(out, "coverage.history") || !strings.Contains(out, "waits.requests=") {
		t.Fatalf("--outcomes did not add the outcomes: %d %s", code, out)
	}
}
