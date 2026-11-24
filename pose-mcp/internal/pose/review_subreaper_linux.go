//go:build linux

package pose

// Orphaned escapees (spec pose-review-run-reaps-lingering-reviewer-processes,
// review round 6, N2): a descendant that leaves the adapter's process group
// with a session of its own and clears its environment matches neither the
// group kill nor the sealed-checkout entry. The dispatching process makes
// itself a child subreaper, so such a process, once its parent ends, is
// reparented to the dispatcher instead of to init; after the adapter exits,
// every child of the dispatcher in a session other than the dispatcher's,
// started after the adapter, is an orphan of a review run. One that names
// another run's checkout belongs to that run and is left to it.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// enableReviewSubreaper makes the dispatching process the subreaper of its
// descendants. It is process-wide and idempotent.
func enableReviewSubreaper() error {
	return unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0)
}

// reviewClockTicks is now, in the clock ticks /proc/<pid>/stat uses for a
// process start.
func reviewClockTicks() (uint64, error) {
	raw, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(raw))
	if len(fields) == 0 {
		return 0, fmt.Errorf("unreadable uptime")
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}
	return uint64(seconds * 100), nil // USER_HZ is 100 on Linux
}

type reviewProcStat struct {
	ppid, session int
	start         uint64
}

func readReviewProcStat(pid int) (reviewProcStat, error) {
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return reviewProcStat{}, err
	}
	text := string(raw)
	end := strings.LastIndexByte(text, ')')
	if end < 0 {
		return reviewProcStat{}, fmt.Errorf("process %d stat is unreadable", pid)
	}
	f := strings.Fields(text[end+1:])
	// f[0] is field 3 (state): ppid is field 4, session field 6, start field 22.
	if len(f) < 20 {
		return reviewProcStat{}, fmt.Errorf("process %d stat is unreadable", pid)
	}
	ppid, e1 := strconv.Atoi(f[1])
	session, e2 := strconv.Atoi(f[3])
	start, e3 := strconv.ParseUint(f[19], 10, 64)
	if e1 != nil || e2 != nil || e3 != nil {
		return reviewProcStat{}, fmt.Errorf("process %d stat is unreadable", pid)
	}
	return reviewProcStat{ppid: ppid, session: session, start: start}, nil
}

// reapOrphanedEscapees ends the orphans of review runs reparented to this
// process since since (clock ticks), and reaps them. entry is this run's
// sealed-checkout entry; an orphan naming another run's checkout is left.
func reapOrphanedEscapees(entry []byte, since uint64) ([]string, error) {
	self := os.Getpid()
	selfStat, err := readReviewProcStat(self)
	if err != nil {
		return nil, err
	}
	var ended []string
	deadline := time.Now().Add(reviewReapWait)
	for {
		dirs, err := os.ReadDir("/proc")
		if err != nil {
			return ended, fmt.Errorf("the run's processes could not be listed: %w", err)
		}
		found := 0
		for _, d := range dirs {
			pid, err := strconv.Atoi(d.Name())
			if err != nil || pid == self {
				continue
			}
			st, err := readReviewProcStat(pid)
			if reviewProcessGoneErr(err) {
				continue
			}
			if err != nil {
				return ended, fmt.Errorf("process %d could not be inspected: %w", pid, err)
			}
			if st.ppid != self || st.session == selfStat.session || st.start < since {
				continue
			}
			if other, err := reviewProcessOtherRun(pid, entry); err == nil && other {
				continue
			}
			found++
			if zombie, _ := reviewProcessZombie(pid); !zombie {
				ended = append(ended, strconv.Itoa(pid)+" (orphaned)")
				if err := killPinnedOrphan(pid, st); err != nil {
					return ended, err
				}
			}
			// It is this process's child now: reap it, so it leaves no zombie.
			var status unix.WaitStatus
			_, _ = unix.Wait4(pid, &status, unix.WNOHANG, nil)
		}
		if found == 0 {
			return ended, nil
		}
		if time.Now().After(deadline) {
			return ended, fmt.Errorf("%d orphaned process(es) of the run could not be ended", found)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// reviewProcessOtherRun reports a process whose environment names another
// run's sealed checkout.
func reviewProcessOtherRun(pid int, entry []byte) (bool, error) {
	environ, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "environ"))
	if err != nil {
		return false, err
	}
	prefix := SealedCheckoutEnv + "="
	for _, kv := range strings.Split(string(environ), "\x00") {
		if strings.HasPrefix(kv, prefix) {
			return kv != string(entry), nil
		}
	}
	return false, nil
}

// killPinnedOrphan signals pid through a pidfd, after the pinned process is
// still the orphan seen (same parent, session and start).
func killPinnedOrphan(pid int, seen reviewProcStat) error {
	fd, err := unix.PidfdOpen(pid, 0)
	if errors.Is(err, unix.ESRCH) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("process %d could not be pinned: %w", pid, err)
	}
	defer unix.Close(fd)
	now, err := readReviewProcStat(pid)
	if reviewProcessGoneErr(err) || err == nil && now != seen {
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
