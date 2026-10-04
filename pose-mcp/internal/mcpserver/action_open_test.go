package mcpserver

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-action-requests R1: the MCP equivalent of `pose action open`
// previews by default and records through the same domain function.
func TestToolsCall_ActionOpen_PreviewsThenRecords(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, ".pose/specs/2026-10-04-storage.md")
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte("---\nslug: storage\nstatus: in-progress\n---\n\n# Spec: storage\n\n## 2. Requirements\n\n- R4: Keep reading v1.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	args := `"origin":"spec:storage","kind":"decision","question":"Keep reading v1?","requested_by":"agent:impl","recipient_role":"maintainer",` +
		`"options":{"preserve-v1":"Keep the reader","break-v1":"Consumers migrate"},"recommend":"preserve-v1","targets":["requirement:R4"],"effects":["closeout:block"]`
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_action_open","arguments":{`+args+`}}}`)
	sc, _ := out.Result["structuredContent"].(map[string]any)
	if sc["preview"] != true {
		t.Fatalf("not a preview: %+v", out)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/actions")); err == nil {
		t.Fatal("the preview wrote a journal")
	}
	preview, _ := sc["request"].(map[string]any)
	_, out = post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_action_open","arguments":{`+args+`,"apply":true}}}`)
	sc, _ = out.Result["structuredContent"].(map[string]any)
	request, _ := sc["request"].(map[string]any)
	if request["id"] == nil || request["id"] != preview["id"] || request["request_digest"] != preview["request_digest"] {
		t.Fatalf("apply did not record the previewed request: preview %+v, applied %+v", preview, sc)
	}
	views, err := pose.Store{Root: root}.ListActionRequests()
	if err != nil || len(views) != 1 || views[0].Request.Kind != pose.ActionDecision || len(views[0].Request.Options) != 2 {
		t.Fatalf("journal: %+v %v", views, err)
	}
	// An incomplete request is refused, as on the CLI.
	_, out = post(t, ts, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"pose_action_open","arguments":{"origin":"spec:storage","kind":"approval","question":"q","requested_by":"agent:impl","targets":[],"effects":["closeout:block"],"apply":true}}}`)
	if out.Result["isError"] != true {
		t.Fatalf("a request without targets was accepted: %+v", out)
	}
}
