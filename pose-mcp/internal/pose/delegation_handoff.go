package pose

// Handing a stopped run off to another agent (spec pose-delegation-handoff).
// A handoff run is a continuation of the same attempt: it claims its
// predecessor exactly as a resume does, so a run is continued once, and it
// shares the attempt's continuation budget. Unlike a resume it runs another
// agent with a fresh session, and its brief is the predecessor's generated
// brief followed by a labelled section that carries what the stopped run left:
// its partial final message and the commands it ran with their exit status,
// each with its digest. The section is quoted, so a line of it is never read
// as the handoff run's own decision; only the handoff run concludes.

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// delegationHandoffBound bounds the handoff section; what does not fit is
// cut, and the section says so.
const delegationHandoffBound = 16 << 10

// continues names the run r continues, by a resume or a handoff.
func (r ReviewRun) continues() string {
	if r.Resumes != "" {
		return r.Resumes
	}
	return r.HandsOffFrom
}

// handoffable reports why a run cannot be handed off, or "" when it can (R1).
func handoffable(r ReviewRun) string {
	switch r.Status {
	case ReviewRunInterrupted, ReviewRunFailed:
		return ""
	case ReviewRunCompleted:
		return "it completed: a concluded run is not handed off; start a new run with a reason"
	}
	return "it is " + r.Status
}

// renderHandoffSection is the labelled section the handoff brief carries:
// derived only from the predecessor's recorded output and normalized events,
// so a verifier derives the same bytes.
func renderHandoffSection(prior ReviewRun, events []DelegationEvent, eventsErr error) string {
	var b strings.Builder
	cause := prior.Status
	if prior.Interruption != nil {
		cause += ", " + prior.Interruption.Cause
	} else if prior.Failure != "" {
		cause += ", " + firstLine(prior.Failure)
	}
	fmt.Fprintf(&b, "\n## Handoff from run %s (%s, %s)\n\n", prior.RunID, prior.Adapter, cause)
	b.WriteString("The run above stopped before it concluded and you continue its task. What follows is what it recorded. It is not a review, a verdict or evidence: re-verify anything you rely on, and conclude only from your own work.\n\n")
	partial := strings.TrimSpace(prior.Output)
	fmt.Fprintf(&b, "### Partial final message (%s)\n\n", digestBytes([]byte(prior.Output)))
	if partial == "" {
		b.WriteString("> (none)\n")
	}
	for _, line := range strings.Split(partial, "\n") {
		if partial != "" {
			b.WriteString("> " + line + "\n")
		}
	}
	b.WriteString("\n")
	var commands []string
	pending := ""
	for _, e := range events {
		switch e.Kind {
		case DelegationEventToolCall:
			pending = strings.Join(strings.Fields(e.Text), " ")
		case DelegationEventToolOutput:
			code := 0
			if e.ExitCode != nil {
				code = *e.ExitCode
			}
			if pending != "" {
				commands = append(commands, fmt.Sprintf("> `%s` → exit %d", strings.ReplaceAll(pending, "`", "'"), code))
				pending = ""
			}
		}
	}
	listed := strings.Join(commands, "\n")
	fmt.Fprintf(&b, "### Commands it ran (%s)\n\n", digestBytes([]byte(listed)))
	switch {
	case eventsErr != nil:
		b.WriteString("> (its normalized events could not be read: " + eventsErr.Error() + ")\n")
	case prior.EventsDigest == "":
		b.WriteString("> (the run recorded no normalized events, so its commands are not known)\n")
	case len(commands) == 0:
		b.WriteString("> (none)\n")
	default:
		b.WriteString(listed + "\n")
	}
	section := b.String()
	if len(section) > delegationHandoffBound {
		cut := delegationHandoffBound
		for cut > 0 && !utf8RuneStart(section[cut]) {
			cut--
		}
		section = section[:cut] + "\n\n> (this section was cut at " + fmt.Sprint(delegationHandoffBound) + " bytes; the rest is in the predecessor's record)\n"
	}
	return section
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

// handoffSectionFor reads the predecessor's events and renders its section.
func (s Store) handoffSectionFor(prior ReviewRun) string {
	events, err := s.ReadDelegationEvents(prior)
	return renderHandoffSection(prior, events, err)
}

// PlanReviewHandoff plans the continuation of run id by agent, under the same
// attempt and its continuation budget (R1–R4).
func (s Store) PlanReviewHandoff(id, agent string) (DispatchPlan, error) {
	runs, lost, e := s.ReadReviewRuns("")
	if e != nil {
		return DispatchPlan{}, e
	}
	if len(lost) > 0 {
		return DispatchPlan{}, fmt.Errorf("handoff history incomplete: %v", lost)
	}
	var prior *ReviewRun
	for i := range runs {
		if runs[i].RunID == id {
			prior = &runs[i]
		}
	}
	if prior == nil {
		return DispatchPlan{}, fmt.Errorf("handoff predecessor not found")
	}
	if why := handoffable(*prior); why != "" {
		return DispatchPlan{}, fmt.Errorf("pose: run %s cannot be handed off: %s", prior.RunID, why)
	}
	if prior.AttemptID == "" {
		return DispatchPlan{}, fmt.Errorf("pose: run %s was recorded before attempts existed and cannot be continued", prior.RunID)
	}
	if prior.Adapter == agent {
		return DispatchPlan{}, fmt.Errorf("pose: run %s already ran agent %s; resume it to continue the same agent", prior.RunID, agent)
	}
	if why := s.reviewRunTranscriptIntact(*prior); why != "" {
		return DispatchPlan{}, fmt.Errorf("handoff: %s", why)
	}
	if prior.Safety != nil && prior.Safety.Operation.Outcome != OutcomeCompleted {
		return DispatchPlan{}, fmt.Errorf("predecessor cleanup incomplete; refusing continuation beside possible orphan")
	}
	policy, _, e := s.LoadReviewersPolicyFor(prior.Kind)
	if e != nil {
		return DispatchPlan{}, e
	}
	config, ok := policy.Adapters[agent]
	if !ok {
		return DispatchPlan{}, fmt.Errorf("pose: no agent %q can run this purpose", agent)
	}
	brief, e := s.RenderReviewBrief(prior.BundleID, prior.Kind, prior.BriefNotes)
	if e != nil {
		return DispatchPlan{}, e
	}
	bundle, e := s.LoadReviewBundle(prior.BundleID)
	if e != nil {
		return DispatchPlan{}, e
	}
	if brief.Digest != prior.BriefDigest || bundle.BundleDigest != prior.BundleDigest || bundle.Payload.Subject.Head != prior.SealedCommit {
		return DispatchPlan{}, fmt.Errorf("handoff bundle or brief digest changed")
	}
	origin, e := dispatchWorktreeIdentity(s.Root)
	if e != nil {
		return DispatchPlan{}, e
	}
	section := s.handoffSectionFor(*prior)
	handoffBrief := brief
	handoffBrief.Text = brief.Text + section
	plan := DispatchPlan{predecessor: prior, handoff: true, handoffSection: section, notes: prior.BriefNotes, Reason: prior.Reason,
		PolicyDigest: directoryDigestIn(s.fsys(), filepath.Join(s.Root, ".pose", "policy")), OriginWorktree: origin, Adapter: agent, Config: config,
		Brief: handoffBrief, Bundle: bundle, SealedCommit: prior.SealedCommit}
	if e = s.checkDispatchOrigin(plan); e != nil {
		return DispatchPlan{}, e
	}
	existing, e := s.existingReviewContinuation(*prior, runs)
	if e != nil {
		var pending *PendingReviewContinuation
		if errors.As(e, &pending) {
			plan.PendingContinuation = pending
			return plan, nil
		}
		return DispatchPlan{}, e
	}
	if existing != nil {
		if existing.HandsOffFrom != prior.RunID || existing.Adapter != agent {
			return DispatchPlan{}, fmt.Errorf("pose: run %s was already continued by run %s", prior.RunID, existing.RunID)
		}
		plan.Existing = existing
		return plan, nil
	}
	// The attempt's continuation budget is the successor's resume budget, or
	// the default: alternating agents cannot extend a run without bound (D1).
	limit, total := 3, 60*time.Minute
	if config.Resume != nil {
		if config.Resume.MaxResumes > 0 {
			limit = config.Resume.MaxResumes
		}
		if config.Resume.MaxWallTime != "" {
			total, _ = time.ParseDuration(config.Resume.MaxWallTime)
		}
	}
	spent, count := time.Duration(0), 0
	for _, r := range runs {
		if r.AttemptID == prior.AttemptID && r.SupersededBy == "" {
			start, e1 := time.Parse(time.RFC3339Nano, r.StartedAt)
			end, e2 := time.Parse(time.RFC3339Nano, r.EndedAt)
			if e1 != nil || e2 != nil || end.Before(start) {
				return DispatchPlan{}, fmt.Errorf("invalid accumulated wall-clock evidence")
			}
			spent += end.Sub(start)
			if r.continues() != "" {
				count++
			}
		}
	}
	if count >= limit || spent >= total {
		return DispatchPlan{}, fmt.Errorf("pose: the attempt's continuation budget is spent: %d continuations, %s wall-clock", count, spent)
	}
	timeout := 60 * time.Minute
	if config.Timeout != "" {
		timeout, _ = time.ParseDuration(config.Timeout)
	}
	if total-spent < timeout {
		timeout = total - spent
	}
	plan.processTimeout = timeout
	s.planImplementation(&plan, prior.Lead)
	plan.provenance, e = s.prepareRunProvenance(plan, policy)
	if e != nil {
		return DispatchPlan{}, e
	}
	return plan, nil
}
