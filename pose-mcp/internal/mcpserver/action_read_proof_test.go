package mcpserver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-action-answers-reverify-on-read R5: persisted labels cannot
// bypass proof verification through the actual MCP transport.
func TestActionReadMCPRejectsPersistedProoflessVerifiedAnswer(t *testing.T) {
	root, ts, view := actionResolveFixture(t, `{"schema_version":1,"roles":{"maintainer":["human:maintainer"]},"identity_assurance":"declared"}`)
	store := pose.Store{Root: root}
	_, err := store.ResolveActionRequest(pose.ActionResolution{Type: pose.ActionEventAnswered, Actor: "human:maintainer", Answer: "preserve-v1",
		RequestID: view.Request.ID, RequestDigest: view.Request.RequestDigest, ExpectedRevision: view.Revision, IdempotencyKey: "read-proof"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	call := func(tool string, args map[string]any) rpcResult {
		t.Helper()
		raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": tool, "arguments": args}})
		_, out := post(t, ts, string(raw))
		return out
	}
	valid := call("pose_action_requests", map[string]any{"id": view.Request.ID})
	if valid.Result["isError"] == true || !strings.Contains(toolText(valid), `"assurance":"declared"`) {
		t.Fatalf("declared history: %+v", valid)
	}
	path := filepath.Join(root, ".pose/actions", view.Request.ID+".jsonl")
	journal, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(journal), `"assurance":"declared"`) {
		t.Fatalf("fixture has no declared answer: %s", journal)
	}
	journal = []byte(strings.ReplaceAll(string(journal), `"assurance":"declared"`, `"assurance":"verified"`))
	if err := os.WriteFile(path, journal, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range []map[string]any{{"id": view.Request.ID}, {}, {"present": true, "actor": "human:maintainer"}} {
		out := call("pose_action_requests", args)
		if out.Result["isError"] != true || !strings.Contains(toolText(out), "action-claim-not-verified") {
			t.Fatalf("MCP consumed forged assurance: %+v", out)
		}
	}
	attention := call("pose_obligations", map[string]any{"actor": "human:maintainer"})
	if attention.Result["isError"] == true || !strings.Contains(toolText(attention), `"complete":false`) || !strings.Contains(toolText(attention), `"unavailable"`) {
		t.Fatalf("MCP hid incomplete coverage: %+v", attention)
	}
}
