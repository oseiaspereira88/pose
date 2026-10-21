package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"

	pose "github.com/harne8/pose-mcp/internal/pose"
)

func TestActionTargetCLIExposesStructuredScopeAndDomainSelection(t *testing.T) {
	root := actionCLIFixture(t)
	mustWrite(t, filepath.Join(root, ".pose/specs/beta.md"), "---\nslug: beta\nstatus: draft\n---\n## 2. Requirements\n- R4: Beta material.\n")
	args := append([]string{}, openArgs...)
	// Remove the legacy whole-source effect, keeping the rest of the real CLI request.
	args = args[:len(args)-4]
	args = append(args, "--effect-json", `{"phase":"start","mode":"block","scope":["xref:proj.storage-test/spec:beta"]}`, "--apply", "--json")
	code, out := runPose(t, root, args...)
	if code != 0 {
		t.Fatalf("structured open: %s", out)
	}
	var view pose.ActionRequestView
	if e := json.Unmarshal([]byte(out), &view); e != nil {
		t.Fatal(e)
	}
	if len(view.Request.Effects) != 1 || len(view.Request.Effects[0].Scope) != 1 {
		t.Fatalf("scope lost: %+v", view)
	}
	s := pose.Store{Root: root}
	for _, tc := range []struct {
		slug string
		want int
	}{{"storage", 0}, {"beta", 1}} {
		got, e := s.GovernedEffectRestrictions(pose.PhaseStart, []string{tc.slug})
		if e != nil || len(got) != tc.want {
			t.Fatalf("%s: %+v %v", tc.slug, got, e)
		}
	}
	args[len(args)-3] = `{"phase":"start","mode":"block","scope":["xref:proj.storage-test/spec:missing"]}`
	if code, out := runPose(t, root, args...); code == 0 {
		t.Fatalf("missing destination accepted: %s", out)
	}
}
