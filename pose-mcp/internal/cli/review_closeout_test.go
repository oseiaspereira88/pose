package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func writeCloseoutCLIFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func closeoutCLIFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"},"continuous_closeout":true}`)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
	writeCloseoutCLIFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\ncompleted_at:\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: works\n")
	return root
}

func TestReviewRecordIsDryRunByDefaultAndCloseIsReviewGated(t *testing.T) {
	root := closeoutCLIFixture(t)
	var out, errOut bytes.Buffer
	args := []string{"spec:alpha", "--reviewer", "agent:review-pass", "--decision", "approved", "--evidence", "check:unit"}
	if code := cmdReviewRecord(root, args, &out, &errOut); code != 0 {
		t.Fatalf("dry run code=%d err=%s", code, errOut.String())
	}
	if entries, _ := os.ReadDir(filepath.Join(root, ".pose", "reviews")); len(entries) != 0 {
		t.Fatalf("dry run wrote reviews: %v", entries)
	}
	out.Reset()
	args = append(args, "--apply")
	if code := cmdReviewRecord(root, args, &out, &errOut); code != 0 {
		t.Fatalf("apply code=%d err=%s", code, errOut.String())
	}
	state, err := (posemodel.Store{Root: root}).GetCloseoutState("spec:alpha")
	if err != nil || !state.Review.Approved || state.Terminal {
		t.Fatalf("unexpected pre-close state=%+v err=%v", state, err)
	}
	out.Reset()
	if code := cmdClose(root, []string{"spec:alpha"}, &out, &errOut); code != 0 {
		t.Fatalf("close code=%d err=%s", code, errOut.String())
	}
	state, err = (posemodel.Store{Root: root}).GetCloseoutState("spec:alpha")
	if err != nil || !state.Terminal {
		t.Fatalf("unexpected terminal state=%+v err=%v", state, err)
	}
}

func TestCloseoutCheckJSONUsesTheSameProjection(t *testing.T) {
	root := closeoutCLIFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdCloseoutCheck(root, []string{"spec:alpha", "--json"}, &out, &errOut); code != 1 {
		t.Fatalf("expected blocked code, got %d", code)
	}
	var state posemodel.CloseoutState
	if err := json.Unmarshal(out.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state.Scope != "spec:alpha" || state.Terminal || !strings.Contains(state.NextAction, "review") {
		t.Fatalf("unexpected JSON projection: %+v", state)
	}
}

func TestContinuousCloseoutPersistsTerminalScopeAndRefusesEarlyCompletion(t *testing.T) {
	root := closeoutCLIFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdContinuousCloseout(root, []string{"start", "spec:alpha", "--apply"}, &out, &errOut); code != 0 {
		t.Fatalf("start code=%d err=%s", code, errOut.String())
	}
	out.Reset()
	if code := cmdContinuousCloseout(root, []string{"status"}, &out, &errOut); code != 1 || !strings.Contains(out.String(), "continuous.next_action=record or remediate") {
		t.Fatalf("status code/output mismatch: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	if code := cmdContinuousCloseout(root, []string{"complete", "--apply"}, &out, &errOut); code != 1 {
		t.Fatalf("early complete code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "continuous-closeout.json")); err != nil {
		t.Fatalf("early completion removed selection: %v", err)
	}
}
