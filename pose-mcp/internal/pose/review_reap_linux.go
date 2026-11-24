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

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const reviewReapWait = 5 * time.Second

// reapSealedCheckoutProcesses ends every process whose environment names
// worktree as its sealed checkout and returns a description of each. An
// error means a process could not be ended within the bound.
func reapSealedCheckoutProcesses(worktree string) ([]string, error) {
	if worktree == "" {
		return nil, nil
	}
	entry := []byte(SealedCheckoutEnv + "=" + worktree)
	self := os.Getpid()
	find := func() map[int]string {
		found := map[int]string{}
		dirs, err := os.ReadDir("/proc")
		if err != nil {
			return found
		}
		for _, d := range dirs {
			pid, err := strconv.Atoi(d.Name())
			if err != nil || pid == self {
				continue
			}
			environ, err := os.ReadFile(filepath.Join("/proc", d.Name(), "environ"))
			if err != nil {
				continue
			}
			for _, kv := range bytes.Split(environ, []byte{0}) {
				if bytes.Equal(kv, entry) {
					found[pid] = reviewProcessName(pid)
					break
				}
			}
		}
		return found
	}
	var reaped []string
	seen := map[int]bool{}
	deadline := time.Now().Add(reviewReapWait)
	for {
		alive := find()
		for pid, name := range alive {
			if reviewProcessZombie(pid) {
				delete(alive, pid)
				continue
			}
			if !seen[pid] {
				seen[pid] = true
				reaped = append(reaped, fmt.Sprintf("%d %s", pid, name))
			}
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
		if len(alive) == 0 {
			return reaped, nil
		}
		if time.Now().After(deadline) {
			return reaped, fmt.Errorf("%d process(es) of the run could not be ended", len(alive))
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func reviewProcessName(pid int) string {
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "comm"))
	if err != nil {
		return "?"
	}
	return strings.TrimSpace(string(raw))
}

func reviewProcessZombie(pid int) bool {
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return true // gone
	}
	// The state follows the parenthesised command name.
	if i := bytes.LastIndexByte(raw, ')'); i >= 0 && i+2 < len(raw) {
		return raw[i+2] == 'Z'
	}
	return false
}
