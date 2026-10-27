package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/mcpserver"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// delegateMCP serves root over MCP with the delegate tools running the built
// pose binary, as `pose serve` runs itself.
func delegateMCP(t *testing.T, root string) func(tool string, args map[string]any) map[string]any {
	t.Helper()
	binary := poseBinary(t)
	server := mcpserver.New(posemodel.Store{Root: root}).WithDelegateRunner(func(ctx context.Context, dir string, args []string) ([]byte, int, error) {
		cmd := exec.CommandContext(ctx, binary, append([]string{"delegate"}, args...)...)
		cmd.Dir = dir
		out, err := cmd.Output()
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return out, exit.ExitCode(), nil
		}
		return out, 0, err
	})
	ts := httptest.NewServer(server.Handler("", ""))
	t.Cleanup(ts.Close)
	return func(tool string, args map[string]any) map[string]any {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": tool, "arguments": args}})
		response, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var rpc struct {
			Result map[string]any `json:"result"`
			Error  map[string]any `json:"error"`
		}
		if err := json.NewDecoder(response.Body).Decode(&rpc); err != nil {
			t.Fatal(err)
		}
		if rpc.Error != nil {
			return map[string]any{"rpc_error": rpc.Error}
		}
		if structured, ok := rpc.Result["structuredContent"].(map[string]any); ok {
			return structured
		}
		content, _ := rpc.Result["content"].([]any)
		if len(content) == 1 {
			text, _ := content[0].(map[string]any)["text"].(string)
			var doc map[string]any
			if json.Unmarshal([]byte(text), &doc) == nil {
				return doc
			}
			return map[string]any{"text": text}
		}
		return rpc.Result
	}
}

// R1–R5, R7: over MCP a run is previewed, started detached, read without
// blocking and waited for within a bound; a rerun on an unchanged bundle
// without a reason is refused with the CLI's reason code; no result carries a
// transcript, a session or an environment value.
func TestDelegateToolsOverMCPMatchTheCLI(t *testing.T) {
	root := dispatchInstance(t)
	slow := `"slow": {"vendor": "test", "model": "slow-1", "timeout": "30s",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; sleep 4; echo decision: approved"]}}`
	writeDelegation(t, root, delegationPolicy(approvingAgent, slow))
	call := delegateMCP(t, root)
	preview := call("pose_delegate_start", map[string]any{"purpose": "review", "target": "spec:alpha", "agent": "ok"})
	if preview["agent"] != "ok" || preview["apply"] == nil || preview["independence"] == nil {
		t.Fatalf("preview: %v", preview)
	}
	started := call("pose_delegate_start", map[string]any{"purpose": "review", "target": "spec:alpha", "agent": "ok", "apply": true})
	attempt, _ := started["attempt"].(string)
	if attempt == "" || started["agent"] != "ok" {
		t.Fatalf("start: %v", started)
	}
	waited := call("pose_delegate_wait", map[string]any{"run": attempt, "timeout_seconds": 60})
	if waited["decision"] != "approved" || waited["state"] != "completed" {
		t.Fatalf("wait: %v", waited)
	}
	status := call("pose_delegate_status", map[string]any{"run": attempt})
	if status["state"] != "completed" || status["decision"] != "approved" {
		t.Fatalf("status: %v", status)
	}
	raw, _ := json.Marshal([]any{preview, started, waited, status})
	for _, leak := range []string{`"output"`, `"transcript`, `"session`, `"final"`, `"env_removed"`, `"argv"`, `"log"`} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("a tool result carries %s: %s", leak, raw)
		}
	}
	refused := call("pose_delegate_start", map[string]any{"purpose": "review", "target": "spec:alpha", "agent": "ok", "apply": true})
	_, cliOut, _ := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", "ok", "--json")
	if refused["reason_code"] != posemodel.DelegationReasonRerunWithoutReason || !strings.Contains(cliOut, `"reason_code": "`+posemodel.DelegationReasonRerunWithoutReason+`"`) {
		t.Fatalf("a rerun without a reason: mcp %v cli %s", refused, cliOut)
	}
	running := call("pose_delegate_start", map[string]any{"purpose": "review", "target": "spec:alpha", "agent": "slow", "reason": "a slower agent", "apply": true})
	slowAttempt, _ := running["attempt"].(string)
	if slowAttempt == "" {
		t.Fatalf("slow start: %v", running)
	}
	if still := call("pose_delegate_wait", map[string]any{"run": slowAttempt, "timeout_seconds": 1}); still["state"] != "still-running" {
		t.Fatalf("a bounded wait did not return: %v", still)
	}
	if live := call("pose_delegate_status", map[string]any{"run": slowAttempt}); live["state"] != "running" {
		t.Fatalf("status of a running run: %v", live)
	}
	if done := call("pose_delegate_wait", map[string]any{"run": slowAttempt, "timeout_seconds": 60}); done["state"] != "completed" {
		t.Fatalf("the slow run: %v", done)
	}
	if task := call("pose_delegate_start", map[string]any{"purpose": "task", "target": "x"}); !strings.Contains(string(mustJSON(task)), "a task is not started over MCP") {
		t.Fatalf("a task started over MCP: %v", task)
	}
}

func mustJSON(v any) []byte {
	raw, _ := json.Marshal(v)
	return raw
}

// R4: over MCP a stopped run is resumed, or handed off to another agent,
// detached, as the CLI does.
func TestDelegateToolsOverMCPContinueStoppedRuns(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(limitingAgent, continuingAgent))
	call := delegateMCP(t, root)
	stopped := func(label string) string {
		t.Helper()
		started := call("pose_delegate_start", map[string]any{"purpose": "review", "target": "spec:alpha", "agent": "stopper", "label": label, "reason": label, "apply": true})
		attempt, _ := started["attempt"].(string)
		if waited := call("pose_delegate_wait", map[string]any{"run": attempt, "timeout_seconds": 60}); waited["state"] != "interrupted" {
			t.Fatalf("%s did not stop: %v %v", label, started, waited)
		}
		return attempt
	}
	first := stopped("to-hand-off")
	if preview := call("pose_delegate_handoff", map[string]any{"run": first, "to": "continuer"}); preview["apply"] == nil || preview["error"] != nil {
		t.Fatalf("handoff preview: %v", preview)
	}
	handed := call("pose_delegate_handoff", map[string]any{"run": first, "to": "continuer", "apply": true})
	if handed["attempt"] != first {
		t.Fatalf("handoff: %v", handed)
	}
	if done := call("pose_delegate_wait", map[string]any{"run": first, "timeout_seconds": 60}); done["state"] != "completed" {
		t.Fatalf("the handoff run: %v", done)
	}
	second := stopped("to-resume")
	resumed := call("pose_delegate_resume", map[string]any{"run": second, "apply": true})
	if resumed["attempt"] != second {
		t.Fatalf("resume: %v", resumed)
	}
	if done := call("pose_delegate_wait", map[string]any{"run": second, "timeout_seconds": 60}); done["state"] != "completed" || done["decision"] != "approved" {
		t.Fatalf("the resumed run: %v", done)
	}
}
