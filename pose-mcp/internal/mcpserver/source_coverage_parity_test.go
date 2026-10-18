package mcpserver

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-source-coverage-integrity V1: the CLI, the MCP tool and Attention see the
// same unresolved set and the same loss for one repository.
func TestFollowupParityAcrossCLIMCPAndAttention(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "pose")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "../../cmd/pose")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the native binary: %v\n%s", err, out)
	}
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/project.json", `{"schema_version":1,"project_id":"proj.parity"}`)
	write(".pose/policy/review.json", `{"schema_version":2,"enabled":false}`)
	write(".pose/specs/2026-10-01-late.md", "---\nslug: late\nstatus: done\ncreated_at: 2026-10-01\n---\n\n# Spec: late\n\n## 7. Final Report\n\n### Follow-ups\n\n- Missing mandatory correction (owner:@core crit:high review:2020-01-01)\n- [open] Confirmed regression (owner:@core crit:high review:2020-01-01)\n")
	if err := os.Symlink("absent-target.md", filepath.Join(root, ".pose", "specs", "2026-10-11-unreadable.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=f", "-c", "user.email=f@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "fixture"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	t.Setenv("POSE_EXECUTABLE", binary)
	t.Setenv("POSE_USAGE_DISABLED", "1")
	t.Setenv("POSE_FOLLOWUP_TODAY", "2026-10-12")

	cli := exec.Command(binary, "followups", "--open", "--json")
	cli.Dir = root
	cliOut, err := cli.Output()
	if err != nil {
		t.Fatalf("pose followups: %v", err)
	}
	var fromCLI map[string]any
	if err := json.Unmarshal(cliOut, &fromCLI); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(New(posemodel.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":201,"method":"tools/call","params":{"name":"pose_get_followups","arguments":{}}}`)
	if out.Error != nil || out.Result["isError"] != false {
		t.Fatalf("pose_get_followups: error=%+v result=%v", out.Error, out.Result)
	}
	fromMCP := out.Result["structuredContent"].(map[string]any)
	for _, key := range []string{"open", "overdue", "coverage_complete"} {
		if fromMCP[key] != fromCLI[key] {
			t.Fatalf("%s differs: mcp=%v cli=%v", key, fromMCP[key], fromCLI[key])
		}
	}
	if fromMCP["coverage_complete"] != false || fromMCP["open"].(float64) != 2 {
		t.Fatalf("the tool hides the loss or miscounts: %v", fromMCP)
	}
	losses := fromMCP["coverage"].(map[string]any)["losses"].([]any)
	if len(losses) != 1 || losses[0].(map[string]any)["artifact"] != ".pose/specs/2026-10-11-unreadable.md" {
		t.Fatalf("losses = %v", losses)
	}

	_, att := post(t, ts, `{"jsonrpc":"2.0","id":202,"method":"tools/call","params":{"name":"pose_obligations","arguments":{}}}`)
	if att.Error != nil || att.Result["isError"] != false {
		t.Fatalf("pose_obligations: error=%+v result=%v", att.Error, att.Result)
	}
	report := att.Result["structuredContent"].(map[string]any)
	projected := 0
	for _, o := range report["obligations"].([]any) {
		if o.(map[string]any)["source"].(map[string]any)["producer"] == "followups" {
			projected++
		}
	}
	if projected != int(fromCLI["open"].(float64)) {
		t.Fatalf("Attention holds %d unresolved follow-ups, the backlog %v", projected, fromCLI["open"])
	}
	incomplete := false
	for _, c := range report["coverage"].([]any) {
		cov := c.(map[string]any)
		if cov["producer"] == "followups" && cov["state"] == "incomplete" && strings.Contains(cov["detail"].(string), "unreadable") {
			incomplete = true
		}
	}
	if !incomplete || report["complete"] != false {
		t.Fatalf("Attention does not say its follow-up coverage is incomplete: complete=%v coverage=%v", report["complete"], report["coverage"])
	}
}
