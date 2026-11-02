package cli

import (
	"os/exec"
	"strings"
	"testing"
)

func TestReleaseCheckRefusesAReplacedTagTree(t *testing.T) {
	root := cliReleaseRepo(t)
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	good := git("rev-parse", "HEAD")
	git("rm", "-q", "-r", ".pose")
	git("commit", "-q", "-m", "no snapshot")
	bad := git("rev-parse", "HEAD")
	git("tag", "v1.0.0")
	git("replace", bad, good)
	_, out := runPose(t, root, "release", "check", "--version", "v1.0.0", "--json")
	if !strings.Contains(out, "snapshot") {
		t.Fatalf("a replace ref supplied the snapshot of a tag: %s", out)
	}
}
