package cli

// pose delegate: the one surface for delegated executions (spec
// pose-delegation-surface; ADR 2026-10-26-delegation-is-a-first-class-execution-surface).
//
// pose review governs the subject (bundle, brief, attestation, verify);
// pose delegate governs executions: who runs, with which access, and what
// happened. review, adjudicate and smoke run through the delivered dispatch
// path; the lifecycle commands read the same run records.

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const delegateUsageText = "Usage: pose delegate <review|adjudicate|smoke|task|ls|watch|wait|resume|handoff|cancel|agents> [args] [--agent auto|<name>] [--access read|sandboxed|write|full] [--label <name>] [--lead <vendor>] [--dry-run] [--json]"

// Versioned JSON documents of pose delegate (R10).
const (
	delegateRunSchema    = "pose.delegate.run/v1"
	delegatePlanSchema   = "pose.delegate.plan/v1"
	delegateListSchema   = "pose.delegate.ls/v1"
	delegateCancelSchema = "pose.delegate.cancel/v1"
	delegateAgentsSchema = "pose.delegate.agents/v1"
	delegateErrorSchema  = "pose.delegate.error/v1"
)

// Subcommands named by the ADR whose behaviour a later spec delivers.
var delegatePlanned = map[string]string{
	"watch":   "pose-delegated-review-completion-signal",
	"wait":    "pose-delegated-review-completion-signal",
	"handoff": "pose-delegation-handoff",
	"task":    "pose-delegation-task",
}

var (
	delegateLabelRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	delegateLeadRE  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
)

type delegateFlags struct {
	agent, access, label, lead, reason, noteFile string
	dryRun, jsonOut, active, wait                bool
	waitMax                                      time.Duration
	positional                                   []string
}

// parseDelegateFlags gives every subcommand the same flags with the same
// meaning (R2); allowed names the flags this subcommand takes.
func parseDelegateFlags(args []string, allowed ...string) (delegateFlags, string) {
	f := delegateFlags{waitMax: 24 * time.Hour}
	permit := map[string]bool{}
	for _, name := range allowed {
		permit[name] = true
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, inline, hasInline := strings.Cut(arg, "=")
		if !strings.HasPrefix(arg, "-") {
			f.positional = append(f.positional, arg)
			continue
		}
		if name == "--detach" {
			return f, "--detach is not available yet; it is delivered by spec pose-delegated-review-completion-signal"
		}
		if !permit[name] {
			return f, "unknown or inapplicable flag " + name
		}
		switch name {
		case "--dry-run", "--json", "--active":
			if hasInline {
				return f, name + " takes no value"
			}
			switch name {
			case "--dry-run":
				f.dryRun = true
			case "--json":
				f.jsonOut = true
			default:
				f.active = true
			}
		case "--wait-until-reset":
			f.wait = true
			if hasInline {
				d, err := time.ParseDuration(inline)
				if err != nil || d <= 0 || d > 24*time.Hour {
					return f, "--wait-until-reset takes a duration up to 24h"
				}
				f.waitMax = d
			}
		default:
			value := inline
			if !hasInline {
				if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
					return f, name + " needs a value"
				}
				value = args[i+1]
				i++
			}
			switch name {
			case "--agent":
				f.agent = value
			case "--access":
				f.access = value
			case "--label":
				if !delegateLabelRE.MatchString(value) {
					return f, "--label takes letters, digits, '.', '_' and '-' (64 at most)"
				}
				f.label = value
			case "--lead":
				if !delegateLeadRE.MatchString(value) {
					return f, "--lead takes a vendor name such as openai or anthropic"
				}
				f.lead = value
			case "--reason":
				f.reason = value
			case "--note-file":
				f.noteFile = value
			}
		}
	}
	return f, ""
}

func cmdDelegate(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		dispatchCommandHelp("delegate", nil, stdout, cliLocaleValue())
		return 0
	}
	sub, rest := args[0], args[1:]
	if spec, planned := delegatePlanned[sub]; planned {
		return delegateUsage(stdout, stderr, rest, sub, "not available yet; it is delivered by spec "+spec)
	}
	switch sub {
	case posemodel.DelegationPurposeReview, posemodel.DelegationPurposeAdjudicate, posemodel.DelegationPurposeSmoke:
		return cmdDelegatePurpose(root, sub, rest, stdout, stderr)
	case "ls":
		return cmdDelegateLs(root, rest, stdout, stderr)
	case "resume":
		return cmdDelegateResume(root, rest, stdout, stderr)
	case "cancel":
		return cmdDelegateCancel(root, rest, stdout, stderr)
	case "agents":
		return cmdDelegateAgents(root, rest, stdout, stderr)
	}
	return delegateUsage(stdout, stderr, rest, sub, "unknown subcommand\n"+delegateUsageText)
}

// delegateUsage reports a usage error: exit 2, as text, or as the error
// document when the caller asked for --json (R10).
func delegateUsage(stdout, stderr io.Writer, args []string, command, msg string) int {
	for _, arg := range args {
		if arg == "--json" {
			writeDelegateJSON(stdout, map[string]any{"schema": delegateErrorSchema, "command": command, "error": msg, "usage": true})
			return 2
		}
	}
	return usageError(stderr, "pose delegate "+command+": "+msg)
}

func writeDelegateJSON(stdout io.Writer, doc any) {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(doc)
}

// delegateFail reports a failure as text or as the error document.
func delegateFail(stdout, stderr io.Writer, jsonOut bool, command string, code int, err error) int {
	msg := strings.TrimPrefix(err.Error(), "pose: ")
	if jsonOut {
		doc := map[string]any{"schema": delegateErrorSchema, "command": command, "error": msg}
		var ambiguous *posemodel.ErrDelegationRefAmbiguous
		if errors.As(err, &ambiguous) {
			doc["candidates"] = ambiguous.Candidates
		}
		writeDelegateJSON(stdout, doc)
		return code
	}
	if code == 2 {
		return usageError(stderr, "pose delegate "+command+": "+msg)
	}
	render(stdout, stderr).Failure("pose delegate " + command + ": " + msg)
	return code
}

// resolveDelegateAgent applies --agent: a name is taken as given; auto (or no
// flag) is the only agent that can run the purpose (R4). With several, the
// choice belongs to spec pose-delegation-routing; until then it is refused.
func resolveDelegateAgent(store posemodel.Store, kind, requested string) (string, int, error) {
	if requested != "" && requested != "auto" {
		return requested, 0, nil
	}
	policy, found, err := store.LoadReviewersPolicyFor(kind)
	if err != nil {
		return "", 1, err
	}
	if !found || len(policy.Adapters) == 0 {
		return "", 1, posemodel.ErrNoReviewerAdapter
	}
	names := make([]string, 0, len(policy.Adapters))
	for name := range policy.Adapters {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 1 {
		return names[0], 0, nil
	}
	return "", 2, fmt.Errorf("several agents can run this purpose (%s); name one with --agent", strings.Join(names, ", "))
}

func cmdDelegatePurpose(root, purpose string, args []string, stdout, stderr io.Writer) int {
	f, problem := parseDelegateFlags(args, "--agent", "--access", "--label", "--lead", "--reason", "--note-file", "--dry-run", "--json")
	if problem != "" {
		return delegateUsage(stdout, stderr, args, purpose, problem)
	}
	if len(f.positional) != 1 {
		return delegateUsage(stdout, stderr, args, purpose, "Usage: pose delegate "+purpose+" <scope|bundle-id> [--agent auto|<name>] [--label <name>] [--lead <vendor>] [--reason <text>] [--note-file <path>] [--dry-run] [--json]")
	}
	// R3: the review-path purposes read only.
	if f.access != "" && f.access != posemodel.DelegationAccessRead {
		return delegateUsage(stdout, stderr, args, purpose, "runs with read access only; --access "+f.access+" is for pose delegate task")
	}
	notes := ""
	if f.noteFile != "" {
		raw, err := posemodel.ReadReviewAdapterFile(f.noteFile, posemodel.ReviewNotesMaxBytes)
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, err)
		}
		notes = string(raw)
	}
	store := cliGovernedStore(root)
	target := f.positional[0]
	ref := target
	if !strings.HasPrefix(target, "rvb-") {
		resolved, err := resolveReviewBundleScope(store, target)
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, err)
		}
		ref = resolved
	}
	kind := posemodel.DelegationKindForPurpose(purpose)
	agent, code, err := resolveDelegateAgent(store, kind, f.agent)
	if errors.Is(err, posemodel.ErrNoReviewerAdapter) {
		return delegateNoAgent(stdout, stderr, f.jsonOut, purpose)
	}
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, purpose, code, err)
	}
	plan, err := store.PlanReviewDispatch(ref, kind, notes, agent, f.reason)
	if errors.Is(err, posemodel.ErrNoReviewerAdapter) {
		return delegateNoAgent(stdout, stderr, f.jsonOut, purpose)
	}
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, err)
	}
	plan.Label, plan.Purpose, plan.Lead = f.label, purpose, f.lead
	if f.dryRun {
		return delegatePrintPlan(stdout, stderr, f.jsonOut, plan)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	run, err := store.RunReviewDispatchContext(ctx, plan, time.Now)
	return delegatePrintRun(stdout, stderr, f.jsonOut, purpose, run, err)
}

func delegateNoAgent(stdout, stderr io.Writer, jsonOut bool, command string) int {
	example, _ := posemodel.CanonicalDelegationPolicy(posemodel.ExampleDelegationPolicy())
	err := fmt.Errorf("no agent is configured; add .pose/policy/delegation.json, for example:\n%s", example)
	return delegateFail(stdout, stderr, jsonOut, command, 1, err)
}

// redactArgv hides the values of the credentials an agent is authorized to
// receive, wherever a template carries them (security of R12).
func redactArgv(argv []string, config posemodel.ReviewerAdapter) []string {
	out := append([]string(nil), argv...)
	if config.Containment == nil {
		return out
	}
	for _, name := range config.Containment.Credentials {
		value := os.Getenv(name)
		if len(value) < 4 {
			continue
		}
		for i := range out {
			out[i] = strings.ReplaceAll(out[i], value, "[credential]")
		}
	}
	return out
}

func delegatePrintPlan(stdout, stderr io.Writer, jsonOut bool, plan posemodel.DispatchPlan) int {
	argv := redactArgv(posemodel.ReviewerArgv(plan.Config.Command, "{output}", "", plan.Effort, plan.Config.Effort), plan.Config)
	workdir := "a disposable worktree at " + plan.SealedCommit
	if jsonOut {
		writeDelegateJSON(stdout, map[string]any{
			"schema": delegatePlanSchema, "purpose": plan.Purpose, "agent": plan.Adapter, "vendor": plan.Config.Vendor, "model": plan.Config.Model,
			"access": posemodel.DelegationAccessRead, "workdir": workdir, "argv": argv, "scope": plan.Brief.Scope, "bundle": plan.Bundle.BundleID,
			"brief_digest": plan.Brief.Digest, "prior_runs": plan.PriorRuns, "label": plan.Label, "lead": plan.Lead, "reserves_attempt": false,
		})
		return 0
	}
	out := render(stdout, stderr)
	out.Field("delegate.purpose", plan.Purpose)
	out.Field("delegate.agent", fmt.Sprintf("%s (%s %s)", plan.Adapter, plan.Config.Vendor, plan.Config.Model))
	out.Field("delegate.access", posemodel.DelegationAccessRead)
	out.Field("delegate.workdir", workdir)
	raw, _ := json.Marshal(argv)
	out.Field("delegate.argv", string(raw))
	out.Field("delegate.scope", plan.Brief.Scope)
	out.Field("delegate.bundle", plan.Bundle.BundleID)
	out.Field("delegate.brief_digest", plan.Brief.Digest)
	out.Field("delegate.prior_runs", fmt.Sprint(plan.PriorRuns))
	if plan.Label != "" {
		out.Field("delegate.label", plan.Label)
	}
	out.Field("delegate.dry_run", "true (no attempt reserved)")
	return 0
}

func delegatePrintRun(stdout, stderr io.Writer, jsonOut bool, command string, run posemodel.ReviewRun, err error) int {
	code := 0
	if err != nil || run.Status != posemodel.ReviewRunCompleted {
		code = 1
	}
	if jsonOut {
		doc := map[string]any{"schema": delegateRunSchema, "purpose": run.Purpose, "agent": run.Adapter, "vendor": run.Vendor, "model": run.Model,
			"label": run.Label, "scope": run.Scope, "bundle": run.BundleID, "attempt": run.AttemptID, "run": run.RunID, "status": run.Status,
			"failure": run.Failure, "record": run.Path, "events": run.EventsDigest != "", "env_removed": run.EnvRemoved,
			"attestation": "not recorded: a run's conclusion is recorded only by the engine's verified step"}
		if run.Interruption != nil {
			doc["interruption"] = run.Interruption
		}
		if err != nil {
			doc["error"] = strings.TrimPrefix(err.Error(), "pose: ")
		}
		writeDelegateJSON(stdout, doc)
		return code
	}
	out := render(stdout, stderr)
	if run.RunID != "" {
		out.Field("delegate.purpose", run.Purpose)
		out.Field("delegate.agent", fmt.Sprintf("%s (%s %s)", run.Adapter, run.Vendor, run.Model))
		if run.Label != "" {
			out.Field("delegate.label", run.Label)
		}
		out.Field("delegate.attempt", run.AttemptID)
		out.Field("delegate.run", run.RunID)
		out.Field("delegate.status", run.Status)
		if run.Failure != "" {
			out.Field("delegate.failure", run.Failure)
		}
		if run.Interruption != nil {
			out.Field("delegate.interruption", fmt.Sprintf("cause=%s resumable=%t reset_at=%s", run.Interruption.Cause, run.Interruption.Resumable, run.Interruption.ResetAt))
		}
		out.Field("delegate.record", run.Path)
		out.Field("delegate.attestation", "not recorded: a run's conclusion is recorded only by the engine's verified step")
	}
	if err != nil {
		out.Failure("pose delegate " + command + ": " + strings.TrimPrefix(err.Error(), "pose: "))
	}
	return code
}

func delegateAge(at string) string {
	t, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func cmdDelegateLs(root string, args []string, stdout, stderr io.Writer) int {
	f, problem := parseDelegateFlags(args, "--json", "--active")
	if problem != "" || len(f.positional) > 1 {
		if problem == "" {
			problem = "one reference at most"
		}
		return delegateUsage(stdout, stderr, args, "ls", problem+"\nUsage: pose delegate ls [<run>|<bundle-id>] [--active] [--json]")
	}
	store := posemodel.Store{Root: root}
	attempts, lost, err := store.ListDelegationAttempts()
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "ls", 1, err)
	}
	if len(f.positional) == 1 {
		ref := f.positional[0]
		if strings.HasPrefix(ref, "rvb-") {
			filtered := []posemodel.DelegationAttempt{}
			for _, a := range attempts {
				if a.BundleID == ref {
					filtered = append(filtered, a)
				}
			}
			attempts = filtered
		} else {
			one, err := posemodel.ResolveDelegationRef(attempts, ref)
			if err != nil {
				return delegateFail(stdout, stderr, f.jsonOut, "ls", 2, err)
			}
			attempts = []posemodel.DelegationAttempt{one}
		}
	}
	if f.active {
		filtered := []posemodel.DelegationAttempt{}
		for _, a := range attempts {
			if a.Active {
				filtered = append(filtered, a)
			}
		}
		attempts = filtered
	}
	if lost == nil {
		lost = []string{}
	}
	if f.jsonOut {
		writeDelegateJSON(stdout, map[string]any{"schema": delegateListSchema, "attempts": attempts, "losses": lost})
	} else {
		out := render(stdout, stderr)
		if len(attempts) == 0 {
			out.Field("delegate.runs", "none")
		}
		for _, a := range attempts {
			value := fmt.Sprintf("state=%s purpose=%s agent=%s turns=%d age=%s", a.State, a.Purpose, a.Agent, a.Turns, delegateAge(a.StartedAt))
			if a.Label != "" {
				value += " label=" + a.Label
			}
			if a.Active && a.Liveness != "" {
				value += " owner=" + a.Liveness
			}
			if a.LatestRun != "" {
				value += " run=" + a.LatestRun
			}
			if a.LastStep != "" {
				value += " last=" + a.LastStep
			}
			out.Field("delegate."+a.AttemptID, value)
		}
		for _, loss := range lost {
			out.Failure("delegated run history incomplete: " + loss)
		}
	}
	if len(lost) > 0 {
		return 1
	}
	return 0
}

func cmdDelegateResume(root string, args []string, stdout, stderr io.Writer) int {
	f, problem := parseDelegateFlags(args, "--wait-until-reset", "--dry-run", "--json")
	if problem != "" || len(f.positional) != 1 {
		if problem == "" {
			problem = "one run reference"
		}
		return delegateUsage(stdout, stderr, args, "resume", problem+"\nUsage: pose delegate resume <run> [--wait-until-reset[=<max>]] [--dry-run] [--json]")
	}
	store := cliGovernedStore(root)
	attempts, _, err := store.ListDelegationAttempts()
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, err)
	}
	attempt, err := posemodel.ResolveDelegationRef(attempts, f.positional[0])
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "resume", 2, err)
	}
	if attempt.Active {
		return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, fmt.Errorf("attempt %s is still running", attempt.AttemptID))
	}
	latest := attempt.Latest()
	if latest == nil || latest.Status != posemodel.ReviewRunInterrupted {
		state := "unknown"
		if latest != nil {
			state = latest.Status
		}
		return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, fmt.Errorf("the latest run of attempt %s is %s, not interrupted", attempt.AttemptID, state))
	}
	plan, err := store.PlanReviewResume(latest.RunID)
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, err)
	}
	if plan.PendingContinuation != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, fmt.Errorf("a continuation of %s is pending (%s)", latest.RunID, plan.PendingContinuation.ID))
	}
	var delay time.Duration
	if f.wait {
		var seed [1]byte
		if _, err := rand.Read(seed[:]); err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, err)
		}
		delay, err = posemodel.ReviewResumeDelay(*latest, time.Now(), f.waitMax, time.Duration(seed[0])*time.Second/255)
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, err)
		}
	}
	if f.dryRun {
		argv := redactArgv(posemodel.ReviewerArgv(plan.Config.Resume.Command, "{output}", "{session}", plan.Effort, plan.Config.Effort), plan.Config)
		if f.jsonOut {
			writeDelegateJSON(stdout, map[string]any{"schema": delegatePlanSchema, "purpose": attempt.Purpose, "agent": plan.Adapter, "vendor": plan.Config.Vendor,
				"model": plan.Config.Model, "access": posemodel.DelegationAccessRead, "workdir": "a disposable worktree at " + plan.SealedCommit, "argv": argv,
				"resumes": latest.RunID, "attempt": attempt.AttemptID, "wait": delay.String(), "reserves_attempt": false})
			return 0
		}
		out := render(stdout, stderr)
		out.Field("delegate.resumes", latest.RunID)
		out.Field("delegate.attempt", attempt.AttemptID)
		out.Field("delegate.agent", fmt.Sprintf("%s (%s %s)", plan.Adapter, plan.Config.Vendor, plan.Config.Model))
		raw, _ := json.Marshal(argv)
		out.Field("delegate.argv", string(raw))
		if f.wait {
			out.Field("delegate.wait", delay.String())
		}
		out.Field("delegate.dry_run", "true (nothing claimed)")
		return 0
	}
	if f.wait && plan.Existing == nil && delay > 0 {
		if !f.jsonOut {
			render(stdout, stderr).Field("delegate.wait", delay.String())
		}
		time.Sleep(delay)
		if plan, err = store.PlanReviewResume(latest.RunID); err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, err)
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	run, err := store.RunReviewDispatchContext(ctx, plan, time.Now)
	return delegatePrintRun(stdout, stderr, f.jsonOut, "resume", run, err)
}

func cmdDelegateCancel(root string, args []string, stdout, stderr io.Writer) int {
	f, problem := parseDelegateFlags(args, "--json")
	if problem != "" || len(f.positional) != 1 {
		if problem == "" {
			problem = "one run reference"
		}
		return delegateUsage(stdout, stderr, args, "cancel", problem+"\nUsage: pose delegate cancel <run> [--json]")
	}
	store := posemodel.Store{Root: root}
	attempts, _, err := store.ListDelegationAttempts()
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "cancel", 1, err)
	}
	attempt, err := posemodel.ResolveDelegationRef(attempts, f.positional[0])
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "cancel", 2, err)
	}
	if !attempt.Active {
		return delegateFail(stdout, stderr, f.jsonOut, "cancel", 1, fmt.Errorf("attempt %s is %s, not running", attempt.AttemptID, attempt.State))
	}
	if _, err := store.RequestDelegationCancel(attempt.AttemptID); err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "cancel", 1, err)
	}
	deadline := time.Now().Add(60 * time.Second)
	for store.DelegationStillActive(attempt.AttemptID) && time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
	}
	stopped := !store.DelegationStillActive(attempt.AttemptID)
	status, run := "", ""
	if after, _, err := store.ListDelegationAttempts(); err == nil {
		for _, a := range after {
			if a.AttemptID == attempt.AttemptID && a.Latest() != nil {
				status, run = a.Latest().Status, a.Latest().RunID
			}
		}
	}
	if f.jsonOut {
		writeDelegateJSON(stdout, map[string]any{"schema": delegateCancelSchema, "attempt": attempt.AttemptID, "requested": true, "stopped": stopped, "run": run, "status": status})
	} else {
		out := render(stdout, stderr)
		out.Field("delegate.attempt", attempt.AttemptID)
		out.Field("delegate.cancel", "requested")
		out.Field("delegate.stopped", fmt.Sprint(stopped))
		if run != "" {
			out.Field("delegate.run", run)
			out.Field("delegate.status", status)
		}
	}
	if !stopped {
		return 1
	}
	return 0
}

// delegateAgentRow is one line of pose delegate agents (R14).
type delegateAgentRow struct {
	Name         string   `json:"name"`
	Vendor       string   `json:"vendor"`
	Model        string   `json:"model"`
	CLI          string   `json:"cli"`
	OnPath       bool     `json:"on_path"`
	Access       []string `json:"access"`
	Events       string   `json:"events"`
	LimitResetAt string   `json:"limit_reset_at,omitempty"`
	Limited      bool     `json:"limited"`
}

func cmdDelegateAgents(root string, args []string, stdout, stderr io.Writer) int {
	f, problem := parseDelegateFlags(args, "--json")
	if problem != "" || len(f.positional) > 0 {
		if problem == "" {
			problem = "no arguments"
		}
		return delegateUsage(stdout, stderr, args, "agents", problem+"\nUsage: pose delegate agents [--json]")
	}
	store := posemodel.Store{Root: root}
	policy, path, found, err := store.LoadDelegationPolicy()
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "agents", 1, err)
	}
	resets := map[string]string{}
	if runs, _, err := store.ReadReviewRuns(""); err == nil {
		for _, run := range runs {
			if run.Interruption != nil && run.Interruption.Cause == "usage-limit" && run.Interruption.ResetAt > resets[run.Adapter] {
				resets[run.Adapter] = run.Interruption.ResetAt
			}
		}
	}
	rows := []delegateAgentRow{}
	for _, name := range policy.AgentNames() {
		agent := policy.Agents[name]
		access := make([]string, 0, len(agent.Commands))
		for level := range agent.Commands {
			access = append(access, level)
		}
		sort.Strings(access)
		cli := agent.CLI
		if cli == "" && len(access) > 0 {
			cli = agent.Commands[access[0]][0]
		}
		_, lookErr := exec.LookPath(cli)
		events := agent.Events
		if events == "" {
			events = posemodel.DelegationEventsPlain
		}
		row := delegateAgentRow{Name: name, Vendor: agent.Vendor, Model: agent.Model, CLI: cli, OnPath: lookErr == nil, Access: access, Events: events, LimitResetAt: resets[name]}
		if t, err := time.Parse(time.RFC3339Nano, row.LimitResetAt); err == nil && t.After(time.Now()) {
			row.Limited = true
		}
		rows = append(rows, row)
	}
	if f.jsonOut {
		doc := map[string]any{"schema": delegateAgentsSchema, "configured": found, "agents": rows}
		if found {
			doc["policy"] = path
		}
		writeDelegateJSON(stdout, doc)
		return 0
	}
	out := render(stdout, stderr)
	if !found {
		out.Field("delegate.agents", "none configured (.pose/policy/delegation.json)")
		return 0
	}
	out.Field("delegate.policy", path)
	for _, row := range rows {
		value := fmt.Sprintf("vendor=%s model=%s cli=%s on_path=%t access=%s events=%s", row.Vendor, row.Model, row.CLI, row.OnPath, strings.Join(row.Access, ","), row.Events)
		if row.LimitResetAt != "" {
			value += " last_limit_reset=" + row.LimitResetAt
			if row.Limited {
				value += " (limited now)"
			}
		}
		out.Field("delegate.agent."+row.Name, value)
	}
	return 0
}
