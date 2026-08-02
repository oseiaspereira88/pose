package mcpserver

import "testing"

func TestCloseoutStateToolUsesProjectScopedProjection(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":91,"method":"tools/call","params":{"name":"pose_closeout_state","arguments":{"scope":"spec:alpha"}}}`)
	if out.Error != nil || out.Result["isError"] != false {
		t.Fatalf("closeout tool failed: error=%+v result=%v", out.Error, out.Result)
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["scope"] != "spec:alpha" || structured["terminal"] != true {
		t.Fatalf("unexpected closeout projection: %v", structured)
	}
}

func TestCloseoutStateToolRejectsMissingAndTraversalScope(t *testing.T) {
	ts := newTestServer(t, "")
	for _, arguments := range []string{`{}`, `{"scope":"spec:../../etc"}`} {
		_, out := post(t, ts, `{"jsonrpc":"2.0","id":92,"method":"tools/call","params":{"name":"pose_closeout_state","arguments":`+arguments+`}}`)
		if out.Error != nil || out.Result["isError"] != true {
			t.Fatalf("arguments %s did not fail closed: error=%+v result=%v", arguments, out.Error, out.Result)
		}
	}
}
