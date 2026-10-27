package cli

// pose delegate: the one surface for delegated executions (spec
// pose-delegation-surface; ADR 2026-10-26-delegation-is-a-first-class-execution-surface).
//
// pose review governs the subject (bundle, brief, attestation, verify);
// pose delegate governs executions: who runs, with which access, and what
// happened. review, adjudicate and smoke run through the delivered dispatch
// path; the lifecycle commands read the same run records.
//
// Every subcommand takes the common flags with one meaning (R2): where a
// command starts a run they set it (--agent, --access, --label, --lead);
// where it names or lists runs they select them, and a run that does not
// match is not the one named; --dry-run changes nothing anywhere.

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const delegateUsageText = "pose delegate <review|adjudicate|smoke|task|ls|watch|wait|resume|handoff|cancel|agents> [args] [--agent auto|<name>] [--access read|sandboxed|write|full] [--label <name>] [--lead <vendor>] [--dry-run] [--json]"

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
var delegatePlanned = map[string]string{}

var (
	delegateLabelRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
)

// commonDelegateFlags are taken by every subcommand.
var commonDelegateFlags = []string{"--agent", "--access", "--label", "--lead", "--dry-run", "--json"}

// tr picks the message for the session's locale.
func tr(english, portuguese string) string { return cliText(cliLocaleValue(), english, portuguese) }

type delegateFlags struct {
	agent, access, label, lead, reason, noteFile string
	to, cd, effort, promptFile                   string
	addDirs                                      []string
	noNetwork, record                            bool
	dryRun, jsonOut, active, wait, detach        bool
	verbose, noFollow                            bool
	lines                                        int
	waitMax, timeout, refresh                    time.Duration
	positional                                   []string
}

// parseDelegateFlags reads the common flags and the extra ones a subcommand
// takes; problem is a message for the person, in their locale.
func parseDelegateFlags(args []string, extra ...string) (delegateFlags, string) {
	f := delegateFlags{waitMax: 24 * time.Hour}
	permit := map[string]bool{}
	for _, name := range append(append([]string(nil), commonDelegateFlags...), extra...) {
		permit[name] = true
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, inline, hasInline := strings.Cut(arg, "=")
		if !strings.HasPrefix(arg, "-") {
			f.positional = append(f.positional, arg)
			continue
		}
		if !permit[name] {
			return f, tr("unknown or inapplicable flag "+name, "flag desconhecida ou inaplicável "+name)
		}
		switch name {
		case "--dry-run", "--json", "--active", "--detach", "-v", "--no-follow", "--no-network", "--record":
			if hasInline {
				return f, tr(name+" takes no value", name+" não recebe valor")
			}
			switch name {
			case "--dry-run":
				f.dryRun = true
			case "--json":
				f.jsonOut = true
			case "--detach":
				f.detach = true
			case "-v":
				f.verbose = true
			case "--no-network":
				f.noNetwork = true
			case "--record":
				f.record = true
			case "--no-follow":
				f.noFollow = true
			default:
				f.active = true
			}
		case "--wait-until-reset":
			f.wait = true
			if hasInline {
				d, err := time.ParseDuration(inline)
				if err != nil || d <= 0 || d > 24*time.Hour {
					return f, tr("--wait-until-reset takes a duration up to 24h", "--wait-until-reset recebe uma duração de até 24h")
				}
				f.waitMax = d
			}
		default:
			value := inline
			if !hasInline {
				if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
					return f, tr(name+" needs a value", name+" precisa de um valor")
				}
				value = args[i+1]
				i++
			}
			switch name {
			case "--agent":
				f.agent = value
			case "--access":
				if !containsString([]string{posemodel.DelegationAccessRead, posemodel.DelegationAccessSandboxed, posemodel.DelegationAccessWrite, posemodel.DelegationAccessFull}, value) {
					return f, tr("--access takes read, sandboxed, write or full", "--access recebe read, sandboxed, write ou full")
				}
				f.access = value
			case "--label":
				if !delegateLabelRE.MatchString(value) {
					return f, tr("--label takes letters, digits, '.', '_' and '-' (64 at most)", "--label aceita letras, dígitos, '.', '_' e '-' (no máximo 64)")
				}
				f.label = value
			case "--lead":
				vendor, model, err := posemodel.ParseLead(value)
				if err != nil {
					return f, tr("--lead takes <vendor>[/<model>], such as openai or anthropic/claude-opus-5-5", "--lead recebe <vendor>[/<modelo>], como openai ou anthropic/claude-opus-5-5")
				}
				f.lead = vendor
				if model != "" {
					f.lead += "/" + model
				}
			case "--reason":
				f.reason = value
			case "--lines":
				n, err := strconv.Atoi(value)
				if err != nil || n < 1 || n > 1000 {
					return f, tr("--lines takes a number from 1 to 1000", "--lines recebe um número de 1 a 1000")
				}
				f.lines = n
			case "--refresh":
				d, err := time.ParseDuration(value)
				if err != nil || d < 0 || d > time.Hour {
					return f, tr("--refresh takes a duration from 0 to 1h (0 turns the preview's refresh off)", "--refresh recebe uma duração de 0 a 1h (0 desliga a atualização do preview)")
				}
				f.refresh = d
			case "--timeout":
				d, err := time.ParseDuration(value)
				if err != nil || d <= 0 {
					return f, tr("--timeout takes a positive duration", "--timeout recebe uma duração positiva")
				}
				f.timeout = d
			case "--to":
				f.to = value
			case "--cd":
				f.cd = value
			case "--add-dir":
				f.addDirs = append(f.addDirs, value)
			case "--effort":
				f.effort = value
			case "--prompt":
				f.promptFile = value
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
		return delegateUsage(stdout, stderr, rest, sub, tr("not available yet; it is delivered by spec "+spec, "ainda não está disponível; é entregue pela spec "+spec))
	}
	switch sub {
	case posemodel.DelegationPurposeReview, posemodel.DelegationPurposeAdjudicate, posemodel.DelegationPurposeSmoke:
		return cmdDelegatePurpose(root, sub, rest, stdout, stderr)
	case posemodel.DelegationPurposeTask:
		return cmdDelegateTask(root, rest, stdout, stderr)
	case "ls":
		return cmdDelegateLs(root, rest, stdout, stderr)
	case "resume":
		return cmdDelegateResume(root, rest, stdout, stderr)
	case "cancel":
		return cmdDelegateCancel(root, rest, stdout, stderr)
	case "agents":
		return cmdDelegateAgents(root, rest, stdout, stderr)
	case "wait":
		return cmdDelegateWait(root, rest, stdout, stderr)
	case "watch":
		return cmdDelegateWatch(root, rest, stdout, stderr)
	case "handoff":
		return cmdDelegateHandoff(root, rest, stdout, stderr)
	}
	return delegateUsage(stdout, stderr, rest, sub, tr("unknown subcommand", "subcomando desconhecido")+"\n"+tr("Usage: ", "Uso: ")+delegateUsageText)
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
		// The reason code of a refusal is the same over MCP (spec
		// pose-delegation-mcp R5).
		if code := posemodel.DelegationReasonCode(err); code != "" {
			doc["reason_code"] = code
		}
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

// refError renders a reference error in the person's locale.
func refError(err error) error {
	var ambiguous *posemodel.ErrDelegationRefAmbiguous
	if errors.As(err, &ambiguous) {
		return localizedRefError{err: ambiguous, msg: tr(err.Error(), "a referência nomeia mais de um run delegado: "+strings.Join(ambiguous.Candidates, ", "))}
	}
	var unknown *posemodel.ErrDelegationRefUnknown
	if errors.As(err, &unknown) {
		return errors.New(tr(err.Error(), "nenhum run delegado corresponde à referência (um id de run, um id de tentativa, um label ou um prefixo de sessão de seis caracteres ou mais)"))
	}
	return err
}

// localizedRefError words a reference error in the person's locale and keeps
// the engine's error, so the error document still carries its candidates.
type localizedRefError struct {
	err error
	msg string
}

func (e localizedRefError) Error() string { return e.msg }
func (e localizedRefError) Unwrap() error { return e.err }

// mismatch names the first selecting flag an attempt does not match.
func (f delegateFlags) mismatch(a posemodel.DelegationAttempt) string {
	switch {
	case f.agent != "" && f.agent != "auto" && f.agent != a.Agent:
		return "--agent " + f.agent
	case f.access != "" && f.access != a.Access:
		return "--access " + f.access
	case f.label != "" && f.label != a.Label:
		return "--label " + f.label
	case f.lead != "" && f.lead != a.Lead && leadVendor(f.lead) != a.Lead:
		return "--lead " + f.lead
	}
	return ""
}

// namedAttempt resolves the one reference a lifecycle command takes: the
// positional reference, or --label when there is none; the other common
// flags then select, and a run that does not match is refused.
func namedAttempt(attempts []posemodel.DelegationAttempt, f delegateFlags) (posemodel.DelegationAttempt, int, error) {
	ref := ""
	switch len(f.positional) {
	case 0:
		ref = f.label
	case 1:
		ref = f.positional[0]
	default:
		return posemodel.DelegationAttempt{}, 2, errors.New(tr("one run reference at most", "no máximo uma referência de run"))
	}
	if ref == "" {
		return posemodel.DelegationAttempt{}, 2, errors.New(tr("name a run (a run id, an attempt id, a label or a session prefix)", "nomeie um run (id de run, id de tentativa, label ou prefixo de sessão)"))
	}
	attempt, err := posemodel.ResolveDelegationRef(attempts, ref)
	if err != nil {
		return attempt, 2, refError(err)
	}
	if flag := f.mismatch(attempt); flag != "" {
		return attempt, 2, errors.New(tr("attempt "+attempt.AttemptID+" does not match "+flag, "a tentativa "+attempt.AttemptID+" não corresponde a "+flag))
	}
	return attempt, 0, nil
}

// routeDelegateAgent applies --agent (spec pose-delegation-routing R5–R9): a
// name is taken as given and the run still records the independence it
// achieves; auto (or no flag) routes among the agents that can run the
// purpose. The returned warnings go to stderr.
func routeDelegateAgent(store posemodel.Store, kind, ref, notes string, f delegateFlags, now time.Time) (string, *posemodel.DelegationRoutingRecord, string, int, error) {
	readers, found, err := store.LoadReviewersPolicyFor(kind)
	if err != nil {
		return "", nil, "", 1, err
	}
	if !found || len(readers.Adapters) == 0 {
		return "", nil, "", 1, posemodel.ErrNoReviewerAdapter
	}
	policy, _, _, err := store.LoadDelegationPolicy()
	if err != nil {
		return "", nil, "", 1, err
	}
	brief, err := store.RenderReviewBrief(ref, kind, notes)
	if err != nil {
		return "", nil, "", 1, err
	}
	bundle, err := store.LoadReviewBundle(brief.BundleID)
	if err != nil {
		return "", nil, "", 1, err
	}
	if f.agent != "" && f.agent != "auto" {
		// A named agent is recorded by the plan itself (R9).
		return f.agent, nil, bundle.BundleID, 0, nil
	}
	impl, from, err := store.RoutingImplementation(bundle, policy, f.lead)
	if err != nil {
		return "", nil, "", 2, err
	}
	eligible := make([]string, 0, len(readers.Adapters))
	for name := range readers.Adapters {
		eligible = append(eligible, name)
	}
	avail := store.DelegationAvailabilityOf(policy, now, true)
	agent, record, err := posemodel.RouteDelegation(policy, eligible, impl, from, avail, now)
	if err != nil {
		return "", &record, bundle.BundleID, 1, err
	}
	return agent, &record, bundle.BundleID, 0, nil
}

// leadVendor is the vendor part of a --lead value.
func leadVendor(lead string) string {
	vendor, _, _ := strings.Cut(lead, "/")
	return vendor
}

// waitForRoute sleeps until the reset the wait step chose, without polling,
// or until ctx ends (R7).
func waitForRoute(ctx context.Context, stderr io.Writer, record *posemodel.DelegationRoutingRecord, agent string) error {
	if record == nil || record.Step != posemodel.DelegationStepWait {
		return nil
	}
	until, err := time.Parse(time.RFC3339Nano, record.WaitUntil)
	if err != nil {
		return err
	}
	render(io.Discard, stderr).Hint(tr("pose delegate: every earlier agent is limited; waiting until "+record.WaitUntil+" to start "+agent,
		"pose delegate: todos os agentes anteriores estão limitados; esperando até "+record.WaitUntil+" para iniciar "+agent))
	timer := time.NewTimer(time.Until(until))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func cmdDelegatePurpose(root, purpose string, args []string, stdout, stderr io.Writer) int {
	f, problem := parseDelegateFlags(args, "--reason", "--note-file", "--detach")
	if problem != "" {
		return delegateUsage(stdout, stderr, args, purpose, problem)
	}
	if len(f.positional) != 1 {
		return delegateUsage(stdout, stderr, args, purpose, tr("Usage: ", "Uso: ")+"pose delegate "+purpose+" <scope|bundle-id> [--agent auto|<name>] [--label <name>] [--lead <vendor>] [--reason <text>] [--note-file <path>] [--dry-run] [--json]")
	}
	// A detached run is this command run again in its own session (R1).
	if f.detach && !f.dryRun && !delegateSupervised() {
		return detachDelegateRun(root, purpose, args, f, stdout, stderr)
	}
	// R3: the review-path purposes read only.
	if f.access != "" && f.access != posemodel.DelegationAccessRead {
		return delegateUsage(stdout, stderr, args, purpose, tr("runs with read access only; --access "+f.access+" is for pose delegate task",
			"roda só com acesso de leitura; --access "+f.access+" é para pose delegate task"))
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
	agent, route, routedBundle, code, err := routeDelegateAgent(store, kind, ref, notes, f, time.Now())
	if errors.Is(err, posemodel.ErrNoReviewerAdapter) {
		return delegateNoAgent(stdout, stderr, f.jsonOut, purpose)
	}
	if err != nil {
		var none *posemodel.ErrNoDelegationAgentAvailable
		if errors.As(err, &none) && f.jsonOut {
			writeDelegateJSON(stdout, map[string]any{"schema": delegateErrorSchema, "command": purpose, "error": strings.TrimPrefix(err.Error(), "pose: "), "routing": none.Record, "reason_code": posemodel.DelegationReasonCode(err)})
			return code
		}
		return delegateFail(stdout, stderr, f.jsonOut, purpose, code, err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if !f.dryRun && route != nil && route.Step == posemodel.DelegationStepWait {
		// A detached run hands its attempt back when it starts; one that would
		// first wait hours for a reset has none to hand back (D3).
		if delegateSupervised() {
			return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, errors.New(tr("every agent is limited until "+route.WaitUntil+"; a detached run does not wait for a reset: run it without --detach, or after the reset",
				"todos os agentes estão limitados até "+route.WaitUntil+"; um run destacado não espera um reset: rode sem --detach, ou depois do reset")))
		}
		if err := waitForRoute(ctx, stderr, route, agent); err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, err)
		}
		// The bundle or the policy may have changed during the wait: route
		// again against what the run will read, and start only the same agent
		// (R7, R8).
		again, rerouted, rebundle, code, err := routeDelegateAgent(store, kind, ref, notes, f, time.Now())
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, purpose, code, err)
		}
		if again != agent || rebundle != routedBundle || rerouted.Step == posemodel.DelegationStepWait {
			return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, errors.New(tr("the bundle, the policy or the agents changed while waiting; run pose delegate again",
				"o bundle, a policy ou os agentes mudaram durante a espera; rode pose delegate de novo")))
		}
		route.Implementation, route.ImplementationFrom = rerouted.Implementation, rerouted.ImplementationFrom
	}
	plan, err := store.PlanReviewDispatch(ref, kind, notes, agent, f.reason)
	if errors.Is(err, posemodel.ErrNoReviewerAdapter) {
		return delegateNoAgent(stdout, stderr, f.jsonOut, purpose)
	}
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, err)
	}
	if plan.Bundle.BundleID != routedBundle {
		return delegateFail(stdout, stderr, f.jsonOut, purpose, 1, errors.New(tr("the bundle of the scope changed while the run was being routed; run pose delegate again",
			"o bundle do escopo mudou enquanto o run era roteado; rode pose delegate de novo")))
	}
	plan.Label, plan.Purpose, plan.Lead = f.label, purpose, f.lead
	warnings := plan.Warnings
	if route == nil {
		store.PlanImplementationWithLead(&plan, f.lead)
		warnings = plan.Warnings
		if plan.SameVendorWarning != "" {
			warnings = append(warnings, plan.SameVendorWarning)
		}
	} else {
		plan.Routing = route
	}
	for _, warning := range warnings {
		render(stdout, stderr).Hint("pose delegate " + purpose + ": " + warning)
	}
	impl := plan.Implementation()
	independence := posemodel.AchievedIndependence(impl.Vendor, impl.Model, plan.Config.Vendor, plan.Config.Model)
	if f.dryRun {
		argv := redactArgv(posemodel.ReviewerArgv(plan.Config.Command, "{output}", "", plan.Effort, plan.Config.Effort), plan.Config)
		return delegatePrintPlan(stdout, stderr, f.jsonOut, plan, argv, map[string]any{"scope": plan.Brief.Scope, "bundle": plan.Bundle.BundleID,
			"brief_digest": plan.Brief.Digest, "prior_runs": plan.PriorRuns, "label": plan.Label, "lead": plan.Lead,
			"routing": plan.Routing, "independence": independence, "implementation": posemodel.DescribeImplementation(impl)})
	}
	if delegateSupervised() {
		plan.OnStart = supervisorHandback()
	}
	run, err := store.RunReviewDispatchContext(ctx, plan, time.Now)
	delegateTerminalHook(store, run, err, stdout, stderr, purpose)
	code = delegatePrintRun(stdout, stderr, f.jsonOut, purpose, run, err, delegateRunNextActions(store, run, err)...)
	if err == nil && !f.jsonOut && !delegateSupervised() {
		if next, chose := offerDelegateChoices(root, run, stdout, stderr); chose {
			return next
		}
	}
	return code
}

func delegateNoAgent(stdout, stderr io.Writer, jsonOut bool, command string) int {
	// The document keeps the typed refusal and no example: the example is
	// for a person reading the terminal (spec pose-delegation-mcp R5, R7).
	if jsonOut {
		return delegateFail(stdout, stderr, true, command, 1, posemodel.ErrNoReviewerAdapter)
	}
	example, _ := posemodel.CanonicalDelegationPolicy(posemodel.ExampleDelegationPolicy())
	err := errors.New(tr("no agent is configured; add .pose/policy/delegation.json, for example:\n", "nenhum agente configurado; adicione .pose/policy/delegation.json, por exemplo:\n") + string(example))
	return delegateFail(stdout, stderr, jsonOut, command, 1, err)
}

// delegateGivenRoot checks the root a caller names: an absolute directory
// holding a POSE instance.
func delegateGivenRoot(root string) (string, error) {
	if !filepath.IsAbs(root) {
		return "", errors.New(posemodel.DelegateRootEnv + " must be an absolute path")
	}
	if info, err := os.Stat(filepath.Join(root, ".pose")); err != nil || !info.IsDir() {
		return "", errors.New(posemodel.DelegateRootEnv + " names no POSE instance")
	}
	return filepath.Clean(root), nil
}

// delegateTerminalHook runs the policy's on_terminal hook once the run is
// recorded, and says so when it failed; the run's state does not change (R7).
func delegateTerminalHook(store posemodel.Store, run posemodel.ReviewRun, err error, stdout, stderr io.Writer, command string) {
	if err != nil || run.RunID == "" {
		return
	}
	if outcome := store.RunDelegationTerminalHook(run); outcome != nil && outcome.Failure != "" {
		render(stdout, stderr).Hint("pose delegate " + command + ": on_terminal: " + outcome.Failure)
	}
}

// routeSummary renders a routing decision for a person: the step chosen and
// each agent skipped with its reason.
func routeSummary(route *posemodel.DelegationRoutingRecord) string {
	text := "step=" + route.Step
	if route.WaitUntil != "" {
		text += " wait_until=" + route.WaitUntil
	}
	for _, c := range route.Candidates {
		if c.Skipped != "" {
			text += "; " + c.Agent + " " + tr("skipped", "ignorado") + ": " + c.Skipped
		}
	}
	return text
}

// redactArgv hides the values of the credentials an agent is authorized to
// receive, wherever a template carries them (security of R12).
func redactArgv(argv []string, config posemodel.ReviewerAdapter) []string {
	out := append([]string(nil), argv...)
	if config.Containment == nil {
		return out
	}
	// One pass, the longest value first: a credential that contains another
	// is replaced whole, whatever order the policy lists them in, and a
	// replacement is never scanned again (review round 3 of
	// pose-delegation-task, R3-B1).
	var values []string
	for _, name := range config.Containment.Credentials {
		if value := os.Getenv(name); len(value) >= 4 {
			values = append(values, value)
		}
	}
	sort.Slice(values, func(i, j int) bool { return len(values[i]) > len(values[j]) })
	pairs := make([]string, 0, 2*len(values))
	for _, value := range values {
		pairs = append(pairs, value, "[credential]")
	}
	if len(pairs) == 0 {
		return out
	}
	replacer := strings.NewReplacer(pairs...)
	for i := range out {
		out[i] = replacer.Replace(out[i])
	}
	return out
}

// delegatePrintPlan prints what a run would do (R12): agent, purpose, access,
// working directory and argv, plus the fields of the command at hand.
func delegatePrintPlan(stdout, stderr io.Writer, jsonOut bool, plan posemodel.DispatchPlan, argv []string, extra map[string]any) int {
	workdir := tr("a disposable worktree at ", "um worktree descartável em ") + plan.SealedCommit
	doc := map[string]any{"schema": delegatePlanSchema, "purpose": plan.Purpose, "agent": plan.Adapter, "vendor": plan.Config.Vendor, "model": plan.Config.Model,
		"access": posemodel.DelegationAccessRead, "workdir": workdir, "argv": argv, "reserves_attempt": false}
	for k, v := range extra {
		doc[k] = v
	}
	if jsonOut {
		writeDelegateJSON(stdout, doc)
		return 0
	}
	out := render(stdout, stderr)
	out.Field("delegate.purpose", plan.Purpose)
	out.Field("delegate.agent", fmt.Sprintf("%s (%s %s)", plan.Adapter, plan.Config.Vendor, plan.Config.Model))
	out.Field("delegate.access", posemodel.DelegationAccessRead)
	out.Field("delegate.workdir", workdir)
	raw, _ := json.Marshal(argv)
	out.Field("delegate.argv", string(raw))
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if route, ok := extra[k].(*posemodel.DelegationRoutingRecord); ok {
			if route != nil {
				out.Field("delegate.routing", routeSummary(route))
			}
			continue
		}
		if value := fmt.Sprint(extra[k]); value != "" {
			out.Field("delegate."+k, value)
		}
	}
	out.Field("delegate.dry_run", tr("true (nothing reserved or claimed)", "true (nada reservado nem reivindicado)"))
	return 0
}

func delegatePrintRun(stdout, stderr io.Writer, jsonOut bool, command string, run posemodel.ReviewRun, err error, next ...delegateNextAction) int {
	code := 0
	if err != nil || run.Status != posemodel.ReviewRunCompleted {
		code = 1
	}
	attestation := tr("not recorded: a run's conclusion is recorded only by the engine's verified step", "não registrada: a conclusão de um run só é registrada pelo passo verificado do motor")
	if jsonOut {
		doc := map[string]any{"schema": delegateRunSchema, "purpose": run.Purpose, "agent": run.Adapter, "vendor": run.Vendor, "model": run.Model,
			"label": run.Label, "lead": run.Lead, "scope": run.Scope, "bundle": run.BundleID, "attempt": run.AttemptID, "run": run.RunID, "resumes": run.Resumes, "hands_off_from": run.HandsOffFrom,
			"status": run.Status, "failure": run.Failure, "record": run.Path, "events": run.EventsDigest != "", "env_removed": run.EnvRemoved, "attestation": attestation}
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
	if run.RunID != "" {
		out.Field("delegate.purpose", run.Purpose)
		out.Field("delegate.agent", fmt.Sprintf("%s (%s %s)", run.Adapter, run.Vendor, run.Model))
		if run.Label != "" {
			out.Field("delegate.label", run.Label)
		}
		out.Field("delegate.attempt", run.AttemptID)
		if run.Resumes != "" {
			out.Field("delegate.resumes", run.Resumes)
		}
		if run.HandsOffFrom != "" {
			out.Field("delegate.hands_off_from", run.HandsOffFrom)
		}
		out.Field("delegate.run", run.RunID)
		// R9 outside a terminal: the ways forward of a stopped run.
		for _, action := range next {
			value := action.Command
			if !action.Available {
				value = tr("unavailable: ", "indisponível: ") + action.Why
			} else if value == "" {
				value = action.Why
			}
			out.Field("delegate.next."+action.Action, value)
		}
		out.Field("delegate.status", run.Status)
		if run.Failure != "" {
			out.Field("delegate.failure", run.Failure)
		}
		if run.Interruption != nil {
			out.Field("delegate.interruption", fmt.Sprintf("cause=%s resumable=%t reset_at=%s", run.Interruption.Cause, run.Interruption.Resumable, run.Interruption.ResetAt))
		}
		out.Field("delegate.record", run.Path)
		out.Field("delegate.attestation", attestation)
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
	f, problem := parseDelegateFlags(args, "--active")
	if problem == "" && len(f.positional) > 1 {
		problem = tr("one reference at most", "no máximo uma referência")
	}
	if problem != "" {
		return delegateUsage(stdout, stderr, args, "ls", problem+"\n"+tr("Usage: ", "Uso: ")+"pose delegate ls [<run>|<bundle-id>] [--active] [--agent <name>] [--access <level>] [--label <name>] [--lead <vendor>] [--json]")
	}
	store := posemodel.Store{Root: root}
	attempts, lost, err := store.ListDelegationAttempts()
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "ls", 1, err)
	}
	if len(f.positional) == 1 && !strings.HasPrefix(f.positional[0], "rvb-") {
		one, err := posemodel.ResolveDelegationRef(attempts, f.positional[0])
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "ls", 2, withDelegationLosses(refError(err), lost))
		}
		// A named run is refused when a selecting flag does not match it, as on
		// every other lifecycle command (D8); without a reference they filter.
		if flag := f.mismatch(one); flag != "" {
			return delegateFail(stdout, stderr, f.jsonOut, "ls", 2, errors.New(tr("attempt "+one.AttemptID+" does not match "+flag, "a tentativa "+one.AttemptID+" não corresponde a "+flag)))
		}
		attempts = []posemodel.DelegationAttempt{one}
	}
	selected := []posemodel.DelegationAttempt{}
	for _, a := range attempts {
		if len(f.positional) == 1 && strings.HasPrefix(f.positional[0], "rvb-") && a.BundleID != f.positional[0] {
			continue
		}
		if f.active && !a.Active || f.mismatch(a) != "" {
			continue
		}
		selected = append(selected, a)
	}
	if lost == nil {
		lost = []string{}
	}
	if f.jsonOut {
		// R9: a stopped attempt carries the ways forward it has now.
		type listed struct {
			posemodel.DelegationAttempt
			NextActions []delegateNextAction `json:"next_actions,omitempty"`
		}
		rows := make([]listed, 0, len(selected))
		for _, a := range selected {
			rows = append(rows, listed{a, delegateNextActions(store, a)})
		}
		writeDelegateJSON(stdout, map[string]any{"schema": delegateListSchema, "attempts": rows, "losses": lost})
	} else {
		out := render(stdout, stderr)
		if len(selected) == 0 {
			out.Field("delegate.runs", tr("none", "nenhum"))
		}
		for _, a := range selected {
			value := fmt.Sprintf("state=%s purpose=%s agent=%s turns=%d age=%s", a.State, a.Purpose, a.Agent, a.Turns, delegateAge(a.StartedAt))
			if a.Round > 0 {
				value += fmt.Sprintf(" round=%d", a.Round)
			}
			if a.Label != "" {
				value += " label=" + a.Label
			}
			if a.Active && a.Liveness != "" {
				value += " owner=" + a.Liveness
			}
			if a.HeartbeatAge != "" {
				value += " heartbeat=" + a.HeartbeatAge
			}
			if a.StalledAt != "" {
				value += " stalled_at=" + a.StalledAt
			}
			if a.HookFailure != "" {
				value += " on_terminal_failed=" + strconv.Quote(a.HookFailure)
			}
			if a.Interruption != nil {
				value += fmt.Sprintf(" cause=%s resumable=%t partial=%t", a.Interruption.Cause, a.Interruption.Resumable, a.Partial)
				if a.Interruption.ResetAt != "" {
					value += " reset_at=" + a.Interruption.ResetAt + " time_left=" + a.TimeLeft
				}
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
			out.Failure(tr("delegated run history incomplete: ", "histórico de runs delegados incompleto: ") + loss)
		}
	}
	if len(lost) > 0 {
		return 1
	}
	return 0
}

// resumeTarget is the run a resume continues (R11 with the delivered
// continuity): the run named by its id, or the attempt's latest run when it
// is interrupted, or the interrupted run the latest one continued, whose
// recorded continuation is then returned instead of a new run.
func resumeTarget(attempt posemodel.DelegationAttempt, ref string) (string, error) {
	if containsString(attempt.Runs, ref) {
		return ref, nil
	}
	latest := attempt.Latest()
	switch {
	case latest == nil:
		return "", errors.New(tr("attempt "+attempt.AttemptID+" has no recorded run", "a tentativa "+attempt.AttemptID+" não tem run registrado"))
	case latest.Status == posemodel.ReviewRunInterrupted:
		return latest.RunID, nil
	case latest.Resumes != "":
		return latest.Resumes, nil
	}
	return "", errors.New(tr("the latest run of attempt "+attempt.AttemptID+" is "+latest.Status+", not interrupted",
		"o run mais recente da tentativa "+attempt.AttemptID+" está "+latest.Status+", não interrompido"))
}

func cmdDelegateResume(root string, args []string, stdout, stderr io.Writer) int {
	f, problem := parseDelegateFlags(args, "--wait-until-reset", "--prompt", "--detach")
	if problem != "" {
		return delegateUsage(stdout, stderr, args, "resume", problem+"\n"+tr("Usage: ", "Uso: ")+"pose delegate resume <run> [--wait-until-reset[=<max>]] [--prompt <file|->] [--detach] [--dry-run] [--json]")
	}
	// A detached resume is this command run again in its own session (spec
	// pose-delegation-mcp R4); it hands its attempt back once it starts.
	if f.detach && !f.dryRun && !delegateSupervised() {
		if f.wait {
			return delegateUsage(stdout, stderr, args, "resume", tr("--detach does not wait for a reset: resume after it, or without --detach", "--detach não espera um reset: retome depois dele, ou sem --detach"))
		}
		return detachDelegateRun(root, "resume", args, f, stdout, stderr)
	}
	store := cliGovernedStore(root)
	attempts, _, err := store.ListDelegationAttempts()
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, err)
	}
	attempt, code, err := namedAttempt(attempts, f)
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "resume", code, err)
	}
	if attempt.Active {
		return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, errors.New(tr("attempt "+attempt.AttemptID+" is still running", "a tentativa "+attempt.AttemptID+" ainda está rodando")))
	}
	ref := attempt.AttemptID
	if len(f.positional) == 1 {
		ref = f.positional[0]
	}
	target, err := resumeTarget(attempt, ref)
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, err)
	}
	if attempt.Purpose == posemodel.DelegationPurposeTask {
		return cmdDelegateTaskResume(root, store, f, attempt, target, stdout, stderr)
	}
	if f.promptFile != "" {
		return delegateUsage(stdout, stderr, args, "resume", tr("--prompt is for a task; a review is resumed with its own brief", "--prompt é para uma task; uma review é retomada com o próprio brief"))
	}
	plan, err := store.PlanReviewResume(target)
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, err)
	}
	if plan.PendingContinuation != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, errors.New(tr("a continuation of "+target+" is pending ("+plan.PendingContinuation.ID+")",
			"uma continuação de "+target+" está pendente ("+plan.PendingContinuation.ID+")")))
	}
	var delay time.Duration
	if f.wait && plan.Existing == nil {
		var seed [1]byte
		if _, err := rand.Read(seed[:]); err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, err)
		}
		var prior posemodel.ReviewRun
		if latest := attempt.Latest(); latest != nil && latest.RunID == target {
			prior = *latest
		} else if runs, rerr := store.ListReviewRuns(""); rerr == nil {
			for _, r := range runs {
				if r.RunID == target {
					prior = r
				}
			}
		}
		delay, err = posemodel.ReviewResumeDelay(prior, time.Now(), f.waitMax, time.Duration(seed[0])*time.Second/255)
		if err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, err)
		}
	}
	if f.dryRun {
		extra := map[string]any{"resumes": target, "attempt": attempt.AttemptID, "label": attempt.Label}
		if plan.Existing != nil {
			extra["recorded_continuation"] = plan.Existing.RunID
		}
		if f.wait {
			extra["wait"] = delay.String()
		}
		plan.Purpose = attempt.Purpose
		argv := redactArgv(posemodel.ReviewerArgv(plan.Config.Resume.Command, "{output}", "{session}", plan.Effort, plan.Config.Effort), plan.Config)
		return delegatePrintPlan(stdout, stderr, f.jsonOut, plan, argv, extra)
	}
	if delay > 0 {
		if !f.jsonOut {
			render(stdout, stderr).Field("delegate.wait", delay.String())
		}
		time.Sleep(delay)
		if plan, err = store.PlanReviewResume(target); err != nil {
			return delegateFail(stdout, stderr, f.jsonOut, "resume", 1, err)
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if delegateSupervised() {
		plan.OnStart = supervisorHandback()
	}
	run, err := store.RunReviewDispatchContext(ctx, plan, time.Now)
	delegateTerminalHook(store, run, err, stdout, stderr, "resume")
	code = delegatePrintRun(stdout, stderr, f.jsonOut, "resume", run, err, delegateRunNextActions(store, run, err)...)
	if err == nil && !f.jsonOut {
		if next, chose := offerDelegateChoices(root, run, stdout, stderr); chose {
			return next
		}
	}
	return code
}

func cmdDelegateCancel(root string, args []string, stdout, stderr io.Writer) int {
	f, problem := parseDelegateFlags(args)
	if problem != "" {
		return delegateUsage(stdout, stderr, args, "cancel", problem+"\n"+tr("Usage: ", "Uso: ")+"pose delegate cancel <run> [--dry-run] [--json]")
	}
	store := posemodel.Store{Root: root}
	attempts, lost, err := store.ListDelegationAttempts()
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "cancel", 1, err)
	}
	attempt, code, err := namedAttempt(attempts, f)
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "cancel", code, withDelegationLosses(err, lost))
	}
	if !attempt.Active {
		return delegateFail(stdout, stderr, f.jsonOut, "cancel", 1, withDelegationLosses(errors.New(tr("attempt "+attempt.AttemptID+" is "+attempt.State+", not running",
			"a tentativa "+attempt.AttemptID+" está "+attempt.State+", não rodando")), lost))
	}
	if f.dryRun {
		if f.jsonOut {
			writeDelegateJSON(stdout, map[string]any{"schema": delegateCancelSchema, "attempt": attempt.AttemptID, "requested": false, "dry_run": true, "owner": attempt.Liveness})
			return 0
		}
		out := render(stdout, stderr)
		out.Field("delegate.attempt", attempt.AttemptID)
		out.Field("delegate.owner", attempt.Liveness)
		out.Field("delegate.dry_run", tr("true (no cancellation requested)", "true (nenhum cancelamento pedido)"))
		return 0
	}
	marker, err := store.RequestDelegationCancel(attempt.AttemptID)
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "cancel", 1, err)
	}
	deadline := time.Now().Add(60 * time.Second)
	for store.DelegationRunActive(marker) && time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
	}
	stopped := !store.DelegationRunActive(marker)
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
		out.Field("delegate.cancel", tr("requested", "pedido"))
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

// withDelegationLosses names what the listing could not read beside a run it
// did not find running: the run may be the one an unreadable marker hides.
func withDelegationLosses(err error, lost []string) error {
	if len(lost) == 0 {
		return err
	}
	return fmt.Errorf("%w (%s %s)", err, tr("unread:", "não lido:"), strings.Join(lost, "; "))
}

// delegateAgentRow is one line of pose delegate agents (R14).
type delegateAgentRow struct {
	Name              string   `json:"name"`
	Vendor            string   `json:"vendor"`
	Model             string   `json:"model"`
	CLI               string   `json:"cli"`
	OnPath            bool     `json:"on_path"`
	Access            []string `json:"access"`
	Events            string   `json:"events"`
	LimitResetAt      string   `json:"limit_reset_at,omitempty"`
	Limited           bool     `json:"limited"`
	State             string   `json:"state"`
	LimitedUntil      string   `json:"limited_until,omitempty"`
	Detail            string   `json:"detail,omitempty"`
	IndependentOfLead *bool    `json:"independent_of_lead,omitempty"`
}

func cmdDelegateAgents(root string, args []string, stdout, stderr io.Writer) int {
	f, problem := parseDelegateFlags(args)
	if problem == "" && len(f.positional) > 0 {
		problem = tr("no arguments", "sem argumentos")
	}
	if problem != "" {
		return delegateUsage(stdout, stderr, args, "agents", problem+"\n"+tr("Usage: ", "Uso: ")+"pose delegate agents [--agent <name>] [--access <level>] [--label <name>] [--lead <vendor>] [--json]")
	}
	store := posemodel.Store{Root: root}
	policy, path, found, err := store.LoadDelegationPolicy()
	if err != nil {
		return delegateFail(stdout, stderr, f.jsonOut, "agents", 1, err)
	}
	labelled := map[string]bool{}
	if runs, _, err := store.ReadReviewRuns(""); err == nil {
		for _, run := range runs {
			if f.label != "" && run.Label == f.label {
				labelled[run.Adapter] = true
			}
		}
	}
	// Availability as routing reads it (spec pose-delegation-routing R6).
	avail := store.DelegationAvailabilityOf(policy, time.Now(), true)
	rows := []delegateAgentRow{}
	for _, name := range policy.DeclaredAgentNames() {
		agent := policy.Agents[name]
		access := make([]string, 0, len(agent.Commands))
		for level := range agent.Commands {
			access = append(access, level)
		}
		sort.Strings(access)
		if f.agent != "" && f.agent != "auto" && f.agent != name || f.access != "" && !containsString(access, f.access) || f.label != "" && !labelled[name] {
			continue
		}
		events := agent.Events
		if events == "" {
			events = posemodel.DelegationEventsPlain
		}
		a := avail[name]
		row := delegateAgentRow{Name: name, Vendor: agent.Vendor, Model: agent.Model, CLI: a.CLI, OnPath: a.OnPath, Access: access, Events: events, LimitResetAt: a.LimitResetAt,
			State: a.State, LimitedUntil: a.LimitedUntil, Detail: a.Detail}
		// limited is the state routing reads now; limit_reset_at stays the
		// reset of the last limit ever recorded.
		row.Limited = a.State == posemodel.DelegationAgentLimited
		if f.lead != "" {
			independent := !strings.EqualFold(strings.TrimSpace(agent.Vendor), leadVendor(f.lead))
			row.IndependentOfLead = &independent
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
		out.Field("delegate.agents", tr("none configured (.pose/policy/delegation.json)", "nenhum configurado (.pose/policy/delegation.json)"))
		return 0
	}
	out.Field("delegate.policy", path)
	for _, row := range rows {
		state := row.State
		switch {
		case row.State == posemodel.DelegationAgentLimited && row.LimitedUntil != "":
			state = tr("limited until ", "limitado até ") + row.LimitedUntil
		case row.State == posemodel.DelegationAgentAvailable:
			state = tr("available", "disponível")
		case row.State == posemodel.DelegationAgentMissing:
			state = tr("missing", "ausente")
		case row.Detail != "":
			state += " (" + row.Detail + ")"
		}
		value := fmt.Sprintf("%s · vendor=%s model=%s cli=%s on_path=%t access=%s events=%s", state, row.Vendor, row.Model, row.CLI, row.OnPath, strings.Join(row.Access, ","), row.Events)
		if row.LimitResetAt != "" {
			value += " last_limit_reset=" + row.LimitResetAt
			if row.Limited {
				value += tr(" (limited now)", " (limitado agora)")
			}
		}
		if row.IndependentOfLead != nil {
			value += fmt.Sprintf(" independent_of_lead=%t", *row.IndependentOfLead)
		}
		out.Field("delegate.agent."+row.Name, value)
	}
	return 0
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
