package version_test

import (
	"os"
	"strings"
	"testing"
)

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
