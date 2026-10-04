package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Specs pose-action-requests and pose-action-request-resolution, through Main.

func actionCLIFixture(t *testing.T) string {
	t.Helper()
	// A declared identity, as an installed project has; the derived-identity
	// warning has its own test.
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.storage-test")
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".pose/specs/2026-10-04-storage.md"), "---\nslug: storage\nstatus: in-progress\n---\n\n# Spec: storage\n\n## 2. Requirements\n\n- R4: Keep reading schema v1 records.\n")
	mustWrite(t, filepath.Join(root, ".pose/policy/actions.json"), `{"schema_version":1,"roles":{"maintainer":["human:maintainer"]}}`)
	return root
}

func runPose(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	var out, errB bytes.Buffer
	code := 0
	inDir(t, root, func() { code = Main(args, &out, &errB) })
	return code, out.String() + errB.String()
}

var openArgs = []string{"action", "open", "--origin", "spec:storage", "--kind", "decision", "--question", "Keep reading schema v1?",
	"--option", "preserve-v1=Keep the v1 reader", "--option", "break-v1=Consumers migrate", "--recommend", "preserve-v1",
	"--recipient-role", "maintainer", "--requested-by", "agent:impl", "--target", "requirement:R4",
	"--effect", "execution:block", "--effect", "closeout:block"}

func TestActionOpenPreviewsThenRecordsAndShowsTheBoundContent(t *testing.T) {
	root := actionCLIFixture(t)
	code, out := runPose(t, root, openArgs...)
	if code != 0 || !strings.Contains(out, "action.preview=true") {
		t.Fatalf("preview: %d %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/actions")); err == nil {
		t.Fatal("a preview wrote a journal")
	}
	code, out = runPose(t, root, append(append([]string{}, openArgs...), "--apply", "--json")...)
	if code != 0 {
		t.Fatalf("apply: %s", out)
	}
	var view posemodel.ActionRequestView
	if err := json.Unmarshal([]byte(out), &view); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	_, shown := runPose(t, root, "action", "show", view.Request.ID)
	for _, want := range []string{"action.question=Keep reading schema v1?", "action.option.preserve-v1=Keep the v1 reader (recommended)", "action.option.break-v1=Consumers migrate", "action.recipient=role:maintainer", "action.request_digest=" + view.Request.RequestDigest, "action.effect=closeout/block"} {
		if !strings.Contains(shown, want) {
			t.Errorf("show lacks %q:\n%s", want, shown)
		}
	}
	_, attention := runPose(t, root, "state", "--attention")
	if !strings.Contains(attention, view.Request.ID) || !strings.Contains(attention, "role:maintainer") {
		t.Fatalf("Attention does not show the request:\n%s", attention)
	}
}

func TestActionResolveRefusesTheWrongActorAndRecordsTheRightOne(t *testing.T) {
	root := actionCLIFixture(t)
	_, out := runPose(t, root, append(append([]string{}, openArgs...), "--apply", "--json")...)
	var view posemodel.ActionRequestView
	_ = json.Unmarshal([]byte(out), &view)
	resolve := func(actor, key string) (int, string) {
		return runPose(t, root, "action", "resolve", view.Request.ID, "--actor", actor, "--answer", "preserve-v1", "--request-digest", view.Request.RequestDigest,
			"--expected-revision", strconv.Itoa(view.Revision), "--idempotency-key", key, "--apply")
	}
	if code, out := resolve("agent:impl", "k1"); code == 0 || !strings.Contains(out, "action-actor-not-authorized") {
		t.Fatalf("the requester answered its own decision: %d %s", code, out)
	}
	if code, out := runPose(t, root, "action", "resolve", view.Request.ID, "--actor", "human:maintainer", "--answer", "preserve-v1", "--request-digest", view.Request.RequestDigest, "--idempotency-key", "k2", "--apply"); code == 0 || !strings.Contains(out, "--expected-revision is required") {
		t.Fatalf("a resolution without a revision: %d %s", code, out)
	}
	code, out := resolve("human:maintainer", "k3")
	if code != 0 || !strings.Contains(out, "action.answer=preserve-v1 by human:maintainer (declared)") || !strings.Contains(out, "declared, not verified") {
		t.Fatalf("the authorized answer: %d %s", code, out)
	}
	_, listed := runPose(t, root, "action", "list", "--json")
	var views []posemodel.ActionRequestView
	if err := json.Unmarshal([]byte(listed), &views); err != nil || len(views) != 1 || views[0].Satisfaction != posemodel.SatisfactionSatisfied {
		t.Fatalf("list: %v %s", err, listed)
	}
}

// Found by the agency-readiness pilot rehearsal: a request qualified with a
// directory-derived project id must say so.
func TestActionOpenDisclosesADirectoryDerivedIdentity(t *testing.T) {
	root := actionCLIFixture(t)
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	var out, errB bytes.Buffer
	code := 0
	inDir(t, root, func() { code = Main(append(append([]string{}, openArgs...), "--apply", "--json"), &out, &errB) })
	if code != 0 {
		t.Fatalf("open: %s %s", out.String(), errB.String())
	}
	var view posemodel.ActionRequestView
	if err := json.Unmarshal(out.Bytes(), &view); err != nil {
		t.Fatalf("the warning broke the JSON on stdout: %v", err)
	}
	if !strings.Contains(errB.String(), "derived from the directory name") {
		t.Fatalf("no identity warning on stderr: %q", errB.String())
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.storage-test")
	errB.Reset()
	out.Reset()
	args := append([]string{}, openArgs...)
	args[7] = "Another question?"
	inDir(t, root, func() { code = Main(append(args, "--apply"), &out, &errB) })
	if code != 0 {
		t.Fatalf("second open: %s %s", out.String(), errB.String())
	}
	if strings.Contains(out.String()+errB.String(), "derived from the directory name") {
		t.Fatal("a declared identity still produced the warning")
	}
}
