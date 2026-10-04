package cli

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// cmdClosePlan is `pose close spec:<slug> --plan|--apply|--resume` (spec
// pose-recoverable-closeout-plan): the closeout as an ordered, recoverable
// plan. --plan previews; --apply runs every mechanical step in order and stops
// at the first one that needs a person (a judgment criterion) or fails;
// --resume is --apply after an interruption — the plan is recomputed from the
// repository, so done steps are not repeated.
func cmdClosePlan(root string, args []string, stdout, stderr io.Writer) int {
	const usage = "Usage: pose close spec:<slug> --plan [--json] | --apply|--resume [--digest <plan digest>] [--reviewer agent:<id>] [--skip-evidence]"
	ref, mode, digest, reviewer, jsonOutput, skipEvidence := "", "", "", "", false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--plan", "--apply", "--resume":
			mode = strings.TrimPrefix(args[i], "--")
		case "--json":
			jsonOutput = true
		case "--skip-evidence":
			skipEvidence = true
		case "--digest", "--reviewer":
			if i+1 >= len(args) {
				return usageError(stderr, usage)
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
	store := cliGovernedStore(root)
	out := render(stdout, stderr)
	plan, err := store.PlanCloseout(ref)
	if err != nil {
		out.Failure("pose close: " + err.Error())
		return 1
	}
	if mode == "plan" {
		if jsonOutput {
			return writeJSON(stdout, plan)
		}
		renderCloseoutPlan(out, plan)
		if checkpoint, ok := store.ReadCloseoutCheckpoint(ref); ok {
			out.Field("closeout_plan.interrupted", fmt.Sprintf("an apply started %s completed %s; --resume continues", checkpoint.StartedAt, strings.Join(checkpoint.Completed, ",")))
		}
		return 0
	}
	if digest != "" && digest != plan.Digest {
		out.Failure("pose close: the repository changed since the plan was computed (" + digest + " → " + plan.Digest + "); review the new plan with --plan")
		return 1
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
		_ = store.WriteCloseoutCheckpoint(checkpoint)
	}
	evidenceRan := false
	for guard := 0; guard < 12; guard++ {
		next, remaining := plan.NextCloseoutStep()
		if !remaining {
			break
		}
		if next.State == posemodel.CloseoutStateBlocked {
			renderCloseoutPlan(out, plan)
			out.Failure("pose close: step " + next.ID + " is blocked: " + next.Reason)
			record(next.ID, fmt.Errorf("%s", next.Reason))
			return 1
		}
		if next.State == posemodel.CloseoutStateWaiting {
			renderCloseoutPlan(out, plan)
			out.Field("closeout_plan.stopped", "waiting on a reviewer: answer the pending criteria with `pose review attest "+plan.BundleID+" --reviewer <id> --decision approved --evidence <ref> --criterion ID|passed|<evidence>|<conclusion> ... --apply`, then `pose close "+ref+" --resume`")
			record(next.ID, fmt.Errorf("waiting on a reviewer"))
			return 3
		}
		var stepErr error
		var buffer bytes.Buffer
		switch next.ID {
		case posemodel.CloseoutStepEvidence, posemodel.CloseoutStepIndex:
			// Evidence and index run as a pair, in that order: results are
			// regenerated into results_path, then indexed, so the seal reads
			// this run and not a previous one. They run once per apply; if
			// the bundle still reports blockers afterwards, those are real.
			if evidenceRan {
				stepErr = fmt.Errorf("evidence was regenerated and indexed, and the bundle still reports: %s", next.Reason)
				break
			}
			evidenceRan = true
			if next.ID == posemodel.CloseoutStepEvidence && !skipEvidence {
				policy, policyErr := posemodel.LoadDeliveryPolicy(root)
				if policyErr != nil {
					stepErr = policyErr
					break
				}
				if code := cmdValidate(root, []string{"--tolerant", "--json-out", policy.ResultsPath}, &buffer, &buffer); code > 1 {
					stepErr = fmt.Errorf("validate exited %d", code)
					break
				}
				record(posemodel.CloseoutStepEvidence, nil)
				out.Field("closeout_plan.step."+posemodel.CloseoutStepEvidence, "done")
			}
			buffer.Reset()
			if code := cmdIndex(root, nil, &buffer, &buffer); code != 0 {
				stepErr = fmt.Errorf("index exited %d: %s", code, lastLine(buffer.String()))
			}
			next.ID = posemodel.CloseoutStepIndex
		case posemodel.CloseoutStepSeal:
			_, stepErr = store.SealReviewBundle(ref, time.Now())
		case posemodel.CloseoutStepMechanical:
			if reviewer == "" {
				stepErr = fmt.Errorf("--reviewer is required to record the mechanical attestation; it names the reviewing execution and is never invented")
				break
			}
			_, stepErr = store.AutoAttestReviewBundle(plan.BundleID, reviewer, true, time.Now())
		case posemodel.CloseoutStepJudgment:
			// Pending only before sealing; after re-planning it is waiting or done.
		case posemodel.CloseoutStepVerify:
			verification, verifyErr := store.VerifyReviewBundle(ref)
			if verifyErr != nil {
				stepErr = verifyErr
			} else if !verification.Approved {
				stepErr = fmt.Errorf("review is not approved: %s", strings.Join(verification.Blockers, "; "))
			}
		case posemodel.CloseoutStepTransition:
			if code := cmdCloseLocal(root, ref, nil, &buffer, &buffer); code != 0 {
				stepErr = fmt.Errorf("guarded transition refused: %s", strings.TrimSpace(buffer.String()))
			}
		}
		if stepErr != nil {
			record(next.ID, stepErr)
			out.Failure("pose close: step " + next.ID + " failed: " + stepErr.Error() + " (fix the cause, then `pose close " + ref + " --resume`)")
			return 1
		}
		record(next.ID, nil)
		out.Field("closeout_plan.step."+next.ID, "done")
		replanned, replanErr := store.PlanCloseout(ref)
		if replanErr != nil {
			out.Failure("pose close: " + replanErr.Error())
			return 1
		}
		if after, ok := replanned.NextCloseoutStep(); ok && after.ID == next.ID && after.State == next.State && next.ID != posemodel.CloseoutStepJudgment && next.ID != posemodel.CloseoutStepIndex {
			record(next.ID, fmt.Errorf("the step ran but the repository does not reflect it"))
			out.Failure("pose close: step " + next.ID + " ran but did not take effect: " + after.Reason)
			return 1
		}
		plan = replanned
	}
	if _, remaining := plan.NextCloseoutStep(); remaining {
		out.Failure("pose close: the plan did not converge")
		return 1
	}
	store.ClearCloseoutCheckpoint(ref)
	out.Field("closeout_plan.result", "closed "+ref+"; commit the closeout (spec, index, results, bundle, attestation) in one commit with its POSE-Spec trailer")
	return 0
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
