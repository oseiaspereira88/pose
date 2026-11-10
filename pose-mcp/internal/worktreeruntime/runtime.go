// Package worktreeruntime is the optional runtime of one worktree: a process that
// answers handshake, status and stop on a protected local socket (spec
// pose-optional-worktree-runtime). It hosts no domain operation and confers no
// authority: a client that attaches gains nothing it did not have standalone.
package worktreeruntime

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const (
	// Protocol is the wire protocol; Schema names the message shapes.
	Protocol = 1
	Schema   = "pose.runtime.v1"
	// EnvDir overrides the directory of the endpoints (tests, unusual layouts).
	EnvDir = "POSE_RUNTIME_DIR"

	maxRequest = 64 << 10
	maxReply   = 16 << 20
	// A hosted operation with no deadline of its own may take this long; a longer one is cut to it.
	MaxHostedDuration = 10 * time.Minute
	requestDeadline   = 5 * time.Second
	defaultDrain      = 5 * time.Second
	socketNameDigits  = 12
)

// Identity is what must be the same on both ends of an attach. The location part (user,
// worktree, Git common directory) names the endpoint; the engine part (engine, build,
// protocol, schema) is compared at the handshake, so a new CLI that finds an old
// process after an update is refused by name instead of being given a different endpoint.
type Identity struct {
	UID          int    `json:"uid"`
	Worktree     string `json:"worktree"`
	GitCommonDir string `json:"git_common_dir"`
	Engine       string `json:"engine"`
	Build        string `json:"build"`
	Protocol     int    `json:"protocol"`
	Schema       string `json:"schema"`
}

// ResolveIdentity reads the identity of the worktree at root for this engine. The root's
// symlinks are resolved and the Git common directory is read, so neither project_id nor
// the common directory alone names a mutable tree.
func ResolveIdentity(root, engine, build string) (Identity, error) {
	worktree, err := filepath.EvalSymlinks(root)
	if err == nil {
		worktree, err = filepath.Abs(worktree)
	}
	if err != nil {
		return Identity{}, fmt.Errorf("the worktree %s cannot be resolved: %w", root, err)
	}
	out, err := posemodel.RawHistoryGit(worktree, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return Identity{}, fmt.Errorf("%s is not a Git worktree, or git failed (%v): the runtime is identified by it", worktree, err)
	}
	common := strings.TrimSpace(string(out))
	if resolved, err := filepath.EvalSymlinks(common); err == nil {
		common = resolved
	}
	return Identity{UID: os.Getuid(), Worktree: worktree, GitCommonDir: common, Engine: engine, Build: build, Protocol: Protocol, Schema: Schema}, nil
}

// location is the digest of the part that names the endpoint.
func (i Identity) location() string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d\x00%s\x00%s", i.UID, i.Worktree, i.GitCommonDir)))
	return hex.EncodeToString(sum[:])[:socketNameDigits]
}

// Mismatches names each field of the engine part where other differs from i.
func (i Identity) Mismatches(other Identity) []string {
	var out []string
	add := func(name string, mine, theirs any) {
		if fmt.Sprint(mine) != fmt.Sprint(theirs) {
			out = append(out, fmt.Sprintf("%s: this side %v, the other %v", name, mine, theirs))
		}
	}
	add("uid", i.UID, other.UID)
	add("worktree", i.Worktree, other.Worktree)
	add("git_common_dir", i.GitCommonDir, other.GitCommonDir)
	add("engine", i.Engine, other.Engine)
	add("build", i.Build, other.Build)
	add("protocol", i.Protocol, other.Protocol)
	add("schema", i.Schema, other.Schema)
	return out
}

// DefaultDir is the directory of the endpoints of the current user.
func DefaultDir() string {
	if dir := os.Getenv(EnvDir); dir != "" {
		return dir
	}
	if run := os.Getenv("XDG_RUNTIME_DIR"); filepath.IsAbs(run) {
		return filepath.Join(run, "pose")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("pose-runtime-%d", os.Getuid()))
}

// SocketPath is where the endpoint of the identity lives in dir.
func SocketPath(dir string, id Identity) string {
	return filepath.Join(dir, id.location()+".sock")
}

// Errors a client can tell apart.
var (
	ErrUnsupported   = errors.New("the worktree runtime is not supported on this platform: every command runs standalone")
	ErrNoRuntime     = errors.New("no runtime is running for this worktree")
	ErrAlreadyServed = errors.New("a runtime is already serving this worktree")
)

// IncompatibleError is an attach refused at the handshake.
type IncompatibleError struct{ Mismatch []string }

func (e *IncompatibleError) Error() string {
	return "the running runtime is not compatible: " + strings.Join(e.Mismatch, "; ")
}

// OutcomeUnknownError is a request that was sent and whose answer did not arrive whole: the
// operation may have run, so it is not repeated in another process unasked.
type OutcomeUnknownError struct{ Cause error }

func (e *OutcomeUnknownError) Error() string {
	return "the runtime did not answer a request it received: " + e.Cause.Error()
}

func (e *OutcomeUnknownError) Unwrap() error { return e.Cause }

// UnsafeError is an endpoint or directory that fails the ownership, mode or type checks.
type UnsafeError struct{ Path, Reason string }

func (e *UnsafeError) Error() string { return "unsafe runtime endpoint " + e.Path + ": " + e.Reason }

// EnsureDir creates dir (one level, mode 0700) when it is absent and checks it is a real
// directory, owned by the user and closed to group and others.
func EnsureDir(dir string) error {
	if !Supported() {
		return ErrUnsupported
	}
	info, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		info, err = os.Lstat(dir)
	}
	if err != nil {
		return err
	}
	return checkDir(dir, info)
}

func checkDir(dir string, info os.FileInfo) error {
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		return &UnsafeError{dir, "it is a symbolic link"}
	case !info.IsDir():
		return &UnsafeError{dir, "it is not a directory"}
	case info.Mode().Perm()&0o077 != 0:
		return &UnsafeError{dir, fmt.Sprintf("its mode %o lets group or others in", info.Mode().Perm())}
	}
	if uid, ok := fileOwnerUID(info); ok && int(uid) != os.Getuid() {
		return &UnsafeError{dir, fmt.Sprintf("it is owned by uid %d, not %d", uid, os.Getuid())}
	}
	return nil
}

// checkSocket is the check of an endpoint before it is dialled or replaced: a socket owned
// by the user, not a link or a file someone put there.
func checkSocket(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return nil, &UnsafeError{path, "it is not a socket"}
	}
	if uid, ok := fileOwnerUID(info); ok && int(uid) != os.Getuid() {
		return nil, &UnsafeError{path, fmt.Sprintf("it is owned by uid %d, not %d", uid, os.Getuid())}
	}
	return info, nil
}

// Request is one message to a runtime. Args, Env and DeadlineMS belong to a hosted operation:
// the command line, the digest of the environment the caller ran in and how long it may take.
type Request struct {
	Op         string    `json:"op"`
	Identity   *Identity `json:"identity,omitempty"`
	DrainMS    int       `json:"drain_ms,omitempty"`
	Args       []string  `json:"args,omitempty"`
	Env        string    `json:"env,omitempty"`
	DeadlineMS int       `json:"deadline_ms,omitempty"`
}

// Outcomes of a hosted operation. A refused one did not start; an error, a deadline and an
// unknown outcome may have had effects, so a caller never repeats them elsewhere unasked.
const (
	OutcomeCompleted = "completed"
	OutcomeRefused   = "refused"
	OutcomeDeadline  = "deadline"
	OutcomeError     = "error"
)

// Reply is the answer of a runtime.
type Reply struct {
	OK        bool      `json:"ok"`
	Mismatch  []string  `json:"mismatch,omitempty"`
	Error     string    `json:"error,omitempty"`
	Identity  *Identity `json:"identity,omitempty"`
	Epoch     string    `json:"epoch,omitempty"`
	PID       int       `json:"pid,omitempty"`
	StartedAt string    `json:"started_at,omitempty"`
	UptimeMS  int64     `json:"uptime_ms,omitempty"`
	IdleMS    int64     `json:"idle_timeout_ms,omitempty"`
	InFlight  int64     `json:"in_flight"`
	Served    int64     `json:"served"`
	// Drained and Abandoned answer a stop: the requests in flight that finished within the
	// drain deadline and the ones that did not.
	Drained   int64 `json:"drained,omitempty"`
	Abandoned int64 `json:"abandoned,omitempty"`
	// A hosted operation's typed result.
	Outcome   string `json:"outcome,omitempty"`
	Reason    string `json:"reason,omitempty"`
	ExitCode  int    `json:"exit_code,omitempty"`
	Stdout    string `json:"stdout,omitempty"`
	Stderr    string `json:"stderr,omitempty"`
	ElapsedMS int64  `json:"elapsed_ms,omitempty"`
	MemoHits  int64  `json:"memo_hits,omitempty"`
}

// Options start a runtime.
type Options struct {
	Identity Identity
	Dir      string
	// Idle stops the runtime after this long without a request; zero never does.
	Idle time.Duration
	// Busy, when set, counts work a handler left running after it answered (a command whose
	// deadline passed): it is in flight for the idle timeout, the drain and the status.
	Busy func() int64
	// Handler, when set, is called for each request the runtime does not answer itself, and
	// counts as in flight while it runs; it reports whether it hosts the operation.
	Handler func(ctx context.Context, req Request) (Reply, bool)
}

// Serve runs the runtime until ctx ends, a client stops it or it idles out. The epoch of
// each run is new: nothing a previous process knew is inherited.
func Serve(ctx context.Context, opts Options) error {
	if !Supported() {
		return ErrUnsupported
	}
	if err := EnsureDir(opts.Dir); err != nil {
		return err
	}
	path := SocketPath(opts.Dir, opts.Identity)
	// One lock per endpoint, held for the life of the runtime: of two concurrent starts only
	// one holds it, so only the holder checks for, replaces or removes the socket.
	unlock, err := lockEndpoint(path + ".lock")
	if err != nil {
		return err
	}
	defer unlock()
	if err := clearStale(path); err != nil {
		return err
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	// The listener would unlink the path on close whatever is there by then; the guard below
	// removes it only if it is still ours.
	if ul, ok := listener.(*net.UnixListener); ok {
		ul.SetUnlinkOnClose(false)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		listener.Close()
		return err
	}
	ours, _ := os.Lstat(path)
	// The endpoint is removed only if it is still the one this process created.
	defer func() {
		if now, err := os.Lstat(path); err == nil && ours != nil && os.SameFile(ours, now) {
			os.Remove(path)
		}
	}()
	var token [8]byte
	_, _ = rand.Read(token[:])
	s := &server{opts: opts, listener: listener, epoch: hex.EncodeToString(token[:]), started: time.Now(), last: time.Now().UnixNano(), done: make(chan struct{})}
	return s.run(ctx)
}

// clearStale removes an endpoint nobody answers on, and refuses one somebody does or one
// that is not a socket of the user.
func clearStale(path string) error {
	if _, err := checkSocket(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	conn, err := net.DialTimeout("unix", path, time.Second)
	if err == nil {
		conn.Close()
		return ErrAlreadyServed
	}
	return os.Remove(path)
}

type server struct {
	opts     Options
	listener net.Listener
	epoch    string
	started  time.Time
	last     int64 // unix nanos of the last request, atomic
	inflight atomic.Int64
	served   atomic.Int64
	wg       sync.WaitGroup
	done     chan struct{}
	once     sync.Once
}

func (s *server) run(ctx context.Context) error {
	go func() {
		select {
		case <-ctx.Done():
			s.shutdown()
		case <-s.done:
		}
	}()
	if s.opts.Idle > 0 {
		go s.watchIdle()
	}
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				s.wg.Wait()
				return nil
			default:
				return err
			}
		}
		s.inflight.Add(1)
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer s.inflight.Add(-1)
			s.handle(ctx, conn)
		}()
	}
}

func (s *server) shutdown() { s.once.Do(func() { close(s.done); s.listener.Close() }) }

func (s *server) watchIdle() {
	tick := time.NewTicker(minDuration(s.opts.Idle/4, time.Second))
	defer tick.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-tick.C:
			if s.inflight.Load()+s.busy() == 0 && time.Since(time.Unix(0, atomic.LoadInt64(&s.last))) >= s.opts.Idle {
				s.shutdown()
				return
			}
		}
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b && a > 0 {
		return a
	}
	return b
}

func (s *server) handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	atomic.StoreInt64(&s.last, time.Now().UnixNano())
	// The operating system says who is on the other end before a byte of it is read.
	if uc, ok := conn.(*net.UnixConn); !ok {
		return
	} else if uid, err := peerUID(uc); err != nil || int(uid) != os.Getuid() {
		return
	}
	_ = conn.SetDeadline(time.Now().Add(requestDeadline))
	line, err := bufio.NewReaderSize(io.LimitReader(conn, maxRequest), 4096).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return
	}
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		writeReply(conn, Reply{Error: "the request is not valid JSON"})
		return
	}
	s.served.Add(1)
	switch req.Op {
	case "handshake":
		if req.Identity == nil {
			writeReply(conn, Reply{Error: "a handshake carries the client's identity"})
			return
		}
		if mism := s.opts.Identity.Mismatches(*req.Identity); len(mism) > 0 {
			writeReply(conn, Reply{Mismatch: mism, Error: "incompatible", PID: os.Getpid()})
			return
		}
		writeReply(conn, s.state(true))
	case "status":
		writeReply(conn, s.state(true))
	case "stop":
		drain := defaultDrain
		if req.DrainMS > 0 {
			drain = time.Duration(req.DrainMS) * time.Millisecond
		}
		// No new connection is accepted while the requests in flight drain.
		s.shutdown()
		// The reply of a long drain must still be writable.
		_ = conn.SetDeadline(time.Now().Add(drain + requestDeadline))
		reply := s.drain(drain)
		writeReply(conn, reply)
	default:
		if s.opts.Handler != nil {
			// The reply of a long operation must still be writable.
			_ = conn.SetDeadline(time.Now().Add(hostedDuration(req.DeadlineMS) + requestDeadline))
			if reply, ok := s.opts.Handler(ctx, req); ok {
				id := s.opts.Identity
				reply.Identity, reply.Epoch, reply.PID = &id, s.epoch, os.Getpid()
				writeReply(conn, reply)
				return
			}
		}
		writeReply(conn, Reply{Error: fmt.Sprintf("the runtime does not host the operation %q", req.Op)})
	}
}

// drain waits, up to the deadline, for the requests in flight besides this one.
func (s *server) drain(deadline time.Duration) Reply {
	before := s.inflight.Load() - 1
	end := time.Now().Add(deadline)
	for s.inflight.Load()+s.busy() > 1 && time.Now().Before(end) {
		time.Sleep(5 * time.Millisecond)
	}
	left := s.inflight.Load() + s.busy() - 1
	if left < 0 {
		left = 0
	}
	if before < 0 {
		before = 0
	}
	r := s.state(true)
	r.Drained, r.Abandoned = before-left, left
	return r
}

func (s *server) busy() int64 {
	if s.opts.Busy == nil {
		return 0
	}
	return s.opts.Busy()
}

func (s *server) state(ok bool) Reply {
	id := s.opts.Identity
	return Reply{OK: ok, Identity: &id, Epoch: s.epoch, PID: os.Getpid(), StartedAt: s.started.UTC().Format(time.RFC3339),
		UptimeMS: time.Since(s.started).Milliseconds(), IdleMS: s.opts.Idle.Milliseconds(), InFlight: s.inflight.Load() + s.busy() - 1, Served: s.served.Load()}
}

func writeReply(conn net.Conn, r Reply) {
	// Without HTML escaping a < or & in a command's output costs one byte, not six; the output
	// travels as UTF-8 text, so bytes that are not valid UTF-8 come back replaced.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(r)
	_, _ = conn.Write(buf.Bytes())
}

// Client talks to the runtime of one identity.
type Client struct {
	Identity Identity
	Dir      string
}

func (c Client) roundTrip(ctx context.Context, req Request) (Reply, error) {
	if !Supported() {
		return Reply{}, ErrUnsupported
	}
	if info, err := os.Lstat(c.Dir); err == nil {
		if err := checkDir(c.Dir, info); err != nil {
			return Reply{}, err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		return Reply{}, ErrNoRuntime
	} else {
		return Reply{}, err
	}
	path := SocketPath(c.Dir, c.Identity)
	if _, err := checkSocket(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Reply{}, ErrNoRuntime
		}
		return Reply{}, err
	}
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		// Nobody answers: a stale endpoint is no runtime.
		return Reply{}, ErrNoRuntime
	}
	defer conn.Close()
	if uc, ok := conn.(*net.UnixConn); ok {
		if uid, err := peerUID(uc); err != nil || int(uid) != os.Getuid() {
			return Reply{}, &UnsafeError{path, "the process that answers is not the user's"}
		}
	}
	wait := time.Duration(req.DrainMS) * time.Millisecond
	if req.Op != "handshake" && req.Op != "status" && req.Op != "stop" {
		wait = hostedDuration(req.DeadlineMS)
	}
	_ = conn.SetDeadline(time.Now().Add(requestDeadline + wait))
	raw, _ := json.Marshal(req)
	if _, err := conn.Write(append(raw, '\n')); err != nil {
		return Reply{}, &OutcomeUnknownError{Cause: err}
	}
	line, err := bufio.NewReaderSize(io.LimitReader(conn, maxReply), 64<<10).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return Reply{}, &OutcomeUnknownError{Cause: err}
	}
	var reply Reply
	if uerr := json.Unmarshal(line, &reply); uerr != nil {
		// A reply that arrives in part (the connection closed, or the reply was cut at its
		// bound) is a request that may have been served: its outcome is unknown.
		cause := uerr
		if err != nil {
			cause = fmt.Errorf("%w (%v)", uerr, err)
		}
		return Reply{}, &OutcomeUnknownError{Cause: fmt.Errorf("the runtime answered something that is not a whole reply: %w", cause)}
	}
	return reply, nil
}

// Handshake attaches: it sends the identity and gets the state of the runtime back, or an
// IncompatibleError naming each field that differs.
func (c Client) Handshake(ctx context.Context) (Reply, error) {
	id := c.Identity
	reply, err := c.roundTrip(ctx, Request{Op: "handshake", Identity: &id})
	if err != nil {
		return Reply{}, err
	}
	if len(reply.Mismatch) > 0 {
		sort.Strings(reply.Mismatch)
		return reply, &IncompatibleError{Mismatch: reply.Mismatch}
	}
	if !reply.OK {
		return reply, errors.New(reply.Error)
	}
	return reply, nil
}

// Status is the state of the runtime without comparing identities; the caller reads the
// identity it carries.
func (c Client) Status(ctx context.Context) (Reply, error) {
	return c.roundTrip(ctx, Request{Op: "status"})
}

// hostedDuration is how long a hosted operation may take: its own deadline, or the default,
// and never more than the maximum.
func hostedDuration(deadlineMS int) time.Duration {
	d := time.Duration(deadlineMS) * time.Millisecond
	if d <= 0 || d > MaxHostedDuration {
		return MaxHostedDuration
	}
	return d
}

// Run sends a hosted operation: the command line, the digest of the caller's environment and
// the longest it may take (zero for the runtime's default).
func (c Client) Run(ctx context.Context, args []string, env string, deadline time.Duration) (Reply, error) {
	return c.roundTrip(ctx, Request{Op: "run", Args: args, Env: env, DeadlineMS: int(deadline.Milliseconds())})
}

// Do sends an operation the runtime hosts and returns its reply.
func (c Client) Do(ctx context.Context, op string) (Reply, error) {
	return c.roundTrip(ctx, Request{Op: op})
}

// WaitReleased returns when the endpoint of the identity can be taken: nothing answers and the
// previous holder has released its lock, or the timeout passed.
func (c Client) WaitReleased(ctx context.Context, timeout time.Duration) bool {
	if !Supported() {
		return true
	}
	end := time.Now().Add(timeout)
	for time.Now().Before(end) && ctx.Err() == nil {
		if _, err := c.Status(ctx); errors.Is(err, ErrNoRuntime) {
			if unlock, err := lockEndpoint(SocketPath(c.Dir, c.Identity) + ".lock"); err == nil {
				unlock()
				return true
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// Stop asks the runtime to drain within the deadline and exit.
func (c Client) Stop(ctx context.Context, drain time.Duration) (Reply, error) {
	return c.roundTrip(ctx, Request{Op: "stop", DrainMS: int(drain.Milliseconds())})
}
