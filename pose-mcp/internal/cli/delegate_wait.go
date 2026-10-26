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
	delegateWaitStep = 500 * time.Millisecond
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
		raw, _ := json.Marshal(map[string]string{"attempt": attempt})
		return posemodel.PublishFile(path, append(raw, '\n'), 0o600, posemodel.PublishAuthoritative, posemodel.PublishExpectation{Mode: posemodel.ExpectAbsent})
	}
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
	_ = os.Remove(handback)
	if attempt == "" {
		tail, _ := os.ReadFile(logPath)
		if len(tail) > 2048 {
			tail = tail[len(tail)-2048:]
		}
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, errors.New(tr("the detached run ended without handing back its attempt (an attempt it reserved is listed by pose delegate ls); its log ("+rel+") ends with:\n", "o run destacado terminou sem devolver a tentativa (uma tentativa que ele reservou aparece em pose delegate ls); o log ("+rel+") termina com:\n")+strings.TrimSpace(string(tail))))
	}
	if f.jsonOut {
		writeDelegateJSON(stdout, map[string]any{"schema": delegateDetachSchema, "attempt": attempt, "supervisor_pid": pid, "log": rel,
			"marker": ".pose/review-private/active/" + attempt + ".json", "records": ".pose/review-runs/"})
		return 0
	}
	out := render(stdout, stderr)
	out.Field("delegate.attempt", attempt)
	out.Field("delegate.supervisor_pid", fmt.Sprint(pid))
	out.Field("delegate.log", rel)
	out.Field("delegate.next", "pose delegate wait "+attempt)
	return 0
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
