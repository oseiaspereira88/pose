package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
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
// agent and no file ever holds it.
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
		t.Fatalf("the piped prompt was written to a file: %v", spools)
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
	refused := []string{"delegate", "task", prompt, "--agent", "worker", "--access", "write", "--no-network"}
	for _, mode := range [][]string{{"--json"}, nil} {
		fgCode, fgOut := runBuilt(t, root, append(append([]string{}, refused...), mode...)...)
		code, out = runBuilt(t, root, append(append(append([]string{}, refused...), mode...), "--detach")...)
		if fgCode != 2 || code != fgCode || out != fgOut || strings.Contains(out, ".log") {
			t.Fatalf("a refused detached task %v: foreground %d %q, detached %d %q", mode, fgCode, fgOut, code, out)
		}
	}
	writeDelegation(t, root, `{"schema_version": 2, "agents": {}}`)
	none := []string{"delegate", "task", prompt, "--json"}
	fgCode, fgOut := runBuilt(t, root, none...)
	code, out = runBuilt(t, root, append(none, "--detach")...)
	if fgCode != 1 || code != fgCode || out != fgOut {
		t.Fatalf("no agent: foreground %d %q, detached %d %q", fgCode, fgOut, code, out)
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

// bigPrompt is larger than any pipe buffer.
func bigPrompt() string {
	return strings.Repeat("0123456789abcdef", 1<<16) // 1 MiB
}

// R2, R10 (review round 2, F2F3-R2-01): a piped prompt larger than a pipe
// buffer reaches a detached task whole.
func TestDelegateTaskDetachedPromptArrivesWhole(t *testing.T) {
	root, _, _ := taskInstance(t)
	writeDelegation(t, root, taskPolicy(`"default_access": "read"`, `"sink": {"vendor": "other", "model": "s-1", "timeout": "30s", "commands": {"read": ["sh", "-c", "cat >/dev/null; echo done"]}}`))
	prompt := bigPrompt()[:1<<20-1]
	code, out := runBuiltWithStdin(t, root, prompt, "delegate", "task", "-", "--agent", "sink", "--detach", "--json")
	doc := detachedFrom(t, code, out)
	if code, out := runBuilt(t, root, "delegate", "wait", doc.Attempt); code != 0 {
		t.Fatalf("wait: %d %s", code, out)
	}
	store := posemodel.Store{Root: root}
	runs, _ := store.ReadDelegationTaskRuns()
	if len(runs) != 1 {
		t.Fatalf("runs: %v", runs)
	}
	if got, err := store.ReadDelegationTaskPrompt(runs[0]); err != nil || got != prompt {
		t.Fatalf("the detached task got %d of %d prompt bytes: %v", len(got), len(prompt), err)
	}
}

// R10 (F2F3-R2-01): a supervisor directory another user could write is
// refused before any run starts.
func TestDelegateTaskDetachRefusesAWritableSupervisorDirectory(t *testing.T) {
	root, _, prompt := taskInstance(t)
	supervisors := filepath.Join(root, ".pose", "review-private", "supervisors")
	if err := os.MkdirAll(supervisors, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{supervisors, filepath.Dir(supervisors)} {
		if err := os.Chmod(dir, 0o777); err != nil {
			t.Fatal(err)
		}
		code, out := runBuilt(t, root, "delegate", "task", prompt, "--agent", "worker", "--detach", "--json")
		if code != 1 || !strings.Contains(out, "another user can write "+dir) {
			t.Fatalf("a writable %s: %d %s", dir, code, out)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if runs, _ := (posemodel.Store{Root: root}).ReadDelegationTaskRuns(); len(runs) != 0 {
		t.Fatalf("a refused detach ran: %v", runs)
	}
}

// R10 (F2F3-R2-01): a hand-back published before the run read its prompt —
// what a writer of the supervisor directory could forge — is not believed:
// the caller waits for the prompt to go through, finds no run reserved the
// attempt, stops the run and reports it; no run completes on a cut prompt.
func TestDelegateTaskDetachDoesNotBelieveAForgedHandback(t *testing.T) {
	root, _, _ := taskInstance(t)
	wrapper := filepath.Join(t.TempDir(), "pose-forging")
	script := "#!/bin/sh\nprintf '%s\\n' '{\"attempt\":\"tat-forged\"}' > \"$POSE_DELEGATE_HANDBACK\"\nsleep 1\nexec '" + poseBinary(t) + "' \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	savedExe, savedIn := delegateExecutable, delegateStdin
	defer func() { delegateExecutable, delegateStdin = savedExe, savedIn }()
	delegateExecutable = func() (string, error) { return wrapper, nil }
	delegateStdin = strings.NewReader(bigPrompt()[:1<<20-1])
	code, out, errOut := runPoseSplit(t, root, "delegate", "task", "-", "--agent", "worker", "--detach", "--json")
	if code != 1 || !strings.Contains(out, "which no run reserved") || strings.Contains(out, delegateDetachSchema) {
		t.Fatalf("a forged hand-back was believed: %d %s %s", code, out, errOut)
	}
	runs, _ := posemodel.Store{Root: root}.ReadDelegationTaskRuns()
	for _, run := range runs {
		if run.Status == posemodel.ReviewRunCompleted {
			t.Fatalf("a run completed after a forged hand-back: %+v", run)
		}
	}

	// A forged hand-back naming a real attempt is believed, but only once the
	// prompt is through: the run it started never runs a cut prompt.
	writeDelegation(t, root, taskPolicy(`"default_access": "read"`, `"sink": {"vendor": "other", "model": "s-1", "timeout": "30s", "commands": {"read": ["sh", "-c", "cat >/dev/null; echo done"]}}`))
	delegateExecutable = savedExe
	delegateStdin = strings.NewReader("first")
	code, out, _ = runPoseSplit(t, root, "delegate", "task", "-", "--agent", "sink", "--json")
	var first map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &first) != nil {
		t.Fatalf("a first task: %d %s", code, out)
	}
	real, _ := first["attempt"].(string)
	script = strings.Replace(script, "tat-forged", real, 1)
	if err := os.WriteFile(wrapper, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	delegateExecutable = func() (string, error) { return wrapper, nil }
	prompt := bigPrompt()[:1<<20-1]
	delegateStdin = strings.NewReader(prompt)
	started := time.Now()
	runPoseSplit(t, root, "delegate", "task", "-", "--agent", "sink", "--detach", "--json")
	// The run starts reading a second after the forged hand-back: a caller
	// that returned sooner would have exited with the pipe cut short.
	if elapsed := time.Since(started); elapsed < 900*time.Millisecond {
		t.Fatalf("the caller returned %s after a forged hand-back, before the run read its prompt", elapsed)
	}
	store := posemodel.Store{Root: root}
	deadline := time.Now().Add(30 * time.Second)
	for {
		runs, _ = store.ReadDelegationTaskRuns()
		if len(runs) >= 2 && !store.DelegationActiveMarkerExists(runs[len(runs)-1].AttemptID) || time.Now().After(deadline) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	for _, run := range runs {
		if run.AttemptID == real || run.Status != posemodel.ReviewRunCompleted {
			continue
		}
		if got, err := store.ReadDelegationTaskPrompt(run); err != nil || got != prompt {
			t.Fatalf("a run completed on %d of %d prompt bytes: %v", len(got), len(prompt), err)
		}
	}
}

// swappingReader retargets a link the first time it is read: the moment a
// writer of the link's directory could choose, after the supervisor
// directory was checked.
type swappingReader struct {
	r            io.Reader
	link, target string
	swapped      bool
}

func (s *swappingReader) Read(p []byte) (int, error) {
	if !s.swapped {
		s.swapped = true
		if err := os.Remove(s.link); err == nil {
			_ = os.Symlink(s.target, s.link)
		}
	}
	return s.r.Read(p)
}

// R10 (review round 3, F2F3-R3-01): a root reached through a link that
// another user could retarget is resolved once; the log, the hand-back, the
// run's own root and the reservation all stay in the tree that was checked.
func TestDelegateTaskDetachStaysInTheCheckedRoot(t *testing.T) {
	root, _, _ := taskInstance(t)
	writeDelegation(t, root, taskPolicy(`"default_access": "read"`, `"sink": {"vendor": "other", "model": "s-1", "timeout": "30s", "commands": {"read": ["sh", "-c", "cat >/dev/null; echo done"]}}`))
	public := filepath.Join(t.TempDir(), "public")
	if err := os.Mkdir(public, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(public, 0o777); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(public, "project")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	elsewhere := t.TempDir()
	t.Setenv(posemodel.DelegateRootEnv, link)
	savedExe, savedIn := delegateExecutable, delegateStdin
	defer func() { delegateExecutable, delegateStdin = savedExe, savedIn }()
	built := poseBinary(t)
	delegateExecutable = func() (string, error) { return built, nil }
	delegateStdin = &swappingReader{r: strings.NewReader("checked tree"), link: link, target: elsewhere}
	// The caller stands outside the project, as an MCP host does: only the
	// pinned root tells the run which project it is in.
	code, out, errOut := runPoseSplit(t, t.TempDir(), "delegate", "task", "-", "--agent", "sink", "--detach", "--json")
	doc := detachedFrom(t, code, out+errOut)
	if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
		t.Fatalf("the retargeted tree was written: %v", entries)
	}
	t.Setenv(posemodel.DelegateRootEnv, root)
	if code, out := runBuilt(t, root, "delegate", "wait", doc.Attempt); code != 0 {
		t.Fatalf("the run in the checked root: %d %s", code, out)
	}
	if got := latestTaskOutput(t, root, doc.Attempt); !strings.Contains(got, "done") {
		t.Fatalf("output: %q", got)
	}
}

// R1, R10 (review round 4, F2F3-R4-01): through a stable link to the root,
// a detached task and a foreground one keep one private history: the caller
// waits on what it detached, and resumes a foreground task detached.
func TestDelegateTaskHistoryIsOneThroughARootLink(t *testing.T) {
	root, _, prompt := taskInstance(t)
	link := filepath.Join(t.TempDir(), "project")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv(posemodel.DelegateRootEnv, link)
	code, out := runBuilt(t, root, "delegate", "task", prompt, "--agent", "worker", "--detach", "--json")
	doc := detachedFrom(t, code, out)
	if code, out := runBuilt(t, root, "delegate", "wait", doc.Attempt, "--json"); code != 0 || !strings.Contains(out, `"state":"completed"`) {
		t.Fatalf("wait through the link: %d %s", code, out)
	}
	if code, out := runBuilt(t, root, "delegate", "task", prompt, "--agent", "limited", "--label", "half"); code != 75 {
		t.Fatalf("a foreground task through the link: %d %s", code, out)
	}
	code, out = runBuilt(t, root, "delegate", "resume", "half", "--detach", "--json")
	resumed := detachedFrom(t, code, out)
	if code, out := runBuilt(t, root, "delegate", "wait", resumed.Attempt); code != 0 {
		t.Fatalf("the detached resume through the link: %d %s", code, out)
	}
	t.Setenv(posemodel.DelegateRootEnv, root)
	if code, out := runBuilt(t, root, "delegate", "ls", "half", "--json"); code != 0 || !strings.Contains(out, `"state": "completed"`) {
		t.Fatalf("the same history from the resolved root: %d %s", code, out)
	}
}

// R10 (F2F3-R4-02): a caller whose directory is gone is refused, never
// started in the root or anywhere else.
func TestDelegateTaskDetachRefusesAMissingCurrentDirectory(t *testing.T) {
	root, _, prompt := taskInstance(t)
	built := poseBinary(t)
	saved := delegateExecutable
	defer func() { delegateExecutable = saved }()
	delegateExecutable = func() (string, error) { return built, nil }
	gone := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(gone); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PWD", gone)
	t.Setenv(posemodel.DelegateRootEnv, root)
	var out, errB strings.Builder
	code := Main([]string{"delegate", "task", prompt, "--agent", "worker", "--detach", "--json"}, &out, &errB)
	if code != 1 || !strings.Contains(out.String(), "current directory cannot be read") {
		t.Fatalf("a missing current directory: %d %s %s", code, out.String(), errB.String())
	}
	if runs, _ := (posemodel.Store{Root: root}).ReadDelegationTaskRuns(); len(runs) != 0 {
		t.Fatalf("a task ran from a missing directory: %v", runs)
	}
}

// R11 (review round 5, F2F3-R5-01): a task an earlier engine recorded under
// a link's spelling of the root is continued through that link, detached as
// in the foreground: resume and handoff find their predecessor, claim it
// beside its record and record the new run under the resolved root.
func TestDelegateTaskDetachedContinuationFindsTheLegacyHistory(t *testing.T) {
	for _, verb := range []string{"resume", "handoff"} {
		t.Run(verb, func(t *testing.T) {
			root, state, prompt := taskInstance(t)
			link := filepath.Join(t.TempDir(), "project")
			if err := os.Symlink(root, link); err != nil {
				t.Fatal(err)
			}
			t.Setenv(posemodel.DelegateRootEnv, link)
			if code, out := runBuilt(t, root, "delegate", "task", prompt, "--agent", "limited", "--label", "half"); code != 75 {
				t.Fatalf("a limited task: %d %s", code, out)
			}
			// Move the record where an earlier engine kept it: keyed by the
			// link's spelling, not the resolved root.
			resolvedSum := sha256.Sum256([]byte(mustResolve(t, root)))
			linkSum := sha256.Sum256([]byte(link))
			base := filepath.Join(mustResolve(t, state), "pose", "delegate")
			current, legacy := filepath.Join(base, hex.EncodeToString(resolvedSum[:])[:16]), filepath.Join(base, hex.EncodeToString(linkSum[:])[:16])
			if err := os.Rename(current, legacy); err != nil {
				t.Fatal(err)
			}
			if code, out := runBuilt(t, root, "delegate", "ls", "half", "--json"); code != 0 || !strings.Contains(out, `"action": "`+verb+`"`) {
				t.Fatalf("ls through the link: %d %s", code, out)
			}
			args := []string{"delegate", verb, "half", "--detach", "--json"}
			if verb == "handoff" {
				args = []string{"delegate", "handoff", "half", "--to", "worker", "--detach", "--json"}
			}
			code, out := runBuilt(t, root, args...)
			doc := detachedFrom(t, code, out)
			if code, out := runBuilt(t, root, "delegate", "wait", doc.Attempt); code != 0 {
				t.Fatalf("wait: %d %s", code, out)
			}
			if claims, _ := filepath.Glob(filepath.Join(legacy, "*.continued")); len(claims) != 1 {
				t.Fatalf("the claim beside the legacy predecessor: %v", claims)
			}
			if fresh, _ := filepath.Glob(filepath.Join(current, "trn-*.json")); len(fresh) != 1 {
				t.Fatalf("the continuation under the resolved root: %v", fresh)
			}
			// A retry answers with the recorded continuation, or refuses; it
			// never runs again.
			runBuilt(t, root, args...)
			if fresh, _ := filepath.Glob(filepath.Join(current, "trn-*.json")); len(fresh) != 1 {
				t.Fatalf("a retried %s ran again: %v", verb, fresh)
			}
		})
	}
}

func mustResolve(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// R11: only a detached run takes its caller's spelling of the root; a stray
// value elsewhere is ignored.
func TestOnlyADetachedRunTakesItsCallersRootSpelling(t *testing.T) {
	defer func() { posemodel.DetachedCallerRoot = "" }()
	t.Setenv(posemodel.DelegateLegacyRootEnv, "/elsewhere/project")
	t.Setenv(delegateHandback, "")
	readDetachedCallerRoot()
	if posemodel.DetachedCallerRoot != "" {
		t.Fatalf("a foreground command took %q", posemodel.DetachedCallerRoot)
	}
	t.Setenv(delegateHandback, filepath.Join(t.TempDir(), "h"))
	readDetachedCallerRoot()
	if posemodel.DetachedCallerRoot != "/elsewhere/project" {
		t.Fatalf("a detached run did not take its caller's spelling: %q", posemodel.DetachedCallerRoot)
	}
}

// R4, R11 (review round 6, F2F3-R6-01): a legacy-root value inherited from
// the caller's environment never reaches a detached run: another project's
// task is refused detached as it is in the foreground.
func TestDelegateTaskDetachIgnoresAnInheritedLegacyRoot(t *testing.T) {
	other, _, prompt := taskInstance(t)
	state := os.Getenv("XDG_STATE_HOME")
	t.Setenv(posemodel.DelegateRootEnv, other)
	if code, out := runBuilt(t, other, "delegate", "task", prompt, "--agent", "limited", "--label", "foreign"); code != 75 {
		t.Fatalf("the other project's task: %d %s", code, out)
	}
	runs, _ := posemodel.Store{Root: other}.ReadDelegationTaskRuns()
	if len(runs) != 1 {
		t.Fatalf("runs: %v", runs)
	}
	foreign := runs[0].RunID
	root, _, _ := taskInstance(t)
	t.Setenv("XDG_STATE_HOME", state)
	t.Setenv(posemodel.DelegateRootEnv, root)
	t.Setenv(posemodel.DelegateLegacyRootEnv, other)
	fgCode, fgOut := runBuilt(t, root, "delegate", "handoff", foreign, "--to", "worker", "--json")
	code, out := runBuilt(t, root, "delegate", "handoff", foreign, "--to", "worker", "--detach", "--json")
	if fgCode != 2 || code != fgCode || out != fgOut {
		t.Fatalf("another project's task: foreground %d %q, detached %d %q", fgCode, fgOut, code, out)
	}
	if claims, _ := filepath.Glob(filepath.Join(filepath.Dir(runs[0].Path), "*.continued")); len(claims) != 0 {
		t.Fatalf("another project's task was claimed: %v", claims)
	}
}

// F2F3-R6-01: a detached run's control variables are never inherited.
func TestADetachedRunInheritsNoControlVariable(t *testing.T) {
	env := withoutDelegateControl([]string{"PATH=/bin", delegateHandback + "=x", delegatePromptPiped + "=1", posemodel.DelegateRootEnv + "=/a", posemodel.DelegateLegacyRootEnv + "=/b", "HOME=/h"})
	if strings.Join(env, " ") != "PATH=/bin HOME=/h" {
		t.Fatalf("inherited: %v", env)
	}
}

// Spec pose-delegation-parity-gate SA-04 (review round 1, PG-R1-02): write
// access runs the agent's write command as declared, its automatic approver
// included: the agent asks the approver before it writes, the approver
// answers, and the record keeps the approver's invocation; read access
// carries none of it.
func TestDelegateTaskWriteRunsTheAgentsApprover(t *testing.T) {
	root, _, prompt := taskInstance(t)
	approver := filepath.Join(t.TempDir(), "approver")
	asked := filepath.Join(t.TempDir(), "asked")
	if err := os.WriteFile(approver, []byte("#!/bin/sh\necho \"$*\" >> '"+asked+"'\n[ \"$1\" = write ] && echo approved\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	approving := `"approving": {"vendor": "other", "model": "a-1", "timeout": "30s",
      "commands": {
        "read": ["sh", "-c", "cat >/dev/null; echo read-only"],
        "write": ["sh", "-c", "cat >/dev/null; if [ \"$(\"$2\" write made.txt)\" = approved ]; then echo made > made.txt; echo wrote; else echo refused; fi", "x", "--approver", "` + approver + `"]}}`
	writeDelegation(t, root, taskPolicy(`"default_access": "read"`, approving))
	dir := t.TempDir()
	code, out, errOut := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "approving", "--access", "write", "--cd", dir, "--json")
	var doc map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || !strings.Contains(fmt.Sprint(doc["final"]), "wrote") {
		t.Fatalf("write with an approver: %d %s %s", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(dir, "made.txt")); err != nil {
		t.Fatalf("the approved write did not happen: %v", err)
	}
	if raw, _ := os.ReadFile(asked); strings.TrimSpace(string(raw)) != "write made.txt" {
		t.Fatalf("the approver was asked %q", raw)
	}
	runs, _ := posemodel.Store{Root: root}.ReadDelegationTaskRuns()
	if len(runs) != 1 || !strings.Contains(strings.Join(runs[0].Argv, " "), "--approver "+approver) {
		t.Fatalf("the record does not keep the approver's invocation: %+v", runs)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "task", prompt, "--agent", "approving", "--dry-run", "--json"); code != 0 || strings.Contains(out, "--approver") {
		t.Fatalf("read access carries the approver: %d %s", code, out)
	}
}

// Spec pose-delegation-parity-gate SA-07 (review round 1, PG-R1-01): a task
// continues a session POSE never recorded — one a personal launcher opened —
// with the named agent's resume command for the access, the default
// continuation message unless a prompt is given; the session never reaches
// the record, and a session without a named agent, of the wrong shape or for
// an agent with no resume command is refused with exit 2.
func TestDelegateTaskContinuesASessionPOSENeverRecorded(t *testing.T) {
	root, _, prompt := taskInstance(t)
	got := filepath.Join(t.TempDir(), "session")
	resuming := `"resuming": {"vendor": "test", "model": "r-1", "timeout": "30s",
      "commands": {"read": ["sh", "-c", "cat >/dev/null; echo fresh"]},
      "session": {"pattern": "session: (\\S+)"},
      "resume": {"command": ["sh", "-c", "in=$(cat); printf %s \"$0\" > '` + got + `'; echo \"resumed:$in\"", "{session}"]}}`
	writeDelegation(t, root, taskPolicy(`"default_access": "read"`, resuming, taskWorker))
	code, out, errOut := runPoseSplit(t, root, "delegate", "task", "--session", "launcher-sess-0042", "--agent", "resuming", "--json")
	var doc map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc["resumes_session"] != true ||
		!strings.Contains(fmt.Sprint(doc["final"]), "resumed:"+posemodel.DelegationTaskContinueMessage[:20]) {
		t.Fatalf("a launcher's session: %d %s %s", code, out, errOut)
	}
	if raw, _ := os.ReadFile(got); string(raw) != "launcher-sess-0042" {
		t.Fatalf("the agent resumed %q", raw)
	}
	if code, out, _ := runPoseSplit(t, root, "delegate", "task", prompt, "--session", "launcher-sess-0042", "--agent", "resuming"); code != 0 || !strings.Contains(out, "resumed:hello task") {
		t.Fatalf("a session with a prompt: %d %s", code, out)
	}
	runs, _ := posemodel.Store{Root: root}.ReadDelegationTaskRuns()
	if len(runs) != 2 {
		t.Fatalf("runs: %+v", runs)
	}
	for _, run := range runs {
		raw, _ := os.ReadFile(run.Path)
		if strings.Contains(string(raw), "launcher-sess-0042") || !run.ResumesSession || !strings.Contains(strings.Join(run.Argv, " "), "{session}") {
			t.Fatalf("the record of a continued session: %s", raw)
		}
	}
	for _, args := range [][]string{
		{"--session", "launcher-sess-0042"},
		{"--session", "launcher-sess-0042", "--agent", "auto"},
		{"--session", "bad session", "--agent", "resuming"},
		{"--session", "launcher-sess-0042", "--agent", "worker"},
	} {
		if code, _, errOut := runPoseSplit(t, root, append([]string{"delegate", "task"}, args...)...); code != 2 {
			t.Fatalf("%v: %d %s", args, code, errOut)
		}
	}
}
