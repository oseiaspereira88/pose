package cli

import (
	"encoding/json"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// limitingAgent stops on a usage limit after printing a partial draft that
// carries a decision line; continuingAgent approves only when its brief
// carries the handoff section.
const limitingAgent = `"stopper": {"vendor": "test", "model": "s-1",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; echo 'session: sess-stop-001'; echo 'decision: approved'; echo 'LIMIT reset: 2099-01-01T00:00:00Z'; exit 75"]},
      "limit": {"exit_codes": [75], "reset_pattern": "reset: (\\S+)"},
      "session": {"pattern": "session: (\\S+)"},
      "resume": {"command": ["sh", "-c", "cat >/dev/null; echo decision: approved", "{session}"], "max_resumes": 2}}`

const continuingAgent = `"continuer": {"vendor": "other", "model": "c-1",
      "commands": {"read": ["sh", "-c", "in=$(cat); printf '%s' \"$in\" | grep -q '## Handoff from run' && printf '%s' \"$in\" | grep -q '^> decision: approved' && echo decision: changes-requested"]}}`

func handoffInstance(t *testing.T) string {
	t.Helper()
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(limitingAgent, continuingAgent))
	if code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", "stopper", "--label", "stopped"); code != 1 || !strings.Contains(out, "interrupted") {
		t.Fatalf("first run: %d %s %s", code, out, errOut)
	}
	return root
}

// R1–R6: a stopped run is handed off to another agent under the same
// attempt; the brief carries the quoted partial, only the handoff run
// concludes, a repeated handoff returns the recorded run, and the
// predecessor can no longer be resumed.
func TestDelegateHandoffContinuesTheAttemptWithAnotherAgent(t *testing.T) {
	root := handoffInstance(t)
	code, out, errOut := runPoseSplit(t, root, "delegate", "handoff", "stopped", "--to", "continuer", "--json")
	var result map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &result) != nil || result["status"] != "completed" {
		t.Fatalf("handoff: %d %s %s", code, out, errOut)
	}
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	if len(runs) != 2 {
		t.Fatalf("runs: %+v", runs)
	}
	first, second := runs[0], runs[1]
	if first.HandsOffFrom != "" {
		first, second = second, first
	}
	if second.HandsOffFrom != first.RunID || second.AttemptID != first.AttemptID || second.Adapter != "continuer" || second.HandoffDigest == "" || second.Resumes != "" {
		t.Fatalf("handoff run: %+v", second)
	}
	// Only the handoff run concludes: its own draft requests changes, the
	// predecessor's quoted approval does not count.
	if code, out, _ := runPoseSplit(t, root, "delegate", "wait", "stopped", "--json"); code != 10 || !strings.Contains(out, `"decision":"changes-requested"`) {
		t.Fatalf("wait: %d %s", code, out)
	}
	code, out, _ = runPoseSplit(t, root, "delegate", "handoff", "stopped", "--to", "continuer", "--json")
	if !strings.Contains(out, second.RunID) {
		t.Fatalf("a repeated handoff did not return the recorded run: %d %s", code, out)
	}
	if code, _, errOut := runPoseSplit(t, root, "delegate", "resume", first.RunID); code == 0 {
		t.Fatalf("a handed-off run was resumed: %s", errOut)
	}
	if findings, err := (posemodel.Store{Root: root}).ReviewRunHistoryFindings(); err != nil || len(findings) > 0 {
		t.Fatalf("history after a handoff: %v %v", findings, err)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "watch", "stopped", "--no-follow"); code != 0 || !strings.Contains(out, "⇄ handed off "+first.RunID) || !strings.Contains(out, "continued by "+second.RunID) {
		t.Fatalf("watch: %d %s", code, out)
	}
}

// R1, R2: a completed run and the same agent are refused, and alternating
// agents stop at the continuation budget of the agent that opened the
// attempt, even when a successor declares a larger one or none.
func TestDelegateHandoffRefusesAndIsBounded(t *testing.T) {
	root := handoffInstance(t)
	if code, out, _ := runPoseSplit(t, root, "delegate", "handoff", "stopped", "--to", "stopper"); code != 1 || !strings.Contains(out+"", "") {
		t.Fatalf("the same agent: %d", code)
	}
	second := strings.Replace(limitingAgent, `"stopper"`, `"stopper-b"`, 1)
	second = strings.Replace(second, `"vendor": "test"`, `"vendor": "other"`, 1)
	second = strings.Replace(second, `"max_resumes": 2`, `"max_resumes": 10`, 1)
	first := strings.Replace(limitingAgent, `"max_resumes": 2`, `"max_resumes": 1`, 1)
	writeDelegation(t, root, delegationPolicy(first, second, continuingAgent))
	if code, out, _ := runPoseSplit(t, root, "delegate", "handoff", "stopped", "--to", "stopper-b"); code != 1 || !strings.Contains(out, "interrupted") {
		t.Fatalf("first handoff: %d %s", code, out)
	}
	// continuer declares no budget and stopper-b ten; the attempt's is one.
	code, _, errOut := runPoseSplit(t, root, "delegate", "handoff", "stopped", "--to", "continuer")
	if code != 1 || !strings.Contains(errOut, "continuation budget is spent") {
		t.Fatalf("a second handoff past the budget: %d %s", code, errOut)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "ls", "stopped", "--json"); code != 0 || strings.Contains(out, `"available": true,
          "command": "pose delegate handoff`) || !strings.Contains(out, "continuation budget is spent") {
		t.Fatalf("an unexecutable handoff offered: %s", out)
	}
	writeDelegation(t, root, delegationPolicy(approvingAgent, continuingAgent))
	runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", "ok", "--label", "done", "--reason", "a concluded run")
	if code, _, errOut := runPoseSplit(t, root, "delegate", "handoff", "done", "--to", "continuer"); code != 1 || !strings.Contains(errOut, "completed") {
		t.Fatalf("a completed run handed off: %d %s", code, errOut)
	}
}

// R7–R9: --to auto skips the stopped agent; a stopped run lists its next
// actions in JSON and, at a terminal, hands off when asked.
func TestDelegateStoppedRunsOfferTheirWaysForward(t *testing.T) {
	root := handoffInstance(t)
	code, out, _ := runPoseSplit(t, root, "delegate", "ls", "stopped", "--json")
	if code != 0 || !strings.Contains(out, `"action": "handoff"`) || !strings.Contains(out, `"action": "wait"`) {
		t.Fatalf("next actions: %d %s", code, out)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "handoff", "stopped", "--to", "auto", "--dry-run", "--json"); code != 0 || !strings.Contains(out, `"agent": "continuer"`) {
		t.Fatalf("auto: %d %s", code, out)
	}
	saved, savedIn := delegateInteractive, delegateStdin
	defer func() { delegateInteractive, delegateStdin = saved, savedIn }()
	delegateInteractive = func() bool { return true }
	delegateStdin = strings.NewReader("h\n")
	code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", "stopper", "--label", "asked", "--reason", "again")
	if !strings.Contains(errOut, "[h] pose delegate handoff") || !strings.Contains(out, "hands_off_from") {
		t.Fatalf("terminal choices: %d %s %s", code, out, errOut)
	}
}

// R8, R9: the ways forward are judged as their commands would judge them —
// a wait past the bound is not offered — cancelling is offered even as the
// only choice, the stopping command's own JSON result carries them, and a
// handoff that stops at a terminal asks again.
func TestDelegateStoppedRunActionsAreExecutable(t *testing.T) {
	root := handoffInstance(t)
	code, out, _ := runPoseSplit(t, root, "delegate", "wait", "stopped", "--json")
	var waited struct {
		Next []delegateNextAction `json:"next_actions"`
	}
	if json.Unmarshal([]byte(out), &waited) != nil {
		t.Fatalf("wait: %d %s", code, out)
	}
	byName := map[string]delegateNextAction{}
	for _, a := range waited.Next {
		byName[a.Action] = a
	}
	if w := byName["wait"]; w.Available || !strings.Contains(w.Why, "maximum wait") {
		t.Fatalf("a wait past its bound offered: %+v", w)
	}
	if c := byName["cancel"]; !c.Available || c.Command != "" || !byName["handoff"].Available {
		t.Fatalf("actions: %+v", waited.Next)
	}
	code, out, _ = runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", "stopper", "--label", "again", "--reason", "again", "--json")
	if code != 1 || !strings.Contains(out, `"next_actions"`) || !strings.Contains(out, `"action": "cancel"`) {
		t.Fatalf("the stopping run's result: %d %s", code, out)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", "stopper", "--label", "plain", "--reason", "plain"); code != 1 || !strings.Contains(out, "delegate.next.handoff=pose delegate handoff") {
		t.Fatalf("the plain result: %d %s", code, out)
	}
	saved, savedIn := delegateInteractive, delegateStdin
	defer func() { delegateInteractive, delegateStdin = saved, savedIn }()
	delegateInteractive = func() bool { return true }
	other := strings.Replace(limitingAgent, `"stopper"`, `"stopper-b"`, 1)
	writeDelegation(t, root, delegationPolicy(limitingAgent, strings.Replace(other, `"vendor": "test"`, `"vendor": "other"`, 1)))
	delegateStdin = strings.NewReader("c\n")
	if code, out, errOut := runPoseSplit(t, root, "delegate", "handoff", "stopped", "--to", "stopper-b"); code != 1 || !strings.Contains(out, "interrupted") || !strings.Contains(errOut, "[c] ") {
		t.Fatalf("a stopped handoff did not ask: %d %s %s", code, out, errOut)
	}
	// One agent, no session to resume and none to hand off to: cancelling is
	// still offered.
	root = dispatchInstance(t)
	lone := strings.Replace(limitingAgent, "echo 'session: sess-stop-001'; ", "", 1)
	writeDelegation(t, root, delegationPolicy(lone))
	delegateStdin = strings.NewReader("c\n")
	if code, _, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", "stopper", "--label", "lone"); code != 1 || !strings.Contains(errOut, "[c] ") {
		t.Fatalf("cancelling not offered alone: %d %s", code, errOut)
	}
}

// F01, F02: repeating a handoff with --to auto returns the recorded
// continuation; naming another agent is refused.
func TestDelegateHandoffAutoRetryReturnsTheContinuation(t *testing.T) {
	root := handoffInstance(t)
	runs, _ := posemodel.Store{Root: root}.ListReviewRuns("")
	first := runs[0].RunID
	if code, out, errOut := runPoseSplit(t, root, "delegate", "handoff", first, "--to", "auto", "--json"); code != 0 {
		t.Fatalf("handoff: %d %s %s", code, out, errOut)
	}
	code, out, errOut := runPoseSplit(t, root, "delegate", "handoff", first, "--to", "auto", "--json")
	if code != 0 || !strings.Contains(out, `"hands_off_from": "`+first) {
		t.Fatalf("an auto retry: %d %s %s", code, out, errOut)
	}
	// Another agent named explicitly is refused, naming the continuation.
	if code, out, errOut := runPoseSplit(t, root, "delegate", "handoff", first, "--to", "stopper"); code != 1 || !strings.Contains(out+errOut, "already handed off to continuer") {
		t.Fatalf("a conflicting destination: %d %s %s", code, out, errOut)
	}
	if runs, _ := (posemodel.Store{Root: root}).ListReviewRuns(""); len(runs) != 2 {
		t.Fatalf("runs: %d", len(runs))
	}
}
