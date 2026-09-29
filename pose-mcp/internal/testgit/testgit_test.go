package testgit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// commitSpawnsMaintenance commits once in a fresh repository and reports
// whether git started its detached automatic maintenance, read from git's own
// trace2 event stream rather than inferred from timing.
func commitSpawnsMaintenance(t *testing.T) bool {
	t.Helper()
	dir := t.TempDir()
	trace := filepath.Join(t.TempDir(), "trace.json")
	run := func(env []string, args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.invalid"}, args...)...)
		cmd.Env = append(os.Environ(), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run(nil, "init", "-q")
	run([]string{"GIT_TRACE2_EVENT=" + trace}, "commit", "-q", "--allow-empty", "-m", "probe")
	raw, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Contains(string(raw), `"git","maintenance"`)
}

func TestIsolatedCommitStartsNoBackgroundMaintenance(t *testing.T) {
	t.Run("default configuration starts it", func(t *testing.T) {
		// The defect the isolation exists for: an empty global config, as on a
		// CI runner, leaves automatic maintenance on.
		t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
		if !commitSpawnsMaintenance(t) {
			t.Skip("this git does not run automatic maintenance after a commit")
		}
	})
	t.Run("isolated configuration does not", func(t *testing.T) {
		t.Setenv("GIT_CONFIG_GLOBAL", os.Getenv("GIT_CONFIG_GLOBAL"))
		cleanup, err := Isolate()
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup()
		if commitSpawnsMaintenance(t) {
			t.Fatal("a commit under Isolate still started git maintenance")
		}
	})
}
