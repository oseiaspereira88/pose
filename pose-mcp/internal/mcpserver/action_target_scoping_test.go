package mcpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

func TestActionTargetMCPForeignMembershipLossReportsUnknownWithoutGlobalBlocker(t *testing.T) {
	parent, child := t.TempDir(), t.TempDir()
	write := func(root, rel, body string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if e := os.MkdirAll(filepath.Dir(p), 0755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p, []byte(body), 0644); e != nil {
			t.Fatal(e)
		}
	}
	write(parent, ".pose/project.json", `{"schema_version":1,"project_id":"parent"}`)
	write(child, ".pose/project.json", `{"schema_version":1,"project_id":"child"}`)
	for _, slug := range []string{"alpha", "beta"} {
		write(parent, ".pose/specs/"+slug+".md", "---\nslug: "+slug+"\nstatus: draft\n---\n## 2. Requirements\n- R4: Material.\n")
	}
	roadmap := "---\nslug: remote\nstatus: in-progress\n---\n## Milestone: first\n- specs: xref:parent/spec:beta\n"
	write(child, ".pose/roadmaps/remote.md", roadmap)
	var allowChild atomic.Bool
	allowChild.Store(true)
	opa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Input struct {
				Project string `json:"project_id"`
			} `json:"input"`
		}
		if e := json.NewDecoder(r.Body).Decode(&input); e != nil {
			http.Error(w, "invalid", 400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"allow": input.Input.Project == "parent" || allowChild.Load()}})
	}))
	t.Cleanup(opa.Close)
	roots := pose.NewRoots(pose.RootsConfig{DefaultRoot: parent, DefaultProjectID: "parent", Explicit: map[string]string{"child": child}})
	ts := httptest.NewServer(NewWithRootsAndPolicy(roots, NewPolicyGate(PolicyConfig{OPAURL: opa.URL, HTTPClient: opa.Client()})).Handler("", ""))
	t.Cleanup(ts.Close)
	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		args["project_id"] = "parent"
		raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}})
		_, out := post(t, ts, string(raw))
		if out.Result["isError"] == true {
			t.Fatalf("MCP %s: %+v", name, out)
		}
		sc, _ := out.Result["structuredContent"].(map[string]any)
		return sc
	}
	call("pose_action_open", map[string]any{"origin": "spec:alpha", "kind": "input", "question": "Input?", "requested_by": "agent:impl", "targets": []string{"self"}, "effects": []any{map[string]any{"phase": "start", "mode": "block", "scope": []string{"xref:child/roadmap:remote"}}}, "apply": true})
	report := call("pose_obligations", map[string]any{"scope": "spec:beta"})
	raw, _ := json.Marshal(report)
	if !strings.Contains(string(raw), "action-open") {
		t.Fatalf("authorized membership not expanded: %s", raw)
	}
	for _, mode := range []string{"unreadable", "unauthorized"} {
		if mode == "unreadable" {
			if e := os.Remove(filepath.Join(child, ".pose/roadmaps/remote.md")); e != nil {
				t.Fatal(e)
			}
		} else {
			write(child, ".pose/roadmaps/remote.md", roadmap)
			allowChild.Store(false)
		}
		report = call("pose_obligations", map[string]any{"scope": "spec:beta"})
		raw, _ = json.Marshal(report)
		coverage, _ := report["coverage"].([]any)
		found := false
		for _, item := range coverage {
			c, _ := item.(map[string]any)
			if c["producer"] == "action-requests" {
				found = c["state"] == pose.CoverageStateIncomplete
			}
		}
		if !found {
			t.Fatalf("%s foreign membership claimed current: %s", mode, raw)
		}
		if strings.Contains(string(raw), "action-open") {
			t.Fatalf("%s invented a global blocker: %s", mode, raw)
		}
		if strings.Contains(string(raw), parent) || strings.Contains(string(raw), child) {
			t.Fatalf("root leaked: %s", raw)
		}
	}
}
