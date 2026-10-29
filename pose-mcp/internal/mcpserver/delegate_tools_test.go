package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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

// Review round 2, B05 and F04: the bound is the operation's — an apply a
// status does not take leaves it at the status bound — and timeout_seconds
// 0 is refused as out of range, not read as the default.
func TestDelegateToolsBoundIsTheOperations(t *testing.T) {
	root := t.TempDir()
	var deadline time.Time
	server := New(pose.Store{Root: root}).WithDelegateRunner(func(ctx context.Context, dir string, args []string) ([]byte, int, error) {
		deadline, _ = ctx.Deadline()
		raw, _ := json.Marshal(map[string]any{"schema": "pose.delegate.ls/v1", "attempts": []any{map[string]any{"attempt": "rat-x", "active": true}}})
		return raw, 0, nil
	})
	entered := time.Now()
	args, _ := json.Marshal(map[string]any{"run": "rat-x", "apply": true})
	if _, err := server.dispatchDelegate(context.Background(), root, "pose_delegate_status", args); err != nil {
		t.Fatal(err)
	}
	if deadline.After(entered.Add(delegateStatusBound + time.Second)) {
		t.Fatalf("a status with apply ran until %v", deadline.Sub(entered))
	}
	zero, _ := json.Marshal(map[string]any{"run": "rat-x", "timeout_seconds": 0})
	if _, err := server.dispatchDelegate(context.Background(), root, "pose_delegate_wait", zero); err == nil {
		t.Fatal("timeout_seconds 0 was accepted")
	}
}

// Review round 2, B02 and B03: what crosses is codes, identifiers, times,
// flags and counts. A credential named in a failure, a log path in an error,
// a skipped candidate's reason, a way forward's why and a loss's file name
// stay on the host; a failing exit without a refusal fails closed.
func TestDelegateToolsProjectWhatTheyReturn(t *testing.T) {
	root := t.TempDir()
	const secret = "SYNTHETIC_ENV_VALUE_123456"
	docs := map[string]map[string]any{
		"ls": {"schema": "pose.delegate.ls/v1", "losses": []any{"SYNTHETIC_PRIVATE_SESSION_REFERENCE.log.json (not a review run)"}, "attempts": []any{map[string]any{
			"attempt": "rat-x", "active": false, "status": "failed", "failure": "the run changed files: [\"" + secret + ".log\"]",
			"interruption": map[string]any{"cause": "/private/SYNTHETIC_LOG_PATH.log", "resumable": true, "reset_at": "soon"},
		}}},
		"handoff": {"schema": "pose.delegate.error/v1", "error": "agent unavailable: /private/SYNTHETIC_LOG_PATH.log", "reason_code": "agent-unavailable", "escalation": "/private/SYNTHETIC_LOG_PATH.log", "escalation_required": "/private/SYNTHETIC_LOG_PATH.log",
			"routing": map[string]any{"requested": "auto", "step": "preferred", "candidates": []any{map[string]any{"agent": "b", "state": "unavailable", "skipped": "cli /private/SYNTHETIC_LOG_PATH.log missing"}}}},
		"resume": {"schema": "pose.delegate.wait/v1", "state": "stopped", "next_actions": []any{
			map[string]any{"action": "resume", "available": false, "why": "/private/SYNTHETIC_LOG_PATH.log", "command": "pose delegate resume rat-x"},
			map[string]any{"action": "handoff", "available": true, "command": "pose delegate handoff rat-x --to '" + secret + "'"},
		}},
	}
	server := New(pose.Store{Root: root}).WithDelegateRunner(func(ctx context.Context, dir string, args []string) ([]byte, int, error) {
		if args[0] == "wait" {
			return []byte(`{"schema":"pose.delegate.error/v1"}`), 1, nil
		}
		raw, _ := json.Marshal(docs[args[0]])
		if args[0] == "handoff" {
			return raw, 1, nil
		}
		return raw, 0, nil
	})
	call := func(tool string, args map[string]any) string {
		raw, _ := json.Marshal(args)
		result, err := server.dispatchDelegate(context.Background(), root, tool, raw)
		if err != nil {
			return "error: " + err.Error()
		}
		out, _ := json.Marshal(result)
		return string(out)
	}
	for _, out := range []string{
		call("pose_delegate_status", map[string]any{"run": "rat-x"}),
		call("pose_delegate_handoff", map[string]any{"run": "rat-x", "to": "auto"}),
		call("pose_delegate_resume", map[string]any{"run": "rat-x"}),
	} {
		for _, private := range []string{secret, "SYNTHETIC_LOG_PATH", "SYNTHETIC_PRIVATE_SESSION"} {
			if strings.Contains(out, private) {
				t.Errorf("%s crossed: %s", private, out)
			}
		}
	}
	if out := call("pose_delegate_handoff", map[string]any{"run": "rat-x", "to": "auto"}); !strings.Contains(out, `"reason_code":"agent-unavailable"`) || !strings.Contains(out, `"skipped":true`) {
		t.Errorf("the refusal lost its code or the skip: %s", out)
	}
	if out := call("pose_delegate_resume", map[string]any{"run": "rat-x"}); !strings.Contains(out, `"command":"pose delegate resume rat-x"`) {
		t.Errorf("a way forward lost its command: %s", out)
	}
	docs["resume"] = map[string]any{"schema": "pose.delegate.error/v1"}
	server.delegateRunner = func(ctx context.Context, dir string, args []string) ([]byte, int, error) {
		return []byte(`{"schema":"pose.delegate.error/v1"}`), 1, nil
	}
	if out := call("pose_delegate_resume", map[string]any{"run": "rat-x", "apply": true}); !strings.HasPrefix(out, "error:") {
		t.Errorf("a failing exit without a refusal answered: %s", out)
	}
}

// Review round 3: a status keeps failed when the ls row it reads was
// projected already (F06); a runner that cannot start names no host path
// (F07).
func TestDelegateToolsStatusKeepsFailedAndHidesRunnerErrors(t *testing.T) {
	root := t.TempDir()
	server := New(pose.Store{Root: root}).WithDelegateRunner(func(ctx context.Context, dir string, args []string) ([]byte, int, error) {
		raw, _ := json.Marshal(map[string]any{"schema": "pose.delegate.ls/v1", "attempts": []any{map[string]any{"attempt": "rat-x", "active": true, "status": "failed", "failure": "x"}}})
		return raw, 0, nil
	})
	args, _ := json.Marshal(map[string]any{"run": "rat-x"})
	status, err := server.dispatchDelegate(context.Background(), root, "pose_delegate_status", args)
	if err != nil || status.(map[string]any)["failed"] != true {
		t.Fatalf("status: %v %v", status, err)
	}
	server.delegateRunner = func(ctx context.Context, dir string, args []string) ([]byte, int, error) {
		return nil, -1, errors.New("chdir /SYNTHETIC_MISSING_PROJECT: no such file or directory")
	}
	if _, err := server.dispatchDelegate(context.Background(), root, "pose_delegate_status", args); err == nil || strings.Contains(err.Error(), "SYNTHETIC") {
		t.Fatalf("runner error: %v", err)
	}
}
