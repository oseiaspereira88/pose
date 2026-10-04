package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// Spec pose-adaptive-assessment-freshness R4: no workflow, skill or AGENTS
// template still tells an agent to rescan every component unconditionally.
func TestNoWorkflowAlwaysRunsDiscovery(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	unconditional := regexp.MustCompile("pose assess discover(?: --component <dir>| \\[--component <dir>\\]| --update-state)")
	paths := []string{"AGENTS.md", "locales/pt-BR/AGENTS.md",
		".agents/skills/pose-feature/SKILL.md", ".agents/skills/pose-spec-closeout/SKILL.md",
		"locales/pt-BR/.agents/skills/pose-feature/SKILL.md", "locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md"}
	for _, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		if loc := unconditional.FindIndex(raw); loc != nil {
			t.Errorf("%s still runs discovery unconditionally: %q", rel, string(raw[loc[0]:loc[1]]))
		}
	}
}
