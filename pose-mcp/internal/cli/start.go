package cli

// pose start — atomic start (spec pose-abm-atomic-start). Preview is the
// default and writes nothing; --apply needs the atomic_start_version
// capability and the digest of the reviewed preview.

import (
	"fmt"
	"io"
	"strings"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const startUsage = "Usage: pose start spec:<slug> [--json] | --apply --digest <sha256> [--json] | --status [--json] | --cancel"

func cmdStart(root string, args []string, stdout, stderr io.Writer) int {
	r := render(stdout, stderr)
	usage := func(reason string) int {
		r.Usage(startUsage)
		if reason != "" {
			r.Failure(reason)
		}
		return 2
	}
	if len(args) == 0 || !strings.HasPrefix(args[0], "spec:") {
		return usage("a spec:<slug> scope is required")
	}
	slug := strings.TrimPrefix(args[0], "spec:")
	if posemodel.ValidateSlug(slug) != nil {
		return usage("invalid spec slug")
	}
	apply, status, cancel, jsonOutput, digest := false, false, false, false, ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--apply":
			apply = true
		case "--status":
			status = true
		case "--cancel":
			cancel = true
		case "--json":
			jsonOutput = true
		case "--digest":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return usage("--digest needs a value")
			}
			i++
			digest = args[i]
		default:
			return usage("unknown option: " + args[i])
		}
	}
	// Modes are exclusive and each flag belongs to one of them: a second mode
	// or a flag the mode does not take is refused, never resolved by
	// priority or order (spec pose-cli-mcp-operation-results R3).
	modes := 0
	for _, on := range []bool{apply, status, cancel} {
		if on {
			modes++
		}
	}
	switch {
	case modes > 1:
		return usage("--apply, --status and --cancel are exclusive modes")
	case digest != "" && !apply:
		return usage("--digest is taken only by --apply")
	case cancel && jsonOutput:
		return usage("--cancel has no JSON output")
	}
	store := cliGovernedStore(root)
	switch {
	case cancel:
		if err := store.CancelStart(slug); err != nil {
			r.Failure("pose start: " + err.Error())
			return 1
		}
		r.Field("start.cancelled", "spec:"+slug)
		return 0
	case status:
		st, err := store.GetStartStatus(slug)
		if err != nil && jsonOutput {
			return writeOperationFailure(stdout, stderr, "start.status", "spec:"+slug, "status-unavailable", "pose start: "+err.Error())
		}
		if err != nil {
			r.Failure("pose start: " + err.Error())
			return 1
		}
		if jsonOutput {
			return writeJSON(stdout, st)
		}
		r.Field("start.phase", st.Phase)
		r.Field("start.status", st.Status)
		for _, node := range st.Nodes {
			r.Field("start.node."+node.ID, node.Origin)
		}
		for _, id := range st.Removed {
			r.Field("start.removed", id)
		}
		r.Field("start.obligations", strings.Join(st.Obligations, ","))
		for _, item := range st.Reconciliation {
			r.Field("start.reconcile", item)
		}
		return 0
	case apply:
		if digest == "" {
			return usage("--apply needs --digest from a reviewed preview")
		}
		plan, err := store.PreviewStart(slug)
		if err != nil {
			if jsonOutput {
				return writeOperationFailure(stdout, stderr, "start.apply", "spec:"+slug, "preview-unavailable", "pose start: "+err.Error())
			}
			r.Failure("pose start: " + err.Error())
			return 1
		}
		// Repeating a completed start is a no-op. An interrupted one still
		// previews the same plan, because the spec has not moved yet.
		if plan.Digest != digest {
			// An apply that recorded its baseline for this digest and already
			// moved the spec is finished from its record. When the spec has not
			// moved, nothing was published and the current plan judges the
			// start below.
			if record, ok, err := store.GetStartRecord(slug); err == nil && ok && record.PlanDigest == digest && record.Phase == "baseline-recorded" {
				resumed, err := store.ResumeStart(slug, digest)
				switch {
				case err == nil:
					if jsonOutput {
						return writeJSON(stdout, resumed)
					}
					r.Field("start.result", "resumed")
					r.Field("start.phase", resumed.Phase)
					r.Field("start.baseline", resumed.Baseline.Digest)
					return 0
				case err.Error() != "start: transition-not-published":
					return renderStartApplyFailure(r, stdout, jsonOutput, startApplyFailure(store, slug, digest, resumed, err), err)
				}
			}
			if record, ok, err := store.GetStartRecord(slug); err == nil && ok && record.PlanDigest == digest && record.Phase == "started" {
				if jsonOutput {
					return writeJSON(stdout, record)
				}
				r.Field("start.result", "already-started")
				r.Field("start.phase", record.Phase)
				r.Field("start.baseline", record.Baseline.Digest)
				return 0
			}
		}
		record, err := store.ApplyStart(plan, digest, nil)
		if err != nil {
			return renderStartApplyFailure(r, stdout, jsonOutput, startApplyFailure(store, slug, digest, record, err), err)
		}
		if jsonOutput {
			return writeJSON(stdout, record)
		}
		r.Field("start.phase", record.Phase)
		r.Field("start.baseline", record.Baseline.Digest)
		return 0
	default:
		plan, err := store.PreviewStart(slug)
		if err != nil {
			if jsonOutput {
				return writeOperationFailure(stdout, stderr, "start.preview", "spec:"+slug, "preview-unavailable", "pose start: "+err.Error())
			}
			r.Failure("pose start: " + err.Error())
			return 1
		}
		if jsonOutput {
			return writeJSON(stdout, plan)
		}
		r.Field("start.ready", fmt.Sprint(plan.Ready))
		if plan.Reason != "" {
			r.Field("start.reason", plan.Reason)
		}
		for _, waiting := range plan.WaitingOn {
			r.Field("start.waiting_on", waiting.Ref+": "+waiting.Reason)
		}
		r.Field("start.obligations", strings.Join(plan.Obligations, ","))
		r.Field("start.baseline", plan.Baseline.Digest)
		r.Field("start.digest", plan.Digest)
		return 0
	}
}

// renderStartApplyFailure renders one apply failure for both channels from
// the same result: the JSON document, or the recorded effects, the
// unconfirmed steps and the next action as fields, then the error.
func renderStartApplyFailure(r *cliout.Renderer, stdout io.Writer, jsonOutput bool, result posemodel.OperationResult, err error) int {
	if jsonOutput {
		r.Failure("pose start: " + err.Error())
		_ = writeJSON(stdout, result)
		return 1
	}
	for _, effect := range result.Effects {
		r.Field("start."+effect.Effect, effect.Ref)
	}
	for _, uncertain := range result.Unconfirmed {
		r.Field("start.unconfirmed."+uncertain.Step, uncertain.Reason)
	}
	if result.NextAction != "" {
		r.Field("start.next_action", result.NextAction)
	}
	r.Failure("pose start: " + err.Error())
	return 1
}

// startApplyFailure is the result of an apply that returned an error. The
// domain may have published the baseline before it failed, so the record on
// disk is read back: a baseline recorded for this plan is a confirmed,
// resumable state, and the spec transition is left unconfirmed (spec
// pose-cli-mcp-operation-results R6).
func startApplyFailure(store posemodel.Store, slug, digest string, returned posemodel.StartRecord, err error) posemodel.OperationResult {
	result := posemodel.NewOperationResult("start.apply", "spec:"+slug)
	// The domain's refusals are "start: <code>"; anything else is an
	// operational error of the apply.
	cause := "apply-failed"
	if code := strings.TrimPrefix(err.Error(), "start: "); code != err.Error() && !strings.ContainsAny(code, " :") {
		cause = code
	}
	persisted, ok, readErr := store.GetStartRecord(slug)
	switch {
	case readErr == nil && ok && persisted.PlanDigest == digest && persisted.Phase != "":
		result.Fail(posemodel.OutcomeFailed, "transition", cause, err.Error())
		result.Effects = append(result.Effects, posemodel.OperationEffect{Step: "baseline", Effect: persisted.Phase, Ref: ".pose/starts/" + slug + ".json"})
		if persisted.Phase != "started" {
			result.Unconfirmed = append(result.Unconfirmed, posemodel.OperationUncertain{Step: "transition", Reason: "the spec transition may have been written before the failure; the start record reconciles it on the next apply"})
		}
		if cause == "compare-and-swap-conflict" {
			// The spec changed under the start: repeating it cannot finish.
			result.NextAction = "the spec no longer matches the recorded baseline; reconcile it with `pose start spec:" + slug + " --status`, or `pose start spec:" + slug + " --cancel` and preview again"
			break
		}
		result.Resumable = true
		result.NextAction = "fix the cause, then repeat `pose start spec:" + slug + " --apply --digest " + digest + "`; the recorded baseline is reused and an already moved spec is reconciled"
		result.Data = persisted
	case returned.Phase != "":
		// A record for another plan: nothing was written by this call.
		result.Fail(posemodel.OutcomeConflict, "", cause, err.Error())
		result.Data = returned
	default:
		result.Fail(posemodel.OutcomeFailed, "", cause, err.Error())
	}
	return result
}
