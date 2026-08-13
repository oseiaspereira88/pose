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

func TestReviewPlanToolUsesTheSameProjectScopedProjection(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":93,"method":"tools/call","params":{"name":"pose_review_plan","arguments":{"scope":"spec:alpha"}}}`)
	if out.Error != nil || out.Result["isError"] != false {
		t.Fatalf("review plan tool failed: error=%+v result=%v", out.Error, out.Result)
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["scope"] != "spec:alpha" || structured["plan_digest"] == "" || structured["base_profile"] != "spec-closeout@1" {
		t.Fatalf("unexpected review plan projection: %v", structured)
	}
}

func TestReviewPlanToolRejectsMissingAndTraversalScope(t *testing.T) {
	ts := newTestServer(t, "")
	for _, arguments := range []string{`{}`, `{"scope":"spec:../../etc"}`} {
		_, out := post(t, ts, `{"jsonrpc":"2.0","id":94,"method":"tools/call","params":{"name":"pose_review_plan","arguments":`+arguments+`}}`)
		if out.Error != nil || out.Result["isError"] != true {
			t.Fatalf("arguments %s did not fail closed: error=%+v result=%v", arguments, out.Error, out.Result)
		}
	}
}

func TestReviewBundleToolIsReadOnlyAndProjectScoped(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":95,"method":"tools/call","params":{"name":"pose_review_bundle","arguments":{"scope":"spec:alpha"}}}`)
	if out.Error != nil || out.Result["isError"] != false {
		t.Fatalf("review bundle tool failed: error=%+v result=%v", out.Error, out.Result)
	}
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if structured["scope"] != "spec:alpha" || structured["state"] == "" || structured["next_action"] == "" {
		t.Fatalf("unexpected review bundle projection: %v", structured)
	}
}
