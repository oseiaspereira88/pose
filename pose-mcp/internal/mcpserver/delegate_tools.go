package mcpserver

// Delegated runs over MCP (spec pose-delegation-mcp).
//
// The tools run `pose delegate … --json` in the project's root and return its
// document reduced to the normalized summary: the CLI is the one place that
// plans, refuses and runs, so a refusal over MCP is the CLI's, with the same
// reason code (R5), and no tool widens what the CLI allows. A start, resume
// or handoff is always detached and returns once the run has reserved its
// attempt; a status never blocks, and a wait blocks at most its bound (R1–R4,
// non-functional). No result carries a transcript, a prompt, a session
// reference or an environment value (R7).

import (
	"context"

	"encoding/json"
	"errors"
	"fmt"
	"github.com/harne8/pose-mcp/internal/pose"
	"os"
	"os/exec"
	"strings"
	"time"
)

// DelegateRunner runs `pose delegate <args>` in root and returns its stdout
// and exit code.
type DelegateRunner func(ctx context.Context, root string, args []string) ([]byte, int, error)

// WithDelegateRunner sets how the delegate tools run the CLI; tests point it
// at a built binary.
func (s *Server) WithDelegateRunner(r DelegateRunner) *Server {
	s.delegateRunner = r
	return s
}

// Bounds of the delegate tools: a start, resume or handoff waits for the
// detached run to hand its attempt back (the CLI's own bound is two minutes);
// a status reads; a wait waits at most delegateWaitMax.
const (
	delegateStartBound  = 150 * time.Second
	delegateStatusBound = 30 * time.Second
	delegateWaitMax     = 300 * time.Second
	delegateWaitDefault = 30 * time.Second
)

func defaultDelegateRunner(ctx context.Context, root string, args []string) ([]byte, int, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, -1, err
	}
	cmd := exec.CommandContext(ctx, self, append([]string{"delegate"}, args...)...)
	cmd.Dir = root
	cmd.Env = DelegateEnv(root)
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return out, exit.ExitCode(), nil
	}
	if err != nil {
		return nil, -1, err
	}
	return out, 0, nil
}

// DelegateEnv is the environment a delegate subprocess runs with: the
// server's, and the root the server resolved and authorized, which the CLI
// uses instead of rediscovering one (R6; review round 1, B01).
func DelegateEnv(root string) []string {
	return append(os.Environ(), pose.DelegateRootEnv+"="+root)
}

// delegateSummaryKeys are the fields a delegate tool returns: the normalized
// summary of a run, its attempt and its ways forward (R7).
var delegateSummaryKeys = map[string]bool{
	"schema": true, "command": true, "purpose": true, "agent": true, "vendor": true, "model": true, "label": true, "lead": true,
	"scope": true, "bundle": true, "attempt": true, "run": true, "runs": true, "latest_run": true, "resumes": true, "hands_off_from": true,
	"status": true, "state": true, "failure": true, "interruption": true, "next_actions": true, "error": true, "reason_code": true,
	"usage": true, "routing": true, "independence": true, "implementation": true, "decision": true, "resumable": true, "reset_at": true,
	"exit_code": true, "active": true, "liveness": true, "heartbeat_age": true, "stalled_at": true, "turns": true,
	"round": true, "started_at": true, "ended_at": true, "access": true, "time_left": true, "partial": true, "cancel_requested": true,
	"attestation": true, "prior_runs": true, "reserves_attempt": true, "supervisor_pid": true, "losses": true, "candidates": true,
	// attempts is the ls document's list; each row is reduced again by status.
	"attempts": true,
}

// delegateSummary reduces a CLI document to the summary keys.
func delegateSummary(doc map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range doc {
		if delegateSummaryKeys[k] {
			out[k] = v
		}
	}
	return out
}

// runDelegate runs the CLI with --json within the call's deadline and returns
// its document, reduced, with the CLI's exit code. The deadline is the
// call's, shared by every subprocess it runs (review round 1, B05).
func (s *Server) runDelegate(ctx context.Context, root string, args ...string) (map[string]any, int, error) {
	runner := s.delegateRunner
	if runner == nil {
		runner = defaultDelegateRunner
	}
	out, code, err := runner(ctx, root, append(args, "--json"))
	if ctx.Err() != nil {
		return nil, code, fmt.Errorf("pose delegate %s did not answer within the call's bound", args[0])
	}
	if err != nil {
		return nil, code, err
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, code, fmt.Errorf("pose delegate %s returned no document (exit %d)", args[0], code)
	}
	return delegateSummary(doc), code, nil
}

// delegateRefused turns a CLI error document into the tool's error, keeping
// its reason code (R5).
func delegateRefused(tool string, doc map[string]any) (any, error) {
	if losses, ok := doc["losses"].([]any); ok && len(losses) > 0 && doc["error"] == nil {
		// A history that could not be read whole answers nothing about the
		// run (review round 1, F01).
		return map[string]any{"refused": true, "error": "the delegated run history could not be read whole", "losses": losses}, nil
	}
	if msg, ok := doc["error"].(string); ok {
		result := map[string]any{"refused": true, "error": msg}
		for _, k := range []string{"reason_code", "routing", "candidates", "usage"} {
			if v, ok := doc[k]; ok && v != "" {
				result[k] = v
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("%s: the CLI refused without an error document", tool)
}

func (s *Server) dispatchDelegate(ctx context.Context, root, name string, args json.RawMessage) (any, error) {
	var a struct {
		Purpose        string `json:"purpose"`
		Target         string `json:"target"`
		Agent          string `json:"agent"`
		Reason         string `json:"reason"`
		Label          string `json:"label"`
		Lead           string `json:"lead"`
		Run            string `json:"run"`
		To             string `json:"to"`
		TimeoutSeconds int    `json:"timeout_seconds"`
		Apply          bool   `json:"apply"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("%s: invalid arguments", name)
	}
	if a.TimeoutSeconds < 0 || a.TimeoutSeconds > int(delegateWaitMax/time.Second) {
		return nil, fmt.Errorf("%s: timeout_seconds must be from 1 to %d", name, int(delegateWaitMax/time.Second))
	}
	// One deadline per call, shared by every subprocess it runs (B05).
	bound := delegateStatusBound
	switch {
	case name == "pose_delegate_wait":
		wait := delegateWaitDefault
		if a.TimeoutSeconds > 0 {
			wait = time.Duration(a.TimeoutSeconds) * time.Second
		}
		bound = wait + delegateStatusBound
	case a.Apply:
		bound = delegateStartBound
	}
	ctx, cancel := context.WithTimeout(ctx, bound)
	defer cancel()
	// R6: the call runs in the project the server resolved for it, under the
	// shared project-scope contract — the default project only when it is
	// unambiguous — after the server's authorization admitted it.
	for field, value := range map[string]string{"purpose": a.Purpose, "target": a.Target, "agent": a.Agent, "reason": a.Reason, "label": a.Label, "lead": a.Lead, "run": a.Run, "to": a.To} {
		if strings.HasPrefix(value, "-") || strings.ContainsAny(value, "\x00\n") {
			return nil, fmt.Errorf("%s: %s is not a value the CLI takes", name, field)
		}
	}
	mode := "--dry-run"
	if a.Apply {
		mode = "--detach"
	}
	common := func(cli []string) []string {
		for flag, value := range map[string]string{"--agent": a.Agent, "--label": a.Label, "--lead": a.Lead, "--reason": a.Reason} {
			if value != "" {
				cli = append(cli, flag, value)
			}
		}
		return cli
	}
	var doc map[string]any
	var code int
	var err error
	switch name {
	case "pose_delegate_start":
		if a.Purpose == "" {
			return nil, fmt.Errorf("%s: required argument %q missing", name, "purpose")
		}
		switch a.Purpose {
		case "review", "adjudicate", "smoke":
		case "task":
			return nil, fmt.Errorf("%s: a task is not started over MCP: its prompt and access are the caller's, and pose delegate task does not run detached yet", name)
		default:
			return nil, fmt.Errorf("%s: purpose is review, adjudicate or smoke", name)
		}
		if a.Target == "" {
			return nil, fmt.Errorf("%s: required argument %q missing", name, "target")
		}
		// The plan names the agent and the independence the run achieves; the
		// detached run then reserves its attempt (R1).
		plan, pcode, perr := s.runDelegate(ctx, root, common([]string{a.Purpose, a.Target, "--dry-run"})...)
		if perr != nil {
			return nil, perr
		}
		if pcode != 0 || !a.Apply {
			if pcode != 0 {
				return delegateRefused(name, plan)
			}
			plan["apply"] = "false (call again with apply: true to start the run detached)"
			return plan, nil
		}
		doc, code, err = s.runDelegate(ctx, root, common([]string{a.Purpose, a.Target, mode})...)
		if err == nil && code == 0 {
			doc["agent"], doc["independence"], doc["routing"] = plan["agent"], plan["independence"], plan["routing"]
		}
	case "pose_delegate_status":
		if a.Run == "" {
			return nil, fmt.Errorf("%s: required argument %q missing", name, "run")
		}
		listed, lcode, lerr := s.runDelegate(ctx, root, "ls", a.Run)
		if lerr != nil {
			return nil, lerr
		}
		if lcode != 0 {
			return delegateRefused(name, listed)
		}
		attempts, _ := listed["attempts"].([]any)
		if len(attempts) != 1 {
			return nil, fmt.Errorf("%s: %s names no single attempt", name, a.Run)
		}
		row, _ := attempts[0].(map[string]any)
		status := delegateSummary(row)
		if active, _ := row["active"].(bool); !active {
			// Terminal: the outcome a waiter reads, without waiting (R2).
			outcome, _, oerr := s.runDelegate(ctx, root, "wait", a.Run, "--timeout", "1ms")
			if oerr == nil {
				for _, k := range []string{"decision", "exit_code", "resumable", "reset_at"} {
					if v, ok := outcome[k]; ok {
						status[k] = v
					}
				}
			}
		}
		return status, nil
	case "pose_delegate_wait":
		if a.Run == "" {
			return nil, fmt.Errorf("%s: required argument %q missing", name, "run")
		}
		timeout := delegateWaitDefault
		if a.TimeoutSeconds > 0 {
			timeout = time.Duration(a.TimeoutSeconds) * time.Second
		}
		doc, code, err = s.runDelegate(ctx, root, "wait", a.Run, "--timeout", timeout.String())
		if err == nil && code == 50 {
			// R3: the bound passed; the run goes on.
			doc["state"] = "still-running"
			doc["waited"] = timeout.String()
			return doc, nil
		}
	case "pose_delegate_resume":
		if a.Run == "" {
			return nil, fmt.Errorf("%s: required argument %q missing", name, "run")
		}
		doc, code, err = s.runDelegate(ctx, root, "resume", a.Run, mode)
	case "pose_delegate_handoff":
		if a.Run == "" || a.To == "" {
			return nil, fmt.Errorf("%s: required arguments %q and %q", name, "run", "to")
		}
		doc, code, err = s.runDelegate(ctx, root, "handoff", a.Run, "--to", a.To, mode)
	default:
		return nil, fmt.Errorf("unknown delegate tool %s", name)
	}
	if err != nil {
		return nil, err
	}
	if _, refused := doc["error"]; refused {
		return delegateRefused(name, doc)
	}
	if !a.Apply && (name == "pose_delegate_resume" || name == "pose_delegate_handoff") {
		doc["apply"] = "false (call again with apply: true to run it detached)"
	}
	return doc, nil
}

func delegateToolDefinitions() []map[string]any {
	project := map[string]any{"type": "string", "description": "Optional project to scope the .pose root (multi-project); omit for the default root"}
	run := map[string]any{"type": "string", "description": "A run id, an attempt id or a label"}
	apply := map[string]any{"type": "boolean", "description": "Start the run detached; without it the call previews the plan and changes nothing"}
	return []map[string]any{
		{
			"name": "pose_delegate_start",
			"description": "Start a delegated review, adjudication or smoke run of a scope or bundle, as `pose delegate <purpose> --detach` does: " +
				"always detached, returning once the run has reserved its attempt, with the agent, the routing and the independence it achieves. " +
				"Without apply it previews the plan. Every refusal of the CLI is returned with its reason_code (rerun-without-reason, " +
				"budget-exhausted, containment-incompatible, agent-unavailable, no-agent). The result is the normalized summary: no " +
				"transcript, prompt, session reference or environment value.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"purpose":    map[string]any{"type": "string", "description": "review, adjudicate or smoke"},
					"target":     map[string]any{"type": "string", "description": "A scope (spec:<slug>, …) or a sealed bundle id"},
					"agent":      map[string]any{"type": "string", "description": "An agent name, or auto (the default) to route"},
					"reason":     map[string]any{"type": "string", "description": "Why a completed run on an unchanged bundle is run again"},
					"label":      map[string]any{"type": "string", "description": "A label to name the run by"},
					"lead":       map[string]any{"type": "string", "description": "The implementation's vendor[/model], when the bundle does not record it"},
					"apply":      apply,
					"project_id": project,
				},
				"required": []string{"purpose", "target"},
			},
		},
		{
			"name": "pose_delegate_status",
			"description": "Read a delegated run without blocking: its state, heartbeat age, turns, last step, round, the decision and " +
				"exit code once terminal, and next_actions with whether each is available now and why not.",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{"run": run, "project_id": project},
				"required":   []string{"run"},
			},
		},
		{
			"name": "pose_delegate_wait",
			"description": "Wait for a delegated run to end, at most timeout_seconds (default 30, at most 300): the terminal result " +
				"(state, decision, exit code, resumability), or state still-running when the bound passes.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"run":             run,
					"timeout_seconds": map[string]any{"type": "integer", "description": "How long to wait, in seconds (1..300)"},
					"project_id":      project,
				},
				"required": []string{"run"},
			},
		},
		{
			"name": "pose_delegate_resume",
			"description": "Resume a stopped delegated run with its agent and session, as `pose delegate resume --detach` does, returning " +
				"once the continuation has reserved its run; without apply it previews. Refusals carry the CLI's reason_code.",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{"run": run, "apply": apply, "project_id": project},
				"required":   []string{"run"},
			},
		},
		{
			"name": "pose_delegate_handoff",
			"description": "Hand a stopped delegated run off to another agent under the same attempt, as `pose delegate handoff --detach` " +
				"does; without apply it previews. Refusals carry the CLI's reason_code.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"run":        run,
					"to":         map[string]any{"type": "string", "description": "The agent to hand the run to, or auto"},
					"apply":      apply,
					"project_id": project,
				},
				"required": []string{"run", "to"},
			},
		},
	}
}
