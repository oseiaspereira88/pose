package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// taskWorker prints what reached it: the directory, its arguments and its
// prompt; its write command writes a file where it runs.
const taskWorker = `"worker": {"vendor": "other", "model": "w-1", "timeout": "30s",
      "commands": {
        "read": ["sh", "-c", "in=$(cat); echo \"read:$PWD:$in\""],
        "sandboxed": ["sh", "-c", "cat >/dev/null; echo \"sandboxed:$*\"", "x", "{no_network}"],
        "write": ["sh", "-c", "cat >/dev/null; echo made > made.txt; echo \"write:$*\"", "x", "{add_dirs}", "{outside_git}"],
        "full": ["sh", "-c", "cat >/dev/null; echo full"]},
      "outside_git_args": ["--skip-git"], "add_dir_args": ["--add-dir", "{dir}"], "no_network_args": ["--offline"]}`

// taskLimited stops on a usage limit and resumes printing the prompt it got.
const taskLimited = `"limited": {"vendor": "test", "model": "l-1", "timeout": "30s",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; echo 'session: task-sess-001'; echo 'half done'; echo 'LIMIT reset: 2000-01-01T00:00:00Z'; exit 75"]},
      "limit": {"exit_codes": [75], "reset_pattern": "reset: (\\S+)"},
      "session": {"pattern": "session: (\\S+)"},
      "resume": {"command": ["sh", "-c", "in=$(cat); echo \"resumed:$in\"", "{session}"], "max_resumes": 2}}`

func taskPolicy(extra string, agents ...string) string {
	return `{"schema_version": 2, "agents": {` + strings.Join(agents, ",") + `}, "purposes": {"task": {` + extra + `}}}`
}

// taskInstance is a project with the task agents and a private state
// directory, and the path of a prompt file.
func taskInstance(t *testing.T) (string, string, string) {
	t.Helper()
	root := dispatchInstance(t)
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	writeDelegation(t, root, taskPolicy(`"default_access": "read"`, taskWorker, taskLimited))
	prompt := filepath.Join(t.TempDir(), "prompt.txt")
	if err := os.WriteFile(prompt, []byte("hello task"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, state, prompt
}

// R1, R2, R4, R5, R6: a task runs its access's command in --cd with the prompt
// on stdin, prints its final message and returns the agent's exit code; its
// record is owner-only under the state directory, or under .pose with
// --record; outside Git it gets the outside_git_args; full warns.
func TestDelegateTaskRunsWithItsAccessAndRecordsPrivately(t *testing.T) {
	root, state, prompt := taskInstance(t)
	code, out, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "worker")
	if code != 0 || strings.TrimSpace(out) != "read:"+root+":hello task" {
		t.Fatalf("default access: %d %q %s", code, out, errOut)
	}
	files, _ := filepath.Glob(filepath.Join(state, "pose", "delegate", "*", "trn-*"))
	if len(files) < 4 {
		t.Fatalf("record files: %v", files)
	}
	for _, f := range append(files, filepath.Dir(files[0])) {
		info, err := os.Stat(f)
		if err != nil || info.Mode().Perm()&0o077 != 0 {
			t.Fatalf("%s is not owner-only: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "delegate-runs")); err == nil {
		t.Fatal("a task without --record wrote under .pose")
	}
	outside := t.TempDir()
	code, out, errOut = runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "worker", "--access", "write", "--cd", outside, "--add-dir", root)
	if code != 0 || !strings.Contains(out, "--add-dir "+root) || !strings.Contains(out, "--skip-git") {
		t.Fatalf("write outside git: %d %q %s", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(outside, "made.txt")); err != nil {
		t.Fatalf("the task did not run in --cd: %v", err)
	}
	if code, _, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "worker", "--access", "full"); code != 0 || !strings.Contains(errOut, "no sandbox") {
		t.Fatalf("full did not warn: %d %s", code, errOut)
	}
	code, out, _ = runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "worker", "--record", "--label", "kept", "--json")
	var doc map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc["final"] == nil || doc["status"] != "completed" {
		t.Fatalf("json: %d %s", code, out)
	}
	if kept, _ := filepath.Glob(filepath.Join(root, ".pose", "delegate-runs", "trn-*.json")); len(kept) != 1 {
		t.Fatalf("--record: %v", kept)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "ls", "kept", "--json"); code != 0 || !strings.Contains(out, `"purpose": "task"`) {
		t.Fatalf("ls: %d %s", code, out)
	}
}

// R2, R3: --no-network is refused with write or full and reaches a
// sandboxed command through its fragment; a flag an agent's template cannot
// carry is refused with exit 2; full is never a default.
func TestDelegateTaskRefusesWhatItCannotEnforce(t *testing.T) {
	root, _, prompt := taskInstance(t)
	if code, _, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "worker", "--access", "write", "--no-network"); code != 2 || !strings.Contains(errOut, "--no-network") {
		t.Fatalf("no-network with write: %d %s", code, errOut)
	}
	if code, _, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "worker", "--add-dir", root); code != 2 || !strings.Contains(errOut, "add_dir_args") {
		t.Fatalf("add-dir on read: %d %s", code, errOut)
	}
	if code, _, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "limited", "--access", "write"); code != 2 {
		t.Fatalf("an access the agent lacks: %d %s", code, errOut)
	}
	code, out, _ := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "worker", "--access", "sandboxed", "--no-network", "--dry-run", "--json")
	if code != 0 || !strings.Contains(out, `"--offline"`) || !strings.Contains(out, `"reserves_attempt": false`) {
		t.Fatalf("no-network fragment: %d %s", code, out)
	}
	writeDelegation(t, root, taskPolicy(`"default_access": "full"`, taskWorker))
	if code, _, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "worker"); code == 0 || !strings.Contains(errOut, "never a default") {
		t.Fatalf("full as a default: %d %s", code, errOut)
	}
}

// R7, D2: no review reader sees a task run, and attestation refuses it by name.
func TestDelegateTaskIsNeverEvidence(t *testing.T) {
	root, _, prompt := taskInstance(t)
	code, out, _ := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "worker", "--json")
	var doc map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil {
		t.Fatalf("task: %d %s", code, out)
	}
	runID, _ := doc["run"].(string)
	store := posemodel.Store{Root: root}
	if runs, err := store.ListReviewRuns(""); err != nil || len(runs) != 0 {
		t.Fatalf("a review reader read a task run: %v %v", runs, err)
	}
	if _, err := store.LoadProvenanceReviewRun(runID); err == nil || !strings.Contains(err.Error(), "task run") {
		t.Fatalf("provenance read a task run: %v", err)
	}
	code, _, errOut := runPoseSplit(t, root, "review", "attest", "spec:alpha", "--reviewer", "agent:x", "--decision", "approved", "--evidence", "e", "--run", runID)
	if code == 0 || !strings.Contains(errOut, "is a task run") {
		t.Fatalf("attest --run took a task run: %d %s", code, errOut)
	}
}

// R8, R9: a task stopped by a limit lists its ways forward and resumes with
// the default continuation message; a handoff carries the stopped run's
// prompt and section to another agent.
func TestDelegateTaskResumesAndHandsOff(t *testing.T) {
	root, _, prompt := taskInstance(t)
	code, out, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "limited", "--label", "half")
	if code != 75 || !strings.Contains(out, "half done") {
		t.Fatalf("limited task: %d %q %s", code, out, errOut)
	}
	code, out, _ = runPoseSplit(t, root, "delegate", "ls", "half", "--json")
	if code != 0 || !strings.Contains(out, `"action": "resume"`) || !strings.Contains(out, `"action": "handoff"`) {
		t.Fatalf("next actions: %d %s", code, out)
	}
	code, out, errOut = runPoseSplit(t, root, "delegate", "resume", "half")
	if code != 0 || !strings.Contains(out, "resumed:"+posemodel.DelegationTaskContinueMessage[:20]) {
		t.Fatalf("resume: %d %q %s", code, out, errOut)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "wait", "half", "--json"); code != 0 || !strings.Contains(out, `"completed"`) {
		t.Fatalf("wait: %d %s", code, out)
	}
	if code, _, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "limited", "--label", "moved"); code != 75 {
		t.Fatalf("second limited task: %d %s", code, errOut)
	}
	code, out, errOut = runPoseSplit(t, root, "delegate", "handoff", "moved", "--to", "worker")
	if code != 0 || !strings.Contains(out, "hello task") || !strings.Contains(out, "## Handoff from run") || !strings.Contains(out, "> half done") {
		t.Fatalf("handoff: %d %q %s", code, out, errOut)
	}
	runs, _ := posemodel.Store{Root: root}.ReadDelegationTaskRuns()
	stopped := ""
	for _, r := range runs {
		if r.Label == "moved" && r.Status == posemodel.ReviewRunInterrupted {
			stopped = r.RunID
		}
	}
	if _, err := (posemodel.Store{Root: root}).PlanDelegationTaskResume(stopped, ""); err == nil || !strings.Contains(err.Error(), "handed off") {
		t.Fatalf("a handed-off task resumed: %v", err)
	}
	if code, _, _ := runPoseSplit(t, root, "delegate", "resume", "moved"); code == 0 {
		t.Fatal("a completed handoff was resumed")
	}
}
