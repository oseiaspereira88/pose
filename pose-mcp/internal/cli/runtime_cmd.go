package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/harne8/pose-mcp/internal/version"
	"github.com/harne8/pose-mcp/internal/worktreeruntime"
)

// Spec pose-optional-worktree-runtime. The runtime is opt-in: nothing here starts it unless
// asked, and every other command runs standalone whether it runs or not.

const runtimeStatusSchema = "pose.runtime.status/1"

// runtimeSpawn starts the runtime of root detached; tests replace it.
var runtimeSpawn = func(root string, idle time.Duration) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(self, "runtime", "serve", "--idle", idle.String())
	cmd.Dir = root
	cmd.Env = os.Environ()
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	detachProcess(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func runtimeBuild() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				return s.Value
			}
		}
	}
	return "unknown"
}

type runtimeFlags struct {
	jsonOut bool
	idle    time.Duration
	drain   time.Duration
}

func parseRuntimeFlags(args []string) (runtimeFlags, string) {
	f := runtimeFlags{idle: 30 * time.Minute, drain: 5 * time.Second}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			f.jsonOut = true
		case "--idle", "--drain":
			if i+1 >= len(args) {
				return f, args[i] + " needs a duration"
			}
			d, err := time.ParseDuration(args[i+1])
			if err != nil || d < 0 {
				return f, args[i] + ": " + args[i+1] + " is not a duration"
			}
			if args[i] == "--idle" {
				f.idle = d
			} else {
				f.drain = d
			}
			i++
		default:
			return f, "unknown argument " + args[i]
		}
	}
	return f, ""
}

// cmdRuntime is `pose runtime serve|start|attach|status|stop|restart`.
func cmdRuntime(root string, args []string, stdout, stderr io.Writer) int {
	out := render(stdout, stderr)
	if len(args) == 0 {
		out.Usage(tr("usage: pose runtime <serve|start|attach|status|stop|restart> [--idle <d>] [--drain <d>] [--json]", "uso: pose runtime <serve|start|attach|status|stop|restart> [--idle <d>] [--drain <d>] [--json]"))
		return 2
	}
	sub := args[0]
	f, problem := parseRuntimeFlags(args[1:])
	if problem != "" {
		out.Failure("pose runtime: " + problem)
		return 2
	}
	id, err := worktreeruntime.ResolveIdentity(root, version.Version, runtimeBuild())
	if err != nil {
		out.Failure("pose runtime: " + err.Error())
		return 2
	}
	client := worktreeruntime.Client{Identity: id, Dir: worktreeruntime.DefaultDir()}
	ctx := context.Background()
	switch sub {
	case "serve":
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
		serveCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		go func() { <-sigs; cancel() }()
		if err := worktreeruntime.Serve(serveCtx, worktreeruntime.Options{Identity: id, Dir: client.Dir, Idle: f.idle}); err != nil {
			return runtimeFailure(out, stdout, f, "serve", err)
		}
		return 0
	case "status", "attach":
		reply, err := client.Handshake(ctx)
		return runtimeReport(out, stdout, f, sub, reply, err)
	case "stop":
		reply, err := client.Stop(ctx, f.drain)
		if errors.Is(err, worktreeruntime.ErrNoRuntime) || errors.Is(err, worktreeruntime.ErrUnsupported) {
			return runtimeReport(out, stdout, f, sub, reply, err)
		}
		if err != nil {
			return runtimeFailure(out, stdout, f, sub, err)
		}
		return runtimeReport(out, stdout, f, "stopped", reply, nil)
	case "start", "restart":
		if sub == "restart" {
			if _, err := client.Stop(ctx, f.drain); err != nil && !errors.Is(err, worktreeruntime.ErrNoRuntime) {
				return runtimeFailure(out, stdout, f, sub, err)
			}
			waitRuntimeGone(ctx, client)
		}
		if reply, err := client.Handshake(ctx); err == nil {
			return runtimeReport(out, stdout, f, "already-running", reply, nil)
		} else if !errors.Is(err, worktreeruntime.ErrNoRuntime) {
			return runtimeFailure(out, stdout, f, sub, err)
		}
		if !worktreeruntime.Supported() {
			return runtimeFailure(out, stdout, f, sub, worktreeruntime.ErrUnsupported)
		}
		if err := runtimeSpawn(root, f.idle); err != nil {
			return runtimeFailure(out, stdout, f, sub, err)
		}
		var reply worktreeruntime.Reply
		for i := 0; i < 100; i++ {
			if reply, err = client.Handshake(ctx); err == nil {
				return runtimeReport(out, stdout, f, "started", reply, nil)
			}
			time.Sleep(50 * time.Millisecond)
		}
		return runtimeFailure(out, stdout, f, sub, fmt.Errorf("the runtime did not answer after it was started: %w", err))
	}
	out.Failure("pose runtime: " + tr("unknown subcommand ", "subcomando desconhecido ") + sub)
	return 2
}

func waitRuntimeGone(ctx context.Context, client worktreeruntime.Client) {
	for i := 0; i < 100; i++ {
		if _, err := client.Status(ctx); err != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// runtimeReport writes the state a subcommand ends in. Not running is a state, not a fault:
// every command runs standalone then; only a runtime that refuses the attach, or an attach
// that was asked for and cannot be had, is a failure.
func runtimeReport(out interface{ Field(string, string) }, stdout io.Writer, f runtimeFlags, sub string, reply worktreeruntime.Reply, err error) int {
	state, code, detail := "running", 0, ""
	switch {
	case errors.Is(err, worktreeruntime.ErrUnsupported):
		state, detail = "unsupported", err.Error()
	case errors.Is(err, worktreeruntime.ErrNoRuntime):
		state, detail = "standalone", tr("no runtime is running for this worktree: commands run standalone", "nenhum runtime está rodando para este worktree: os comandos rodam standalone")
	case err != nil:
		var inc *worktreeruntime.IncompatibleError
		if errors.As(err, &inc) {
			state, code, detail = "incompatible", 1, err.Error()+tr("; stop it with `pose runtime stop` from its own engine", "; pare-o com `pose runtime stop` do próprio engine")
		} else {
			return runtimeFailureFields(out, stdout, f, sub, err)
		}
	default:
		state = sub
		if sub == "status" {
			state = "running"
		}
	}
	if sub == "attach" && (state == "standalone" || state == "unsupported") {
		code = 1
	}
	if f.jsonOut {
		doc := map[string]any{"schema": runtimeStatusSchema, "state": state, "detail": detail, "supported": worktreeruntime.Supported()}
		if reply.Epoch != "" {
			doc["epoch"], doc["pid"], doc["started_at"], doc["in_flight"], doc["served"] = reply.Epoch, reply.PID, reply.StartedAt, reply.InFlight, reply.Served
			doc["drained"], doc["abandoned"] = reply.Drained, reply.Abandoned
		}
		raw, _ := json.Marshal(doc)
		fmt.Fprintln(stdout, string(raw))
		return code
	}
	out.Field("runtime.state", state)
	if detail != "" {
		out.Field("runtime.detail", detail)
	}
	if reply.Epoch != "" {
		out.Field("runtime.epoch", reply.Epoch)
		out.Field("runtime.pid", fmt.Sprint(reply.PID))
		out.Field("runtime.started_at", reply.StartedAt)
		out.Field("runtime.in_flight", fmt.Sprint(reply.InFlight))
		if state == "stopped" {
			out.Field("runtime.drained", fmt.Sprint(reply.Drained))
			out.Field("runtime.abandoned", fmt.Sprint(reply.Abandoned))
		}
	}
	return code
}

func runtimeFailure(out interface {
	Field(string, string)
	Failure(string)
}, stdout io.Writer, f runtimeFlags, sub string, err error) int {
	out.Failure("pose runtime " + sub + ": " + err.Error())
	return 1
}

func runtimeFailureFields(out interface{ Field(string, string) }, stdout io.Writer, f runtimeFlags, sub string, err error) int {
	out.Field("runtime.error", strings.TrimSpace(err.Error()))
	return 1
}
