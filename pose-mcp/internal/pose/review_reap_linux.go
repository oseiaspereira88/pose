//go:build linux

package pose

// Leftover reviewer processes (spec
// pose-review-run-reaps-lingering-reviewer-processes R1): an adapter such as
// `codex exec` runs each command in a session of its own, so killing the
// adapter's process group misses a command the reviewer left running, and
// that command keeps writing into the run's runtime directory while the
// cleanup inventory reads it. Every process of the run inherits
// SealedCheckoutEnv naming the run's checkout, so the dispatch finds them by
// that entry, ends them and waits for them before the inventory.
//
// Each process is pinned by a pidfd before it is signalled, and its
// environment is read again through the pinned process, so a pid reused by
// an unrelated process between the scan and the signal is never killed
// (review round 1, N3). A scan that cannot list or read the state of the
// processes fails. A process of this user whose environment cannot be read
// (one that made itself non-inspectable) cannot be matched: one that already
// existed before the adapter started is not the run's and is skipped; any
// other may be the run's, so the scan fails and the run stays unconcludable
// (review round 2, N2).

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const reviewReapWait = 5 * time.Second

// reapSealedCheckoutProcesses ends every process whose environment names
// worktree as its sealed checkout and returns a description of each. An
// error means the run's processes could not be proven ended within the bound.
func reapSealedCheckoutProcesses(worktree string, baseline map[int]string) ([]string, int, error) {
	if worktree == "" {
		return nil, 0, nil
	}
	entry := []byte(SealedCheckoutEnv + "=" + worktree)
	var reaped []string
	seen := map[int]bool{}
	deadline := time.Now().Add(reviewReapWait)
	unreadable := 0
	for {
		alive, opaque, err := reviewMarkedProcesses(entry)
		if err != nil {
			return reaped, len(opaque), err
		}
		unreadable = 0
		for pid, start := range opaque {
			if baseline[pid] != start {
				return reaped, len(opaque), fmt.Errorf("process %d of this user started during the run and cannot be inspected; it may be the run's and was not ended", pid)
			}
			unreadable++
		}
		for _, pid := range alive {
			if !seen[pid] {
				seen[pid] = true
				reaped = append(reaped, fmt.Sprintf("%d %s", pid, reviewProcessName(pid)))
			}
			if err := killPinnedMarkedProcess(pid, entry); err != nil {
				return reaped, unreadable, err
			}
		}
		if len(alive) == 0 {
			return reaped, unreadable, nil
		}
		if time.Now().After(deadline) {
			return reaped, unreadable, fmt.Errorf("%d process(es) of the run could not be ended", len(alive))
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// reviewMarkedProcesses lists the live processes whose environment holds
// entry, and counts this user's processes whose environment could not be
// read. A process that is gone is skipped.
func reviewMarkedProcesses(entry []byte) ([]int, map[int]string, error) {
	dirs, err := os.ReadDir("/proc")
	if err != nil {
		return nil, nil, fmt.Errorf("the run's processes could not be listed: %w", err)
	}
	unreadable := map[int]string{}
	self, uid := os.Getpid(), os.Getuid()
	var out []int
	for _, d := range dirs {
		pid, err := strconv.Atoi(d.Name())
		if err != nil || pid == self {
			continue
		}
		marked, err := reviewProcessMarked(pid, entry)
		switch {
		case reviewProcessGoneErr(err):
			continue
		case err != nil:
			if owner, oerr := reviewProcessOwner(pid); oerr == nil && owner != uid {
				continue // another user's process never inherits this run's environment
			} else if reviewProcessGoneErr(oerr) {
				continue
			}
			start, serr := processStartTime(pid)
			if reviewProcessGoneErr(serr) {
				continue
			}
			unreadable[pid] = start
			continue
		case !marked:
			continue
		}
		zombie, err := reviewProcessZombie(pid)
		if reviewProcessGoneErr(err) || err == nil && zombie {
			continue
		}
		if err != nil {
			return nil, unreadable, fmt.Errorf("process %d state could not be read: %w", pid, err)
		}
		out = append(out, pid)
	}
	return out, unreadable, nil
}

// killPinnedMarkedProcess signals pid only through a pidfd opened on it,
// after the environment read through that same pinned process still holds
// entry.
func killPinnedMarkedProcess(pid int, entry []byte) error {
	fd, err := unix.PidfdOpen(pid, 0)
	if errors.Is(err, unix.ESRCH) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("process %d could not be pinned: %w", pid, err)
	}
	defer unix.Close(fd)
	// The pidfd names one process for its lifetime; if pid was reused before
	// it was opened, the new process does not carry this run's entry.
	marked, err := reviewProcessMarked(pid, entry)
	if reviewProcessGoneErr(err) || err == nil && !marked {
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

func reviewProcessMarked(pid int, entry []byte) (bool, error) {
	environ, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "environ"))
	if err != nil {
		return false, err
	}
	for _, kv := range bytes.Split(environ, []byte{0}) {
		if bytes.Equal(kv, entry) {
			return true, nil
		}
	}
	return false, nil
}

func reviewProcessOwner(pid int) (int, error) {
	info, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
	if err != nil {
		return 0, err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("process %d owner unavailable", pid)
	}
	return int(st.Uid), nil
}

func reviewProcessGoneErr(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ESRCH)
}

func reviewProcessName(pid int) string {
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "comm"))
	if err != nil {
		return "?"
	}
	return strings.TrimSpace(string(raw))
}

func reviewProcessZombie(pid int) (bool, error) {
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return false, err
	}
	// The state follows the parenthesised command name.
	if i := bytes.LastIndexByte(raw, ')'); i >= 0 && i+2 < len(raw) {
		return raw[i+2] == 'Z', nil
	}
	return false, fmt.Errorf("process %d state is unreadable", pid)
}

// reviewOpaqueProcessBaseline records, before the adapter starts, the
// processes of this user whose environment cannot be read, by pid and start
// time: they cannot be the run's.
func reviewOpaqueProcessBaseline() map[int]string {
	_, opaque, _ := reviewMarkedProcesses([]byte(SealedCheckoutEnv + "=\x00"))
	return opaque
}
