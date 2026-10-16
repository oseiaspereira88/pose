package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-journal-checkpoint-and-lock-recovery R9: MCP reports a journal
// that does not read where it breaks, and a start lock with its owner's state.
func TestMCPLocatesAJournalThatDoesNotRead(t *testing.T) {
	root, ts, view := actionResolveFixture(t, `{"schema_version":1,"roles":{"maintainer":["human:maintainer"]}}`)
	path := filepath.Join(root, ".pose/actions", view.Request.ID+".jsonl")
	intact, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(append([]byte{}, intact...), []byte(`{"schema_version":1,"seq":2,`)...), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range []map[string]any{{"id": view.Request.ID}, {}} {
		raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "pose_action_requests", "arguments": args}})
		_, out := post(t, ts, string(raw))
		sc, _ := out.Result["structuredContent"].(map[string]any)
		journal, _ := sc["journal"].(map[string]any)
		if out.Result["isError"] != true || sc["error_code"] != "action_journal_invalid" || journal["kind"] != pose.JournalInterruptedTail ||
			journal["offset"] != float64(len(intact)) || journal["request"] != view.Request.ID || !strings.Contains(sc["next_action"].(string), "truncating the journal to") {
			t.Fatalf("%v: %+v", args, out.Result)
		}
	}
}

func TestMCPStartStatusShowsAHeldLock(t *testing.T) {
	root := t.TempDir()
	for path, body := range map[string]string{
		".pose/policy/review.json":       `{"schema_version":1,"enabled":false,"atomic_start_version":1}`,
		".pose/specs/2026-09-27-work.md": "---\nslug: work\nstatus: draft\ncreated_at: 2026-09-27\n---\n\n# Spec: work\n\n## 2. Requirements\n\n- R1: One.\n",
		".pose/starts/.work.lock":        "",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, err := New(pose.Store{Root: root}).dispatch(context.Background(), "pose_start_status", json.RawMessage(`{"slug":"work"}`))
	if err != nil {
		t.Fatal(err)
	}
	status, _ := out.(map[string]any)["status"].(pose.StartStatus)
	if status.Lock == nil || status.Lock.State != pose.LockOwnerUnknown || status.Lock.OwnerDigest == "" {
		t.Fatalf("the MCP status hid the lock: %+v", status)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/starts/.work.lock")); err != nil {
		t.Fatal("the MCP status removed the lock")
	}
}
