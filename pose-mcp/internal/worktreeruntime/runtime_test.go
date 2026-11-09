package worktreeruntime

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Spec pose-optional-worktree-runtime.

func repo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("no git: %v %s", err, out)
	}
	return dir
}

func skipUnsupported(t *testing.T) {
	t.Helper()
	if !Supported() {
		t.Skip("no protected local endpoint on this platform")
	}
}

func shortDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(os.Getenv("TMPDIR"), "rt")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	dir = filepath.Join(dir, "r")
	return dir
}

func start(t *testing.T, id Identity, dir string, idle time.Duration) (context.CancelFunc, chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, Options{Identity: id, Dir: dir, Idle: idle}) }()
	client := Client{Identity: id, Dir: dir}
	for i := 0; i < 200; i++ {
		if _, err := client.Status(context.Background()); err == nil {
			return cancel, done
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	t.Fatal("the runtime did not come up")
	return nil, nil
}

func identity(t *testing.T, root, engine string) Identity {
	t.Helper()
	id, err := ResolveIdentity(root, engine, "b1")
	if err != nil {
		t.Skip(err)
	}
	return id
}

// R1: the identity names user, worktree and Git common directory; two worktrees of one
// repository do not share an endpoint, and the handshake refuses another engine by field.
func TestTheIdentityNamesTheWorktreeAndTheHandshakeRefusesAnotherEngine(t *testing.T) {
	skipUnsupported(t)
	a, b := repo(t), repo(t)
	ia, ib := identity(t, a, "7.4.1"), identity(t, b, "7.4.1")
	if ia.location() == ib.location() {
		t.Fatal("two worktrees share an endpoint name")
	}
	link := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(a, link); err != nil {
		t.Skip(err)
	}
	if got := identity(t, link, "7.4.1"); got.Worktree != ia.Worktree || got.location() != ia.location() {
		t.Fatalf("a symlink to the worktree is another identity: %v vs %v", got.Worktree, ia.Worktree)
	}
	dir := shortDir(t)
	cancel, done := start(t, ia, dir, 0)
	defer func() { cancel(); <-done }()
	if _, err := (Client{Identity: ia, Dir: dir}).Handshake(context.Background()); err != nil {
		t.Fatalf("a compatible attach was refused: %v", err)
	}
	newer := ia
	newer.Engine = "7.5.0"
	_, err := Client{Identity: newer, Dir: dir}.Handshake(context.Background())
	var inc *IncompatibleError
	if !errors.As(err, &inc) || len(inc.Mismatch) != 1 || !strings.HasPrefix(inc.Mismatch[0], "engine:") {
		t.Fatalf("a newer CLI against the old process: %v", err)
	}
	other := ia
	other.Protocol = 2
	if _, err := (Client{Identity: other, Dir: dir}).Handshake(context.Background()); err == nil {
		t.Fatal("another protocol attached")
	}
}

// R2: the directory and the endpoint are the user's alone; a link, a loose mode or a file
// that is not a socket is refused, and a stale socket is replaced.
func TestTheEndpointIsProtectedAndAStaleOneIsReplaced(t *testing.T) {
	skipUnsupported(t)
	id := identity(t, repo(t), "7.4.1")
	dir := shortDir(t)
	if err := EnsureDir(dir); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(dir); info.Mode().Perm() != 0o700 {
		t.Fatalf("the directory mode is %o", info.Mode().Perm())
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var unsafe *UnsafeError
	if err := EnsureDir(dir); !errors.As(err, &unsafe) {
		t.Fatalf("a directory open to others was accepted: %v", err)
	}
	_ = os.Chmod(dir, 0o700)
	link := dir + "-link"
	if err := os.Symlink(dir, link); err == nil {
		if err := EnsureDir(link); !errors.As(err, &unsafe) {
			t.Fatalf("a symlinked directory was accepted: %v", err)
		}
	}
	path := SocketPath(dir, id)
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Serve(context.Background(), Options{Identity: id, Dir: dir}); !errors.As(err, &unsafe) {
		t.Fatalf("a regular file where the socket goes was replaced: %v", err)
	}
	os.Remove(path)
	// A socket nobody answers on is stale: Serve takes the name over.
	l, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	l.(*net.UnixListener).SetUnlinkOnClose(false)
	l.Close()
	if _, err := (Client{Identity: id, Dir: dir}).Status(context.Background()); !errors.Is(err, ErrNoRuntime) {
		t.Fatalf("a stale endpoint is not 'no runtime': %v", err)
	}
	cancel, done := start(t, id, dir, 0)
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("the socket mode: %v %v", info, err)
	}
	if err := Serve(context.Background(), Options{Identity: id, Dir: dir}); !errors.Is(err, ErrAlreadyServed) {
		t.Fatalf("a second runtime for the same worktree started: %v", err)
	}
	cancel()
	<-done
	if _, err := os.Lstat(path); err == nil {
		t.Fatal("the endpoint was left behind")
	}
}

// R3: stop drains and says what finished; a restart has a new epoch; the idle timeout ends
// an idle runtime and removes its endpoint.
func TestTheLifecycleDrainsRestartsWithANewEpochAndIdlesOut(t *testing.T) {
	skipUnsupported(t)
	id := identity(t, repo(t), "7.4.1")
	dir := shortDir(t)
	cancel, done := start(t, id, dir, 0)
	c := Client{Identity: id, Dir: dir}
	first, err := c.Status(context.Background())
	if err != nil || first.Epoch == "" || first.PID != os.Getpid() {
		t.Fatalf("status: %+v %v", first, err)
	}
	stopped, err := c.Stop(context.Background(), time.Second)
	if err != nil || !stopped.OK || stopped.Abandoned != 0 {
		t.Fatalf("stop: %+v %v", stopped, err)
	}
	if err := <-done; err != nil {
		t.Fatalf("serve after stop: %v", err)
	}
	cancel()
	if _, err := c.Status(context.Background()); !errors.Is(err, ErrNoRuntime) {
		t.Fatalf("the runtime answers after stop: %v", err)
	}
	cancel2, done2 := start(t, id, dir, 0)
	second, _ := c.Status(context.Background())
	if second.Epoch == first.Epoch {
		t.Fatal("a restart inherited the epoch of the previous process")
	}
	cancel2()
	<-done2
	cancel3, done3 := start(t, id, dir, 120*time.Millisecond)
	defer cancel3()
	select {
	case err := <-done3:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("an idle runtime did not stop")
	}
}

// R4: with no runtime, a client gets ErrNoRuntime (the caller runs standalone) and nothing
// is started; an unsupported platform says so.
func TestWithNoRuntimeTheClientSaysSoAndStartsNothing(t *testing.T) {
	id := identity(t, repo(t), "7.4.1")
	dir := shortDir(t)
	_, err := Client{Identity: id, Dir: dir}.Handshake(context.Background())
	if Supported() {
		if !errors.Is(err, ErrNoRuntime) {
			t.Fatalf("handshake with no runtime: %v", err)
		}
		if _, statErr := os.Stat(dir); statErr == nil {
			t.Fatal("the client created the endpoint directory: attach must not autostart or prepare one")
		}
	} else if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("unsupported platform: %v", err)
	}
	_ = runtime.GOOS
}
