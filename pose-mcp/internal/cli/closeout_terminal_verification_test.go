package cli

import (
	"bytes"
	"encoding/json"
	pose "github.com/harne8/pose-mcp/internal/pose"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTerminalVerificationDoneWithoutReviewIsBlockedAndReadonly(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	path := filepath.Join(root, ".pose/specs/bundle/spec.md")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	done := strings.Replace(string(before), "status: in-progress", "status: done", 1)
	mustWrite(t, path, done)
	for _, mode := range []string{"--apply", "--resume"} {
		code, raw := runPose(t, root, "close", "spec:bundle", mode, "--json")
		var result pose.OperationResult
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			t.Fatalf("decode %v %s", err, raw)
		}
		if code == 0 || result.Outcome != pose.OutcomeBlocked || len(result.Effects) > 0 {
			t.Fatalf("done was certified: %d %s", code, raw)
		}
	}
	got, _ := os.ReadFile(path)
	if string(got) != done {
		t.Fatal("done spec rewritten")
	}
	if countFiles(t, filepath.Join(root, ".pose/review-bundles")) != 0 || countFiles(t, filepath.Join(root, ".pose/closeout-plans")) != 0 {
		t.Fatal("repair fabricated writes")
	}
}

func TestTerminalVerificationValidNoopPreservesCheckpointAndHistory(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	if code, out := runPose(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer"); code != 0 {
		t.Fatalf("positive close %d %s", code, out)
	}
	path := filepath.Join(root, ".pose/specs/bundle/spec.md")
	before, _ := os.ReadFile(path)
	checkpoint := filepath.Join(root, ".pose/closeout-plans/spec-bundle.json")
	mustWrite(t, checkpoint, "{malformed old checkpoint\n")
	for _, mode := range []string{"--apply", "--resume"} {
		code, raw := runPose(t, root, "close", "spec:bundle", mode, "--json")
		var result pose.OperationResult
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			t.Fatal(err)
		}
		if code != 0 || result.Outcome != pose.OutcomeAlreadySatisfied || len(result.Effects) > 0 || len(result.StepsRun) > 0 || !strings.Contains(raw, `"terminal": true`) {
			t.Fatalf("positive noop %d %s", code, raw)
		}
	}
	if code, out := runPose(t, root, "close", "spec:bundle"); code != 0 || !strings.Contains(out, "no transition executed") {
		t.Fatalf("plain close %d %s", code, out)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("noop rewrote done spec")
	}
	raw, _ := os.ReadFile(checkpoint)
	if string(raw) != "{malformed old checkpoint\n" {
		t.Fatal("noop rewrote checkpoint")
	}
}

func TestTerminalVerificationNoopDoesNotConfirmTransitionEffect(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	if code, out := runPose(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer"); code != 0 {
		t.Fatalf("positive close %d %s", code, out)
	}
	var out, errOut bytes.Buffer
	transitioned := false
	if code := cmdCloseLocalWithEffect(root, "spec:bundle", nil, &out, &errOut, func() { transitioned = true }); code != 0 || transitioned {
		t.Fatalf("noop confirmed foreign effect %d %t %s%s", code, transitioned, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	result := pose.NewOperationResult("close.apply", "spec:bundle")
	result.Outcome = pose.OutcomeCompleted
	renderClosePlanResult(render(&out, &errOut), result)
	if strings.Contains(out.String(), "result=closed") || !strings.Contains(out.String(), "no lifecycle transition was executed by this call") {
		t.Fatalf("fabricated completion %s", out.String())
	}
}
