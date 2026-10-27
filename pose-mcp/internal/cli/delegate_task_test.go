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

// taskRecordOf reads the task record of run id from the state directory.
func taskRecordOf(t *testing.T, state, id string) (map[string]any, string) {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(state, "pose", "delegate", "*", id+".json"))
	if len(matches) != 1 {
		t.Fatalf("record of %s: %v", id, matches)
	}
	raw, _ := os.ReadFile(matches[0])
	var record map[string]any
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	return record, filepath.Dir(matches[0])
}

// Review round 1, B1, B2, F1: the record of a resumed task keeps {session} in
// its argv, never the session; a session printed only on stderr is redacted
// there; a resume keeps the effort the task started with.
func TestDelegateTaskKeepsSessionsOutOfItsRecord(t *testing.T) {
	root, state, prompt := taskInstance(t)
	effortful := `"effortful": {"vendor": "test", "model": "e-1", "timeout": "30s",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; echo 'session: effort-sess-01' >&2; echo \"$@\"; exit 75", "x", "{effort}"]},
      "effort": {"default": "low", "levels": {"low": ["--low"], "high": ["--high"]}},
      "limit": {"exit_codes": [75]}, "session": {"pattern": "session: (\\S+)"},
      "resume": {"command": ["sh", "-c", "cat >/dev/null; echo \"resumed $@\"", "x", "{session}", "{effort}"]}}`
	writeDelegation(t, root, taskPolicy(`"default_access": "read"`, taskWorker, effortful))
	code, out, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "effortful", "--effort", "high", "--json")
	var first map[string]any
	if code != 75 || json.Unmarshal([]byte(out), &first) != nil {
		t.Fatalf("task: %d %s %s", code, out, errOut)
	}
	_, dir := taskRecordOf(t, state, first["run"].(string))
	stderrFile, _ := os.ReadFile(filepath.Join(dir, first["run"].(string)+".stderr"))
	if strings.Contains(string(stderrFile), "effort-sess-01") {
		t.Fatalf("a session on stderr was kept: %s", stderrFile)
	}
	code, out, errOut = runPoseSplit(t, root, "delegate", "resume", first["run"].(string), "--json")
	var second map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &second) != nil || !strings.Contains(second["final"].(string), "--high") {
		t.Fatalf("resume lost its effort: %d %s %s", code, out, errOut)
	}
	record, _ := taskRecordOf(t, state, second["run"].(string))
	raw, _ := json.Marshal(record)
	if strings.Contains(string(raw), "effort-sess-01") || !strings.Contains(string(raw), "{session}") {
		t.Fatalf("the resumed record keeps the session: %s", raw)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "watch", second["run"].(string), "--no-follow"); code != 0 || !strings.Contains(out, second["run"].(string)) {
		t.Fatalf("watch does not replay a task: %d %s", code, out)
	}
}

// Review round 1, B3, B5, B6, B7: the final message obeys output_bytes; a
// task-only agent's continuity is validated before anything runs; resume
// commands per access stand alone; outside Git, declared outside_git_args
// with no place in the template are refused.
func TestDelegateTaskPolicyAndBudgetsHold(t *testing.T) {
	root, _, prompt := taskInstance(t)
	bounded := `"bounded": {"vendor": "test", "model": "b-1", "timeout": "30s",
      "containment": {"schema_version": 1, "profile": "trusted-process", "stream_bytes": 4096, "output_bytes": 16},
      "commands": {"read": ["sh", "-c", "cat >/dev/null; head -c 100 /dev/zero | tr '\\000' x > \"$1\"", "x", "{output}"]}}`
	writeDelegation(t, root, taskPolicy("", bounded))
	if code, out, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "bounded", "--json"); code == 0 || !strings.Contains(out, "output_bytes") {
		t.Fatalf("an output beyond its budget: %d %s %s", code, out, errOut)
	}
	writer := `"writer": {"vendor": "test", "model": "w-2", "timeout": "30s",
      "commands": {"write": ["sh", "-c", "cat >/dev/null; echo 'session: write-sess-1'; exit 75"]},
      "limit": {"exit_codes": [75], "output_patterns": ["["]}}`
	writeDelegation(t, root, taskPolicy("", writer))
	if code, _, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "writer", "--access", "write"); code != 1 || !strings.Contains(errOut, "writer") {
		t.Fatalf("a task-only agent's invalid pattern: %d %s", code, errOut)
	}
	writer = `"writer": {"vendor": "test", "model": "w-2", "timeout": "30s",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; echo read"], "write": ["sh", "-c", "cat >/dev/null; echo 'session: write-sess-1'; exit 75"]},
      "limit": {"exit_codes": [75]}, "session": {"pattern": "session: (\\S+)"},
      "resume": {"commands": {"write": ["sh", "-c", "cat >/dev/null; echo resumed-write", "{session}"]}}}`
	writeDelegation(t, root, taskPolicy("", writer))
	if code, _, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "writer", "--access", "write", "--label", "w"); code != 75 {
		t.Fatalf("write task: %d %s", code, errOut)
	}
	if code, out, errOut := runPoseSplit(t, root, "delegate", "resume", "w"); code != 0 || !strings.Contains(out, "resumed-write") {
		t.Fatalf("resume with resume.commands alone: %d %s %s", code, out, errOut)
	}
	outside := `"outside": {"vendor": "test", "model": "o-1", "timeout": "30s",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; echo ran"]}, "outside_git_args": ["--skip-git"]}`
	writeDelegation(t, root, taskPolicy("", outside))
	if code, _, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "outside", "--cd", t.TempDir()); code != 2 || !strings.Contains(errOut, "{outside_git}") {
		t.Fatalf("outside_git_args with no place: %d %s", code, errOut)
	}
}

// Review round 2, B1, B2, B3: a credential or a session is redacted before
// the stream's tail stands in for the final message; the dry run hides an
// authorized credential; a session printed on both streams is one session.
func TestDelegateTaskRedactsBeforeItCutsAndShows(t *testing.T) {
	root, state, prompt := taskInstance(t)
	t.Setenv("TASK_TOKEN", "tok-0123456789abcdef")
	secretive := `"secretive": {"vendor": "test", "model": "s-2", "timeout": "30s",
      "containment": {"schema_version": 1, "profile": "trusted-process", "credentials": ["TASK_TOKEN"], "output_bytes": 16},
      "commands": {"read": ["sh", "-c", "cat >/dev/null; printf '%s' \"padding-padding-padding-padding $TASK_TOKEN\"", "x", "--token=tok-0123456789abcdef"]}}`
	twice := `"twice": {"vendor": "test", "model": "t-2", "timeout": "30s",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; echo 'session: same-sess-01'; echo 'session: same-sess-01' >&2; echo done"]},
      "session": {"pattern": "session: (\\S+)"}}`
	writeDelegation(t, root, taskPolicy("", secretive, twice))
	code, out, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "secretive", "--json")
	var doc map[string]any
	if json.Unmarshal([]byte(out), &doc) != nil || strings.Contains(out, "0123456789") || strings.Contains(out, "456789abcdef") {
		t.Fatalf("a credential suffix survived the tail: %d %s %s", code, out, errOut)
	}
	if run, _ := doc["run"].(string); run != "" {
		matches, _ := filepath.Glob(filepath.Join(state, "pose", "delegate", "*", run+"*"))
		for _, f := range matches {
			raw, _ := os.ReadFile(f)
			if strings.Contains(string(raw), "456789abcdef") {
				t.Fatalf("%s keeps a credential suffix", f)
			}
		}
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "secretive", "--dry-run", "--json"); code != 0 || strings.Contains(out, "0123456789") || !strings.Contains(out, "[credential]") {
		t.Fatalf("the dry run shows a credential: %d %s", code, out)
	}
	if code, out, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "twice", "--json"); code != 0 || !strings.Contains(out, `"completed"`) || strings.Contains(out, "same-sess-01") {
		t.Fatalf("one session on both streams: %d %s %s", code, out, errOut)
	}
}

// Review round 3: overlapping credentials hide whole in the dry run whatever
// their order (R3-B1); an anchored session pattern still recognizes the
// session (R3-B2); a limit marker in the session's line still stops the run
// (R3-B3); a run that ends before its argv is redacted records none (R3-B4).
func TestDelegateTaskRedactionHoldsOnEveryPath(t *testing.T) {
	root, _, prompt := taskInstance(t)
	t.Setenv("SHORT_TOKEN", "tok-overlap-01")
	t.Setenv("LONG_TOKEN", "tok-overlap-01-sensitive-suffix")
	overlap := `"overlap": {"vendor": "test", "model": "v-1", "timeout": "30s",
      "containment": {"schema_version": 1, "profile": "trusted-process", "credentials": ["SHORT_TOKEN", "LONG_TOKEN"]},
      "commands": {"read": ["sh", "-c", "cat >/dev/null; echo ok", "x", "--token=tok-overlap-01-sensitive-suffix"]}}`
	anchored := `"anchored": {"vendor": "test", "model": "a-1", "timeout": "30s",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; printf 'session: anchored-sess-01'"]},
      "session": {"pattern": "^session: (\\S+)$"}}`
	marked := `"marked": {"vendor": "test", "model": "m-1", "timeout": "30s",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; echo 'session: marked-sess-01'; echo 'session: marked-sess-01 LIMIT' >&2"]},
      "limit": {"output_patterns": ["LIMIT"]}, "session": {"pattern": "session: (\\S+)[^\\n]*"},
      "resume": {"command": ["sh", "-c", "cat >/dev/null; echo resumed", "{session}"]}}`
	literal := `"literal": {"vendor": "test", "model": "l-2", "timeout": "30s",
      "containment": {"schema_version": 1, "profile": "trusted-process", "output_bytes": 16},
      "commands": {"read": ["sh", "-c", "cat >/dev/null; echo \"session: $1\"; head -c 100 /dev/zero | tr '\\000' x > \"$2\"", "x", "literal-sess-01", "{output}"]},
      "session": {"pattern": "session: (\\S+)"}}`
	writeDelegation(t, root, taskPolicy("", overlap, anchored, marked, literal))
	for _, args := range [][]string{{"--dry-run", "--json"}, {"--dry-run"}} {
		code, out, errOut := runPoseSplit(t, root, append([]string{"delegate", "task", prompt, "--agent", "overlap"}, args...)...)
		if code != 0 || strings.Contains(out+errOut, "sensitive-suffix") {
			t.Fatalf("an overlapping credential shows %v: %s %s", args, out, errOut)
		}
	}
	if code, out, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "anchored", "--json"); code != 0 || strings.Contains(out, "anchored-sess-01") {
		t.Fatalf("an anchored session was published: %d %s %s", code, out, errOut)
	}
	if code, out, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "marked", "--json"); !strings.Contains(out, `"interrupted"`) || strings.Contains(out, "marked-sess-01") {
		t.Fatalf("a limit in the session's line was lost: %d %s %s", code, out, errOut)
	}
	code, out, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "literal", "--record", "--json")
	if code == 0 {
		t.Fatalf("an output beyond its budget passed: %s %s", out, errOut)
	}
	records, _ := filepath.Glob(filepath.Join(root, ".pose", "delegate-runs", "trn-*.json"))
	for _, record := range records {
		raw, _ := os.ReadFile(record)
		if strings.Contains(string(raw), "literal-sess-01") {
			t.Fatalf("a failed run recorded a literal session: %s", raw)
		}
	}
}
