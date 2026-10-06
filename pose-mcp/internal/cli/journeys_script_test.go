package cli

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Spec pose-install-and-upgrade-journeys: the journeys download the latest
// published release, so they run when POSE_JOURNEYS=1 (the validation matrix
// sets it; CI also runs the script directly).
func TestJourneysInstallAndUpgradeFromThePublishedRelease(t *testing.T) {
	if os.Getenv("POSE_JOURNEYS") != "1" {
		t.Skip("set POSE_JOURNEYS=1: the upgrade journey downloads the latest published release")
	}
	for _, tool := range []string{"bash", "python3", "ssh-keygen", "go", "curl", "sha256sum"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatalf("%s is required for the journeys", tool)
		}
	}
	out, err := exec.Command("bash", "../../../tests/journeys/install-and-upgrade.sh").CombinedOutput()
	if err != nil {
		t.Fatalf("a journey diverged: %v\n%s", err, out)
	}
	for _, journey := range []string{"journey 1:", "journey 2a:", "journey 2b:"} {
		if !strings.Contains(string(out), journey) {
			t.Fatalf("journey %s did not run\n%s", journey, out)
		}
	}
}
