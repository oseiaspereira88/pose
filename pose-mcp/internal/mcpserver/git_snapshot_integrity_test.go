package mcpserver

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestGitSnapshotMCPPlanPreservesUnknownAndHiddenChanges(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose/specs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose/specs/alpha.md"), []byte("---\nslug: alpha\nstatus: in-progress\n---\n# Alpha\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "input.go"), []byte("package input\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_closeout_state","arguments":{"scope":"spec:alpha","include_plan":true}}}`
	_, unknown := post(t, ts, request)
	raw, err := json.Marshal(unknown)
	if err != nil || !strings.Contains(string(raw), "Git snapshot unavailable") {
		t.Fatalf("unknown lost: %s %v", raw, err)
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-qm", "fixture")
	_, control := post(t, ts, request)
	sc, _ := control.Result["structuredContent"].(map[string]any)
	plan, _ := sc["execution_plan"].(map[string]any)
	direct, err := (pose.Store{Root: root}).PlanCloseout("spec:alpha")
	if err != nil || plan["digest"] != direct.Digest {
		t.Fatalf("domain parity: %+v %+v %v", plan, direct, err)
	}
	git("update-index", "--skip-worktree", "input.go")
	if err := os.WriteFile(filepath.Join(root, "input.go"), []byte("package changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, hidden := post(t, ts, request)
	raw, _ = json.Marshal(hidden)
	if !strings.Contains(string(raw), "input.go") || !strings.Contains(string(raw), "blocked") {
		t.Fatalf("hidden edit lost: %s", raw)
	}
}
