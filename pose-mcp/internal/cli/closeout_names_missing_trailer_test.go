package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// unattributedEvidenceFixture is a spec whose validation ran at a commit no change
// set attributes to it, as after a commit without the POSE-Spec line.
func unattributedEvidenceFixture(t *testing.T) string {
	t.Helper()
	root := reviewBundleCLIFixture(t)
	path := filepath.Join(root, ".pose/indexes/delivery-integrity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := regexp.MustCompile(`"change_sets":\[.*\],`).ReplaceAllString(string(raw), `"change_sets":[],`)
	text = regexp.MustCompile(`("outcome":"pass",)"provenance_digest":"[^"]*"`).ReplaceAllString(text, `$1"git_head":"deadbeef"`)
	if !strings.Contains(text, `"git_head":"deadbeef"`) {
		t.Fatal("the fixture was not rewritten")
	}
	mustWrite(t, path, text)
	artifactGit(t, root, "add", "-A")
	artifactGit(t, root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "unattributed")
	return root
}

func TestCloseoutPlanNamesTheMissingTrailerOnce(t *testing.T) {
	root := unattributedEvidenceFixture(t)
	_, out := runPose(t, root, "close", "spec:bundle", "--plan")
	if !strings.Contains(out, "`POSE-Spec: bundle`") || !strings.Contains(out, "pose artifact-check --spec bundle") {
		t.Fatalf("the plan does not name the fix: %s", out)
	}
	if strings.Contains(out, "no passed structured validation evidence") {
		t.Fatalf("the plan reports a consequence as a second gap: %s", out)
	}
}
