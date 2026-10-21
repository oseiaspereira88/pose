package mcpserver

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestActionTargetMCPStructuredAndLegacyEffects(t *testing.T) {
	root := t.TempDir()
	for _, slug := range []string{"alpha", "beta"} {
		p := filepath.Join(root, ".pose/specs/"+slug+".md")
		if e := os.MkdirAll(filepath.Dir(p), 0755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p, []byte("---\nslug: "+slug+"\nstatus: draft\n---\n## 2. Requirements\n- R4: "+slug+" material.\n"), 0644); e != nil {
			t.Fatal(e)
		}
	}
	project := pose.Store{Root: root}.CurrentObligationSnapshot().Project
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	args := map[string]any{"origin": "spec:alpha", "kind": "input", "question": "Which input?", "requested_by": "agent:impl", "targets": []string{"self"}, "effects": []any{"review:advisory", map[string]any{"phase": "start", "mode": "block", "scope": []string{"xref:" + project + "/spec:beta"}}}, "apply": true}
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "pose_action_open", "arguments": args}})
	_, out := post(t, ts, string(raw))
	if out.Result["isError"] == true {
		t.Fatalf("structured open: %+v", out)
	}
	s := pose.Store{Root: root}
	views, e := s.ListActionRequests()
	if e != nil || len(views) != 1 || len(views[0].Request.Effects) != 2 {
		t.Fatalf("journal: %+v %v", views, e)
	}
	for _, tc := range []struct {
		slug string
		want int
	}{{"alpha", 0}, {"beta", 1}} {
		got, e := s.GovernedEffectRestrictions(pose.PhaseStart, []string{tc.slug})
		if e != nil || len(got) != tc.want {
			t.Fatalf("%s: %+v %v", tc.slug, got, e)
		}
	}
	args["effects"] = []any{map[string]any{"phase": "start", "mode": "block", "scope": []string{"xref:" + project + "/spec:missing"}}}
	raw, _ = json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "pose_action_open", "arguments": args}})
	_, out = post(t, ts, string(raw))
	if out.Result["isError"] != true {
		t.Fatal("missing destination accepted")
	}
}
