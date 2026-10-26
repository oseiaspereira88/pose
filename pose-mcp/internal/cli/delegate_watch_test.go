package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// R10: a resumed attempt replays as one task, with a marker at the resume and
// the interrupted run naming the run that continued it.
func TestDelegateWatchReplaysTheChain(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(resumableAgent))
	if code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--label", "chain"); code != 1 || !strings.Contains(out, "interrupted") {
		t.Fatalf("first run: %d %s %s", code, out, errOut)
	}
	if code, _, errOut := runPoseSplit(t, root, "delegate", "resume", "chain"); code != 0 {
		t.Fatalf("resume: %d %s", code, errOut)
	}
	code, out, _ := runPoseSplit(t, root, "delegate", "watch", "chain", "--no-follow")
	first := strings.Index(out, "▶ run ")
	resumed := strings.Index(out, "↻ resumed ")
	if code != 0 || first < 0 || resumed < first || !strings.Contains(out, "interrupted (usage-limit), continued by ") || !strings.Contains(out, "■ completed") {
		t.Fatalf("chain: %d %s", code, out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Fatal("colours on an output that is not a terminal")
	}
	code, out, _ = runPoseSplit(t, root, "delegate", "watch", "chain", "--no-follow", "--json")
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		var doc map[string]any
		if json.Unmarshal([]byte(line), &doc) != nil {
			t.Fatalf("--json is not NDJSON: %q", line)
		}
	}
	if code != 0 || !strings.Contains(out, `"resumes"`) {
		t.Fatalf("json chain: %d %s", code, out)
	}
}

// R11: without a reference, outside a terminal, several candidates exit 2 and
// are listed; at a terminal a numbered menu picks one without fzf, and fzf is
// given a preview of the run under the cursor.
func TestDelegateWatchPicksARun(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(approvingAgent))
	runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--label", "one")
	runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--label", "two", "--reason", "again")
	code, out, _ := runPoseSplit(t, root, "delegate", "watch", "--json")
	if code != 2 || strings.Count(out, "att-") < 2 && !strings.Contains(out, "candidates") {
		t.Fatalf("no reference outside a terminal: %d %s", code, out)
	}
	saved, savedIn, savedPicker := delegateInteractive, delegateStdin, delegatePicker
	defer func() { delegateInteractive, delegateStdin, delegatePicker = saved, savedIn, savedPicker }()
	delegateInteractive = func() bool { return true }
	delegatePicker = "no-such-picker-for-pose"
	delegateStdin = strings.NewReader("2\n")
	code, out, errOut := runPoseSplit(t, root, "delegate", "watch", "--no-follow")
	if code != 0 || !strings.Contains(errOut, "1) ") || !strings.Contains(errOut, "2) ") || !strings.Contains(out, "■ completed") {
		t.Fatalf("menu: %d %s %s", code, out, errOut)
	}
	bin := t.TempDir()
	argsFile := filepath.Join(bin, "args")
	fake := filepath.Join(bin, "fake-fzf")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\nhead -n 1\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	delegatePicker = fake
	code, out, errOut = runPoseSplit(t, root, "delegate", "watch", "--no-follow", "--refresh", "0")
	recorded, _ := os.ReadFile(argsFile)
	if code != 0 || !strings.Contains(string(recorded), "delegate watch {1} --no-follow") || strings.Contains(string(recorded), "while :") || !strings.Contains(out, "■ completed") {
		t.Fatalf("fzf: %d %s %s args=%s", code, out, errOut, recorded)
	}
	runPoseSplit(t, root, "delegate", "watch", "--no-follow")
	recorded, _ = os.ReadFile(argsFile)
	if !strings.Contains(string(recorded), "while :") || !strings.Contains(string(recorded), "sleep 2") {
		t.Fatalf("the preview does not refresh by default: %s", recorded)
	}
}

const streamingAgent = `{"schema_version": 2, "agents": {
  "streamer": {"vendor": "test", "model": "s-1", "cli": "sh", "events": "codex-jsonl", "commands": {"read": ["sh", "-c",
    "cat >/dev/null; echo '{\"type\":\"turn.started\"}'; sleep 1; echo '{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"looking at the bundle\"}}'; sleep 2; echo '{\"type\":\"item.completed\",\"item\":{\"type\":\"agent_message\",\"text\":\"decision: approved\"}}'"]}}
}, "purposes": {"review": {"max_runs_per_bundle": 9}}}`

// R9: watch follows a running run's events live and stops by itself at the
// end, adding from the record only what the live events did not carry.
func TestDelegateWatchFollowsLive(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, streamingAgent)
	code, out := runBuilt(t, root, "delegate", "review", "spec:alpha", "--detach", "--json")
	var detached struct{ Attempt string }
	if code != 0 || json.Unmarshal([]byte(out), &detached) != nil {
		t.Fatalf("detach: %d %s", code, out)
	}
	started := time.Now()
	code, out = runBuilt(t, root, "delegate", "watch", detached.Attempt)
	if code != 0 || !strings.Contains(out, "▶ live: streamer") || strings.Count(out, "looking at the bundle") != 1 || !strings.Contains(out, "■ completed") {
		t.Fatalf("live watch: %d %s", code, out)
	}
	if time.Since(started) > time.Minute {
		t.Fatal("watch did not stop at the end of the run")
	}
	if entries, _ := os.ReadDir(filepath.Join(root, ".pose", "review-private", "live")); len(entries) != 0 {
		t.Fatalf("the live events outlived the record: %v", entries)
	}
}

// R7: on_terminal runs once after the run ends with only the run and its
// state in its environment; its failure is recorded beside the run and does
// not change the run's state.
func TestDelegateOnTerminalHook(t *testing.T) {
	root := dispatchInstance(t)
	t.Setenv("REVIEWER_TOKEN", "tok-hook-1234567890")
	policy := `{"schema_version": 2, "agents": {
  "ok": {"vendor": "test", "model": "ok-1", "cli": "sh", "commands": {"read": ["sh", "-c", "cat >/dev/null; echo decision: approved"]},
    "on_terminal": {"command": ["sh", "-c", "env > hook-env.txt"]}}
}, "purposes": {"review": {"max_runs_per_bundle": 9, "on_terminal": {"command": ["sh", "-c", "exit 3"]}}}}`
	writeDelegation(t, root, policy)
	if code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--label", "hooked"); code != 0 {
		t.Fatalf("run: %d %s %s", code, out, errOut)
	}
	env, err := os.ReadFile(filepath.Join(root, ".pose", "review-runs", "hook-env.txt"))
	if err != nil || !strings.Contains(string(env), "POSE_DELEGATE_RUN=rrn-") || !strings.Contains(string(env), "POSE_DELEGATE_STATE=completed") || strings.Contains(string(env), "tok-hook") || strings.Contains(string(env), "REVIEWER_TOKEN") {
		t.Fatalf("the hook's environment: %s %v", env, err)
	}
	os.Remove(filepath.Join(root, ".pose", "review-runs", "hook-env.txt"))
	// Without an agent hook, the purpose's runs, and its failure is recorded.
	writeDelegation(t, root, strings.Replace(policy, `,
    "on_terminal": {"command": ["sh", "-c", "env > hook-env.txt"]}`, "", 1))
	code, _, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--label", "failing-hook", "--reason", "again")
	if code != 0 || !strings.Contains(errOut, "on_terminal: the hook failed") {
		t.Fatalf("a failing hook: %d %s", code, errOut)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "ls", "failing-hook"); code != 0 || !strings.Contains(out, "state=completed") || !strings.Contains(out, "on_terminal_failed=") {
		t.Fatalf("ls: %d %s", code, out)
	}
	if code, _, _ := runPoseSplit(t, root, "delegate", "wait", "failing-hook"); code != 0 {
		t.Fatalf("a failing hook changed the wait: %d", code)
	}
	for _, bad := range []string{`{"command": []}`, `{"command": ["sh", "{output}"]}`, `{"command": ["sh"], "timeout": "10m"}`} {
		writeDelegation(t, root, `{"schema_version": 2, "agents": {}, "purposes": {"review": {"on_terminal": `+bad+`}}}`)
		if code, _, _ := runPoseSplit(t, root, "delegate", "agents", "--json"); code != 1 {
			t.Fatalf("hook accepted: %s", bad)
		}
	}
}
