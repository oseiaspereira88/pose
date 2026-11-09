package pose

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// The common operation layer (spec pose-common-domain-and-cli-mcp-handoff).
//
// An MCP route is one of five things, and which one decides how it may share code
// with the CLI:
//
//   - domain-direct: the adapter calls the domain package;
//   - cli-wrapper: the adapter asks the CLI to run (a process, or, for a read-only
//     operation whose semantics the CLI package owns, the same function in process);
//   - read-only-planning: the adapter returns a plan that writes nothing;
//   - admitted-write: the adapter reaches a write point of the domain, which
//     revalidates authority, assurance, policy and subjects itself;
//   - external-handoff: the operation belongs to another actor or surface; the
//     adapter returns a typed handoff and executes nothing.
//
// stdio is not an IPC channel shared between clients, and sharing code never lends
// one transport the identity, signature or human presence of another.

// RouteClass names how an MCP route relates to the CLI.
type RouteClass string

const (
	RouteDomainDirect    RouteClass = "domain-direct"
	RouteCLIWrapper      RouteClass = "cli-wrapper"
	RouteReadOnlyPlan    RouteClass = "read-only-planning"
	RouteAdmittedWrite   RouteClass = "admitted-write"
	RouteExternalHandoff RouteClass = "external-handoff"
)

// InProcessCommand is a CLI command whose semantics the CLI package owns and that
// is read-only and bounded, so an adapter may run it in process instead of
// spawning the binary.
type InProcessCommand func(root string, args []string, stdout, stderr io.Writer) int

var (
	inProcessMu       sync.RWMutex
	inProcessCommands = map[string]InProcessCommand{}

	// cliSpawnAttempts counts the times a wrapper tried to start the POSE binary:
	// the evidence that an extracted operation no longer does (R1, R9).
	cliSpawnAttempts atomic.Int64
	// cliInProcessRuns counts the runs served in process.
	cliInProcessRuns atomic.Int64
	// inProcessSlots bounds the in-process runs alive at once, abandoned ones
	// included: a run that outlives its deadline keeps its slot until it finishes,
	// so repeated timeouts cannot accumulate work beyond the slots (R1).
	inProcessSlots = make(chan struct{}, inProcessMaxRuns)
)

// inProcessMaxRuns is the number of in-process runs a server keeps alive at once.
const inProcessMaxRuns = 4

// RegisterInProcessCommand lets the CLI package offer a read-only command to the
// wrappers of this package. Registering a command that writes is a defect: the
// wrapper gives it no process boundary and no way to be killed.
func RegisterInProcessCommand(name string, run InProcessCommand) {
	inProcessMu.Lock()
	defer inProcessMu.Unlock()
	inProcessCommands[name] = run
}

func inProcessCommand(name string) (InProcessCommand, bool) {
	inProcessMu.RLock()
	defer inProcessMu.RUnlock()
	run, ok := inProcessCommands[name]
	return run, ok
}

// CLISpawnAttempts and CLIInProcessRuns report the counters for measurement.
func CLISpawnAttempts() int64 { return cliSpawnAttempts.Load() }
func CLIInProcessRuns() int64 { return cliInProcessRuns.Load() }

// inProcessRunsTheSpawnedEngine reports whether running a command in this process
// answers as the spawn of the installed binary would. It does not when POSE_EXECUTABLE
// names another binary, or when the file this process started from has been replaced
// since (`pose update` writes the new engine at the same path): the spawned run would
// use the updated engine, and a running server holds the old one in memory (spec
// pose-mcp-server-survives-self-update R2, which the in-process route must not break).
// A test process has no installed binary to compare with and runs in process.
func inProcessRunsTheSpawnedEngine() bool {
	if os.Getenv("POSE_EXECUTABLE") != "" {
		return false
	}
	if executableAtStartErr != nil || strings.HasSuffix(executableAtStart, ".test") || executableInfoAtStart == nil {
		return true
	}
	now, err := os.Stat(executableAtStart)
	if err != nil {
		return false // the installed binary is gone: the spawn reports it, a stale engine does not
	}
	return os.SameFile(executableInfoAtStart, now)
}

// runInProcess runs a registered read-only command with the same contract runCLI
// gives a spawned one: a non-zero exit is an error carrying the command's message,
// and a deadline or cancellation is an execution failure with no verdict, never a
// failed result. A command cannot be killed, so a stopped run leaves its goroutine
// to finish; registered commands are read-only and bounded, and the disclosure says so.
func (s Store) runInProcess(ctx context.Context, run InProcessCommand, args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, cliTimeout)
	defer cancel()
	failure := func(cleanup string) error {
		return &GateExecutionError{Command: "pose " + strings.Join(args, " "), Cause: executionCause(ctx), Cleanup: cleanup}
	}
	// Cancelled before starting: nothing runs.
	if ctx.Err() != nil {
		return nil, failure("the command did not start; nothing ran and nothing was written")
	}
	// Admission: a bounded number of runs, abandoned ones counted. A caller that cannot
	// get a slot before its context ends has an execution failure, not a verdict.
	select {
	case inProcessSlots <- struct{}{}:
	case <-ctx.Done():
		return nil, failure("the command did not start: every in-process slot is held by a run still finishing")
	}
	type outcome struct {
		stdout, stderr []byte
		code           int
	}
	// A caller that was cancelled while it waited for a slot does not start the command.
	if ctx.Err() != nil {
		<-inProcessSlots
		return nil, failure("the command did not start; nothing ran and nothing was written")
	}
	done := make(chan outcome, 1)
	go func() {
		defer func() { <-inProcessSlots }()
		var stdout, stderr bytes.Buffer
		// A panic in the command must not take the server down, as the exit of a spawned
		// process never did: it becomes a failed run with the panic as its message.
		code := func() (code int) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintf(&stderr, "panic: %v", r)
					code = 2
				}
			}()
			return run(s.Root, args[1:], &stdout, &stderr)
		}()
		done <- outcome{stdout.Bytes(), stderr.Bytes(), code}
	}()
	var res outcome
	select {
	case <-ctx.Done():
		return nil, failure("the read-only command was running in this process and finishes in the background; it wrote nothing")
	case res = <-done:
	}
	// A completion that races a cancellation is not a verdict: the caller asked to stop.
	if ctx.Err() != nil {
		return nil, failure("the read-only command finished after the cancellation; its result was discarded and it wrote nothing")
	}
	cliInProcessRuns.Add(1)
	if res.code != 0 {
		msg := strings.TrimSpace(string(res.stderr))
		if msg == "" {
			msg = strings.TrimSpace(string(res.stdout))
		}
		if msg == "" {
			msg = fmt.Sprintf("exit status %d", res.code)
		}
		return nil, fmt.Errorf("pose %s failed: %s", args[0], msg)
	}
	return res.stdout, nil
}

// OperationHandoff is the typed description of an operation another actor or
// surface performs (spec pose-common-domain-and-cli-mcp-handoff R5). It names
// what to run as structured arguments and what it was computed from; it executes
// nothing, carries no secret, is not an approval, and does not retarget itself:
// when the inputs it names are no longer current, the operation must be planned
// again.
type OperationHandoff struct {
	Operation string `json:"operation"`
	// RequestID identifies the handoff: the same operation, scope and plan give the
	// same id, so a resumption can be matched to the plan it came from.
	RequestID string `json:"request_id"`
	Scope     string `json:"scope"`
	Phase     string `json:"phase"`
	// Argv is the CLI invocation, one element per argument. It is not a shell line.
	Argv []string `json:"argv"`
	// Inputs bind the handoff to the state it was computed from.
	Inputs HandoffInputs `json:"inputs"`
	// RequiredAuthority says who may perform it; the handoff grants none.
	RequiredAuthority string `json:"required_authority"`
	// Executes is always false: the adapter that returned the handoff did not run it.
	Executes bool `json:"executes"`
	// Target is the project and root the plan was computed in. The handoff does not
	// retarget itself: the performer applies it there, after verifying that context.
	Target HandoffTarget `json:"target"`
	// ContextRequirement says what the performer verifies before applying: the
	// adapter never infers a context revision from a context it merely presented
	// (R3), and `pose close --apply` takes no context flag, only the plan digest.
	ContextRequirement string `json:"context_requirement"`
	// Receipt says how the outcome comes back: the CLI result and `pose close
	// --resume` read the state again.
	Receipt string `json:"receipt"`
	// Invalidation says when this handoff stops being valid.
	Invalidation string `json:"invalidation"`
}

// HandoffTarget names the project an MCP caller selected and its root.
type HandoffTarget struct {
	ProjectID string `json:"project_id,omitempty"`
	Root      string `json:"root,omitempty"`
}

// HandoffInputs are the digests a handoff was computed from.
type HandoffInputs struct {
	PlanDigest   string `json:"plan_digest"`
	SpecDigest   string `json:"spec_digest"`
	Revision     string `json:"revision"`
	PolicyDigest string `json:"policy_digest"`
	Generation   string `json:"generation,omitempty"`
}

// CloseoutHandoff describes `pose close <scope> --apply` for a plan that has steps
// left, or nil when nothing is left to apply. The plan is the one the read-only
// planner computed; this function reads it and writes nothing.
func CloseoutHandoff(plan CloseoutPlan, target HandoffTarget) *OperationHandoff {
	if plan.Terminal || plan.Scope == "" || len(plan.Blockers) > 0 {
		return nil
	}
	if plan.ReadView != nil && plan.ReadView.Freshness == ReadViewFreshnessUnknown {
		return nil // inputs not proven stable: the plan must be recomputed, not applied
	}
	pending, authority := false, "maintainer"
	for _, step := range plan.Steps {
		switch step.State {
		case CloseoutStateBlocked:
			return nil // a blocked step stops the apply: there is nothing to hand off
		case CloseoutStatePending, CloseoutStateWaiting:
			pending = true
		}
		if step.ID == CloseoutStepJudgment && step.State == CloseoutStateWaiting {
			authority = "reviewer (a different-actor judgment is pending; the apply stops there)"
		}
	}
	if !pending {
		return nil
	}
	sum := sha256.Sum256([]byte("close\x00" + plan.Scope + "\x00" + plan.Digest))
	handoff := &OperationHandoff{
		Operation: "close",
		RequestID: "hnd-" + hex.EncodeToString(sum[:8]),
		Scope:     plan.Scope,
		Phase:     "closeout",
		Argv:      []string{"pose", "close", plan.Scope, "--apply", "--digest", plan.Digest},
		Inputs: HandoffInputs{
			PlanDigest: plan.Digest, SpecDigest: plan.Inputs.SpecDigest, Revision: plan.Inputs.Revision, PolicyDigest: plan.Inputs.PolicyDigest,
		},
		Target:             target,
		RequiredAuthority:  authority,
		ContextRequirement: "run it in the target root, after `pose context --task " + plan.Scope + " --json` there shows the authority and revision this plan was computed under; `pose close --apply` takes no context flag, its binding is the plan digest, and this handoff carries no context",
		Receipt:            "the CLI result of the apply, then `pose close " + plan.Scope + " --resume` or `pose_closeout_state` with include_plan to read the state again",
		Invalidation:       "any change to the spec, the policy or the revision makes the plan digest stale; the index generation it was read at is recorded in inputs.generation for the performer to compare, and the apply does not check it; plan again instead of applying this handoff",
	}
	if plan.ReadView != nil && len(plan.ReadView.Bindings) > 0 {
		handoff.Inputs.Generation = plan.ReadView.Bindings[0].IndexGeneration
	}
	return handoff
}

// RouteInventory is the declared class of every MCP tool. A tool missing from it,
// or listed here and not advertised, fails TestEveryMCPRouteHasADeclaredClass: the
// inventory cannot drift from the catalog (R1).
type RouteInventory map[string]RouteClass

// SortedTools returns the tools of an inventory in order.
func (r RouteInventory) SortedTools() []string {
	out := make([]string, 0, len(r))
	for name := range r {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
