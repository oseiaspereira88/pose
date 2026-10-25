package mcpserver

import (
	"encoding/json"
	pose "github.com/harne8/pose-mcp/internal/pose"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTerminalVerificationMCPPlanMatchesStateAndDomain(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing-review", true: "legacy"}[legacy], func(t *testing.T) {
			root := t.TempDir()
			created := "2026-10-13"
			if legacy {
				created = "2026-07-01"
			}
			files := map[string]string{
				".pose/specs/alpha.md":                     "---\nslug: alpha\nstatus: done\ncreated_at: " + created + "\ncompleted_at: 2026-10-24\n---\n## 2. Requirements\n- R1: preserve history.\n",
				".pose/policy/review.json":                 `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"}}`,
				".pose/review-profiles/spec-closeout.json": `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`,
			}
			for path, data := range files {
				full := filepath.Join(root, path)
				if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte(data), 0644); err != nil {
					t.Fatal(err)
				}
			}
			for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "fixture"}} {
				cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git: %v %s", err, out)
				}
			}
			ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
			t.Cleanup(ts.Close)
			_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_closeout_state","arguments":{"scope":"spec:alpha","include_plan":true}}}`)
			if out.Error != nil || out.Result["isError"] != false {
				t.Fatalf("tool: %+v", out)
			}
			raw, _ := json.Marshal(out.Result["structuredContent"])
			var view struct {
				pose.CloseoutState
				ExecutionPlan pose.CloseoutPlan `json:"execution_plan"`
			}
			if err := json.Unmarshal(raw, &view); err != nil {
				t.Fatal(err)
			}
			direct, err := (pose.Store{Root: root}).GetCloseoutState("spec:alpha")
			if err != nil {
				t.Fatal(err)
			}
			if view.Terminal != legacy || view.Terminal != view.ExecutionPlan.Terminal || view.Terminal != direct.Terminal || view.ExecutionPlan.Verification == nil {
				t.Fatalf("parity: %s", raw)
			}
			if _, err := os.Stat(filepath.Join(root, ".pose/review-bundles")); !os.IsNotExist(err) {
				t.Fatal("MCP query wrote bundle")
			}
		})
	}
}
