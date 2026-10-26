package cli

// pose delegate handoff and the choices a stopped run offers (spec
// pose-delegation-handoff R1, R7–R9). A handoff continues a stopped run with
// another agent under the same attempt; --to auto routes among the agents
// that can run the purpose, the stopped one excluded and the unavailable ones
// skipped. A run that stops interrupted in a terminal offers waiting for the
// reset, handing off or stopping there; elsewhere, and in ls and wait with
// --json, it carries next_actions instead.

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const delegateHandoffUsage = "pose delegate handoff <run> --to <agent|auto> [--dry-run] [--json]"

func cmdDelegateHandoff(root string, args []string, stdout, stderr io.Writer) int {
	f, problem := parseDelegateFlags(args, "--to")
	if problem == "" && (len(f.positional) != 1 || f.to == "") {
		problem = tr("name the run and the agent to hand it to", "nomeie o run e o agente que o recebe")
	}
	if problem != "" {
		return delegateUsage(stdout, stderr, args, "handoff", problem+"\n"+tr("Usage: ", "Uso: ")+delegateHandoffUsage)
	}
	store := cliGovernedStore(root)
	attempts, lost, err := store.ListDelegationAttempts()
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "handoff", 1, err)
	}
	if len(lost) > 0 {
		return delegateFail(stdout, stderr, f.jsonOut, "handoff", 1, withDelegationLosses(errors.New(tr("the run history could not be read whole", "o histórico de runs não pôde ser lido inteiro")), lost))
	}
	selector := f
	selector.agent = ""
	attempt, code, err := namedAttempt(attempts, selector)
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "handoff", code, err)
	}
	if attempt.Active {
		return delegateFail(stdout, stderr, f.jsonOut, "handoff", 1, errors.New(tr("attempt "+attempt.AttemptID+" is still running", "a tentativa "+attempt.AttemptID+" ainda está rodando")))
	}
	latest := attempt.Latest()
	if latest == nil {
		return delegateFail(stdout, stderr, f.jsonOut, "handoff", 1, errors.New(tr("attempt "+attempt.AttemptID+" left no record to continue; start a new run", "a tentativa "+attempt.AttemptID+" não deixou registro a continuar; inicie um novo run")))
	}
	target := latest.RunID
	if containsString(attempt.Runs, f.positional[0]) {
		target = f.positional[0]
	}
	agent := f.to
	var route *posemodel.DelegationRoutingRecord
	if agent == "auto" {
		chosen, record, err := routeHandoff(store, *latest)
		if err != nil {
			if record != nil && f.jsonOut {
				writeDelegateJSON(stdout, map[string]any{"schema": delegateErrorSchema, "command": "handoff", "error": strings.TrimPrefix(err.Error(), "pose: "), "routing": record})
				return 1
			}
			return delegateFail(stdout, stderr, f.jsonOut, "handoff", 1, err)
		}
		agent, route = chosen, record
	}
	plan, err := store.PlanReviewHandoff(target, agent)
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "handoff", 1, err)
	}
	if plan.PendingContinuation != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "handoff", 1, errors.New(tr("a continuation of "+target+" is pending ("+plan.PendingContinuation.ID+")",
			"uma continuação de "+target+" está pendente ("+plan.PendingContinuation.ID+")")))
	}
	if route != nil {
		plan.Routing = route
	}
	plan.Purpose = attempt.Purpose
	for _, warning := range plan.Warnings {
		render(stdout, stderr).Hint("pose delegate handoff: " + warning)
	}
	if f.dryRun {
		extra := map[string]any{"hands_off_from": target, "attempt": attempt.AttemptID, "label": attempt.Label, "routing": plan.Routing}
		if plan.Existing != nil {
			extra["recorded_continuation"] = plan.Existing.RunID
		}
		argv := redactArgv(posemodel.ReviewerArgv(plan.Config.Command, "{output}", "", plan.Effort, plan.Config.Effort), plan.Config)
		return delegatePrintPlan(stdout, stderr, f.jsonOut, plan, argv, extra)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	run, err := store.RunReviewDispatchContext(ctx, plan, time.Now)
	delegateTerminalHook(store, run, err, stdout, stderr, "handoff")
	return delegatePrintRun(stdout, stderr, f.jsonOut, "handoff", run, err)
}

// routeHandoff routes a handoff among the agents of the stopped run's purpose,
// the stopped agent excluded (R7).
func routeHandoff(store posemodel.Store, prior posemodel.ReviewRun) (string, *posemodel.DelegationRoutingRecord, error) {
	readers, found, err := store.LoadReviewersPolicyFor(prior.Kind)
	if err != nil {
		return "", nil, err
	}
	if !found {
		return "", nil, posemodel.ErrNoReviewerAdapter
	}
	policy, _, _, err := store.LoadDelegationPolicy()
	if err != nil {
		return "", nil, err
	}
	bundle, err := store.LoadReviewBundle(prior.BundleID)
	if err != nil {
		return "", nil, err
	}
	impl, from, err := store.RoutingImplementation(bundle, policy, prior.Lead)
	if err != nil {
		return "", nil, err
	}
	eligible := []string{}
	for name := range readers.Adapters {
		if name != prior.Adapter {
			eligible = append(eligible, name)
		}
	}
	avail := store.DelegationAvailabilityOf(policy, time.Now(), true)
	agent, record, err := posemodel.RouteDelegation(policy, eligible, impl, from, avail, time.Now())
	if err != nil {
		return "", &record, err
	}
	if record.Step == posemodel.DelegationStepWait {
		return "", &record, errors.New(tr("every other agent is limited until "+record.WaitUntil+"; hand off after the reset, or wait for this run's reset and resume it",
			"todos os outros agentes estão limitados até "+record.WaitUntil+"; faça o handoff depois do reset, ou espere o reset deste run e o retome"))
	}
	return agent, &record, nil
}

// delegateNextAction is one way forward for a stopped run (R9).
type delegateNextAction struct {
	Action    string `json:"action"`
	Command   string `json:"command"`
	Available bool   `json:"available"`
	Why       string `json:"why,omitempty"`
}

// delegateNextActions lists what a stopped attempt can do next; nil for an
// attempt that runs or concluded.
func delegateNextActions(store posemodel.Store, a posemodel.DelegationAttempt) []delegateNextAction {
	latest := a.Latest()
	if a.Active || latest == nil || latest.Status == posemodel.ReviewRunCompleted || latest.Status == posemodel.ReviewRunStale {
		return nil
	}
	var actions []delegateNextAction
	if i := latest.Interruption; latest.Status == posemodel.ReviewRunInterrupted && i != nil {
		resume := delegateNextAction{Action: "resume", Command: "pose delegate resume " + a.AttemptID}
		reset, resetErr := time.Parse(time.RFC3339Nano, i.ResetAt)
		switch {
		case !i.Resumable:
			resume.Why = tr("the stop is not resumable", "a parada não é retomável")
		case i.ResetAt != "" && resetErr == nil && reset.After(time.Now()):
			resume.Why = tr("the agent is limited until ", "o agente está limitado até ") + i.ResetAt
			actions = append(actions, delegateNextAction{Action: "wait", Command: "pose delegate resume " + a.AttemptID + " --wait-until-reset", Available: true})
		default:
			resume.Available = true
		}
		actions = append(actions, resume)
	}
	handoff := delegateNextAction{Action: "handoff", Command: "pose delegate handoff " + a.AttemptID + " --to auto"}
	if agent, _, err := routeHandoff(store, *latest); err == nil && agent != "" {
		handoff.Available = true
	} else if err != nil {
		handoff.Why = strings.TrimPrefix(err.Error(), "pose: ")
	}
	return append(actions, handoff)
}

// offerDelegateChoices asks, in a terminal, what to do with a run that just
// stopped interrupted: wait for its reset and resume it, hand it off to the
// first available agent, or stop here; it does nothing until answered (R8).
func offerDelegateChoices(root string, run posemodel.ReviewRun, stdout, stderr io.Writer) (int, bool) {
	if run.Status != posemodel.ReviewRunInterrupted || run.AttemptID == "" || !delegateInteractive() {
		return 0, false
	}
	store := cliGovernedStore(root)
	attempts, _, err := store.ListDelegationAttempts()
	if err != nil {
		return 0, false
	}
	var attempt posemodel.DelegationAttempt
	for _, a := range attempts {
		if a.AttemptID == run.AttemptID {
			attempt = a
		}
	}
	actions := delegateNextActions(store, attempt)
	menu := render(io.Discard, stderr)
	keys := map[string]delegateNextAction{}
	for _, action := range actions {
		if !action.Available || action.Action == "resume" && hasAction(actions, "wait") {
			continue
		}
		key := action.Action[:1]
		keys[key] = action
		menu.Usage("[" + key + "] " + action.Command)
	}
	if len(keys) == 0 {
		return 0, false
	}
	menu.Usage("[c] " + tr("stop here", "parar aqui"))
	menu.Usage(tr("the run stopped; choose: ", "o run parou; escolha: "))
	answer, _ := bufio.NewReader(delegateStdin).ReadString('\n')
	chosen, ok := keys[strings.ToLower(strings.TrimSpace(answer))]
	if !ok {
		return 0, false
	}
	fields := strings.Fields(chosen.Command)[2:]
	switch fields[0] {
	case "resume":
		return cmdDelegateResume(root, fields[1:], stdout, stderr), true
	case "handoff":
		return cmdDelegateHandoff(root, fields[1:], stdout, stderr), true
	}
	return 0, false
}

func hasAction(actions []delegateNextAction, name string) bool {
	for _, a := range actions {
		if a.Action == name {
			return true
		}
	}
	return false
}
