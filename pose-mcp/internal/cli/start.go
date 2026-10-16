package cli

// pose start — atomic start (spec pose-abm-atomic-start). Preview is the
// default and writes nothing; --apply needs the atomic_start_version
// capability and the digest of the reviewed preview.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const startUsage = "Usage: pose start spec:<slug> [--json] | --apply --digest <sha256> [--json] | --status [--json] | --cancel | --recover-lock <owner digest> [--json]"

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
	apply, status, cancel, jsonOutput, digest, recoverLock, recovering := false, false, false, false, "", "", false
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
		case "--recover-lock":
			if i+1 >= len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "-") {
				return usage("--recover-lock needs the owner digest `pose start spec:<slug> --status` shows")
			}
			i++
			recoverLock, recovering = args[i], true
		default:
			return usage("unknown option: " + args[i])
		}
	}
	// Modes are exclusive and each flag belongs to one of them: a second mode
	// or a flag the mode does not take is refused, never resolved by
	// priority or order (spec pose-cli-mcp-operation-results R3).
	modes := 0
	for _, on := range []bool{apply, status, cancel, recovering} {
		if on {
			modes++
		}
	}
	switch {
	case modes > 1:
		return usage("--apply, --status, --cancel and --recover-lock are exclusive modes")
	case digest != "" && !apply:
		return usage("--digest is taken only by --apply")
	case cancel && jsonOutput:
		return usage("--cancel has no JSON output")
	}
	store := cliGovernedStore(root)
	switch {
	case recovering:
		return renderStartLockRecovery(r, stdout, jsonOutput, slug, recoverLock, store)
	case cancel:
		if err := store.CancelStart(slug); err != nil {
			var lockErr *posemodel.StartLockError
			if errors.As(err, &lockErr) {
				renderStartLock(r, slug, lockErr)
			}
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
		if st.Lock != nil {
			renderStartLock(r, slug, st.Lock)
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
				attempt := beginStartAttempt(store, slug, digest)
				resumed, err := store.ResumeStart(slug, digest, attempt.observe)
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
					return renderStartApplyFailure(r, stdout, jsonOutput, startApplyFailure(store, plan.SpecPath, slug, digest, attempt, resumed, err), err)
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
		attempt := beginStartAttempt(store, slug, digest)
		record, err := store.ApplyStart(plan, digest, attempt.observe)
		if err != nil {
			return renderStartApplyFailure(r, stdout, jsonOutput, startApplyFailure(store, plan.SpecPath, slug, digest, attempt, record, err), err)
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
	for _, step := range result.StepsRun {
		r.Field("start.step."+step.ID, step.Result)
	}
	for _, effect := range result.Effects {
		r.Field("start."+effect.Effect, effect.Ref)
	}
	for _, uncertain := range result.Unconfirmed {
		r.Field("start.unconfirmed."+uncertain.Step, uncertain.Reason)
	}
	if lock, ok := result.Data.(*posemodel.StartLockError); ok {
		renderStartLock(r, strings.TrimPrefix(result.Scope, "spec:"), lock)
	} else if result.NextAction != "" {
		r.Field("start.next_action", result.NextAction)
	}
	r.Failure("pose start: " + err.Error())
	return 1
}

// startAttempt is what an apply or a resume found before it ran — the
// recovery state of its digest and whether a start record file existed — and
// the phases it reached, observed without interrupting it.
type startAttempt struct {
	prior         string
	recordExisted bool
	reached       []string
}

func beginStartAttempt(store posemodel.Store, slug, digest string) *startAttempt {
	prior, _ := store.StartRecovery(slug, digest)
	_, err := os.Lstat(startRecordFile(store, slug))
	return &startAttempt{prior: prior, recordExisted: err == nil}
}

func (a *startAttempt) observe(phase string) error {
	a.reached = append(a.reached, phase)
	return nil
}

func startRecordFile(store posemodel.Store, slug string) string {
	return filepath.Join(store.Root, ".pose", "starts", slug+".json")
}

// startApplyFailure is the result of an apply that returned an error. The
// domain may have published the baseline and the spec transition before it
// failed, so the record and the spec on disk are read back: each published
// state is a confirmed effect, and only the step that did not finish is left
// unconfirmed (spec pose-cli-mcp-operation-results R6). steps_run lists only
// what this call executed, from the recovery state before the call (prior)
// and the phases it reached; an effect whose step is not in steps_run was
// published by an earlier call.
func startApplyFailure(store posemodel.Store, specPath, slug, digest string, attempt *startAttempt, returned posemodel.StartRecord, err error) posemodel.OperationResult {
	result := posemodel.NewOperationResult("start.apply", "spec:"+slug)
	var lockErr *posemodel.StartLockError
	if errors.As(err, &lockErr) {
		// Another call holds the start lock: nothing ran in this one.
		outcome := posemodel.OutcomeBlocked
		if lockErr.State == posemodel.LockOwnerDead {
			result.Resumable = true
		}
		result.Fail(outcome, "lock", lockErr.Code, lockErr.Path+": "+lockErr.Reason)
		result.Data = lockErr
		result.NextAction = startLockNextAction(slug, lockErr)
		return result
	}
	// The domain's refusals are "start: <code>"; anything else is an
	// operational error of the apply.
	cause := "apply-failed"
	if code := strings.TrimPrefix(err.Error(), "start: "); code != err.Error() && !strings.ContainsAny(code, " :") {
		cause = code
	}
	retry := "`pose start spec:" + slug + " --apply --digest " + digest + "`"
	reconcile := "`pose start spec:" + slug + " --status` shows the record; `pose start spec:" + slug + " --cancel`, a new preview and its digest start again"
	persisted, _, readErr := store.GetStartRecord(slug)
	if _, statErr := os.Lstat(startRecordFile(store, slug)); readErr != nil && statErr == nil {
		// A record file that does not read as a start record: an exclusive
		// baseline write this call began and did not finish (a short write
		// leaves part of it), or one an earlier call left. It is no
		// confirmed baseline, and no step can repair it automatically.
		recordRef := ".pose/starts/" + slug + ".json"
		if attempt.recordExisted {
			result.Fail(posemodel.OutcomeFailed, "", cause, err.Error())
		} else {
			result.Fail(posemodel.OutcomeFailed, "baseline", cause, err.Error())
			result.StepsRun = append(result.StepsRun, posemodel.OperationStep{ID: "baseline", Result: "failed"})
			result.Unconfirmed = append(result.Unconfirmed, posemodel.OperationUncertain{Step: "baseline", Reason: "the start record " + recordRef + " was created but does not hold a complete record; the write may be partial"})
		}
		result.NextAction = recordRef + " is not a valid start record and the spec was not moved from it; inspect it, remove it once no start is running, then a new `pose start spec:" + slug + "` preview and its digest start again"
		return result
	}
	recovery, _ := store.StartRecovery(slug, digest)
	result.StepsRun = startStepsRun(attempt.prior, recovery, attempt.reached)
	recorded := func() {
		result.Effects = append(result.Effects, posemodel.OperationEffect{Step: "baseline", Effect: persisted.Phase, Ref: ".pose/starts/" + slug + ".json"})
		result.Data = persisted
	}
	switch recovery {
	case posemodel.StartRecoveryPublished:
		result.Fail(posemodel.OutcomeFailed, "record", cause, err.Error())
		recorded()
		result.Effects = append(result.Effects, posemodel.OperationEffect{Step: "transition", Effect: "spec-in-progress", Ref: specPath})
		result.Unconfirmed = append(result.Unconfirmed, posemodel.OperationUncertain{Step: "record", Reason: "the spec is in-progress but the start record does not say started yet"})
		result.Resumable = true
		result.NextAction = "fix the cause, then repeat " + retry + "; it finishes from the start record"
	case posemodel.StartRecoveryRetry:
		result.Fail(posemodel.OutcomeFailed, "transition", cause, err.Error())
		recorded()
		result.Unconfirmed = append(result.Unconfirmed, posemodel.OperationUncertain{Step: "transition", Reason: "the spec transition did not complete; the plan still matches the repository"})
		result.Resumable = true
		result.NextAction = "fix the cause, then repeat " + retry + "; the recorded baseline is reused"
	case posemodel.StartRecoveryStale:
		// Nothing was published and the repository moved on: the recorded
		// digest can never apply again.
		result.Fail(posemodel.OutcomeConflict, "transition", "stale-start-baseline", err.Error())
		recorded()
		result.NextAction = "the recorded baseline no longer matches the repository and its digest cannot apply; " + reconcile
	case posemodel.StartRecoveryConflict:
		result.Fail(posemodel.OutcomeConflict, "transition", "compare-and-swap-conflict", err.Error())
		recorded()
		result.NextAction = "the spec changed after the baseline was recorded; " + reconcile
	default:
		if returned.Phase != "" {
			// A record for another plan: nothing was written by this call.
			result.Fail(posemodel.OutcomeConflict, "", cause, err.Error())
			result.Data = returned
			result.NextAction = "the spec was started with another plan; " + reconcile
			break
		}
		result.Fail(posemodel.OutcomeFailed, "", cause, err.Error())
	}
	return result
}

// startStepsRun is what one apply or resume executed, in order. The baseline
// ran in this call when no record existed for the digest before it and one
// does now. Reaching "baseline-recorded" means the call went on to the
// transition; reaching "transitioned" means the spec is in-progress, which
// this call published unless it was already published before (prior), and
// that it went on to the record. The record step only ever fails here: a
// call that finished it returned no error.
func startStepsRun(prior, recovery string, reached []string) []posemodel.OperationStep {
	steps := []posemodel.OperationStep{}
	if prior == posemodel.StartRecoveryNone && recovery != posemodel.StartRecoveryNone {
		steps = append(steps, posemodel.OperationStep{ID: "baseline", Result: "done"})
	}
	reachedPhase := func(phase string) bool {
		for _, p := range reached {
			if p == phase {
				return true
			}
		}
		return false
	}
	switch {
	case reachedPhase("transitioned"):
		if prior != posemodel.StartRecoveryPublished {
			steps = append(steps, posemodel.OperationStep{ID: "transition", Result: "done"})
		}
		steps = append(steps, posemodel.OperationStep{ID: "record", Result: "failed"})
	case reachedPhase("baseline-recorded"):
		steps = append(steps, posemodel.OperationStep{ID: "transition", Result: "failed"})
	}
	return steps
}

// startLockNextAction is the supported next step for a start lock another
// call holds: wait for a running owner, check an unknown one by hand, and
// remove a dead one only by naming the lock that was inspected (spec
// pose-journal-checkpoint-and-lock-recovery R6/R7/R9).
func startLockNextAction(slug string, lock *posemodel.StartLockError) string {
	switch lock.State {
	case posemodel.LockOwnerActive:
		return "another start is running; wait for it to finish, then repeat this command"
	case posemodel.LockOwnerDead:
		return "the lock's owner is not running; `pose start spec:" + slug + " --recover-lock " + lock.OwnerDigest + "` removes this lock, then repeat this command"
	default:
		return "the lock's owner cannot be checked from here; confirm no start is running, then remove " + lock.Path + " by hand and repeat this command"
	}
}

func renderStartLock(r *cliout.Renderer, slug string, lock *posemodel.StartLockError) {
	r.Field("start.lock", lock.State+" "+lock.Path+": "+lock.Reason)
	if lock.OwnerDigest != "" {
		r.Field("start.lock.owner_digest", lock.OwnerDigest)
	}
	r.Field("start.lock.next_action", startLockNextAction(slug, lock))
}

// renderStartLockRecovery removes a start lock whose owner is proven dead,
// naming the lock the caller inspected.
func renderStartLockRecovery(r *cliout.Renderer, stdout io.Writer, jsonOutput bool, slug, ownerDigest string, store posemodel.Store) int {
	result := posemodel.NewOperationResult("start.recover-lock", "spec:"+slug)
	lock, err := store.RecoverStartLock(slug, ownerDigest)
	if lock != nil {
		result.Data = lock
	}
	if err == nil {
		result.Outcome = posemodel.OutcomeCompleted
		result.StepsRun = append(result.StepsRun, posemodel.OperationStep{ID: "recover-lock", Result: "done"})
		result.Effects = append(result.Effects, posemodel.OperationEffect{Step: "recover-lock", Effect: "lock-removed", Ref: lock.Path})
		result.NextAction = "repeat the start command that found the lock; the start record and the spec were not touched"
	} else {
		code := strings.TrimPrefix(err.Error(), "start: ")
		switch code {
		case "start-lock-changed":
			result.Fail(posemodel.OutcomeConflict, "recover-lock", code, "the lock changed since its owner digest was read")
			result.NextAction = "`pose start spec:" + slug + " --status` shows the lock as it is now"
		case "start-lock-owner-not-dead", "start-lock-absent":
			result.Fail(posemodel.OutcomeBlocked, "recover-lock", code, err.Error())
			if lock != nil {
				result.NextAction = startLockNextAction(slug, lock)
			}
		default:
			result.Fail(posemodel.OutcomeFailed, "recover-lock", "recovery-failed", err.Error())
		}
	}
	if jsonOutput {
		if writeJSON(stdout, result) != 0 {
			return 1
		}
	} else {
		if err == nil {
			r.Field("start.lock_recovered", lock.Path+" ("+lock.Reason+")")
			r.Field("start.next_action", result.NextAction)
		} else {
			if lock != nil {
				renderStartLock(r, slug, lock)
			}
			r.Failure("pose start: " + err.Error())
		}
	}
	if err != nil {
		return 1
	}
	return 0
}
