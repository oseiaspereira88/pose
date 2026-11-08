package pose

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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
)

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

// runInProcess runs a registered read-only command with the same contract runCLI
// gives a spawned one: a non-zero exit is an error carrying the command's message,
// and a deadline or cancellation is an execution failure with no verdict, never a
// failed result. A command cannot be killed, so a stopped run leaves its goroutine
// to finish; registered commands are read-only and bounded, and the disclosure says so.
func (s Store) runInProcess(ctx context.Context, run InProcessCommand, args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, cliTimeout)
	defer cancel()
	type outcome struct {
		stdout, stderr []byte
		code           int
	}
	done := make(chan outcome, 1)
	go func() {
		var stdout, stderr bytes.Buffer
		code := run(s.Root, args[1:], &stdout, &stderr)
		done <- outcome{stdout.Bytes(), stderr.Bytes(), code}
	}()
	select {
	case <-ctx.Done():
		return nil, &GateExecutionError{Command: "pose " + strings.Join(args, " "), Cause: executionCause(ctx), Cleanup: "the read-only command was running in this process and finishes in the background; it wrote nothing"}
	case res := <-done:
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
	// ContextRequirement says what the performer supplies at apply time: the adapter
	// never infers a context revision from a context it merely presented (R3).
	ContextRequirement string `json:"context_requirement"`
	// Receipt says how the outcome comes back: the CLI result and `pose close
	// --resume` read the state again.
	Receipt string `json:"receipt"`
	// Invalidation says when this handoff stops being valid.
	Invalidation string `json:"invalidation"`
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
func CloseoutHandoff(plan CloseoutPlan) *OperationHandoff {
	if plan.Terminal || plan.Scope == "" {
		return nil
	}
	pending, authority := false, "maintainer"
	for _, step := range plan.Steps {
		switch step.State {
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
		Argv:      []string{"pose", "close", plan.Scope, "--apply", "--digest", strings.TrimPrefix(plan.Digest, "sha256:")},
		Inputs: HandoffInputs{
			PlanDigest: plan.Digest, SpecDigest: plan.Inputs.SpecDigest, Revision: plan.Inputs.Revision, PolicyDigest: plan.Inputs.PolicyDigest,
		},
		RequiredAuthority:  authority,
		ContextRequirement: "pass --expect-context <digest> from a fresh `pose context --task " + plan.Scope + " --json` when applying; this handoff does not carry one",
		Receipt:            "the CLI result of the apply, then `pose close " + plan.Scope + " --resume` or `pose_closeout_state` with include_plan to read the state again",
		Invalidation:       "any change to the spec, the policy, the revision or the index generation makes the plan digest stale; plan again instead of applying this handoff",
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
