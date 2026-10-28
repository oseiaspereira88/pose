package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const reviewRequestUsage = "Usage: pose review request <scope|bundle-id> --kind delegated-review [--purpose review|adjudicate|smoke] [--agents <name,...>] [--note-file <path>] " +
	"[--requested-by <principal>] [--target <kind:ID|self>]... [--effect <phase>:<block|advisory>]... [--question <text>] [--apply] [--json]"

// cmdReviewRequest opens a delegated-review action request (spec
// pose-delegated-review-action-fulfillment R1, R4): the material is read
// from the current bundle, its brief and the policies, and a scope whose plan
// reserves the review for a person is asked of a person instead (R7).
func cmdReviewRequest(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return usageError(stderr, reviewRequestUsage)
	}
	target := args[0]
	kind, purpose, agents, noteFile, requestedBy, question := "", posemodel.DelegationPurposeReview, []string(nil), "", "agent:pose-cli", ""
	var targets []posemodel.NodeRef
	var effects []posemodel.ObligationEffect
	apply, jsonOutput := false, false
	for i := 1; i < len(args); i++ {
		flag := args[i]
		switch flag {
		case "--apply":
			apply = true
			continue
		case "--json":
			jsonOutput = true
			continue
		}
		if i+1 >= len(args) {
			return usageError(stderr, reviewRequestUsage)
		}
		i++
		value := args[i]
		switch flag {
		case "--kind":
			kind = value
		case "--purpose":
			purpose = value
		case "--agents":
			for _, a := range strings.Split(value, ",") {
				if a = strings.TrimSpace(a); a != "" {
					agents = append(agents, a)
				}
			}
		case "--note-file":
			noteFile = value
		case "--requested-by":
			requestedBy = value
		case "--question":
			question = value
		case "--target":
			ref, err := posemodel.ParseActionTarget(value)
			if err != nil {
				return usageError(stderr, "pose review request: "+err.Error())
			}
			targets = append(targets, ref)
		case "--effect":
			phase, mode, ok := strings.Cut(value, ":")
			if !ok {
				return usageError(stderr, reviewRequestUsage)
			}
			effects = append(effects, posemodel.ObligationEffect{Phase: phase, Mode: mode})
		default:
			return usageError(stderr, reviewRequestUsage)
		}
	}
	if kind != posemodel.ActionDelegatedReview {
		return usageError(stderr, "pose review request: --kind delegated-review is the kind this command opens")
	}
	notes := ""
	if noteFile != "" {
		raw, err := os.ReadFile(noteFile)
		if err != nil {
			return usageError(stderr, "pose review request: "+err.Error())
		}
		notes = string(raw)
	}
	store := posemodel.Store{Root: root}
	out := render(stdout, stderr)
	material, err := store.PrepareDelegatedReviewMaterial(target, purpose, notes, agents)
	if err != nil {
		out.Failure("pose review request: " + err.Error())
		return 1
	}
	scope, err := posemodel.ParseScopeRef(material.Scope)
	if err != nil || scope.Kind != "spec" {
		out.Failure("pose review request: a delegated-review request is opened on a spec scope")
		return 1
	}
	if len(targets) == 0 {
		targets = []posemodel.NodeRef{{}}
	}
	if len(effects) == 0 {
		effects = []posemodel.ObligationEffect{{Phase: posemodel.PhaseCloseout, Mode: posemodel.EffectBlock}}
	}
	if question == "" {
		question = fmt.Sprintf("Have an accepted agent run the %s of %s from its sealed bundle %s, and record its conclusion", purpose, material.Scope, material.BundleID)
	}
	r := posemodel.ActionRequest{Origin: material.Scope, Kind: posemodel.ActionDelegatedReview, Question: question, RequestedBy: posemodel.ActionPrincipal{Principal: requestedBy},
		Targets: targets, Effects: effects, Review: &material}
	now := time.Now()
	if store.PersonRequiredByPlan(material) {
		// The plan reserves this review for a person: no delegated run
		// satisfies it, so the person is asked instead (R7).
		if !apply {
			if jsonOutput {
				return writeJSON(stdout, map[string]any{"person_required": true, "scope": material.Scope, "bundle_id": material.BundleID, "apply": false})
			}
			out.Field("review_request.person_required", "true")
			out.Field("review_request.apply", "false (rerun with --apply to ask the maintainer)")
			return 0
		}
		prepared, err := store.PrepareActionRequest(r, now)
		if err != nil {
			out.Failure("pose review request: " + err.Error())
			return 1
		}
		id, err := store.EscalateDelegatedReview(prepared, posemodel.EscalationPersonRequired, "The plan of "+material.Scope+" reserves its review for a person: who reviews it?",
			"Bundle "+material.BundleID+" requires mandatory-human review; a delegated run cannot satisfy it.", nil, now)
		if err != nil {
			out.Failure("pose review request: " + err.Error())
			return 1
		}
		if jsonOutput {
			return writeJSON(stdout, map[string]any{"person_required": true, "scope": material.Scope, "bundle_id": material.BundleID, "escalation": id})
		}
		out.Field("review_request.person_required", "true")
		out.Field("review_request.escalation", id)
		return 0
	}
	if !apply {
		preflight, err := store.PreflightActionOpen(r, nil, now)
		if err != nil {
			out.Failure("pose review request: " + err.Error())
			return 1
		}
		if jsonOutput {
			return writeJSON(stdout, preflight.Request)
		}
		out.Field("review_request.preview", "true")
		out.Field("review_request.id", preflight.Request.ID)
		out.Field("review_request.request_digest", preflight.Request.RequestDigest)
		out.Field("review_request.brief_digest", material.BriefDigest)
		out.Field("review_request.destination", material.Destination)
		out.Field("review_request.apply", "false (rerun with --apply to open it)")
		return 0
	}
	view, err := store.OpenActionRequest(r, now)
	if err != nil {
		out.Failure("pose review request: " + err.Error())
		return 1
	}
	if jsonOutput {
		return writeJSON(stdout, view)
	}
	renderActionView(out, view)
	return 0
}

const actionFulfilUsage = "Usage: pose action fulfil <act-id> --run <run-id> --runner <agent:principal> --idempotency-key <key> --expected-revision <n> [--request-digest <sha256>] " +
	"[--channel cli|ci|conductor] [--envelope <file>] [--apply] [--json]"

// cmdActionFulfil submits a fulfilment envelope (spec
// pose-delegated-review-action-fulfillment R2, R5, R9): a local runner, a CI
// job and Harne8's Conductor submit the same envelope, verified by the same
// engine.
func cmdActionFulfil(root string, args []string, stdout, stderr io.Writer) int {
	var env posemodel.ActionFulfillmentEnvelope
	env.Channel = "cli"
	apply, jsonOutput, envelopeFile := false, false, ""
	for i := 0; i < len(args); i++ {
		flag := args[i]
		switch flag {
		case "--apply":
			apply = true
			continue
		case "--json":
			jsonOutput = true
			continue
		}
		if !strings.HasPrefix(flag, "-") && env.RequestID == "" {
			env.RequestID = flag
			continue
		}
		if i+1 >= len(args) {
			return usageError(stderr, actionFulfilUsage)
		}
		i++
		value := args[i]
		switch flag {
		case "--run":
			env.RunID = value
		case "--runner":
			env.Runner = value
		case "--idempotency-key":
			env.IdempotencyKey = value
		case "--expected-revision":
			n, err := strconv.Atoi(value)
			if err != nil {
				return usageError(stderr, actionFulfilUsage)
			}
			env.ExpectedRevision = n
		case "--request-digest":
			env.RequestDigest = value
		case "--channel":
			env.Channel = value
		case "--envelope":
			envelopeFile = value
		default:
			return usageError(stderr, actionFulfilUsage)
		}
	}
	if envelopeFile != "" {
		raw, err := os.ReadFile(envelopeFile)
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err != nil || dec.Decode(&env) != nil {
			return usageError(stderr, "pose action fulfil: the envelope is not a fulfilment envelope")
		}
	}
	store := posemodel.Store{Root: root}
	out := render(stdout, stderr)
	if env.RequestID == "" || env.RunID == "" {
		return usageError(stderr, actionFulfilUsage)
	}
	if env.RequestDigest == "" {
		// The digest the runner read; without it the current one is named,
		// which a revision check still binds.
		view, err := store.LoadActionRequest(env.RequestID)
		if err != nil {
			out.Failure("pose action fulfil: " + err.Error())
			return 1
		}
		env.RequestDigest = view.Request.RequestDigest
	}
	now := time.Now()
	if !apply {
		_, event, err := store.PreflightFulfillment(env, now)
		if err != nil {
			out.Failure("pose action fulfil: " + err.Error())
			return 1
		}
		if jsonOutput {
			return writeJSON(stdout, event)
		}
		out.Field("action.fulfil.preview", "true")
		out.Field("action.fulfil.run", event.Fulfillment.RunID)
		out.Field("action.fulfil.decision", event.Fulfillment.Decision)
		out.Field("action.fulfil.recorded", event.Fulfillment.Recorded)
		out.Field("action.fulfil.apply", "false (rerun with --apply to record it)")
		return 0
	}
	view, err := store.FulfilActionRequest(env, now)
	if err != nil {
		out.Failure("pose action fulfil: " + err.Error())
		return 1
	}
	if jsonOutput {
		return writeJSON(stdout, view)
	}
	renderActionView(out, view)
	return 0
}

// delegateRequestFor reads the delegated-review request a run is asked for
// and checks the run would be the one it asks: open, of this purpose, scope
// and notes, with an agent it accepts — which it names when it accepts one.
func delegateRequestFor(store posemodel.Store, f *delegateFlags, purpose, ref, notes string) (posemodel.ActionRequestView, error) {
	view, err := store.LoadActionRequest(f.request)
	if err != nil {
		return view, err
	}
	r := view.Request
	switch {
	case r.Kind != posemodel.ActionDelegatedReview || r.Review == nil:
		return view, fmt.Errorf("pose: %s is a %s request, not delegated-review", r.ID, r.Kind)
	case view.State != posemodel.ActionStateOpen:
		return view, fmt.Errorf("pose: request %s is %s", r.ID, view.State)
	case r.Review.Purpose != purpose:
		return view, fmt.Errorf("pose: request %s asks for %s, not %s", r.ID, r.Review.Purpose, purpose)
	case ref != r.Review.Scope && ref != r.Review.BundleID:
		return view, fmt.Errorf("pose: request %s is for %s", r.ID, r.Review.Scope)
	case posemodel.NotesDigest(notes) != r.Review.NotesDigest:
		return view, fmt.Errorf("pose: request %s was asked with other notes; pass the same with --note-file", r.ID)
	}
	if agents := r.Review.Agents; len(agents) > 0 {
		switch {
		case (f.agent == "" || f.agent == "auto") && len(agents) == 1:
			f.agent = agents[0]
		case f.agent == "" || f.agent == "auto":
			return view, fmt.Errorf("pose: request %s accepts %s; name one with --agent", r.ID, strings.Join(agents, ", "))
		case !containsString(agents, f.agent):
			return view, fmt.Errorf("pose: request %s does not accept agent %s (%s)", r.ID, f.agent, strings.Join(agents, ", "))
		}
	}
	return view, nil
}

// delegateFulfilRequest submits the run that completed as the fulfilment of
// the request it was asked for, once the engine recorded its conclusion.
func delegateFulfilRequest(store posemodel.Store, request posemodel.ActionRequestView, run posemodel.ReviewRun, err error, stdout, stderr io.Writer) {
	out := render(stdout, stderr)
	if err != nil || run.Status != posemodel.ReviewRunCompleted {
		return
	}
	if c, _ := store.ReadReviewConclusion(run.RunID); c == nil || c.State != posemodel.ReviewConclusionRecorded {
		out.Hint("pose delegate: request " + request.Request.ID + " is not fulfilled: the engine recorded no conclusion of " + run.RunID +
			"; conclude it with pose review conclude " + run.RunID + " --issuer <name> --apply, then pose action fulfil " + request.Request.ID + " --run " + run.RunID)
		return
	}
	current, lerr := store.LoadActionRequest(request.Request.ID)
	if lerr != nil {
		out.Hint("pose delegate: request " + request.Request.ID + " is not fulfilled: " + lerr.Error())
		return
	}
	view, ferr := store.FulfilActionRequest(posemodel.ActionFulfillmentEnvelope{RequestID: current.Request.ID, RequestDigest: current.Request.RequestDigest,
		ExpectedRevision: current.Revision, IdempotencyKey: "delegate-" + run.RunID, RunID: run.RunID, Runner: "agent:" + run.Adapter, Channel: "cli"}, time.Now())
	if ferr != nil {
		out.Hint("pose delegate: request " + request.Request.ID + " is not fulfilled: " + strings.TrimPrefix(ferr.Error(), "pose: "))
		return
	}
	line := "pose delegate: request " + view.Request.ID + " " + view.State + " (" + view.Answer + ")"
	if len(view.Events) > 0 {
		if f := view.Events[len(view.Events)-1].Fulfillment; f != nil && len(f.Escalations) > 0 {
			line += "; asked of a person: " + strings.Join(f.Escalations, ", ")
		}
	}
	out.Hint(line)
}

// delegateEscalateRefusal asks a person when the request's run is refused
// because the attempt budget is spent: no retry restores it (R7, R10).
func delegateEscalateRefusal(store posemodel.Store, request *posemodel.ActionRequestView, err error, dryRun bool, stdout, stderr io.Writer) {
	if request == nil || dryRun || posemodel.DelegationReasonCode(err) != posemodel.DelegationReasonBudgetExhausted {
		return
	}
	id, eerr := store.EscalateDelegatedReview(request.Request, posemodel.EscalationBudgetExhausted,
		"The attempt budget of "+request.Request.Review.Scope+" is spent without the requested review: what now?",
		"Request "+request.Request.ID+" could not run: "+strings.TrimPrefix(err.Error(), "pose: "), nil, time.Now())
	if eerr != nil {
		render(stdout, stderr).Hint("pose delegate: the person could not be asked: " + eerr.Error())
		return
	}
	render(stdout, stderr).Hint("pose delegate: the attempt budget is spent; asked of a person: " + id)
}
