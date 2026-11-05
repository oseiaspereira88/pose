package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// A run's terminal record/absent marker is not a supervisor join: conclusion,
// fulfilment and on_terminal can still write into the fixture afterwards.
// Keep ownership at the root, independently of successful test assertions.
var detachedFixtures = struct {
	sync.Mutex
	roots map[string]*detachedFixture
}{roots: make(map[string]*detachedFixture)}

type detachedFixture struct {
	root    string
	mu      sync.Mutex
	runs    []*fixtureSupervisor
	joined  bool
	joinErr error
}

type fixtureSupervisor struct {
	pid     int
	attempt string
	stat    *os.File
	start   string
}

func newDetachedFixtureRoot(t *testing.T) string {
	t.Helper()
	// Unlike TempDir, this directory is retained if joining fails: removing a
	// root with an unaccounted writer would just reproduce the cleanup race.
	root, err := os.MkdirTemp("", "pose-dispatch-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	f := &detachedFixture{root: root}
	detachedFixtures.Lock()
	detachedFixtures.roots[root] = f
	detachedFixtures.Unlock()
	t.Cleanup(func() {
		defer func() {
			detachedFixtures.Lock()
			delete(detachedFixtures.roots, root)
			detachedFixtures.Unlock()
		}()
		if err := f.join(); err != nil {
			t.Errorf("detached fixture retained at %s: %v", root, err)
			return
		}
		if err := os.RemoveAll(root); err != nil {
			t.Errorf("detached fixture RemoveAll: %v", err)
		}
	})
	return root
}

func detachedFixtureFor(root string) *detachedFixture {
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	detachedFixtures.Lock()
	defer detachedFixtures.Unlock()
	for dir := filepath.Clean(root); ; dir = filepath.Dir(dir) {
		if f := detachedFixtures.roots[dir]; f != nil {
			return f
		}
		if filepath.Dir(dir) == dir {
			return nil
		}
	}
}

// Observe the raw CLI response, before MCP filters it or assertions can fail.
// A pinned proc stat descriptor refers to that process even after PID reuse;
// reads after its removal fail rather than following a replacement PID.
// This fixture intentionally requires Linux procfs, as used by these tests.
func ownDetachedFixtureOutput(root string, out []byte) (ownErr error) {
	f := detachedFixtureFor(root)
	if f == nil {
		return nil
	}
	defer func() {
		if ownErr != nil {
			f.mu.Lock()
			f.joinErr = errors.Join(f.joinErr, ownErr)
			f.mu.Unlock()
		}
	}()
	var doc struct {
		Attempt string `json:"attempt"`
		PID     int    `json:"supervisor_pid"`
	}
	if json.Unmarshal(out, &doc) != nil || doc.PID == 0 {
		return nil // preview, refusal, wait or other non-start response
	}
	if doc.PID < 0 || doc.Attempt == "" {
		return fmt.Errorf("invalid detached supervisor identity: pid %d attempt %q", doc.PID, doc.Attempt)
	}
	if _, err := os.Stat("/proc/self/stat"); err != nil {
		return fmt.Errorf("detached fixture needs procfs: %w", err)
	}
	p := &fixtureSupervisor{pid: doc.PID, attempt: doc.Attempt}
	file, err := os.Open(filepath.Join("/proc", strconv.Itoa(p.pid), "stat"))
	if os.IsNotExist(err) {
		return nil // already exited and reaped
	}
	if err != nil {
		return fmt.Errorf("pin supervisor %d: %w", p.pid, err)
	}
	p.stat = file
	_, start, err := p.state()
	if err != nil && !fixtureProcessGone(err) {
		file.Close()
		return err
	}
	p.start = start
	f.mu.Lock()
	f.runs = append(f.runs, p) // do not deduplicate by attempt: resumes share it
	f.mu.Unlock()
	return nil
}

func fixtureProcessGone(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH)
}

func (p *fixtureSupervisor) state() (state, start string, err error) {
	var raw [8192]byte
	n, err := p.stat.ReadAt(raw[:], 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", "", err
	}
	text := string(raw[:n])
	end := strings.LastIndexByte(text, ')')
	if end < 0 {
		return "", "", fmt.Errorf("supervisor %d: invalid proc stat", p.pid)
	}
	fields := strings.Fields(text[end+1:])
	if len(fields) < 20 {
		return "", "", fmt.Errorf("supervisor %d: short proc stat", p.pid)
	}
	return fields[0], fields[19], nil // fields 3 (state) and 22 (start time)
}

func (p *fixtureSupervisor) exited() (bool, error) {
	state, start, err := p.state()
	if fixtureProcessGone(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	// A zombie has exited even if its adopting parent has not reaped it yet.
	return state == "Z" || state == "X" || start != p.start, nil
}

func (p *fixtureSupervisor) wait(ctx context.Context, waiting chan<- struct{}) error {
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		gone, err := p.exited()
		if err != nil || gone {
			return err
		}
		if waiting != nil {
			close(waiting) // deterministic observation that join found it alive
			waiting = nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("supervisor %d (%s) did not exit: %w", p.pid, p.attempt, ctx.Err())
		case <-tick.C:
		}
	}
}

func (f *detachedFixture) join() error {
	return f.joinObserved(nil)
}

func (f *detachedFixture) joinObserved(waiting chan<- struct{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.joined {
		return f.joinErr
	}
	f.joined = true
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	failures := []error{f.joinErr}
	for _, p := range f.runs {
		gone, err := p.exited()
		if err == nil && !gone {
			// Gates have already been released by their cleanup. Request native
			// cancellation too, so assertion failures need not await agent timeout.
			// A terminal run can refuse cancellation while its hook still runs;
			// only the process join below establishes that cleanup is safe.
			cmd := exec.CommandContext(ctx, builtPose, "delegate", "cancel", p.attempt, "--json")
			cmd.Dir = f.root
			_, _ = cmd.CombinedOutput()
		}
		if err == nil {
			err = p.wait(ctx, waiting)
			waiting = nil
		}
		if err != nil {
			failures = append(failures, err)
		}
	}
	for _, p := range f.runs {
		if err := p.stat.Close(); err != nil {
			failures = append(failures, err)
		}
	}
	f.joinErr = errors.Join(failures...)
	return f.joinErr
}

// The hook is a deterministic terminal barrier, not a delay chosen to hit
// RemoveAll. Exercise both successful and failing hooks: neither changes wait.
func TestDetachedFixtureJoinsSupervisorAfterTerminalHook(t *testing.T) {
	for _, hookCode := range []int{0, 3} {
		var root string
		t.Run(strconv.Itoa(hookCode), func(t *testing.T) {
			root = dispatchInstance(t)
			held := newGate(t)
			entered := filepath.Join(root, "terminal-entered")
			finished := filepath.Join(root, "terminal-finished")
			hook := "printf entered > " + shellQuote(entered) + "; " + held.wait +
				"; printf finished > " + shellQuote(finished) + "; exit " + strconv.Itoa(hookCode)
			writeDelegation(t, root, `{"schema_version":2,"agents":{
  "ok":{"vendor":"test","model":"ok-1","commands":{"read":["sh","-c","cat >/dev/null; echo decision: approved"]},
  "on_terminal":{"command":["sh","-c",`+strconv.Quote(hook)+`]}}
},"purposes":{"review":{"max_runs_per_bundle":9}}}`)
			code, out := runBuilt(t, root, "delegate", "review", "spec:alpha", "--detach", "--json")
			var doc struct{ Attempt string }
			if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc.Attempt == "" {
				t.Fatalf("detach: %d %s", code, out)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			tick := time.NewTicker(20 * time.Millisecond)
			defer tick.Stop()
			for {
				if _, err := os.Stat(entered); err == nil {
					break
				} else if !os.IsNotExist(err) {
					t.Fatal(err)
				}
				select {
				case <-ctx.Done():
					t.Fatal("supervisor never entered on_terminal")
				case <-tick.C:
				}
			}
			if code, out := runBuilt(t, root, "delegate", "wait", doc.Attempt, "--timeout", "200ms", "--json"); code != 0 || !strings.Contains(out, `"state":"completed"`) || !strings.Contains(out, `"decision":"approved"`) {
				t.Fatalf("wait while on_terminal is blocked: %d %s", code, out)
			}
			if _, err := os.Stat(filepath.Join(root, ".pose", "review-private", "active", doc.Attempt+".json")); !os.IsNotExist(err) {
				t.Fatalf("terminal run still has an active marker: %v", err)
			}
			f := detachedFixtureFor(root)
			if f == nil || len(f.runs) != 1 {
				t.Fatal("the fixture did not acquire its supervisor")
			}
			waiting := make(chan struct{})
			joined := make(chan error, 1)
			go func() { joined <- f.joinObserved(waiting) }()
			select {
			case <-waiting: // join observed the pinned supervisor still alive
			case err := <-joined:
				t.Fatalf("join returned before the hook was released: %v", err)
			case <-ctx.Done():
				t.Fatal("join never observed the blocked supervisor")
			}
			select {
			case err := <-joined:
				t.Fatalf("join passed the terminal barrier: %v", err)
			default:
			}
			if _, err := os.Stat(finished); !os.IsNotExist(err) {
				t.Fatalf("hook passed its barrier: %v", err)
			}
			held.open()
			select {
			case err := <-joined:
				if err != nil {
					t.Fatal(err)
				}
			case <-ctx.Done():
				t.Fatal("join did not observe supervisor exit after release")
			}
			if raw, err := os.ReadFile(finished); err != nil || string(raw) != "finished" {
				t.Fatalf("hook's last write: %q %v", raw, err)
			}
			if code, out := runBuilt(t, root, "delegate", "ls", doc.Attempt); code != 0 || strings.Contains(out, "on_terminal_failed=") != (hookCode != 0) {
				t.Fatalf("hook outcome after join: %d %s", code, out)
			}
			if code, out := runBuilt(t, root, "delegate", "wait", doc.Attempt, "--json"); code != 0 || !strings.Contains(out, `"state":"completed"`) {
				t.Fatalf("wait after supervisor exit: %d %s", code, out)
			}
		})
		// This is the real root cleanup, including every post-terminal write.
		if root != "" {
			if _, err := os.Stat(root); !os.IsNotExist(err) {
				t.Errorf("fixture root was not safely removed after join: %v", err)
			}
		}
	}
}

func TestDetachedFixtureCleanupReleasesAndJoinsEveryStart(t *testing.T) {
	var root string
	t.Run("early-return", func(t *testing.T) {
		root = dispatchInstance(t)
		held := newGate(t)
		writeDelegation(t, root, `{"schema_version":2,"agents":{
  "ok":{"vendor":"test","model":"ok-1","commands":{"read":["sh","-c","cat >/dev/null; echo decision: approved"]},
  "on_terminal":{"command":["sh","-c",`+strconv.Quote(held.wait)+`]}}
},"purposes":{"review":{"max_runs_per_bundle":9}}}`)
		for _, reason := range []string{"first held run", "second held run"} {
			code, out := runBuilt(t, root, "delegate", "review", "spec:alpha", "--detach", "--reason", reason, "--json")
			var doc struct{ Attempt string }
			if code != 0 || json.Unmarshal([]byte(out), &doc) != nil || doc.Attempt == "" {
				t.Fatalf("detach: %d %s", code, out)
			}
			// A second attempt on the same bundle requires the first run's
			// terminal result. Its supervisor remains held in on_terminal.
			if code, out := runBuilt(t, root, "delegate", "wait", doc.Attempt, "--timeout", "20s", "--json"); code != 0 || !strings.Contains(out, `"state":"completed"`) {
				t.Fatalf("run did not reach the terminal hook: %d %s", code, out)
			}
		}
		if f := detachedFixtureFor(root); f == nil || len(f.runs) != 2 {
			t.Fatal("the fixture did not acquire both supervisors")
		}
		// No explicit release or supervisor join. Like Fatal, this early return
		// relies entirely on registered cleanup to release and join both runs.
		return
	})
	if root != "" {
		if _, err := os.Stat(root); !os.IsNotExist(err) {
			t.Errorf("early-return fixture was not safely removed: %v", err)
		}
	}
}
