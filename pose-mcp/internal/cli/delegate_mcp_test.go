package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		cmd.Dir, cmd.Env = dir, mcpserver.DelegateEnv(dir)
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

// Review round 1 of pose-delegation-mcp: the subprocess stays in the
// authorized root (B01); no result carries the run's messages or a log
// (B02); a detached continuation refused by its budget, and a project with
// no agent, keep their reason codes (B03, B04); timeout_seconds is bounded
// (F04).
func TestDelegateToolsOverMCPHoldAgainstTheFirstReview(t *testing.T) {
	root := dispatchInstance(t)
	chatty := `"chatty": {"vendor": "test", "model": "chatty-1", "timeout": "30s", "events": "codex-jsonl",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; printf '%s\\n' '{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"SYNTHETIC_PRIVATE_PROMPT /private/x.log\"}}'; sleep 4; echo decision: approved"]}}`
	writeDelegation(t, root, delegationPolicy(chatty, limitingAgent, strings.Replace(strings.Replace(limitingAgent, `"stopper"`, `"stopper-b"`, 1), `"vendor": "test"`, `"vendor": "other"`, 1), continuingAgent))
	call := delegateMCP(t, root)
	started := call("pose_delegate_start", map[string]any{"purpose": "review", "target": "spec:alpha", "agent": "chatty", "apply": true})
	attempt, _ := started["attempt"].(string)
	if attempt == "" {
		t.Fatalf("start: %v", started)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		status := call("pose_delegate_status", map[string]any{"run": attempt})
		if strings.Contains(string(mustJSON(status)), "SYNTHETIC_PRIVATE") {
			t.Fatalf("a status carries the run's message: %v", status)
		}
		time.Sleep(300 * time.Millisecond)
	}
	call("pose_delegate_wait", map[string]any{"run": attempt, "timeout_seconds": 60})
	if bad := call("pose_delegate_wait", map[string]any{"run": attempt, "timeout_seconds": 99999}); !strings.Contains(string(mustJSON(bad)), "timeout_seconds") {
		t.Fatalf("an unbounded wait was accepted: %v", bad)
	}
	stopped := call("pose_delegate_start", map[string]any{"purpose": "review", "target": "spec:alpha", "agent": "stopper", "reason": "budget", "apply": true})
	first, _ := stopped["attempt"].(string)
	call("pose_delegate_wait", map[string]any{"run": first, "timeout_seconds": 60})
	call("pose_delegate_handoff", map[string]any{"run": first, "to": "stopper-b", "apply": true})
	call("pose_delegate_wait", map[string]any{"run": first, "timeout_seconds": 60})
	call("pose_delegate_handoff", map[string]any{"run": first, "to": "stopper", "apply": true})
	call("pose_delegate_wait", map[string]any{"run": first, "timeout_seconds": 60})
	refused := call("pose_delegate_handoff", map[string]any{"run": first, "to": "continuer", "apply": true})
	if refused["reason_code"] != posemodel.DelegationReasonBudgetExhausted || strings.Contains(string(mustJSON(refused)), "supervisors") {
		t.Fatalf("a detached refusal lost its code or carried a log: %v", refused)
	}
	empty := dispatchInstance(t)
	if r := delegateMCP(t, empty)("pose_delegate_start", map[string]any{"purpose": "review", "target": "spec:alpha", "apply": true}); r["reason_code"] != posemodel.DelegationReasonNoAgent || strings.Contains(string(mustJSON(r)), "commands") {
		t.Fatalf("no agent: %v", r)
	}
	// A server rooted at a nested instance runs there, never in the parent.
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(filepath.Join(child, ".pose", "policy"), 0o755); err != nil {
		t.Fatal(err)
	}
	before, _, _ := posemodel.Store{Root: root}.ListDelegationAttempts()
	r := delegateMCP(t, child)("pose_delegate_start", map[string]any{"purpose": "review", "target": "spec:alpha", "agent": "chatty", "reason": "nested", "apply": true})
	after, _, _ := posemodel.Store{Root: root}.ListDelegationAttempts()
	if len(after) != len(before) || r["attempt"] != nil {
		t.Fatalf("a nested server ran in the parent: %v (%d → %d)", r, len(before), len(after))
	}
}

// Review round 2, B03: a detached handoff refused by routing — an error
// document with nested objects — keeps its reason code over MCP, and
// carries no log.
func TestDelegateToolsOverMCPKeepADetachedRoutingRefusal(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(limitingAgent))
	call := delegateMCP(t, root)
	started := call("pose_delegate_start", map[string]any{"purpose": "review", "target": "spec:alpha", "agent": "stopper", "apply": true})
	attempt, _ := started["attempt"].(string)
	if waited := call("pose_delegate_wait", map[string]any{"run": attempt, "timeout_seconds": 60}); waited["state"] != "interrupted" {
		t.Fatalf("the run did not stop: %v %v", started, waited)
	}
	writeDelegation(t, root, delegationPolicy(limitingAgent, `"absent": {"vendor": "other", "model": "a-1", "commands": {"read": ["/nonexistent-SYNTHETIC_PRIVATE_PROMPT/cli"]}}`))
	preview := call("pose_delegate_handoff", map[string]any{"run": attempt, "to": "auto"})
	refused := call("pose_delegate_handoff", map[string]any{"run": attempt, "to": "auto", "apply": true})
	out := string(mustJSON(refused))
	if refused["refused"] != true || refused["reason_code"] != preview["reason_code"] || refused["reason_code"] == nil || strings.Contains(out, "supervisors") || strings.Contains(out, "SYNTHETIC") {
		t.Fatalf("preview %v, applied %s", preview, out)
	}
}

// Review round 3, B06: repeating a resume or a handoff of a predecessor
// whose continuation exists returns that continuation detached, over MCP,
// as the foreground CLI does, and runs no agent again.
func TestDelegateToolsOverMCPReturnAnExistingContinuation(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(limitingAgent, continuingAgent))
	call := delegateMCP(t, root)
	for _, op := range []string{"resume", "handoff"} {
		started := call("pose_delegate_start", map[string]any{"purpose": "review", "target": "spec:alpha", "agent": "stopper", "label": op, "reason": op, "apply": true})
		attempt, _ := started["attempt"].(string)
		waited := call("pose_delegate_wait", map[string]any{"run": attempt, "timeout_seconds": 60})
		predecessor, _ := waited["run"].(string)
		if waited["state"] != "interrupted" || predecessor == "" {
			t.Fatalf("%s: the run did not stop: %v", op, waited)
		}
		args := map[string]any{"run": predecessor, "apply": true}
		if op == "handoff" {
			args["to"] = "continuer"
		}
		first := call("pose_delegate_"+op, args)
		done := call("pose_delegate_wait", map[string]any{"run": attempt, "timeout_seconds": 60})
		if first["attempt"] != attempt || done["state"] != "completed" {
			t.Fatalf("%s: %v then %v", op, first, done)
		}
		before, _, _ := posemodel.Store{Root: root}.ListDelegationAttempts()
		again := call("pose_delegate_"+op, args)
		after, _, _ := posemodel.Store{Root: root}.ListDelegationAttempts()
		if again["attempt"] != attempt || again["refused"] != nil || len(after) != len(before) {
			t.Fatalf("%s repeated: %v", op, again)
		}
		if replay := call("pose_delegate_wait", map[string]any{"run": attempt, "timeout_seconds": 60}); replay["run"] != done["run"] {
			t.Fatalf("%s repeated ran again: %v, first %v", op, replay, done)
		}
	}
}

// Review round 4, B07: a task's existing continuation, too, is returned
// detached and over MCP as the foreground CLI returns it, with and without
// --record, and no agent runs again.
func TestDelegateToolsOverMCPReturnAnExistingTaskContinuation(t *testing.T) {
	for _, record := range []bool{true, false} {
		for _, op := range []string{"resume", "handoff"} {
			t.Run(fmt.Sprintf("%s/record=%t", op, record), func(t *testing.T) {
				root, _, prompt := taskInstance(t)
				initial := []string{"delegate", "task", prompt, "--agent", "limited", "--json"}
				if record {
					initial = append(initial, "--record")
				}
				runPoseSplit(t, root, initial...)
				runs, lost := posemodel.Store{Root: root}.ReadDelegationTaskRuns()
				if len(runs) != 1 || len(lost) > 0 {
					t.Fatalf("runs %d, lost %v", len(runs), lost)
				}
				predecessor := runs[0].RunID
				call := delegateMCP(t, root)
				args := map[string]any{"run": predecessor, "apply": true}
				cli := []string{"delegate", op, predecessor, "--json"}
				if op == "handoff" {
					args["to"] = "worker"
					cli = append(cli, "--to", "worker")
				}
				first := call("pose_delegate_"+op, args)
				attempt, _ := first["attempt"].(string)
				done := call("pose_delegate_wait", map[string]any{"run": attempt, "timeout_seconds": 30})
				if attempt == "" || done["state"] != "completed" {
					t.Fatalf("first %v, then %v", first, done)
				}
				again := call("pose_delegate_"+op, args)
				code, out := runBuilt(t, root, append(cli, "--detach")...)
				if again["attempt"] != attempt || again["refused"] != nil || code != 0 || !strings.Contains(out, attempt) {
					t.Fatalf("repeated: MCP %v, --detach %d %s", again, code, out)
				}
				after, _ := posemodel.Store{Root: root}.ReadDelegationTaskRuns()
				if len(after) != 2 {
					t.Fatalf("a repeat ran an agent: %d runs", len(after))
				}
			})
		}
	}
}

// Review round 4, F08: a run that started hands its attempt back once; the
// file the caller consumed is not written again when the run ends.
func TestDetachedRunHandsBackOnce(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(`"slow": {"vendor": "test", "model": "s-1", "commands": {"read": ["sh", "-c", "cat >/dev/null; sleep 1; echo decision: approved"]}}`))
	call := delegateMCP(t, root)
	started := call("pose_delegate_start", map[string]any{"purpose": "review", "target": "spec:alpha", "agent": "slow", "apply": true})
	call("pose_delegate_wait", map[string]any{"run": started["attempt"], "timeout_seconds": 30})
	time.Sleep(500 * time.Millisecond)
	if left, _ := filepath.Glob(filepath.Join(root, ".pose", "review-private", "supervisors", "*.handback*")); len(left) != 0 {
		t.Fatalf("hand-backs left: %v", left)
	}
}
