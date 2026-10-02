package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoryQuietPrintsOneToleratedOrStrictVerdict(t *testing.T) {
	for _, modified := range []bool{false, true} {
		root := doctorTrailerFixture(t)
		path := filepath.Join(root, ".pose/reports/history/fixture.jsonl")
		mustWrite(t, path, "{}\n")
		if modified {
			artifactGit(t, root, "add", ".")
			artifactGit(t, root, "commit", "-qm", "history baseline")
			mustWrite(t, path, "{}\n{}\n")
		}
		for _, mode := range []string{"--tolerant", "--strict"} {
			out, errOut, code := runCLI(t, root, "history-check", mode, "--quiet")
			want := 0
			if mode == "--strict" {
				want = 1
			}
			if code != want || len(strings.Split(strings.TrimSpace(out), "\n")) != 1 || strings.TrimSpace(out) == "" {
				t.Fatalf("modified=%t mode=%s exit=%d output=%q error=%q", modified, mode, code, out, errOut)
			}
		}
	}
}
