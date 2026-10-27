package cli

// pose delegate task: a general task handed to another agent (spec
// pose-delegation-task). The prompt goes to the agent on stdin in --cd, with
// the access template its caller chose; the final message goes to stdout and
// the agent's exit code is returned (R1, R6). A task never becomes evidence:
// its record lives under the user's state directory unless --record (R5, R7).

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const delegateTaskUsage = "pose delegate task <prompt-file|-> [--cd <dir>] [--agent auto|<name>] [--access read|sandboxed|write|full] [--add-dir <dir>]... [--effort <level>] [--no-network] [--record] [--label <name>] [--lead <vendor>] [--dry-run] [--json]"

func cmdDelegateTask(root string, args []string, stdout, stderr io.Writer) int {
	f, problem := parseDelegateFlags(args, "--cd", "--add-dir", "--effort", "--no-network", "--record")
	if problem == "" && len(f.positional) != 1 {
		problem = tr("name the prompt file, or - to read it from stdin", "nomeie o arquivo do prompt, ou - para lê-lo do stdin")
	}
	if problem != "" {
		return delegateUsage(stdout, stderr, args, "task", problem+"\n"+tr("Usage: ", "Uso: ")+delegateTaskUsage)
	}
	prompt, err := readDelegatePrompt(f.positional[0])
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "task", 2, err)
	}
	store := cliGovernedStore(root)
	policy, _, found, err := store.LoadDelegationPolicy()
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "task", 1, err)
	}
	if !found || len(policy.Agents) == 0 {
		return delegateNoAgent(stdout, stderr, f.jsonOut, "task")
	}
	access := f.access
	if access == "" {
		access = policy.TaskDefaultAccess()
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	agent, route, err := routeTask(store, policy, access, f.agent, f.lead, "")
	if err != nil {
		return delegateTaskFail(stdout, stderr, f.jsonOut, "task", err, route)
	}
	if !f.dryRun && route != nil && route.Step == posemodel.DelegationStepWait {
		if err := waitForRoute(ctx, stderr, route, agent); err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "task", 1, err)
		}
	}
	plan, err := store.PlanDelegationTask(posemodel.DelegationTaskRequest{Agent: agent, Access: access, Dir: f.cd, Prompt: prompt, AddDirs: f.addDirs,
		Effort: f.effort, NoNetwork: f.noNetwork, Record: f.record, Label: f.label, Lead: f.lead, Routing: route})
	if err != nil {
		return delegateTaskFail(stdout, stderr, f.jsonOut, "task", err, nil)
	}
	return runDelegateTask(root, store, ctx, plan, f, "task", nil, stdout, stderr)
}

// readDelegatePrompt reads a prompt file, or stdin for "-".
func readDelegatePrompt(path string) (string, error) {
	var raw []byte
	var err error
	if path == "-" {
		raw, err = io.ReadAll(io.LimitReader(delegateStdin, 1<<20+1))
	} else {
		raw, err = posemodel.ReadReviewAdapterFile(path, 1<<20+1)
	}
	if err != nil {
		return "", err
	}
	if len(raw) > 1<<20 {
		return "", errors.New(tr("the prompt exceeds 1 MiB", "o prompt passa de 1 MiB"))
	}
	return string(raw), nil
}

// delegateTaskFail reports a task refusal: a request the policy or the flags
// refuse exits 2, anything else 1.
func delegateTaskFail(stdout, stderr io.Writer, jsonOut bool, command string, err error, route *posemodel.DelegationRoutingRecord) int {
	if errors.Is(err, posemodel.ErrNoReviewerAdapter) {
		return delegateNoAgent(stdout, stderr, jsonOut, command)
	}
	code := 1
	var usage *posemodel.DelegationTaskUsageError
	if errors.As(err, &usage) {
		code = 2
	}
	if route != nil && jsonOut {
		writeDelegateJSON(stdout, map[string]any{"schema": delegateErrorSchema, "command": command, "error": strings.TrimPrefix(err.Error(), "pose: "), "routing": route})
		return code
	}
	return delegateFail(stdout, stderr, jsonOut, command, code, err)
}

// routeTask applies --agent to a task: a name is taken as given; auto (or no
// flag) routes among the agents with a command for access, the launching
// session as the implementation (R10), without exclude.
func routeTask(store posemodel.Store, policy posemodel.DelegationPolicy, access, named, lead, exclude string) (string, *posemodel.DelegationRoutingRecord, error) {
	if named != "" && named != "auto" {
		return named, nil, nil
	}
	var eligible []string
	for _, name := range policy.TaskAgentNames(access) {
		if name != exclude {
			eligible = append(eligible, name)
		}
	}
	if len(eligible) == 0 {
		return "", nil, &posemodel.DelegationTaskUsageError{Msg: tr("no agent declares a "+access+" command", "nenhum agente declara um comando "+access)}
	}
	var leadModels map[string]string
	if policy.Routing != nil {
		leadModels = policy.Routing.LeadModels
	}
	impl, _, err := posemodel.EnvironmentSessionIdentity(lead, leadModels)
	if err != nil {
		return "", nil, err
	}
	from := posemodel.ImplementationFromSession
	if lead != "" {
		from = posemodel.ImplementationFromLead
	}
	now := time.Now()
	agent, record, err := posemodel.RouteDelegation(policy, eligible, impl, from, store.DelegationAvailabilityOf(policy, now, true), now)
	if err != nil {
		return "", &record, err
	}
	return agent, &record, nil
}

// runDelegateTask prints the plan of a dry run, or runs the task, prints its
// result and, at a terminal, offers the ways forward of a stopped run.
func runDelegateTask(root string, store posemodel.Store, ctx context.Context, plan posemodel.DelegationTaskPlan, f delegateFlags, command string, extra map[string]any, stdout, stderr io.Writer) int {
	for _, warning := range plan.Warnings {
		render(stdout, stderr).Hint("pose delegate " + command + ": " + warning)
	}
	if f.dryRun {
		return delegatePrintTaskPlan(stdout, stderr, f.jsonOut, plan, extra)
	}
	run, err := store.RunDelegationTask(ctx, plan, time.Now)
	delegateTerminalHook(store, run, err, stdout, stderr, command)
	code := delegatePrintTask(stdout, stderr, f.jsonOut, command, run, err, delegateRunNextActions(store, run, err)...)
	if err == nil && !f.jsonOut {
		if next, chose := offerDelegateChoices(root, run, stdout, stderr); chose {
			return next
		}
	}
	return code
}

func delegateTaskRecordPlace(record bool) string {
	if record {
		return ".pose/delegate-runs"
	}
	return tr("the user's state directory (owner-only)", "o diretório de estado do usuário (só o dono)")
}

func delegatePrintTaskPlan(stdout, stderr io.Writer, jsonOut bool, plan posemodel.DelegationTaskPlan, extra map[string]any) int {
	argv := plan.TaskArgv("{output}")
	doc := map[string]any{"schema": delegatePlanSchema, "purpose": posemodel.DelegationPurposeTask, "agent": plan.Agent, "vendor": plan.Config.Vendor, "model": plan.Config.Model,
		"access": plan.Access, "workdir": plan.Dir, "argv": argv, "outside_git": plan.OutsideGit, "record": delegateTaskRecordPlace(plan.Record),
		"label": plan.Label, "lead": plan.Lead, "routing": plan.Routing, "reserves_attempt": false}
	for k, v := range extra {
		doc[k] = v
	}
	if jsonOut {
		writeDelegateJSON(stdout, doc)
		return 0
	}
	out := render(stdout, stderr)
	out.Field("delegate.purpose", posemodel.DelegationPurposeTask)
	out.Field("delegate.agent", fmt.Sprintf("%s (%s %s)", plan.Agent, plan.Config.Vendor, plan.Config.Model))
	out.Field("delegate.access", plan.Access)
	out.Field("delegate.workdir", plan.Dir)
	out.Field("delegate.argv", strings.Join(argv, " "))
	out.Field("delegate.record", delegateTaskRecordPlace(plan.Record))
	if plan.Routing != nil {
		out.Field("delegate.routing", routeSummary(plan.Routing))
	}
	for k, v := range extra {
		if value := fmt.Sprint(v); value != "" {
			out.Field("delegate."+k, value)
		}
	}
	out.Field("delegate.dry_run", tr("true (nothing run or recorded)", "true (nada rodado nem registrado)"))
	return 0
}

// delegatePrintTask prints a task's result: the final message on stdout and
// the run's fields on stderr, or one JSON document; it returns the agent's
// exit code, or 1 when the run failed without one (R6).
func delegatePrintTask(stdout, stderr io.Writer, jsonOut bool, command string, run posemodel.ReviewRun, err error, next ...delegateNextAction) int {
	code := run.ExitCode
	if err != nil || code < 0 || code == 0 && run.Status != posemodel.ReviewRunCompleted {
		code = 1
	}
	if jsonOut {
		doc := map[string]any{"schema": delegateRunSchema, "purpose": posemodel.DelegationPurposeTask, "agent": run.Adapter, "vendor": run.Vendor, "model": run.Model,
			"access": run.Access, "workdir": run.Dir, "label": run.Label, "attempt": run.AttemptID, "run": run.RunID, "resumes": run.Resumes, "hands_off_from": run.HandsOffFrom,
			"status": run.Status, "failure": run.Failure, "exit_code": run.ExitCode, "final": run.Output, "record": run.Path, "env_removed": run.EnvRemoved}
		if run.Interruption != nil {
			doc["interruption"] = run.Interruption
		}
		if len(next) > 0 {
			doc["next_actions"] = next
		}
		if err != nil {
			doc["error"] = strings.TrimPrefix(err.Error(), "pose: ")
		}
		writeDelegateJSON(stdout, doc)
		return code
	}
	out := render(stdout, stderr)
	if run.Output != "" {
		// The final message, verbatim, is the command's result (R6).
		out.ContractLine(strings.TrimSuffix(run.Output, "\n"))
	}
	if run.RunID != "" {
		out.Hint(fmt.Sprintf("pose delegate %s: %s %s (%s, %s) %s", command, run.Status, run.RunID, run.Adapter, run.Access, run.AttemptID))
		if run.Failure != "" {
			out.Hint("pose delegate " + command + ": " + run.Failure)
		}
		for _, action := range next {
			if action.Available && action.Command != "" {
				out.Hint("pose delegate " + command + ": " + action.Action + ": " + action.Command)
			}
		}
	}
	if err != nil {
		out.Failure("pose delegate " + command + ": " + strings.TrimPrefix(err.Error(), "pose: "))
	}
	return code
}

// cmdDelegateTaskResume continues a stopped task with its own agent and
// session; without --prompt it sends the default continuation message (R8).
func cmdDelegateTaskResume(root string, store posemodel.Store, f delegateFlags, attempt posemodel.DelegationAttempt, target string, stdout, stderr io.Writer) int {
	prompt := ""
	if f.promptFile != "" {
		var err error
		if prompt, err = readDelegatePrompt(f.promptFile); err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "resume", 2, err)
		}
	}
	plan, err := store.PlanDelegationTaskResume(target, prompt)
	if err != nil {
		return delegateTaskFail(stdout, stderr, f.jsonOut, "resume", err, nil)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if f.wait && plan.Existing == nil && !f.dryRun {
		latest := attempt.Latest()
		var seed [1]byte
		_, _ = rand.Read(seed[:])
		delay, err := posemodel.ReviewResumeDelay(*latest, time.Now(), f.waitMax, time.Duration(seed[0])*time.Second/255)
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, err)
		}
		if delay > 0 {
			render(io.Discard, stderr).Hint("pose delegate resume: " + tr("waiting ", "esperando ") + delay.String())
			select {
			case <-ctx.Done():
				return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, ctx.Err())
			case <-time.After(delay):
			}
			if plan, err = store.PlanDelegationTaskResume(target, prompt); err != nil {
				return delegateTaskFail(stdout, stderr, f.jsonOut, "resume", err, nil)
			}
		}
	}
	if plan.Existing != nil {
		return delegatePrintTask(stdout, stderr, f.jsonOut, "resume", *plan.Existing, nil)
	}
	return runDelegateTask(root, store, ctx, plan, f, "resume", map[string]any{"resumes": target, "attempt": attempt.AttemptID}, stdout, stderr)
}

// cmdDelegateTaskHandoff continues a stopped task with another agent, its
// prompt the stopped run's followed by the handoff section.
func cmdDelegateTaskHandoff(root string, store posemodel.Store, f delegateFlags, attempt posemodel.DelegationAttempt, latest posemodel.ReviewRun, target string, stdout, stderr io.Writer) int {
	agent := f.to
	runs, _ := store.ReadDelegationTaskRuns()
	for _, r := range runs {
		if r.HandsOffFrom != target {
			continue
		}
		if agent != "auto" && agent != r.Adapter {
			return delegateFail(stdout, stderr, f.jsonOut, "handoff", 1, fmt.Errorf(tr("run %s was already handed off to %s by run %s", "o run %s já foi repassado para %s pelo run %s"), target, r.Adapter, r.RunID))
		}
		agent = r.Adapter
	}
	var route *posemodel.DelegationRoutingRecord
	if agent == "auto" {
		chosen, record, err := routeTaskHandoff(store, latest)
		if err != nil {
			return delegateTaskFail(stdout, stderr, f.jsonOut, "handoff", err, record)
		}
		agent, route = chosen, record
	}
	plan, err := store.PlanDelegationTaskHandoff(target, agent)
	if err != nil {
		return delegateTaskFail(stdout, stderr, f.jsonOut, "handoff", err, nil)
	}
	if plan.Existing != nil {
		return delegatePrintTask(stdout, stderr, f.jsonOut, "handoff", *plan.Existing, nil)
	}
	plan.Routing = route
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runDelegateTask(root, store, ctx, plan, f, "handoff", map[string]any{"hands_off_from": target, "attempt": attempt.AttemptID}, stdout, stderr)
}

// routeTaskHandoff routes the handoff of a task among the agents with a
// command for its access, the stopped agent excluded; it never waits.
func routeTaskHandoff(store posemodel.Store, prior posemodel.ReviewRun) (string, *posemodel.DelegationRoutingRecord, error) {
	policy, _, found, err := store.LoadDelegationPolicy()
	if err != nil {
		return "", nil, err
	}
	if !found {
		return "", nil, posemodel.ErrNoReviewerAdapter
	}
	agent, record, err := routeTask(store, policy, prior.Access, "auto", prior.Lead, prior.Adapter)
	if err != nil {
		return "", record, err
	}
	if record.Step == posemodel.DelegationStepWait {
		return "", record, errors.New(tr("every other agent is limited until "+record.WaitUntil+"; hand off after the reset, or wait for this run's reset and resume it",
			"todos os outros agentes estão limitados até "+record.WaitUntil+"; faça o handoff depois do reset, ou espere o reset deste run e o retome"))
	}
	return agent, record, nil
}

// delegateTaskNextActions are the ways forward of a stopped task, judged by
// the planning their commands run (R9).
func delegateTaskNextActions(store posemodel.Store, a posemodel.DelegationAttempt, latest posemodel.ReviewRun) []delegateNextAction {
	reason := func(err error) string { return strings.TrimPrefix(err.Error(), "pose: ") }
	var actions []delegateNextAction
	if i := latest.Interruption; latest.Status == posemodel.ReviewRunInterrupted && i != nil {
		resume := delegateNextAction{Action: "resume", Command: "pose delegate resume " + a.AttemptID}
		plan, err := store.PlanDelegationTaskResume(latest.RunID, "")
		switch {
		case !i.Resumable:
			resume.Why = tr("the stop is not resumable", "a parada não é retomável")
		case err != nil:
			resume.Why = reason(err)
		case plan.Existing != nil:
			resume.Why = tr("the run was already continued", "o run já foi continuado")
		}
		if resume.Why == "" {
			reset, resetErr := time.Parse(time.RFC3339Nano, i.ResetAt)
			if i.ResetAt != "" && resetErr == nil && reset.After(time.Now()) {
				wait := delegateNextAction{Action: "wait", Command: "pose delegate resume " + a.AttemptID + " --wait-until-reset"}
				if _, err := posemodel.ReviewResumeDelay(latest, time.Now(), 24*time.Hour, 0); err != nil {
					wait.Why = reason(err)
				} else {
					wait.Available = true
				}
				actions = append(actions, wait)
				resume.Why = tr("the agent is limited until ", "o agente está limitado até ") + i.ResetAt
			} else {
				resume.Available = true
			}
		}
		actions = append(actions, resume)
	}
	handoff := delegateNextAction{Action: "handoff", Command: "pose delegate handoff " + a.AttemptID + " --to auto"}
	if agent, _, err := routeTaskHandoff(store, latest); err != nil {
		handoff.Why = reason(err)
	} else if plan, err := store.PlanDelegationTaskHandoff(latest.RunID, agent); err != nil {
		handoff.Why = reason(err)
	} else if plan.Existing != nil {
		handoff.Why = tr("the run was already continued", "o run já foi continuado")
	} else {
		handoff.Available = true
	}
	return append(actions, handoff, delegateCancelAction())
}
