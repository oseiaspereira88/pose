package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// cmdClosePlan is `pose close spec:<slug> --plan|--apply|--resume` (spec
// pose-recoverable-closeout-plan): the closeout as an ordered, recoverable
// plan. --plan previews; --apply runs every mechanical step in order and stops
// at the first one that needs a person (a judgment criterion) or fails;
// --resume is --apply after an interruption — the plan is recomputed from the
// repository, so done steps are not repeated.
//
// Apply and resume build one posemodel.OperationResult (spec
// pose-cli-mcp-operation-results); --json writes it as the only document on
// stdout and the human renderer reads the same value.
func cmdClosePlan(root string, args []string, stdout, stderr io.Writer) int {
	const usage = "Usage: pose close spec:<slug> --plan [--json] | --apply|--resume [--digest <plan digest>] [--reviewer agent:<id>] [--skip-evidence] [--json]"
	ref, mode, digest, reviewer, jsonOutput, skipEvidence := "", "", "", "", false, false
	refuse := func(reason string) int {
		render(io.Discard, stderr).Failure("pose close: " + reason)
		return usageError(stderr, usage)
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--plan", "--apply", "--resume":
			// One mode per call: a second, different mode is refused rather
			// than resolved by its position (R3).
			next := strings.TrimPrefix(args[i], "--")
			if mode != "" && mode != next {
				return refuse("--" + mode + " and --" + next + " are exclusive modes")
			}
			mode = next
		case "--json":
			jsonOutput = true
		case "--skip-evidence":
			skipEvidence = true
		case "--digest", "--reviewer":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return refuse(args[i] + " needs a value")
			}
			i++
			if args[i-1] == "--digest" {
				digest = args[i]
			} else {
				reviewer = args[i]
			}
		default:
			if strings.HasPrefix(args[i], "-") || ref != "" {
				return usageError(stderr, usage)
			}
			ref = args[i]
		}
	}
	if ref == "" || mode == "" {
		return usageError(stderr, usage)
	}
	if mode == "plan" && (digest != "" || reviewer != "" || skipEvidence) {
		return refuse("--digest, --reviewer and --skip-evidence are taken only by --apply and --resume")
	}
	store := cliGovernedStore(root)
	out := render(stdout, stderr)
	if mode == "plan" {
		plan, err := store.PlanCloseout(ref)
		if err != nil && jsonOutput {
			return writeOperationFailure(stdout, stderr, "close.plan", ref, "plan-unavailable", "pose close: "+err.Error())
		}
		if err != nil {
			out.Failure("pose close: " + err.Error())
			return 1
		}
		if jsonOutput {
			return writeJSON(stdout, plan)
		}
		renderCloseoutPlan(out, plan)
		if checkpoint, ok := store.ReadCloseoutCheckpoint(ref); ok {
			out.Field("closeout_plan.interrupted", fmt.Sprintf("an apply started %s completed %s; --resume continues", checkpoint.StartedAt, strings.Join(checkpoint.Completed, ",")))
		}
		return 0
	}
	result := applyClosePlan(root, store, ref, mode, digest, reviewer, skipEvidence)
	if result.Outcome == posemodel.OutcomeWaiting {
		// A stop on another actor is not a failed execution (R8).
		noteCommandUsage(stdout, commandUsageResult{SemanticOutcome: "unknown", ExecutionOutcome: "waiting"})
	}
	if jsonOutput {
		if writeJSON(stdout, result) != 0 {
			return 1
		}
	} else {
		renderClosePlanResult(out, result)
	}
	return closePlanExitCode(result)
}

// observePublishedResults confirms that this validate run published its
// results. validate prints "json: <path>" only after its own write returned
// without error, so that line in this run's output is the fact bound to the
// run; an older file at the path, however recent, is not. The file must also
// be a regular file holding a JSON document.
func observePublishedResults(root, resultsPath, runOutput string) error {
	target := filepath.Join(root, filepath.FromSlash(resultsPath))
	reported := false
	for _, line := range strings.Split(runOutput, "\n") {
		if strings.TrimSpace(line) == "json: "+target {
			reported = true
			break
		}
	}
	if !reported {
		return fmt.Errorf("this run did not report writing %s", resultsPath)
	}
	info, err := os.Stat(target)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a results file", resultsPath)
	}
	raw, err := os.ReadFile(target)
	if err != nil || !json.Valid(raw) {
		return fmt.Errorf("%s does not hold a JSON document", resultsPath)
	}
	return nil
}

// writeOperationFailure is the failure of a route that supports --json and
// keeps its legacy success document: stdout carries one operation result
// with the typed cause, and stderr the human line (spec
// pose-cli-mcp-operation-results R2).
func writeOperationFailure(stdout, stderr io.Writer, operation, scope, cause, message string) int {
	result := posemodel.NewOperationResult(operation, scope)
	result.Fail(posemodel.OutcomeFailed, "", cause, message)
	render(io.Discard, stderr).Failure(message)
	if writeJSON(stdout, result) != 0 {
		return 1
	}
	return 1
}

// closeApplyData is the payload of a close apply/resume result: the plan as
// the repository stands after this call and, at a judgment, what the
// reviewer needs.
type closeApplyData struct {
	Plan          *posemodel.CloseoutPlan `json:"plan,omitempty"`
	BundleID      string                  `json:"bundle_id,omitempty"`
	EvidenceRefs  []string                `json:"evidence_refs,omitempty"`
	AttestCommand string                  `json:"attest_command,omitempty"`
}

// closePlanExitCode keeps the documented exit codes: 0 when the call
// completed or had nothing to do, 3 at a judgment, 1 otherwise.
func closePlanExitCode(result posemodel.OperationResult) int {
	switch result.Outcome {
	case posemodel.OutcomeCompleted, posemodel.OutcomeAlreadySatisfied:
		return 0
	case posemodel.OutcomeWaiting:
		return 3
	default:
		return 1
	}
}

// closeStepEffect is the effect a step confirms once the re-planned
// repository reflects it; verify and judgment write nothing.
func closeStepEffect(stepID string) string {
	switch stepID {
	case posemodel.CloseoutStepEvidence:
		return "results-written"
	case posemodel.CloseoutStepIndex:
		return "index-regenerated"
	case posemodel.CloseoutStepSeal:
		return "bundle-sealed"
	case posemodel.CloseoutStepMechanical:
		return "attestation-recorded"
	case posemodel.CloseoutStepTransition:
		return "spec-closed"
	default:
		return ""
	}
}

func applyClosePlan(root string, store posemodel.Store, ref, mode, digest, reviewer string, skipEvidence bool) posemodel.OperationResult {
	result := posemodel.NewOperationResult("close."+mode, ref)
	resume := "`pose close " + ref + " --resume`"
	data := &closeApplyData{}
	result.Data = data
	plan, err := store.PlanCloseout(ref)
	if err != nil {
		result.Fail(posemodel.OutcomeFailed, "", "plan-unavailable", err.Error())
		return result
	}
	data.Plan = &plan
	if digest != "" && digest != plan.Digest {
		result.Fail(posemodel.OutcomeConflict, "", "plan-digest-mismatch", "the repository changed since the plan was computed ("+digest+" → "+plan.Digest+")")
		result.NextAction = "review the new plan with `pose close " + ref + " --plan`"
		return result
	}
	checkpoint, resumed := store.ReadCloseoutCheckpoint(ref)
	if !resumed {
		checkpoint = posemodel.CloseoutCheckpoint{SchemaVersion: 1, Scope: ref, StartedAt: time.Now().UTC().Format(time.RFC3339), PlanDigest: plan.Digest}
	}
	record := func(stepID string, stepErr error) {
		if stepErr != nil {
			checkpoint.LastError = stepID + ": " + stepErr.Error()
		} else {
			checkpoint.LastError = ""
			checkpoint.Completed = append(checkpoint.Completed, stepID)
		}
		if err := store.WriteCloseoutCheckpoint(checkpoint); err != nil {
			result.Unconfirmed = append(result.Unconfirmed, posemodel.OperationUncertain{Step: "checkpoint", Reason: "progress after " + stepID + " was not recorded: " + err.Error() + "; --resume re-plans from the repository"})
		}
	}
	confirm := func(stepID, ref string) {
		result.StepsRun = append(result.StepsRun, posemodel.OperationStep{ID: stepID, Result: "done"})
		if effect := closeStepEffect(stepID); effect != "" {
			result.Effects = append(result.Effects, posemodel.OperationEffect{Step: stepID, Effect: effect, Ref: ref})
		}
	}
	// fail ends the call at a step that ran. wrote says the step may have
	// written before failing: its effect is then neither confirmed nor
	// presumed rolled back (R6).
	fail := func(stepID, cause string, stepErr error, wrote bool) posemodel.OperationResult {
		record(stepID, stepErr)
		result.StepsRun = append(result.StepsRun, posemodel.OperationStep{ID: stepID, Result: "failed"})
		if wrote {
			result.Unconfirmed = append(result.Unconfirmed, posemodel.OperationUncertain{Step: stepID, Reason: "the step failed after it may have written; inspect it before retrying"})
		}
		result.Fail(posemodel.OutcomeFailed, stepID, cause, stepErr.Error())
		result.Resumable = true
		result.NextAction = "fix the cause, then " + resume
		return result
	}
	ran := false
	evidenceRan := false
	for guard := 0; guard < 12; guard++ {
		next, remaining := plan.NextCloseoutStep()
		if !remaining {
			break
		}
		if next.State == posemodel.CloseoutStateBlocked {
			record(next.ID, fmt.Errorf("%s", next.Reason))
			result.Fail(posemodel.OutcomeBlocked, next.ID, "step-blocked", next.Reason)
			result.Resumable = true
			result.NextAction = "resolve the blocker, then " + resume
			return result
		}
		if next.State == posemodel.CloseoutStateWaiting {
			result.Outcome, result.Step, result.Resumable = posemodel.OutcomeWaiting, next.ID, true
			for _, p := range plan.Pending {
				result.Pending = append(result.Pending, posemodel.OperationPending{ID: p.Criterion, Kind: p.Kind, Reason: p.Reason})
			}
			result.NextAction = "answer the pending criteria with `pose review attest " + plan.BundleID + " --reviewer <id> --decision approved --evidence <ref> --criterion ID|passed|<evidence>|<conclusion> ... --apply`, then " + resume
			data.BundleID = plan.BundleID
			// The reviewer should not have to discover the evidence refs and
			// required tools by trial and error (spec
			// pose-quickstart-real-lifecycle): the sealed bundle names them.
			data.EvidenceRefs, data.AttestCommand = closeoutAttestTemplate(store, plan)
			record(next.ID, fmt.Errorf("waiting on a reviewer"))
			return result
		}
		ran = true
		var stepErr error
		var buffer bytes.Buffer
		stepRef, cause, wrote := "", "step-failed", true
		switch next.ID {
		case posemodel.CloseoutStepEvidence, posemodel.CloseoutStepIndex:
			// Evidence and index run as a pair, in that order: results are
			// regenerated into results_path, then indexed, so the seal reads
			// this run and not a previous one. They run once per apply; if
			// the bundle still reports blockers afterwards, those are real.
			if evidenceRan {
				stepErr, cause, wrote = fmt.Errorf("evidence was regenerated and indexed, and the bundle still reports: %s", next.Reason), "evidence-insufficient", false
				break
			}
			evidenceRan = true
			if next.ID == posemodel.CloseoutStepEvidence && !skipEvidence {
				policy, policyErr := posemodel.LoadDeliveryPolicy(root)
				if policyErr != nil {
					return fail(posemodel.CloseoutStepEvidence, "policy-unavailable", policyErr, false)
				}
				code := cmdValidate(root, []string{"--tolerant", "--json-out", policy.ResultsPath}, &buffer, &buffer)
				if code > 1 {
					return fail(posemodel.CloseoutStepEvidence, "validate-error", fmt.Errorf("validate exited %d: %s", code, lastLine(buffer.String())), true)
				}
				// Exit 1 is both a failing check, whose results are published,
				// and a results file validate could not write. The effect is
				// confirmed only by observing the published document (R1/R6).
				if err := observePublishedResults(root, policy.ResultsPath, buffer.String()); err != nil {
					return fail(posemodel.CloseoutStepEvidence, "results-not-published", fmt.Errorf("validate exited %d and %v: %s", code, err, lastLine(buffer.String())), true)
				}
				record(posemodel.CloseoutStepEvidence, nil)
				confirm(posemodel.CloseoutStepEvidence, policy.ResultsPath)
			}
			buffer.Reset()
			if code := cmdIndex(root, nil, &buffer, &buffer); code != 0 {
				stepErr = fmt.Errorf("index exited %d: %s", code, lastLine(buffer.String()))
			}
			next.ID, stepRef = posemodel.CloseoutStepIndex, ".pose/indexes"
		case posemodel.CloseoutStepSeal:
			var bundle posemodel.ReviewBundle
			bundle, stepErr = store.SealReviewBundle(ref, time.Now())
			stepRef = bundle.BundleID
		case posemodel.CloseoutStepMechanical:
			if reviewer == "" {
				stepErr, cause, wrote = fmt.Errorf("--reviewer is required to record the mechanical attestation; it names the reviewing execution and is never invented"), "reviewer-required", false
				break
			}
			var attestation posemodel.ReviewAttestation
			attestation, stepErr = store.AutoAttestReviewBundle(plan.BundleID, reviewer, true, time.Now())
			stepRef = attestation.AttestationID
		case posemodel.CloseoutStepJudgment:
			// Pending only before sealing; after re-planning it is waiting or done.
			wrote = false
		case posemodel.CloseoutStepVerify:
			wrote = false
			verification, verifyErr := store.VerifyReviewBundle(ref)
			if verifyErr != nil {
				stepErr = verifyErr
			} else if !verification.Approved {
				stepErr, cause = fmt.Errorf("review is not approved: %s", strings.Join(verification.Blockers, "; ")), "review-not-approved"
			}
		case posemodel.CloseoutStepTransition:
			stepRef = ref
			if code := cmdCloseLocal(root, ref, nil, &buffer, &buffer); code != 0 {
				stepErr, cause = fmt.Errorf("guarded transition refused: %s", strings.TrimSpace(buffer.String())), "transition-refused"
			}
		}
		if stepErr != nil {
			return fail(next.ID, cause, stepErr, wrote)
		}
		record(next.ID, nil)
		replanned, replanErr := store.PlanCloseout(ref)
		if replanErr != nil {
			return fail(next.ID, "plan-unavailable", replanErr, true)
		}
		if after, ok := replanned.NextCloseoutStep(); ok && after.ID == next.ID && after.State == next.State && next.ID != posemodel.CloseoutStepJudgment && next.ID != posemodel.CloseoutStepIndex {
			data.Plan = &replanned
			return fail(next.ID, "no-effect", fmt.Errorf("the step ran but the repository does not reflect it: %s", after.Reason), true)
		}
		confirm(next.ID, stepRef)
		plan = replanned
		data.Plan = &plan
	}
	if _, remaining := plan.NextCloseoutStep(); remaining {
		result.Fail(posemodel.OutcomeFailed, "", "not-converged", "the plan did not converge")
		result.Resumable = true
		return result
	}
	store.ClearCloseoutCheckpoint(ref)
	if !ran {
		// Nothing ran: the stored state already says done. That is not a
		// fresh verification of the gates (R4).
		result.Outcome = posemodel.OutcomeAlreadySatisfied
		result.NextAction = "no step ran in this call; `pose closeout-check " + ref + "` verifies the current gates"
		return result
	}
	result.Outcome = posemodel.OutcomeCompleted
	result.NextAction = "commit the closeout (spec, index, results, bundle, attestation) in one commit with its POSE-Spec trailer"
	return result
}

// renderClosePlanResult is the human rendering of a close apply/resume
// result; it keeps the closeout_plan.* lines the quickstart reads.
func renderClosePlanResult(out *cliout.Renderer, result posemodel.OperationResult) {
	data, _ := result.Data.(*closeApplyData)
	for _, step := range result.StepsRun {
		if step.Result == "done" {
			out.Field("closeout_plan.step."+step.ID, "done")
		}
	}
	for _, uncertain := range result.Unconfirmed {
		out.Field("closeout_plan.unconfirmed."+uncertain.Step, uncertain.Reason)
	}
	message := ""
	if result.Error != nil {
		message = result.Error.Message
	}
	switch result.Outcome {
	case posemodel.OutcomeCompleted:
		out.Field("closeout_plan.result", "closed "+result.Scope+"; "+result.NextAction)
	case posemodel.OutcomeAlreadySatisfied:
		out.Field("closeout_plan.result", "already-satisfied "+result.Scope+"; "+result.NextAction)
	case posemodel.OutcomeWaiting:
		if data != nil && data.Plan != nil {
			renderCloseoutPlan(out, *data.Plan)
		}
		out.Field("closeout_plan.stopped", "waiting on a reviewer: "+result.NextAction)
		if data != nil && data.AttestCommand != "" {
			out.Field("closeout_plan.evidence", strings.Join(data.EvidenceRefs, ", "))
			out.Field("closeout_plan.attest", data.AttestCommand)
		}
	case posemodel.OutcomeBlocked:
		if data != nil && data.Plan != nil {
			renderCloseoutPlan(out, *data.Plan)
		}
		out.Failure("pose close: step " + result.Step + " is blocked: " + message)
	case posemodel.OutcomeConflict:
		out.Failure("pose close: " + message + "; " + result.NextAction)
	default:
		if result.Step == "" {
			out.Failure("pose close: " + message)
			return
		}
		out.Failure("pose close: step " + result.Step + " failed: " + message + " (" + result.NextAction + ")")
	}
}

func renderCloseoutPlan(out interface{ Field(string, string) }, plan posemodel.CloseoutPlan) {
	out.Field("closeout_plan.scope", plan.Scope)
	out.Field("closeout_plan.digest", plan.Digest)
	for _, st := range plan.Steps {
		line := st.State
		if st.Reason != "" {
			line += " — " + st.Reason
		}
		out.Field("closeout_plan.step."+st.ID, line)
	}
	for _, p := range plan.Pending {
		out.Field("closeout_plan.pending."+p.Criterion, p.Kind+": "+p.Reason)
	}
	for _, b := range plan.Blockers {
		out.Field("closeout_plan.blocker", b)
	}
}

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	return lines[len(lines)-1]
}

// closeoutAttestTemplate returns the sealed evidence refs and an attest
// command filled with them: one --tool per required tool the review owes,
// one --criterion per pending judgment, each with a conclusion left for the
// reviewer to write.
func closeoutAttestTemplate(store posemodel.Store, plan posemodel.CloseoutPlan) ([]string, string) {
	bundle, err := store.LoadReviewBundle(plan.BundleID)
	if err != nil {
		return nil, ""
	}
	refs := []string{}
	byClass := map[string]string{}
	for _, ev := range bundle.Payload.Evidence {
		ref := ev.EvidenceClass + ":" + ev.ID
		refs = append(refs, ref)
		if _, ok := byClass[ev.EvidenceClass]; !ok && ev.Outcome == "pass" {
			byClass[ev.EvidenceClass] = ref
		}
	}
	if len(refs) == 0 {
		return nil, ""
	}
	pick := func(classes []string) string {
		for _, class := range classes {
			if ref, ok := byClass[class]; ok {
				return ref
			}
		}
		for _, class := range []string{"unit", "integration", "e2e", "build"} {
			if ref, ok := byClass[class]; ok {
				return ref
			}
		}
		return refs[0]
	}
	primary := pick(nil)
	parts := []string{"pose review attest " + plan.BundleID, "--reviewer <id>", "--decision approved", "--evidence " + primary}
	for _, tool := range bundle.Payload.Plan.Tools {
		if tool.Requiredness != "required" || cliReviewToolHasPrecondition(tool, "review-complete") {
			continue
		}
		component := tool.Component
		if component == "" {
			component = "-"
		}
		disposition := "passed|check:" + tool.ID + "|"
		if len(tool.EvidenceClasses) > 0 {
			matched := ""
			for _, class := range tool.EvidenceClasses {
				if ref, ok := byClass[class]; ok {
					matched = ref
					break
				}
			}
			switch {
			case matched != "":
				disposition = "passed|" + matched + "|"
			case cliReviewToolHasPrecondition(tool, "delivery-target-declared"):
				// No sealed evidence of its class and no delivery target in
				// scope: the tool is deferred, never fed evidence of another
				// class it would refuse.
				disposition = "deferred||no delivery target in this scope"
			case tool.ProducerCoverage == "none":
				disposition = "not-used||no registered check produces " + strings.Join(tool.EvidenceClasses, "|") + " here"
			default:
				disposition = "passed|<" + strings.Join(tool.EvidenceClasses, "|") + " evidence>|"
			}
		}
		parts = append(parts, "--tool '"+tool.ID+"|"+component+"|"+disposition+"'")
	}
	pendingIDs := map[string]bool{}
	for _, pending := range plan.Pending {
		pendingIDs[pending.Criterion] = true
		parts = append(parts, "--criterion '"+pending.Criterion+"|passed|"+primary+"|<your conclusion>'")
	}
	for _, criterion := range bundle.Payload.Plan.Criteria {
		// A mechanical criterion with no sealed evidence of its class (a
		// configuration-only spec) is answered as not applicable, with the
		// reviewer's reason — never with evidence of another class.
		if !criterion.Required || pendingIDs[criterion.ID] || len(criterion.EvidenceClasses) == 0 {
			continue
		}
		covered := false
		for _, class := range criterion.EvidenceClasses {
			if _, ok := byClass[class]; ok {
				covered = true
				break
			}
		}
		if !covered {
			parts = append(parts, "--criterion '"+criterion.ID+"|not-applicable||<why no "+strings.Join(criterion.EvidenceClasses, "|")+" evidence applies>'")
		}
	}
	if structure := bundle.Payload.Plan.Structure; structure != nil {
		for _, criterion := range bundle.Payload.Plan.Criteria {
			if !criterion.RequiresStructuralMapping {
				continue
			}
			for _, fact := range structure.Material {
				parts = append(parts, "--mapping '"+criterion.ID+"|"+fact.ID+"|<decision that reaches a requirement, e.g. D1>|<why "+fact.Kind+" "+fact.Action+" "+fact.Subject+">'")
			}
		}
	}
	parts = append(parts, "--apply")
	return refs, strings.Join(parts, " ")
}
