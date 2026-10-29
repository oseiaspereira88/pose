package cli

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// runBuiltWithStdin runs the built pose with stdin, returning its exit code
// and stdout.
func runBuiltWithStdin(t *testing.T, root, stdin string, args ...string) (int, string) {
	t.Helper()
	cmd := exec.Command(poseBinary(t), args...)
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	code := 0
	if exit, ok := err.(*exec.ExitError); ok {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return code, string(out)
}

type detachedDoc struct {
	Schema, Attempt, Log, Records string
}

func detachedFrom(t *testing.T, code int, out string) detachedDoc {
	t.Helper()
	var doc detachedDoc
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc.Schema != delegateDetachSchema || !strings.HasPrefix(doc.Attempt, "tat-") {
		t.Fatalf("detach: %d %s", code, out)
	}
	return doc
}

// latestTaskOutput is the final message of the latest task run of attempt
// or label.
func latestTaskOutput(t *testing.T, root, ref string) string {
	t.Helper()
	runs, lost := posemodel.Store{Root: root}.ReadDelegationTaskRuns()
	if len(lost) > 0 {
		t.Fatalf("task runs lost: %v", lost)
	}
	output := ""
	for _, run := range runs {
		if run.AttemptID == ref || run.Label == ref {
			output = run.Output
		}
	}
	if output == "" {
		t.Fatalf("no task run of %s: %v", ref, runs)
	}
	return output
}

// spec pose-delegation-task-detach-and-state R1, R2: a task detaches with a
// prompt file and with a piped prompt, returns its attempt and the place of
// its record, and wait, ls and watch resolve it; the piped prompt reaches the
// agent and its spool file is gone once the call returns.
func TestDelegateTaskDetaches(t *testing.T) {
	root, state, prompt := taskInstance(t)
	started := time.Now()
	code, out := runBuilt(t, root, "delegate", "task", prompt, "--agent", "worker", "--label", "bg", "--detach", "--json")
	doc := detachedFrom(t, code, out)
	if time.Since(started) > 30*time.Second {
		t.Fatalf("detach waited for the task: %s", time.Since(started))
	}
	resolved, _ := filepath.EvalSymlinks(state)
	if !strings.HasPrefix(doc.Records, filepath.ToSlash(filepath.Join(resolved, "pose", "delegate"))) {
		t.Fatalf("records of a detached task: %q", doc.Records)
	}
	if code, out := runBuilt(t, root, "delegate", "wait", doc.Attempt, "--json"); code != 0 || !strings.Contains(out, `"state":"completed"`) {
		t.Fatalf("wait: %d %s", code, out)
	}
	if got := latestTaskOutput(t, root, "bg"); !strings.Contains(got, "read:"+root+":hello task") {
		t.Fatalf("detached task output: %q", got)
	}
	if code, out := runBuilt(t, root, "delegate", "watch", doc.Attempt, "--no-follow"); code != 0 || !strings.Contains(out, "(worker, task)") || !strings.Contains(out, "completed") {
		t.Fatalf("watch: %d %s", code, out)
	}

	code, out = runBuiltWithStdin(t, root, "piped prompt", "delegate", "task", "-", "--agent", "worker", "--record", "--detach", "--json")
	piped := detachedFrom(t, code, out)
	if piped.Records != ".pose/delegate-runs/" {
		t.Fatalf("records of a detached --record task: %q", piped.Records)
	}
	if spools, _ := filepath.Glob(filepath.Join(root, ".pose", "review-private", "supervisors", "*.prompt")); len(spools) != 0 {
		t.Fatalf("the prompt spool outlived the call: %v", spools)
	}
	if code, out := runBuilt(t, root, "delegate", "wait", piped.Attempt); code != 0 {
		t.Fatalf("wait piped: %d %s", code, out)
	}
	if got := latestTaskOutput(t, root, piped.Attempt); !strings.Contains(got, ":piped prompt") {
		t.Fatalf("the piped prompt did not reach the detached task: %q", got)
	}
}

// R2: a detached resume of a task sends the prompt piped to its caller.
func TestDelegateTaskDetachedResumeTakesAPipedPrompt(t *testing.T) {
	root, _, prompt := taskInstance(t)
	if code, out := runBuilt(t, root, "delegate", "task", prompt, "--agent", "limited", "--label", "half"); code != 75 {
		t.Fatalf("limited task: %d %s", code, out)
	}
	code, out := runBuiltWithStdin(t, root, "go on from the pipe", "delegate", "resume", "half", "--prompt", "-", "--detach", "--json")
	doc := detachedFrom(t, code, out)
	if code, out := runBuilt(t, root, "delegate", "wait", doc.Attempt); code != 0 {
		t.Fatalf("wait: %d %s", code, out)
	}
	if got := latestTaskOutput(t, root, "half"); !strings.Contains(got, "resumed:go on from the pipe") {
		t.Fatalf("the detached resume did not send the piped prompt: %q", got)
	}
}

// R3, R4: a task whose route must wait for a reset is not detached, and a
// task the policy refuses returns that refusal with its reason, not a log.
func TestDelegateTaskDetachRefusesWhatItCannotStart(t *testing.T) {
	root, _, prompt := taskInstance(t)
	reset := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	limitedLater := `"later": {"vendor": "test", "model": "l-2", "timeout": "30s",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; echo 'LIMIT reset: ` + reset + `'; exit 75"]},
      "limit": {"exit_codes": [75], "reset_pattern": "reset: (\\S+)"}}`
	writeDelegation(t, root, taskPolicy(`"default_access": "read"`, limitedLater))
	// Routing learns a limit from the review runs.
	if code, out := runBuilt(t, root, "delegate", "review", "spec:alpha", "--agent", "later", "--lead", "other"); code == 0 {
		t.Fatalf("limited review: %d %s", code, out)
	}
	code, out := runBuilt(t, root, "delegate", "task", prompt, "--agent", "auto", "--lead", "other", "--detach", "--json")
	if code == 0 || !strings.Contains(out, "does not wait for a reset") {
		t.Fatalf("a task that must wait for a reset was detached: %d %s", code, out)
	}
	writeDelegation(t, root, taskPolicy(`"default_access": "read"`, taskWorker))
	code, out = runBuilt(t, root, "delegate", "task", prompt, "--agent", "worker", "--access", "write", "--no-network", "--detach", "--json")
	if code == 0 || !strings.Contains(out, "--no-network") || strings.Contains(out, "log") {
		t.Fatalf("a refused detached task: %d %s", code, out)
	}
}

// R2: in the foreground, a lone - is the task's prompt from stdin and
// --prompt - the resume's; neither is taken for a flag.
func TestDelegateTaskReadsAPromptFromStdin(t *testing.T) {
	root, _, prompt := taskInstance(t)
	saved := delegateStdin
	defer func() { delegateStdin = saved }()
	delegateStdin = strings.NewReader("from stdin")
	if code, out, errOut := runPoseSplit(t, root, "delegate", "task", "-", "--agent", "worker"); code != 0 || strings.TrimSpace(out) != "read:"+root+":from stdin" {
		t.Fatalf("task -: %d %q %s", code, out, errOut)
	}
	if code, _, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "limited", "--label", "half"); code != 75 {
		t.Fatalf("limited task: %d %s", code, errOut)
	}
	delegateStdin = strings.NewReader("resume from stdin")
	if code, out, errOut := runPoseSplit(t, root, "delegate", "resume", "half", "--prompt", "-"); code != 0 || !strings.Contains(out, "resumed:resume from stdin") {
		t.Fatalf("resume --prompt -: %d %q %s", code, out, errOut)
	}
}
