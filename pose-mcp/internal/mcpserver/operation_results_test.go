package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-cli-mcp-operation-results: MCP keeps the domain's typed outcome.

func TestAGateToolStoppedByItsDeadlineIsAnInconclusiveError(t *testing.T) {
	script := filepath.Join(t.TempDir(), "pose")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho checking\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_EXECUTABLE", script)
	t.Setenv("POSE_USAGE_DISABLED", "1")
	server := New(pose.Store{Root: t.TempDir()})
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	req := rpcRequest{JSONRPC: "2.0", ID: []byte(`1`), Method: "tools/call",
		Params: []byte(`{"name":"pose_check","arguments":{"strict":true}}`)}
	resp := server.callToolCtx(ctx, "", "", "", req)
	if resp.Error != nil {
		t.Fatalf("an inconclusive gate is a tool result, not a protocol error: %+v", resp.Error)
	}
	result, _ := resp.Result.(map[string]any)
	structured, _ := result["structuredContent"].(map[string]any)
	if result["isError"] != true || structured == nil {
		t.Fatalf("a killed gate was not an error result: %+v", result)
	}
	if _, verdict := structured["passed"]; verdict {
		t.Fatalf("a killed gate carries a verdict: %+v", structured)
	}
	if structured["error_code"] != "execution_inconclusive" || structured["outcome"] != pose.OutcomeInconclusive || structured["cause"] != pose.ExecutionCauseDeadline || structured["partial_output_complete"] != false {
		t.Fatalf("the inconclusive cause is not typed: %+v", structured)
	}
	if !strings.Contains(structured["partial_output"].(string), "checking") || structured["cleanup"] == "" {
		t.Fatalf("partial output or cleanup limit missing: %+v", structured)
	}
}

func TestListSpecsRefusesFiltersItWouldOtherwiseDrop(t *testing.T) {
	ts := newTestServer(t, "")
	for _, args := range []string{`{"since":"nonsense"}`, `{"since":"-3d"}`, `{"since":""}`, `{"since":"999999999999999999d"}`, `{"recent":-1}`, `{"recent":0}`, `{"limit":-1}`, `{"limit":0}`, `{"sttaus":"done"}`, `{"sort":"name"}`, `{"sort":""}`, `{"sort":null}`, `{"recent":null}`, `{"limit":null}`, `{"since":null}`, `{"status":null}`} {
		_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_list_specs","arguments":`+args+`}}`)
		if out.Result["isError"] != true {
			t.Fatalf("%s listed specs instead of refusing: %+v", args, out.Result)
		}
	}
	// Omitted values keep the unfiltered, unpaginated listing.
	_, all := post(t, ts, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{}}}`)
	if structured, _ := all.Result["structuredContent"].(map[string]any); all.Result["isError"] == true || structured["total"] != float64(2) {
		t.Fatalf("the unfiltered listing changed: %+v", all.Result)
	}
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{"since":"2026-06-02","recent":5}}}`)
	structured, _ := out.Result["structuredContent"].(map[string]any)
	if out.Result["isError"] == true || structured == nil || structured["total"] != float64(1) {
		t.Fatalf("a valid filter stopped working: %+v", out.Result)
	}
}

func TestListSpecsKeepsTheAdvertisedSlugOrder(t *testing.T) {
	ts := newTestServer(t, "")
	_, out := post(t, ts, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pose_list_specs","arguments":{"sort":"slug"}}}`)
	structured, _ := out.Result["structuredContent"].(map[string]any)
	specs, _ := structured["specs"].([]any)
	if out.Result["isError"] == true || len(specs) != 2 || specs[0].(map[string]any)["slug"] != "alpha" || specs[1].(map[string]any)["slug"] != "beta" {
		t.Fatalf("the advertised slug order was refused or not applied: %+v", out.Result)
	}
}
