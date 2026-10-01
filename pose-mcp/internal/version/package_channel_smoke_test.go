package version_test

import (
	"os"
	"strings"
	"testing"
)

func TestPackageChannelVerificationIsManualOnly(t *testing.T) {
	raw, err := os.ReadFile("../../../.github/workflows/package-channels.yml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(raw)
	start := strings.Index(workflow, "\non:\n")
	end := strings.Index(workflow, "\npermissions:")
	if start < 0 || end <= start {
		t.Fatal("workflow trigger boundary missing")
	}
	triggers := workflow[start:end]
	if !strings.Contains(triggers, "\n  workflow_dispatch:") {
		t.Fatal("explicit manual dispatch is required")
	}
	for _, automatic := range []string{"\n  release:", "\n  workflow_run:", "\n  push:", "\n  pull_request:", "\n  schedule:", "\n  workflow_call:"} {
		if strings.Contains(triggers, automatic) {
			t.Errorf("deferred native round has an automatic trigger: %s", automatic)
		}
	}
	for _, required := range []string{"macos-latest", "windows-latest", "RAW_REF: ${{ inputs.tag }}"} {
		if !strings.Contains(workflow, required) {
			t.Errorf("manual native verification is missing %q", required)
		}
	}
	if strings.Contains(workflow, "continue-on-error:") {
		t.Fatal("native verification must retain real failure semantics")
	}
}

func TestPackageChannelSmokeUsesFreshInstance(t *testing.T) {
	raw, err := os.ReadFile("../../../.github/workflows/package-channels.yml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(raw)
	for _, tc := range []struct {
		name    string
		next    string
		init    string
		install string
		enter   string
	}{
		{"Install via Homebrew", "Install via WinGet", "git -C \"$smoke\" init -q", "pose install \"$smoke\" --skip-mcp", "(cd \"$smoke\" && pose doctor --json)"},
		{"Install via WinGet", "", "git -C $smoke init -q", "pose install $smoke --skip-mcp", "Push-Location $smoke"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := strings.Index(workflow, "- name: "+tc.name)
			if start < 0 {
				t.Fatalf("%s step missing", tc.name)
			}
			step := workflow[start:]
			if tc.next != "" {
				end := strings.Index(step, "- name: "+tc.next)
				if end < 0 {
					t.Fatalf("%s boundary missing", tc.next)
				}
				step = step[:end]
			}
			for _, required := range []string{tc.init, tc.install, tc.enter, "pose doctor --json"} {
				if !strings.Contains(step, required) {
					t.Errorf("%s does not use a fresh installed instance: missing %q", tc.name, required)
				}
			}
			if strings.Index(step, tc.install) > strings.Index(step, "pose doctor --json") {
				t.Errorf("%s runs doctor before installing into the fresh instance", tc.name)
			}
		})
	}
}
