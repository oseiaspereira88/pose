//go:build linux

package pose

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// RunReviewShim runs args (the adapter's path, then its argv) as the shim's
// only child, waits for it or for SIGTERM, ends every process left in the
// shim's subtree and returns the adapter's exit status.
func RunReviewShim(args []string) int {
	report := reviewShimReport{}
	var reportFile *os.File
	parent := 0
	if value := os.Getenv(ReviewShimReportEnv); value != "" {
		parent, _ = strconv.Atoi(value)
		// The report pipe is the shim's alone: the adapter never inherits it.
		syscall.CloseOnExec(3)
		reportFile = os.NewFile(3, "pose-review-shim-report")
	}
	_ = os.Unsetenv(ReviewShimReportEnv)
	write := func() {
		if reportFile == nil {
			return
		}
		raw, _ := json.Marshal(report)
		_, _ = reportFile.Write(raw)
		_ = reportFile.Close()
	}
	if len(args) < 2 {
		report.Error = "the review shim was given no adapter"
		write()
		return 125
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM)
	// The dispatcher's death stops the shim the same way as a timeout.
	if err := unix.Prctl(unix.PR_SET_PDEATHSIG, uintptr(syscall.SIGTERM), 0, 0, 0); err != nil {
		report.Error = "the shim could not follow its dispatcher: " + err.Error()
		write()
		return 125
	}
	if parent != 0 && os.Getppid() != parent {
		report.Error = "the dispatcher ended before the shim started"
		write()
		return 125
	}
	if err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0); err != nil {
		report.Error = "the shim could not become the subreaper of the run: " + err.Error()
		write()
		return 125
	}
	cmd := &exec.Cmd{Path: args[0], Args: args[1:], Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr, Env: os.Environ()}
	if err := cmd.Start(); err != nil {
		report.Error = "the adapter could not run: " + err.Error()
		write()
		return 127
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	code, stopped := 0, false
	var signaled syscall.Signal
	select {
	case err := <-done:
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
				signaled, code = status.Signal(), 128+int(status.Signal())
			} else {
				code = exit.ExitCode()
			}
		} else if err != nil {
			report.Error = "the adapter's end could not be read: " + err.Error()
		}
	case <-stop:
		stopped, code = true, 128+int(syscall.SIGTERM)
	}
	ended, err := reapShimSubtree()
	report.Ended = ended
	if err != nil && report.Error == "" {
		report.Error = err.Error()
	}
	write()
	if stopped {
		return code
	}
	// An adapter ended by a signal is reported as ended by it: the shim ends
	// itself the same way, so the dispatcher reads what it would have read
	// without the shim.
	if signaled != 0 {
		signal.Reset(signaled)
		_ = syscall.Kill(os.Getpid(), signaled)
		time.Sleep(time.Second)
	}
	return code
}

// stopReviewShim asks a running shim to end its subtree and exit.
func stopReviewShim(cmd *exec.Cmd) func() error {
	return func() error { return cmd.Process.Signal(syscall.SIGTERM) }
}

// reapShimSubtree ends and reaps every child of the shim, including the
// orphans its subreaper status adopted, until none is left.
func reapShimSubtree() ([]string, error) {
	self := os.Getpid()
	var ended []string
	seen := map[int]bool{}
	deadline := time.Now().Add(reviewReapWait)
	for {
		dirs, err := os.ReadDir("/proc")
		if err != nil {
			return ended, fmt.Errorf("the run's processes could not be listed: %w", err)
		}
		children := 0
		for _, d := range dirs {
			pid, err := strconv.Atoi(d.Name())
			if err != nil || pid == self {
				continue
			}
			st, err := readShimProcStat(pid)
			if reviewProcessGoneErr(err) {
				continue
			}
			if err != nil {
				return ended, fmt.Errorf("process %d could not be inspected: %w", pid, err)
			}
			if st.ppid != self {
				continue
			}
			children++
			if !st.zombie {
				if !seen[pid] {
					seen[pid] = true
					ended = append(ended, strconv.Itoa(pid))
				}
				if err := killPinnedChild(pid, self); err != nil {
					return ended, err
				}
			}
		}
		for {
			pid, err := unix.Wait4(-1, nil, unix.WNOHANG, nil)
			if pid <= 0 || err != nil {
				break
			}
		}
		if children == 0 {
			return ended, nil
		}
		if time.Now().After(deadline) {
			return ended, fmt.Errorf("%d process(es) of the run could not be ended", children)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

type shimProcStat struct {
	ppid   int
	zombie bool
}

func readShimProcStat(pid int) (shimProcStat, error) {
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return shimProcStat{}, err
	}
	text := string(raw)
	end := strings.LastIndexByte(text, ')')
	if end < 0 {
		return shimProcStat{}, fmt.Errorf("process %d stat is unreadable", pid)
	}
	f := strings.Fields(text[end+1:])
	if len(f) < 2 {
		return shimProcStat{}, fmt.Errorf("process %d stat is unreadable", pid)
	}
	ppid, err := strconv.Atoi(f[1])
	if err != nil {
		return shimProcStat{}, fmt.Errorf("process %d stat is unreadable", pid)
	}
	return shimProcStat{ppid: ppid, zombie: f[0] == "Z"}, nil
}

// killPinnedChild signals pid through a pidfd while it is still a child of
// parent: a pid reused by an unrelated process is never signalled.
func killPinnedChild(pid, parent int) error {
	fd, err := unix.PidfdOpen(pid, 0)
	if errors.Is(err, unix.ESRCH) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("process %d could not be pinned: %w", pid, err)
	}
	defer unix.Close(fd)
	st, err := readShimProcStat(pid)
	if reviewProcessGoneErr(err) || err == nil && st.ppid != parent {
		return nil
	}
	if err != nil {
		return fmt.Errorf("process %d could not be inspected: %w", pid, err)
	}
	if err := unix.PidfdSendSignal(fd, unix.SIGKILL, nil, 0); err != nil && !errors.Is(err, unix.ESRCH) {
		return fmt.Errorf("process %d could not be ended: %w", pid, err)
	}
	return nil
}

// parseReviewShimReport reads the report the shim wrote to its pipe.
func parseReviewShimReport(raw []byte) (reviewShimReport, error) {
	var r reviewShimReport
	if len(raw) == 0 {
		return r, fmt.Errorf("the review shim left no report")
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return r, fmt.Errorf("the review shim report is unreadable: %w", err)
	}
	return r, nil
}
