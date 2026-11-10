package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
	"github.com/harne8/pose-mcp/internal/worktreeruntime"
)

// Spec pose-worktree-runtime-resident-cache.

const runtimeRunSchema = "pose.runtime.run/1"

// maxHostedOutput bounds what a hosted command's reply carries; past it the reply says the
// command ran and its output is not sent.
const maxHostedOutput = 2 << 20

// hostedCommands are the commands the runtime runs for a caller: the checks that read the
// project and answer. They run as the user that started the runtime, in its worktree, with
// the environment digest the caller proved equal to the runtime's: a caller never gains,
// through the runtime, an authority it does not have standalone.
var hostedCommands = map[string]bool{
	"review-check": true, "closeout-check": true, "artifact-check": true, "surface-check": true, "roadmap-check": true,
}

// runtimeEnvDigest is the digest of the environment that decides what a command answers: the
// POSE_, HARNE8_, GIT_, CLAUDE and CODEX variables (project roots, identities, session markers
// seals detect), the locale, the home and state directories and NO_COLOR.
func runtimeEnvDigest(environ []string) string {
	keep := func(k string) bool {
		for _, p := range []string{"POSE_", "HARNE8_", "GIT_", "CLAUDE", "CODEX", "XDG_STATE_HOME", "XDG_CONFIG_HOME", "XDG_RUNTIME_DIR"} {
			if strings.HasPrefix(k, p) {
				return true
			}
		}
		switch k {
		case "LC_ALL", "LANG", "LANGUAGE", "HOME", "NO_COLOR", "USER", "LOGNAME":
			return true
		}
		return false
	}
	var pairs []string
	for _, kv := range environ {
		if k, _, ok := strings.Cut(kv, "="); ok && keep(k) {
			pairs = append(pairs, kv)
		}
	}
	sort.Strings(pairs)
	sum := sha256.Sum256([]byte(strings.Join(pairs, "\x00")))
	return hex.EncodeToString(sum[:])
}

// runtimeHostedRun runs a hosted command; tests replace it.
type hostedRunFunc func(args []string, out, errb io.Writer) int

var runtimeHostedRun atomic.Pointer[hostedRunFunc]

// Set at init: mainCommand reaches this file through cmdRuntime, so a variable initialised with
// it would be an initialization cycle.
func init() {
	run := hostedRunFunc(func(args []string, out, errb io.Writer) int { return mainCommand(args, out, errb) })
	runtimeHostedRun.Store(&run)
}

// runtimeHostedHandler runs the commands the runtime hosts (the commands read
// and may refresh the project's derived files, which they publish atomically as separate
// processes do) in the lanes of the admission policy, and answers with a typed result. Anything it does not host or will not run is refused before it
// starts.
func runtimeHostedHandler(envDigest string, busy *atomic.Int64, lanes *hostedLanes) func(context.Context, worktreeruntime.Request) (worktreeruntime.Reply, bool) {
	return func(ctx context.Context, req worktreeruntime.Request) (worktreeruntime.Reply, bool) {
		if req.Op != "run" {
			return worktreeruntime.Reply{}, false
		}
		refuse := func(why string) (worktreeruntime.Reply, bool) {
			return worktreeruntime.Reply{Outcome: worktreeruntime.OutcomeRefused, Reason: why}, true
		}
		switch {
		case len(req.Args) == 0 || !hostedCommands[req.Args[0]]:
			return refuse("the runtime does not host this command")
		case req.Env != envDigest:
			return refuse("the caller's environment differs from the one the runtime started in")
		}
		started := time.Now()
		// One hosted command at a time. The deadline covers the wait for its turn: a command
		// that has not started by then is withdrawn and refused, so its caller may run it
		// standalone without the runtime running it later as well.
		limit := time.Duration(req.DeadlineMS) * time.Millisecond
		if limit <= 0 || limit > worktreeruntime.MaxHostedDuration {
			limit = worktreeruntime.MaxHostedDuration
		}
		timer := time.NewTimer(limit)
		defer timer.Stop()
		lane := lanes.of(req.Args[0])
		if !lane.enter() {
			return refuse("the " + lane.name + " lane of the runtime is full: this command did not start")
		}
		waitStart := time.Now()
		select {
		case lane.slots <- struct{}{}:
			lane.admit(time.Since(waitStart))
		case <-timer.C:
			lane.withdrawn()
			return refuse("the runtime was busy with another command until the deadline: this one did not start")
		case <-ctx.Done():
			lane.withdrawn()
			return refuse("the runtime is stopping: this command did not start")
		}
		done := make(chan worktreeruntime.Reply, 1)
		// The command counts as work in the runtime until it ends, even after its caller was
		// answered at a deadline.
		busy.Add(1)
		go func() {
			defer busy.Add(-1)
			defer func() { <-lane.slots; lane.running.Add(-1) }()
			hitsBefore, _, _, _ := posemodel.ResidentGitMemoStats()
			var out, errb bytes.Buffer
			code := 1
			func() {
				defer func() {
					if r := recover(); r != nil {
						errb.WriteString("pose: the command panicked: " + fmt.Sprint(r) + "\n")
					}
				}()
				code = (*runtimeHostedRun.Load())(req.Args, &out, &errb)
			}()
			hitsAfter, _, _, _ := posemodel.ResidentGitMemoStats()
			if out.Len()+errb.Len() > maxHostedOutput {
				// The command ran; its output is too large to send whole.
				done <- worktreeruntime.Reply{Outcome: worktreeruntime.OutcomeError, Reason: "the command's output exceeds what the runtime sends back", ExitCode: code, ElapsedMS: time.Since(started).Milliseconds()}
				return
			}
			done <- worktreeruntime.Reply{Outcome: worktreeruntime.OutcomeCompleted, ExitCode: code, Stdout: out.String(), Stderr: errb.String(),
				ElapsedMS: time.Since(started).Milliseconds(), MemoHits: hitsAfter - hitsBefore}
		}()
		select {
		case reply := <-done:
			return reply, true
		case <-timer.C:
			// The command is read-only apart from refreshing derived files and cannot be
			// cancelled; it ends on its own and the next one waits for it.
			return worktreeruntime.Reply{Outcome: worktreeruntime.OutcomeDeadline, Reason: "the deadline passed while the command ran; it was not stopped", ElapsedMS: time.Since(started).Milliseconds()}, true
		}
	}
}

// cmdRuntimeRun is `pose runtime run [--envelope] [--timeout <d>] <command> [args...]`: the
// command runs in the runtime when one is running, compatible and willing, and standalone in
// this process otherwise. The exit code and the output are the command's; --envelope prints
// one document with them and with where it ran. A command that reached the runtime and whose
// answer did not come back whole is never rerun here: it may have run.
func cmdRuntimeRun(root string, args []string, stdout, stderr io.Writer) int {
	envelope, timeout := false, time.Duration(0)
	for len(args) > 0 && strings.HasPrefix(args[0], "--") {
		switch args[0] {
		case "--envelope":
			envelope = true
			args = args[1:]
		case "--timeout":
			if len(args) < 2 {
				render(stdout, stderr).Failure("pose runtime run: --timeout needs a duration")
				return 2
			}
			d, err := time.ParseDuration(args[1])
			if err != nil || d <= 0 {
				render(stdout, stderr).Failure("pose runtime run: " + args[1] + " is not a duration")
				return 2
			}
			timeout, args = d, args[2:]
		default:
			render(stdout, stderr).Failure("pose runtime run: unknown flag " + args[0])
			return 2
		}
	}
	if len(args) == 0 {
		render(stdout, stderr).Usage(tr("usage: pose runtime run [--envelope] [--timeout <d>] <command> [args...]", "uso: pose runtime run [--envelope] [--timeout <d>] <comando> [args...]"))
		return 2
	}
	via, reason := "standalone", ""
	var reply worktreeruntime.Reply
	id, err := worktreeruntime.ResolveIdentity(root, version.Version, runtimeBuild())
	if err == nil && hostedCommands[args[0]] {
		client := worktreeruntime.Client{Identity: id, Dir: worktreeruntime.DefaultDir()}
		if _, herr := client.Handshake(context.Background()); herr != nil {
			reason = herr.Error()
		} else {
			var rerr error
			reply, rerr = client.Run(context.Background(), args, runtimeEnvDigest(os.Environ()), timeout)
			var unknown *worktreeruntime.OutcomeUnknownError
			switch {
			case errors.As(rerr, &unknown):
				return runtimeRunUnknown(stdout, stderr, envelope, args, rerr)
			case rerr != nil:
				reason = rerr.Error()
			case reply.Outcome == worktreeruntime.OutcomeCompleted:
				via = "runtime"
			case reply.Outcome == worktreeruntime.OutcomeRefused:
				reason = reply.Reason
			case reply.Outcome == worktreeruntime.OutcomeError:
				// The command ran and ended; only its output was too large to bring back. The
				// outcome is known, so the exit code is the command's.
				via, reply.Stderr = "runtime", reply.Reason+"\n"
			default:
				return runtimeRunUnknown(stdout, stderr, envelope, args, errors.New(reply.Outcome+": "+reply.Reason))
			}
		}
	} else if err != nil {
		reason = err.Error()
	} else {
		reason = "the command is not one the runtime hosts"
	}
	if via == "standalone" {
		var out, errb bytes.Buffer
		started := time.Now()
		code := mainCommand(args, &out, &errb)
		reply = worktreeruntime.Reply{Outcome: worktreeruntime.OutcomeCompleted, ExitCode: code, Stdout: out.String(), Stderr: errb.String(), ElapsedMS: time.Since(started).Milliseconds()}
	}
	if envelope {
		doc := map[string]any{"schema": runtimeRunSchema, "via": via, "outcome": reply.Outcome, "exit_code": reply.ExitCode, "stdout": reply.Stdout, "stderr": reply.Stderr, "elapsed_ms": reply.ElapsedMS}
		if reason != "" {
			doc["fallback_reason"] = reason
		}
		if via == "runtime" {
			doc["epoch"], doc["memo_hits"] = reply.Epoch, reply.MemoHits
		}
		_ = json.NewEncoder(stdout).Encode(doc)
		return reply.ExitCode
	}
	// The command's own output, passed through byte for byte.
	_, _ = stdout.Write([]byte(reply.Stdout))
	_, _ = stderr.Write([]byte(reply.Stderr))
	return reply.ExitCode
}

func runtimeRunUnknown(stdout, stderr io.Writer, envelope bool, args []string, err error) int {
	msg := "pose runtime run " + args[0] + ": " + err.Error() + "; " + tr("it may have run, so it was not repeated here: check its effects before running it again", "pode ter rodado, então não foi repetido aqui: confira seus efeitos antes de rodá-lo de novo")
	if envelope {
		_ = json.NewEncoder(stdout).Encode(map[string]any{"schema": runtimeRunSchema, "via": "runtime", "outcome": "unknown", "exit_code": 1, "stderr": msg})
		return 1
	}
	render(stdout, stderr).Failure(msg)
	return 1
}
