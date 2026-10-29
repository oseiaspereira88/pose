package cli

// Detached runs and waiting on them (spec
// pose-delegated-review-completion-signal R1–R3, R8). A detached run is an
// ordinary pose process, started in its own session so it survives the
// caller's terminal, that runs the attempt in the foreground; the caller
// returns as soon as the attempt is reserved and its marker written. pose
// delegate wait is a plain foreground process whose exit is the signal: a host
// that tracks the process is woken when the run ends.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const (
	delegateWaitSchema   = "pose.delegate.wait/v1"
	delegateDetachSchema = "pose.delegate.detach/v1"
	// delegateHandback names, in a detached run's environment, the file in
	// which it hands its attempt back: a file, not an inherited descriptor,
	// so the hand-back works on every platform.
	delegateHandback = "POSE_DELEGATE_HANDBACK"
	// delegatePromptPiped tells a detached run that its stdin is the pipe
	// on which its caller hands over the prompt the caller read from its own
	// stdin (spec pose-delegation-task-detach-and-state R2).
	delegatePromptPiped = "POSE_DELEGATE_PROMPT_PIPED"
	delegateWaitStep    = 500 * time.Millisecond
)

// delegateDetachBound is how long a detached run has to reserve its attempt;
// delegateStopGrace, how long one asked to stop has to record its run.
var (
	delegateDetachBound = 2 * time.Minute
	delegateStopGrace   = 30 * time.Second
)

// delegateExecutable is the pose binary a detached run starts; tests point it
// at a built binary.
var delegateExecutable = os.Executable

// delegateSupervised reports whether this process is a detached run.
func delegateSupervised() bool { return os.Getenv(delegateHandback) != "" }

// supervisorHandback publishes the attempt where the caller waits for it; a
// hand-back that cannot be published stops the run before its agent starts.
func supervisorHandback() func(attempt string) error {
	path := os.Getenv(delegateHandback)
	if path == "" {
		return nil
	}
	return func(attempt string) error {
		return publishHandback(path, attempt)
	}
}

// handedBack records that this process handed an attempt back: the caller
// consumes the file, so its absence later does not mean nothing was handed
// back (review round 4, F08).
var handedBack bool

func publishHandback(path, attempt string) error {
	raw, _ := json.Marshal(map[string]string{"attempt": attempt})
	err := posemodel.PublishFile(path, append(raw, '\n'), 0o600, posemodel.PublishAuthoritative, posemodel.PublishExpectation{Mode: posemodel.ExpectAbsent})
	if err == nil {
		handedBack = true
	}
	return err
}

// handBackRefusal is how a detached run that refuses before it reserves an
// attempt hands its error document back, with the exit code the foreground
// command returns: in a file beside the hand-back, so the caller returns that
// refusal with its reason code and exit code, never a log (spec
// pose-delegation-mcp R5, R7; review rounds 1 and 2, B03; spec
// pose-delegation-task-detach-and-state R4). The first refusal handed back is
// the answer.
func handBackRefusal(doc map[string]any, code int) {
	path := os.Getenv(delegateHandback)
	if path == "" {
		return
	}
	// A run that handed its attempt back is answered by that attempt.
	if _, err := os.Stat(path); err == nil {
		return
	}
	raw, err := json.Marshal(delegateRefusal{ExitCode: code, Document: doc})
	if err != nil {
		return
	}
	_ = posemodel.PublishFile(path+".refusal", append(raw, '\n'), 0o600, posemodel.PublishAuthoritative, posemodel.PublishExpectation{Mode: posemodel.ExpectAbsent})
}

// delegateRefusal is what a detached run hands back when it refuses.
type delegateRefusal struct {
	ExitCode int            `json:"exit_code"`
	Document map[string]any `json:"document"`
}

// readRefusal is the error document a detached run handed back, when it is
// one: the error schema with an error, and the exit code to return.
func readRefusal(path string) (map[string]any, int) {
	raw, err := os.ReadFile(path + ".refusal")
	if err != nil {
		return nil, 0
	}
	var r delegateRefusal
	if json.Unmarshal(raw, &r) != nil || r.Document["schema"] != delegateErrorSchema {
		return nil, 0
	}
	if msg, ok := r.Document["error"].(string); !ok || msg == "" {
		return nil, 0
	}
	if r.ExitCode != 2 {
		r.ExitCode = 1
	}
	return r.Document, r.ExitCode
}

// handBackExisting hands back the attempt of a run the dispatch answered
// without starting one — a continuation or a run that already exists — so a
// detached caller returns it as the foreground command does (review round 3,
// B06). A run that started handed its attempt back already.
func handBackExisting(run posemodel.ReviewRun, err error) {
	if err == nil {
		handBackAttempt(run.AttemptID)
	}
}

// handBackAttempt hands attempt back unless this process handed one back
// already; a task's existing continuation goes through it too (review round
// 4, B07).
func handBackAttempt(attempt string) {
	path := os.Getenv(delegateHandback)
	if path == "" || attempt == "" || handedBack {
		return
	}
	if _, statErr := os.Stat(path); statErr == nil {
		return
	}
	_ = publishHandback(path, attempt)
}

func readHandback(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var h struct {
		Attempt string `json:"attempt"`
	}
	_ = json.Unmarshal(raw, &h)
	return h.Attempt
}

// detachDelegateRun starts the same command, without --detach, as a detached
// run and returns when it has reserved its attempt (R1). A run that does not
// reserve within the bound is stopped, and an attempt it reserved meanwhile is
// named, never left running unannounced.
func detachDelegateRun(root, purpose string, args []string, f delegateFlags, stdout, stderr io.Writer) int {
	self, err := delegateExecutable()
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, err)
	}
	child := []string{"delegate", purpose}
	for _, arg := range args {
		if arg != "--detach" {
			child = append(child, arg)
		}
	}
	logDir, err := posemodel.Store{Root: root}.DelegationPrivateDir("supervisors")
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, err)
	}
	stamp := time.Now().UTC().Format("20060102T150405.000000000")
	logPath := filepath.Join(logDir, stamp+".log")
	handback := filepath.Join(logDir, stamp+".handback")
	// A prompt piped to the caller is read here, before the run leaves the
	// caller's stdin behind, and written to the run's stdin through a pipe no
	// file or other user can reach (R2, D1).
	var piped *string
	if purpose == "task" && len(f.positional) == 1 && f.positional[0] == "-" || purpose == "resume" && f.promptFile == "-" {
		prompt, err := readDelegatePrompt("-")
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, purpose, 2, err)
		}
		piped = &prompt
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, err)
	}
	defer logFile.Close()
	cmd := exec.Command(self, child...)
	// The run starts where the caller stands, so a relative --note-file or
	// scope reads what the caller meant.
	cmd.Dir = root
	if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, logFile, logFile
	cmd.Env = append(os.Environ(), delegateHandback+"="+handback)
	if piped != nil {
		// The run reads the prompt before it hands its attempt back, so the
		// whole prompt is through the pipe before this call returns.
		cmd.Stdin = strings.NewReader(*piped)
		cmd.Env = append(cmd.Env, delegatePromptPiped+"=1")
	}
	detachProcess(cmd)
	if err := cmd.Start(); err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, err)
	}
	pid := cmd.Process.Pid
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	rel := filepath.ToSlash(strings.TrimPrefix(logPath, root+string(filepath.Separator)))
	deadline := time.After(delegateDetachBound)
	attempt := ""
wait:
	for {
		if attempt = readHandback(handback); attempt != "" {
			break
		}
		select {
		case <-exited:
			attempt = readHandback(handback)
			break wait
		case <-deadline:
			stopDetached(cmd, exited)
			if attempt = readHandback(handback); attempt != "" {
				return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, errors.New(tr("the detached run reserved attempt "+attempt+" after the hand-back bound and was stopped; see pose delegate ls "+attempt,
					"o run destacado reservou a tentativa "+attempt+" depois do limite da devolução e foi parado; veja pose delegate ls "+attempt)))
			}
			return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, errors.New(tr("the detached run did not reserve its attempt within "+delegateDetachBound.String()+" and was stopped; its log is "+rel,
				"o run destacado não reservou a tentativa em "+delegateDetachBound.String()+" e foi parado; o log é "+rel)))
		case <-time.After(100 * time.Millisecond):
		}
	}
	refusal, refusalCode := readRefusal(handback)
	_ = os.Remove(handback)
	_ = os.Remove(handback + ".refusal")
	if attempt == "" {
		// A child that refused handed its error document back: that refusal,
		// with its reason code, is the answer (B03).
		if refusal != nil {
			if f.jsonOut {
				writeDelegateJSON(stdout, refusal)
				return refusalCode
			}
			msg, _ := refusal["error"].(string)
			if refusalCode == 2 {
				return usageError(stderr, "pose delegate "+purpose+": "+msg)
			}
			render(stdout, stderr).Failure("pose delegate " + purpose + ": " + msg)
			return refusalCode
		}
		// A document carries no log: its tail is for the person at the
		// terminal (R7).
		if f.jsonOut {
			return delegateFail(stdout, stderr, true, purpose, 1, errors.New("the detached run ended without handing back its attempt or a refusal; an attempt it reserved is listed by pose delegate ls"))
		}
		tail, _ := os.ReadFile(logPath)
		if len(tail) > 2048 {
			tail = tail[len(tail)-2048:]
		}
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, errors.New(tr("the detached run ended without handing back its attempt (an attempt it reserved is listed by pose delegate ls); its log ("+rel+") ends with:\n", "o run destacado terminou sem devolver a tentativa (uma tentativa que ele reservou aparece em pose delegate ls); o log ("+rel+") termina com:\n")+strings.TrimSpace(string(tail))))
	}
	records := ".pose/review-runs/"
	if strings.HasPrefix(attempt, "tat-") {
		// A task keeps its record apart from review runs (R1).
		recorded := detachedTaskRecorded(root, attempt, f)
		records = delegateTaskRecordPlace(recorded)
		if dir, err := (posemodel.Store{Root: root}).DelegationTaskRecordDir(recorded); err == nil {
			records = filepath.ToSlash(strings.TrimPrefix(dir, root+string(filepath.Separator))) + "/"
		}
	}
	if f.jsonOut {
		writeDelegateJSON(stdout, map[string]any{"schema": delegateDetachSchema, "attempt": attempt, "supervisor_pid": pid, "log": rel,
			"marker": ".pose/review-private/active/" + attempt + ".json", "records": records})
		return 0
	}
	out := render(stdout, stderr)
	out.Field("delegate.attempt", attempt)
	out.Field("delegate.supervisor_pid", fmt.Sprint(pid))
	out.Field("delegate.log", rel)
	out.Field("delegate.record", records)
	out.Field("delegate.next", "pose delegate wait "+attempt)
	return 0
}

// detachedTaskRecorded tells whether a detached task's attempt records under
// .pose: --record says so for a new task; a resume or handoff continues
// where its attempt's first run recorded.
func detachedTaskRecorded(root, attempt string, f delegateFlags) bool {
	if f.record {
		return true
	}
	runs, _ := posemodel.Store{Root: root}.ReadDelegationTaskRuns()
	for _, run := range runs {
		if run.AttemptID == attempt {
			return run.Recorded
		}
	}
	return false
}

// cmdDelegateWait blocks until the run is terminal (R2, R3, R8). While the
// attempt's marker is there it only checks the marker; the history is read
// when the marker goes. A history it could not read whole is never answered
// with a run's state: the losses are reported and the wait exits 1.
func cmdDelegateWait(root string, args []string, stdout, stderr io.Writer) int {
	f, problem := parseDelegateFlags(args, "--timeout")
	if problem == "" && len(f.positional) != 1 {
		problem = tr("name one run", "nomeie um run")
	}
	if problem != "" {
		return delegateUsage(stdout, stderr, args, "wait", problem+"\n"+tr("Usage: ", "Uso: ")+"pose delegate wait <run> [--timeout <d>] [--json]")
	}
	store := posemodel.Store{Root: root}
	var deadline time.Time
	if f.timeout > 0 {
		deadline = time.Now().Add(f.timeout)
	}
	for {
		attempts, lost, err := store.ListDelegationAttempts()
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "wait", 1, err)
		}
		attempt, code, err := namedAttempt(attempts, f)
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "wait", code, withDelegationLosses(err, lost))
		}
		result, terminal := posemodel.DelegationWaitOutcome(attempt)
		if terminal && len(lost) > 0 {
			return delegateFail(stdout, stderr, f.jsonOut, "wait", 1, withDelegationLosses(errors.New(tr("the run history could not be read whole, so the run's end cannot be told", "o histórico de runs não pôde ser lido inteiro, então o fim do run não pode ser dito")), lost))
		}
		if !terminal && !deadline.IsZero() && !time.Now().Before(deadline) {
			result.Code = posemodel.DelegationWaitTimeout
			terminal = true
		}
		if terminal {
			printWaitResult(stdout, f.jsonOut, result, delegateNextActions(store, attempt))
			return result.Code
		}
		// Running: watch only the marker until it goes, or the heartbeat
		// interval passes, so a stall is still noticed.
		recheck := time.Now().Add(10 * time.Second)
		for store.DelegationActiveMarkerExists(attempt.AttemptID) && time.Now().Before(recheck) {
			if !deadline.IsZero() && !time.Now().Before(deadline) {
				break
			}
			step := delegateWaitStep
			if !deadline.IsZero() && time.Until(deadline) < step {
				step = time.Until(deadline)
			}
			time.Sleep(step)
		}
	}
}

// printWaitResult prints the one line of a wait.
func printWaitResult(stdout io.Writer, jsonOut bool, r posemodel.DelegationWaitResult, next []delegateNextAction) {
	if jsonOut {
		raw, _ := json.Marshal(struct {
			Schema string `json:"schema"`
			posemodel.DelegationWaitResult
			NextActions []delegateNextAction `json:"next_actions,omitempty"`
		}{delegateWaitSchema, r, next})
		render(stdout, io.Discard).ContractLine(string(raw))
		return
	}
	line := "attempt=" + r.Attempt + " state=" + r.State
	if r.Run != "" {
		line += " run=" + r.Run
	}
	if r.Decision != "" {
		line += " decision=" + r.Decision
	}
	line += fmt.Sprintf(" resumable=%t", r.Resumable)
	if r.ResetAt != "" {
		line += " reset_at=" + r.ResetAt
	}
	line += fmt.Sprintf(" exit=%d", r.Code)
	render(stdout, io.Discard).ContractLine(line)
}
