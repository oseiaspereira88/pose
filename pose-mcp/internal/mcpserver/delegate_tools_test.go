package mcpserver

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Review round 1, B05: every subprocess of one call shares the call's
// deadline; a start never exceeds its bound by running a preview first.
func TestDelegateToolsShareOneDeadlinePerCall(t *testing.T) {
	root := t.TempDir()
	var deadlines []time.Time
	server := New(pose.Store{Root: root}).WithDelegateRunner(func(ctx context.Context, dir string, args []string) ([]byte, int, error) {
		d, _ := ctx.Deadline()
		deadlines = append(deadlines, d)
		time.Sleep(20 * time.Millisecond)
		doc := map[string]any{"schema": "pose.delegate.plan/v1", "agent": "a"}
		if args[len(args)-2] == "--detach" {
			doc = map[string]any{"schema": "pose.delegate.detach/v1", "attempt": "rat-x"}
		}
		raw, _ := json.Marshal(doc)
		return raw, 0, nil
	})
	entered := time.Now()
	args, _ := json.Marshal(map[string]any{"purpose": "review", "target": "spec:x", "apply": true})
	if _, err := server.dispatchDelegate(context.Background(), root, "pose_delegate_start", args); err != nil {
		t.Fatal(err)
	}
	if len(deadlines) != 2 || !deadlines[0].Equal(deadlines[1]) || deadlines[1].After(entered.Add(delegateStartBound+time.Second)) {
		t.Fatalf("deadlines %v for a call entered at %v", deadlines, entered)
	}
}
