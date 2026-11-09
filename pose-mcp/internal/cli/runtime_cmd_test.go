package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/worktreeruntime"
)

// Spec pose-optional-worktree-runtime R3/R4: the lifecycle commands and the standalone
// behaviour, through the command itself.
func TestRuntimeCommandsRunTheLifecycleAndFallBackToStandalone(t *testing.T) {
	if !worktreeruntime.Supported() {
		t.Skip("no protected local endpoint on this platform")
	}
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("no git: %v %s", err, out)
	}
	dir, err := os.MkdirTemp(os.Getenv("TMPDIR"), "rtc")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	t.Setenv(worktreeruntime.EnvDir, dir+"/r")
	run := func(args ...string) (string, int) {
		var out, errb bytes.Buffer
		code := cmdRuntime(root, args, &out, &errb)
		return out.String() + errb.String(), code
	}
	state := func(args ...string) (string, int) {
		out, code := run(append(args, "--json")...)
		var doc struct {
			State string `json:"state"`
			Epoch string `json:"epoch"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &doc); err != nil {
			t.Fatalf("not JSON: %q", out)
		}
		return doc.State + "|" + doc.Epoch, code
	}
	// No runtime: standalone is a state, not a fault; attach asked for it and cannot have it.
	if s, code := state("status"); !strings.HasPrefix(s, "standalone|") || code != 0 {
		t.Fatalf("status with no runtime: %s %d", s, code)
	}
	if s, code := state("attach"); !strings.HasPrefix(s, "standalone|") || code != 1 {
		t.Fatalf("attach with no runtime: %s %d", s, code)
	}
	if _, err := os.Stat(dir + "/r"); err == nil {
		t.Fatal("asking about a runtime prepared its directory")
	}
	// start: the spawn is replaced by the same runtime in-process.
	var stops []context.CancelFunc
	defer func() {
		for _, stop := range stops {
			stop()
		}
	}()
	original := runtimeSpawn
	runtimeSpawn = func(r string, idle time.Duration) error {
		ctx, cancel := context.WithCancel(context.Background())
		stops = append(stops, cancel)
		go cmdRuntime(r, []string{"serve", "--idle", idle.String()}, new(bytes.Buffer), new(bytes.Buffer))
		_ = ctx
		return nil
	}
	defer func() { runtimeSpawn = original }()
	out, code := state("start")
	if !strings.HasPrefix(out, "started|") || code != 0 {
		t.Fatalf("start: %s %d", out, code)
	}
	first := strings.SplitN(out, "|", 2)[1]
	if s, code := state("start"); !strings.HasPrefix(s, "already-running|") || code != 0 {
		t.Fatalf("a second start: %s %d", s, code)
	}
	if s, code := state("attach"); !strings.HasPrefix(s, "attach|") || code != 0 {
		t.Fatalf("attach: %s %d", s, code)
	}
	if s, code := state("restart"); !strings.HasPrefix(s, "started|") || code != 0 || strings.SplitN(s, "|", 2)[1] == first {
		t.Fatalf("restart did not give a new epoch: %s %d (was %s)", s, code, first)
	}
	if s, code := state("stop"); !strings.HasPrefix(s, "stopped|") || code != 0 {
		t.Fatalf("stop: %s %d", s, code)
	}
	if s, code := state("status"); !strings.HasPrefix(s, "standalone|") || code != 0 {
		t.Fatalf("status after stop: %s %d", s, code)
	}
	if _, code := run("nosuch"); code != 2 {
		t.Fatalf("an unknown subcommand exits %d", code)
	}
}
