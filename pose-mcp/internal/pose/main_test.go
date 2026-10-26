package pose

import (
	"os"
	"testing"

	"github.com/harne8/pose-mcp/internal/testgit"
)

// TestMain keeps git's detached automatic maintenance out of the repositories
// these tests create (spec test-git-repos-run-no-background-maintenance).
//
// It also drops the session variables of the agent hosts: a bundle sealed by a
// test would otherwise record whichever agent ran the suite as its
// implementation (spec pose-delegation-routing R2).
func TestMain(m *testing.M) {
	for _, name := range DelegationSessionVariables() {
		_ = os.Unsetenv(name)
	}
	cleanup, err := testgit.Isolate()
	if err != nil {
		panic(err)
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}
