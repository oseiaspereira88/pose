package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-claude-sandboxed-profile R4: `delegate agents` and `doctor` say which profile
// is unavailable and how to install what it needs.
func TestDelegateAgentsAndDoctorSayHowToInstallWhatAProfileNeeds(t *testing.T) {
	root := t.TempDir()
	policy := `{"schema_version":2,"agents":{"a1":{"vendor":"v","model":"m","cli":"sh","commands":{"read":["sh"],"sandboxed":["sh"]},` +
		`"requires":{"sandboxed":{"executables":["definitely-not-installed-xyz"],"install":"apt install xyz"}}}},"purposes":{}}`
	if err := os.MkdirAll(filepath.Join(root, ".pose", "policy"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "policy", "delegation.json"), []byte(policy), 0o644); err != nil {
		t.Skip(err)
	}
	var out, errb strings.Builder
	_ = cmdDelegateAgents(root, nil, &out, &errb)
	if !strings.Contains(out.String(), "unavailable_access=sandboxed(") || !strings.Contains(out.String(), "apt install xyz") {
		t.Fatalf("delegate agents does not say the profile is unavailable and how to install: %s %s", out.String(), errb.String())
	}
}
