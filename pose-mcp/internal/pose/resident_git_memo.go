package pose

import (
	"container/list"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// ResidentGitMemo keeps, in a process that outlives one operation, the output of the Git reads
// that are pure functions of object ids or of the commit HEAD named when they ran (see
// governancePureGitRead). It is off unless a process turns it on, so nothing changes for a
// command that runs once; the optional worktree runtime turns it on (spec
// pose-worktree-runtime-resident-cache). A hit is the answer Git would give: the key
// holds the commit and the shallow state, and the output of such a read cannot differ for them.
type ResidentGitMemo struct {
	mu      sync.Mutex
	max     int64
	bytes   int64
	entries map[string]*list.Element
	order   *list.List // front is the most recently used
	hits    atomic.Int64
	misses  atomic.Int64
}

type residentGitEntry struct {
	key    string
	stdout []byte
}

var residentGit atomic.Pointer[ResidentGitMemo]

// EnableResidentGitMemo turns the memo on with a byte budget for what it retains.
func EnableResidentGitMemo(maxBytes int64) *ResidentGitMemo {
	m := &ResidentGitMemo{max: maxBytes, entries: map[string]*list.Element{}, order: list.New()}
	residentGit.Store(m)
	return m
}

// DisableResidentGitMemo turns the memo off and drops what it held.
func DisableResidentGitMemo() { residentGit.Store(nil) }

// ResidentGitMemoStats are the hits, misses, entries and retained bytes since it was enabled.
func ResidentGitMemoStats() (hits, misses int64, entries int, bytes int64) {
	m := residentGit.Load()
	if m == nil {
		return 0, 0, 0, 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hits.Load(), m.misses.Load(), len(m.entries), m.bytes
}

// residentGitKey names a read: the root, the commit it was made at (empty for a read by object
// id), the shallow state of the repository and the exact command. A root whose Git directory
// is not a plain .git directory is not memoized: its shallow state is not read here.
func residentGitKey(root, head string, args []string) (string, bool) {
	gitDir := filepath.Join(root, ".git")
	info, err := os.Lstat(gitDir)
	if err != nil || !info.IsDir() {
		return "", false
	}
	shallow := "-"
	if s, err := os.Stat(filepath.Join(gitDir, "shallow")); err == nil {
		shallow = fmt.Sprintf("%d:%d", s.Size(), s.ModTime().UnixNano())
	}
	// The repository's own config can change what a read answers (log.follow, for one).
	config := "-"
	if c, err := os.Stat(filepath.Join(gitDir, "config")); err == nil {
		config = fmt.Sprintf("%d:%d", c.Size(), c.ModTime().UnixNano())
	}
	return strings.Join([]string{filepath.Clean(root), head, shallow, config, strings.Join(args, "\x00")}, "\x01"), true
}

func (m *ResidentGitMemo) get(key string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	el, ok := m.entries[key]
	if !ok {
		m.misses.Add(1)
		return nil, false
	}
	m.order.MoveToFront(el)
	m.hits.Add(1)
	return append([]byte(nil), el.Value.(*residentGitEntry).stdout...), true
}

func (m *ResidentGitMemo) put(key string, stdout []byte) {
	size := int64(len(key) + len(stdout))
	m.mu.Lock()
	defer m.mu.Unlock()
	if size > m.max {
		return
	}
	if el, ok := m.entries[key]; ok {
		m.bytes -= int64(len(el.Value.(*residentGitEntry).key) + len(el.Value.(*residentGitEntry).stdout))
		m.order.Remove(el)
		delete(m.entries, key)
	}
	for m.bytes+size > m.max {
		last := m.order.Back()
		if last == nil {
			break
		}
		e := last.Value.(*residentGitEntry)
		m.bytes -= int64(len(e.key) + len(e.stdout))
		m.order.Remove(last)
		delete(m.entries, e.key)
	}
	m.entries[key] = m.order.PushFront(&residentGitEntry{key: key, stdout: append([]byte(nil), stdout...)})
	m.bytes += size
}
