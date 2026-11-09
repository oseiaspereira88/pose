// Package worktreeruntime is the optional runtime of one worktree: a process that
// answers handshake, status and stop on a protected local socket (spec
// pose-optional-worktree-runtime). It hosts no domain operation and confers no
// authority: a client that attaches gains nothing it did not have standalone.
package worktreeruntime

import (
	"bufio"
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

	maxRequest       = 64 << 10
	requestDeadline  = 5 * time.Second
	defaultDrain     = 5 * time.Second
	socketNameDigits = 12
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
		return Identity{}, fmt.Errorf("%s is not a Git worktree: the runtime is identified by it", worktree)
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

type request struct {
	Op       string    `json:"op"`
	Identity *Identity `json:"identity,omitempty"`
	DrainMS  int       `json:"drain_ms,omitempty"`
}

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
}

// Options start a runtime.
type Options struct {
	Identity Identity
	Dir      string
	// Idle stops the runtime after this long without a request; zero never does.
	Idle time.Duration
	// Handler, when set, is called for each admitted request before the reply of an unknown
	// op; slice 1 hosts none, and the field exists so the in-flight count is the one a later
	// slice's operations will be counted by.
	Handler func(ctx context.Context, op string) (Reply, bool)
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
	if err := clearStale(path); err != nil {
		return err
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		listener.Close()
		return err
	}
	defer os.Remove(path)
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
			if s.inflight.Load() == 0 && time.Since(time.Unix(0, atomic.LoadInt64(&s.last))) >= s.opts.Idle {
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
	var req request
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
			writeReply(conn, Reply{Mismatch: mism, Error: "incompatible"})
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
		reply := s.drain(drain)
		writeReply(conn, reply)
		s.shutdown()
	default:
		if s.opts.Handler != nil {
			if reply, ok := s.opts.Handler(ctx, req.Op); ok {
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
	for s.inflight.Load() > 1 && time.Now().Before(end) {
		time.Sleep(5 * time.Millisecond)
	}
	left := s.inflight.Load() - 1
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

func (s *server) state(ok bool) Reply {
	id := s.opts.Identity
	return Reply{OK: ok, Identity: &id, Epoch: s.epoch, PID: os.Getpid(), StartedAt: s.started.UTC().Format(time.RFC3339),
		UptimeMS: time.Since(s.started).Milliseconds(), IdleMS: s.opts.Idle.Milliseconds(), InFlight: s.inflight.Load() - 1, Served: s.served.Load()}
}

func writeReply(conn net.Conn, r Reply) {
	raw, _ := json.Marshal(r)
	_, _ = conn.Write(append(raw, '\n'))
}

// Client talks to the runtime of one identity.
type Client struct {
	Identity Identity
	Dir      string
}

func (c Client) roundTrip(ctx context.Context, req request) (Reply, error) {
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
	_ = conn.SetDeadline(time.Now().Add(requestDeadline + time.Duration(req.DrainMS)*time.Millisecond))
	raw, _ := json.Marshal(req)
	if _, err := conn.Write(append(raw, '\n')); err != nil {
		return Reply{}, err
	}
	line, err := bufio.NewReader(io.LimitReader(conn, maxRequest)).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return Reply{}, err
	}
	var reply Reply
	if err := json.Unmarshal(line, &reply); err != nil {
		return Reply{}, fmt.Errorf("the runtime answered something that is not a reply: %w", err)
	}
	return reply, nil
}

// Handshake attaches: it sends the identity and gets the state of the runtime back, or an
// IncompatibleError naming each field that differs.
func (c Client) Handshake(ctx context.Context) (Reply, error) {
	id := c.Identity
	reply, err := c.roundTrip(ctx, request{Op: "handshake", Identity: &id})
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
	return c.roundTrip(ctx, request{Op: "status"})
}

// Stop asks the runtime to drain within the deadline and exit.
func (c Client) Stop(ctx context.Context, drain time.Duration) (Reply, error) {
	return c.roundTrip(ctx, request{Op: "stop", DrainMS: int(drain.Milliseconds())})
}
