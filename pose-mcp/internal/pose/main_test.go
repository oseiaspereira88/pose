package pose

import (
	"os"
	"testing"

	"github.com/harne8/pose-mcp/internal/testgit"
)

// TestMain keeps git's detached automatic maintenance out of the repositories
// these tests create (spec test-git-repos-run-no-background-maintenance).
func TestMain(m *testing.M) {
	cleanup, err := testgit.Isolate()
	if err != nil {
		panic(err)
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}
