package cli

// Detached runs and waiting on them (spec
// pose-delegated-review-completion-signal R1–R3, R8). A detached run is an
// ordinary pose process, started in its own session so it survives the
// caller's terminal, that runs the attempt in the foreground; the caller
// returns as soon as the attempt is reserved and its marker written. pose
// delegate wait is a plain foreground process whose exit is the signal: a host
// that tracks the process is woken when the run ends.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const (
	delegateWaitSchema   = "pose.delegate.wait/v1"
	delegateDetachSchema = "pose.delegate.detach/v1"
	// delegateSupervisorFD names, in a detached run's environment, the
	// descriptor on which it hands its attempt back.
	delegateSupervisorFD = "POSE_DELEGATE_SUPERVISOR_FD"
	delegateDetachBound  = 2 * time.Minute
	delegateWaitStep     = 500 * time.Millisecond
)

// delegateExecutable is the pose binary a detached run starts; tests point it
// at a built binary.
var delegateExecutable = os.Executable

// delegateSupervised reports whether this process is a detached run.
func delegateSupervised() bool { return os.Getenv(delegateSupervisorFD) != "" }

// supervisorHandback writes the attempt to the descriptor the caller holds,
// once, and closes it.
func supervisorHandback() func(attempt string) {
	fd, err := strconv.Atoi(os.Getenv(delegateSupervisorFD))
	if err != nil || fd < 3 {
		return nil
	}
	pipe := os.NewFile(uintptr(fd), "pose-delegate-supervisor")
	return func(attempt string) {
		raw, _ := json.Marshal(map[string]string{"attempt": attempt})
		_, _ = pipe.Write(append(raw, '\n'))
		_ = pipe.Close()
	}
}

// detachDelegateRun starts the same command, without --detach, as a detached
// run and returns when it has reserved its attempt (R1).
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
	// The log sits with the engine's private run state, which Git ignores.
	private := filepath.Join(root, ".pose", "review-private")
	logDir := filepath.Join(private, "supervisors")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, err)
	}
	if _, err := os.Lstat(filepath.Join(private, ".gitignore")); errors.Is(err, os.ErrNotExist) {
		_ = os.WriteFile(filepath.Join(private, ".gitignore"), []byte("*\n"), 0o600)
	}
	logPath := filepath.Join(logDir, time.Now().UTC().Format("20060102T150405.000000000")+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, err)
	}
	defer logFile.Close()
	reader, writer, err := os.Pipe()
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, err)
	}
	defer reader.Close()
	cmd := exec.Command(self, child...)
	cmd.Dir = root
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, logFile, logFile
	cmd.ExtraFiles = []*os.File{writer}
	cmd.Env = append(os.Environ(), delegateSupervisorFD+"=3")
	detachProcess(cmd)
	if err := cmd.Start(); err != nil {
		writer.Close()
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, err)
	}
	writer.Close()
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	type handback struct {
		Attempt string `json:"attempt"`
	}
	got := make(chan handback, 1)
	go func() {
		var h handback
		if line, err := bufio.NewReader(reader).ReadBytes('\n'); err == nil {
			_ = json.Unmarshal(line, &h)
		}
		got <- h
	}()
	var h handback
	select {
	case h = <-got:
	case <-time.After(delegateDetachBound):
	}
	rel := filepath.ToSlash(strings.TrimPrefix(logPath, root+string(filepath.Separator)))
	if h.Attempt == "" {
		tail, _ := os.ReadFile(logPath)
		if len(tail) > 2048 {
			tail = tail[len(tail)-2048:]
		}
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, errors.New(tr("the detached run ended before reserving its attempt; its log ("+rel+") ends with:\n", "o run destacado terminou antes de reservar a tentativa; o log ("+rel+") termina com:\n")+strings.TrimSpace(string(tail))))
	}
	if f.jsonOut {
		writeDelegateJSON(stdout, map[string]any{"schema": delegateDetachSchema, "attempt": h.Attempt, "supervisor_pid": pid, "log": rel,
			"marker": ".pose/review-private/active/" + h.Attempt + ".json", "records": ".pose/review-runs/"})
		return 0
	}
	out := render(stdout, stderr)
	out.Field("delegate.attempt", h.Attempt)
	out.Field("delegate.supervisor_pid", fmt.Sprint(pid))
	out.Field("delegate.log", rel)
	out.Field("delegate.next", "pose delegate wait "+h.Attempt)
	return 0
}

// cmdDelegateWait blocks until the run is terminal (R2, R3, R8).
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
		attempts, _, err := store.ListDelegationAttempts()
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "wait", 1, err)
		}
		attempt, code, err := namedAttempt(attempts, f)
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "wait", code, err)
		}
		result, terminal := posemodel.DelegationWaitOutcome(attempt)
		if !terminal && !deadline.IsZero() && !time.Now().Before(deadline) {
			result.Code = posemodel.DelegationWaitTimeout
			terminal = true
		}
		if terminal {
			printWaitResult(stdout, f.jsonOut, result)
			return result.Code
		}
		step := delegateWaitStep
		if !deadline.IsZero() && time.Until(deadline) < step {
			step = time.Until(deadline)
		}
		time.Sleep(step)
	}
}

// printWaitResult prints the one line of a wait.
func printWaitResult(stdout io.Writer, jsonOut bool, r posemodel.DelegationWaitResult) {
	if jsonOut {
		raw, _ := json.Marshal(struct {
			Schema string `json:"schema"`
			posemodel.DelegationWaitResult
		}{delegateWaitSchema, r})
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
