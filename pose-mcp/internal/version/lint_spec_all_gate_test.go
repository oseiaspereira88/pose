package version

// `pose lint-spec --all` exited 1 on this repository for weeks and nobody
// noticed, because nothing ran it; the first time anyone did, it failed for a
// reason unrelated to their change. On 2026-09-29 one closeout batch
// introduced 38 new failures in it without any gate reporting them. CI now
// runs it (spec pose-lint-spec-all-is-a-gate, Decision 1, option A), and this
// pins the step so removing it fails too.

import (
	"os"
	"strings"
	"testing"
)

func TestCIGovernanceJobRunsLintSpecAll(t *testing.T) {
	raw, err := os.ReadFile("../../../.github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range jobBlocks(string(raw))["governance"] {
		if strings.Contains(line, "lint-spec --all") {
			return
		}
	}
	t.Fatal("the governance job does not run `pose lint-spec --all`; a spec that breaks the template would merge unseen")
}
