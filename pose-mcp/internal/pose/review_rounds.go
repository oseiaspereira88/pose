package pose

// Review rounds (spec pose-delegated-review-round-protocol).
//
// The rounds of a scope are the sealed bundles of that scope a completed
// delegated review concluded on, in the order they were sealed. A round after
// the first is briefed with the open findings of the round before, each with
// the identity it was first given, and its draft must account for every one
// of them and say what it swept (R1–R5, D2). The draft is the fenced
// `pose-draft` block of the reviewer's final message: JSON, versioned, read
// strictly; prose around it is the reviewer's and is not read.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ReviewDraftSchema is the schema of a pose-draft block.
const ReviewDraftSchema = "pose.review.draft/v1"

const (
	reviewDraftFence            = "```pose-draft"
	reviewDraftMaxFindings      = 500
	reviewDraftMaxText          = 4000
	ReviewRoundThresholdDefault = 5
)

var reviewFindingID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// ReviewDraftFinding is one finding a draft reports.
type ReviewDraftFinding struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Disposition string `json:"disposition"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Summary     string `json:"summary"`
}

// ReviewDraftPrior is a draft's account of one prior finding (R3, R4).
type ReviewDraftPrior struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Evidence string `json:"evidence"`
}

// ReviewDraft is the pose-draft block of a reviewer's final message.
type ReviewDraft struct {
	Schema        string               `json:"schema"`
	Decision      string               `json:"decision"`
	Findings      []ReviewDraftFinding `json:"findings"`
	PriorFindings []ReviewDraftPrior   `json:"prior_findings,omitempty"`
	Sweep         string               `json:"sweep,omitempty"`
}

// Prior finding statuses (R3).
const (
	PriorFindingFixed      = "fixed"
	PriorFindingPartial    = "partial"
	PriorFindingNotFixed   = "not-fixed"
	PriorFindingSuperseded = "superseded"
)

// ParseReviewDraft reads the last pose-draft block of output: nil when there
// is none, an error when the block is not a valid draft.
func ParseReviewDraft(output string) (*ReviewDraft, error) {
	start := strings.LastIndex(output, reviewDraftFence+"\n")
	if start < 0 {
		return nil, nil
	}
	body := output[start+len(reviewDraftFence)+1:]
	end := strings.Index(body, "\n```")
	if end < 0 && strings.HasPrefix(body, "```") {
		end = 0
	}
	if end < 0 {
		return nil, fmt.Errorf("the pose-draft block is not closed")
	}
	var draft ReviewDraft
	dec := json.NewDecoder(bytes.NewReader([]byte(body[:end])))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&draft); err != nil {
		return nil, fmt.Errorf("the pose-draft block is not a draft: %v", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("the pose-draft block carries data after the draft")
	}
	if err := validateReviewDraft(draft); err != nil {
		return nil, err
	}
	if line := draftReviewDecision(output); line != "unknown" && line != draft.Decision {
		return nil, fmt.Errorf("the draft's decision %s differs from the decision line %s", draft.Decision, line)
	}
	return &draft, nil
}

func validateReviewDraft(d ReviewDraft) error {
	if d.Schema != ReviewDraftSchema {
		return fmt.Errorf("the draft's schema is %q, not %s", d.Schema, ReviewDraftSchema)
	}
	switch d.Decision {
	case "approved", "changes-requested", "rejected":
	default:
		return fmt.Errorf("the draft's decision %q is not approved, changes-requested or rejected", d.Decision)
	}
	if len(d.Findings) > reviewDraftMaxFindings || len(d.PriorFindings) > reviewDraftMaxFindings || len(d.Sweep) > reviewDraftMaxText {
		return fmt.Errorf("the draft exceeds its bounds")
	}
	seen := map[string]bool{}
	for _, f := range d.Findings {
		switch {
		case !reviewFindingID.MatchString(f.ID) || seen[f.ID]:
			return fmt.Errorf("finding id %q is not a unique identifier", f.ID)
		case f.Severity != "high" && f.Severity != "medium" && f.Severity != "low":
			return fmt.Errorf("finding %s: severity %q is not high, medium or low", f.ID, f.Severity)
		case f.Disposition != "blocker" && f.Disposition != "follow-up":
			return fmt.Errorf("finding %s: disposition %q is not blocker or follow-up", f.ID, f.Disposition)
		case strings.TrimSpace(f.Summary) == "" || len(f.Summary) > reviewDraftMaxText || len(f.File) > 1024 || f.Line < 0:
			return fmt.Errorf("finding %s needs a bounded summary and location", f.ID)
		}
		seen[f.ID] = true
	}
	accounted := map[string]bool{}
	for _, p := range d.PriorFindings {
		switch {
		case !reviewFindingID.MatchString(p.ID) || accounted[p.ID]:
			return fmt.Errorf("prior finding id %q is not a unique identifier", p.ID)
		case p.Status != PriorFindingFixed && p.Status != PriorFindingPartial && p.Status != PriorFindingNotFixed && p.Status != PriorFindingSuperseded:
			return fmt.Errorf("prior finding %s: status %q is not fixed, partial, not-fixed or superseded", p.ID, p.Status)
		case len(p.Evidence) > reviewDraftMaxText:
			return fmt.Errorf("prior finding %s: evidence exceeds its bound", p.ID)
		}
		accounted[p.ID] = true
	}
	return nil
}

// ReviewOpenFinding is a finding a round must account for: its identity
// across rounds, the round it was first reported in, its details and, when it
// was carried, the status the last round gave it (R2, R5).
type ReviewOpenFinding struct {
	ID          string
	Round       int
	Severity    string
	Disposition string
	File        string
	Line        int
	Summary     string
	Carried     string
}

// ReviewRound is the round a bundle is in and what it must account for.
type ReviewRound struct {
	Number int
	// Prior is the run that concluded the round before, nil in round 1.
	Prior       *ReviewRun
	PriorBundle *ReviewBundle
	// Open are the findings this round must account for.
	Open []ReviewOpenFinding
	// Broken names why the chain of drafts before this round is broken, so
	// the open findings start again from the rounds after it.
	Broken string
}

// roundFindingID is the identity a finding gets when first reported in
// round: the round's prefix and the reviewer's identifier.
func roundFindingID(round int, id string) string {
	prefix := fmt.Sprintf("R%d-", round)
	if strings.HasPrefix(id, prefix) {
		return id
	}
	return prefix + id
}

// concludingReviewRun is the run a round concluded with on bundle: the latest
// completed review run of it that no run supersedes.
func concludingReviewRun(runs []ReviewRun, bundleID string) *ReviewRun {
	var out *ReviewRun
	for i := range runs {
		r := &runs[i]
		if r.BundleID == bundleID && r.Kind == ReviewBriefKindReview && r.Status == ReviewRunCompleted && r.SupersededBy == "" {
			out = r
		}
	}
	return out
}

// ReviewRoundOf derives the round of bundle from the sealed bundles of its
// scope and the run history (R1): the bundles sealed before it that a
// completed review concluded on are the rounds before it. A previous round
// whose record is not intact makes the derivation fail (security).
func (s Store) ReviewRoundOf(bundle ReviewBundle) (ReviewRound, error) {
	runs, lost, err := s.ReadReviewRuns("")
	if err != nil {
		return ReviewRound{}, err
	}
	if len(lost) > 0 {
		return ReviewRound{}, fmt.Errorf("pose: the review run history is incomplete, so the round of %s cannot be derived: %v", bundle.BundleID, lost)
	}
	bundles, err := s.ListReviewBundles(bundle.Payload.Scope.Ref)
	if err != nil {
		return ReviewRound{}, err
	}
	return reviewRoundFrom(bundle, bundles, runs, s.reviewRunTranscriptIntact)
}

// reviewRoundFrom derives the round of bundle from bundles and runs; intact
// says why a run's record is not intact, or "".
func reviewRoundFrom(bundle ReviewBundle, bundles []ReviewBundle, runs []ReviewRun, intact func(ReviewRun) string) (ReviewRound, error) {
	type concluded struct {
		bundle ReviewBundle
		run    *ReviewRun
	}
	// A round precedes bundle when its concluding run ended no later than the
	// second bundle was sealed in: a bundle is sealed after the rejection it
	// answers. Seals have second precision, runs nanoseconds, so the runs order
	// the chain.
	sealed, sealErr := time.Parse(time.RFC3339, bundle.SealedAt)
	var chain []concluded
	for _, b := range bundles {
		if b.BundleID == bundle.BundleID || b.BundleDigest == bundle.BundleDigest {
			continue
		}
		run := concludingReviewRun(runs, b.BundleID)
		if run == nil {
			continue
		}
		ended, err := time.Parse(time.RFC3339Nano, run.EndedAt)
		if err != nil {
			continue
		}
		if bundle.SealedAt != "" && sealErr == nil && ended.Truncate(time.Second).After(sealed) {
			continue
		}
		chain = append(chain, concluded{b, run})
	}
	sort.SliceStable(chain, func(i, j int) bool { return chain[i].run.EndedAt < chain[j].run.EndedAt })
	round := ReviewRound{Number: len(chain) + 1}
	var open []ReviewOpenFinding
	for i, c := range chain {
		number := i + 1
		if why := intact(*c.run); why != "" {
			return ReviewRound{}, fmt.Errorf("pose: the run of round %d (%s) is not intact, so its findings cannot be carried: %s", number, c.run.RunID, why)
		}
		draft, err := ParseReviewDraft(c.run.Output)
		if draft == nil || err != nil {
			why := "it carries no pose-draft block"
			if err != nil {
				why = err.Error()
			}
			round.Broken = fmt.Sprintf("the draft of round %d (%s) cannot be read: %s", number, c.run.RunID, why)
			open = nil
			continue
		}
		byID := map[string]ReviewOpenFinding{}
		for _, f := range open {
			byID[f.ID] = f
		}
		var next []ReviewOpenFinding
		for _, p := range draft.PriorFindings {
			if f, ok := byID[p.ID]; ok && (p.Status == PriorFindingPartial || p.Status == PriorFindingNotFixed) {
				f.Carried = p.Status
				next = append(next, f)
			}
		}
		for _, f := range draft.Findings {
			next = append(next, ReviewOpenFinding{ID: roundFindingID(number, f.ID), Round: number, Severity: f.Severity, Disposition: f.Disposition, File: f.File, Line: f.Line, Summary: f.Summary})
		}
		open = next
	}
	if len(chain) > 0 {
		last := chain[len(chain)-1]
		round.Prior, round.PriorBundle = last.run, &last.bundle
	}
	round.Open = open
	return round, nil
}

// ValidateRoundDraft judges a draft of round against R3 and R4: from round 2
// on, every open finding is accounted for, a fixed one with evidence, and the
// sweep is stated. It returns the problems; none means valid.
func ValidateRoundDraft(round ReviewRound, output string) []string {
	draft, err := ParseReviewDraft(output)
	if err != nil {
		return []string{err.Error()}
	}
	if round.Number < 2 {
		return nil
	}
	if draft == nil {
		return []string{"a draft of round " + fmt.Sprint(round.Number) + " needs a pose-draft block"}
	}
	var problems []string
	accounted := map[string]ReviewDraftPrior{}
	for _, p := range draft.PriorFindings {
		accounted[p.ID] = p
	}
	for _, f := range round.Open {
		p, ok := accounted[f.ID]
		switch {
		case !ok:
			problems = append(problems, "prior finding "+f.ID+" is not accounted for")
		case p.Status == PriorFindingFixed && strings.TrimSpace(p.Evidence) == "":
			problems = append(problems, "prior finding "+f.ID+" is reported fixed with no evidence")
		}
	}
	if strings.TrimSpace(draft.Sweep) == "" {
		problems = append(problems, "the draft does not state what it swept")
	}
	return problems
}

// renderPriorRound is the brief's section of round 2 and later, built only
// from the ledger, byte-stable for the same ledger state (R2, R3).
func renderPriorRound(round ReviewRound) string {
	var b strings.Builder
	line := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	line("## Previous round")
	line("")
	line("This is round %d of the scope.", round.Number)
	if round.Prior != nil {
		head := ""
		if round.PriorBundle != nil {
			head = round.PriorBundle.Payload.Subject.Head
		}
		line("The previous round is run %s on bundle %s (commit %s); its draft digest is %s.", round.Prior.RunID, round.Prior.BundleID, orNone(head), digestBytes([]byte(round.Prior.Output)))
	}
	if round.Broken != "" {
		line("The chain of drafts is broken: %s. Findings before it are not carried.", round.Broken)
	}
	line("")
	line("Findings to account for, each with the identifier it keeps across rounds:")
	line("")
	if len(round.Open) == 0 {
		line("- none recorded")
	}
	for _, f := range round.Open {
		where := ""
		if f.File != "" {
			where = " " + f.File
			if f.Line > 0 {
				where += fmt.Sprintf(":%d", f.Line)
			}
		}
		carried := ""
		if f.Carried != "" {
			carried = " [reported " + f.Carried + " in a later round]"
		}
		line("- %s (%s, %s, round %d)%s%s — %s", f.ID, f.Severity, f.Disposition, f.Round, where, carried, strings.Join(strings.Fields(f.Summary), " "))
	}
	line("")
	line("Before reporting anything new, account for each finding above in `prior_findings`, by its identifier, as `fixed`, `partial`, `not-fixed` or `superseded`, with your evidence (a command you ran, or file and line); `fixed` needs evidence. Then sweep the whole subject in this round and report every finding you can establish, not only the first; say in `sweep` what you swept. A draft that leaves a finding above unaccounted, or states no sweep, is invalid and is never taken as an approval.")
	line("")
	return b.String()
}

// Rounds modes of routing.rounds (R6).
const (
	ReviewRoundsContinueThread = "continue-thread"
	ReviewRoundsColdSession    = "cold-session"
)

// Thread modes.
const (
	ReviewThreadContinued = "continued"
	ReviewThreadNew       = "new"
)

// ReviewThread is how the session of a round after the first relates to the
// previous round's: continued from its run, or new with the reason (R6).
type ReviewThread struct {
	Round  int    `json:"round"`
	Mode   string `json:"mode"`
	From   string `json:"from,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func (t *ReviewThread) continued() bool { return t != nil && t.Mode == ReviewThreadContinued }

// planReviewThread decides the session of a review round after the first:
// with continue-thread it resumes the session the previous round ended with,
// unless the agent's vendor or the agent changed, the chain of drafts is
// broken, that session cannot be resumed, or the scope reached the round
// threshold; each opens a new session, recording why (R6, D1).
func (s Store) planReviewThread(plan *DispatchPlan) error {
	round, err := s.ReviewRoundOf(plan.Bundle)
	if err != nil {
		return err
	}
	if round.Number < 2 || round.Prior == nil {
		return nil
	}
	policy, _, _, err := s.LoadDelegationPolicy()
	if err != nil {
		return err
	}
	prior := round.Prior
	thread := &ReviewThread{Round: round.Number, Mode: ReviewThreadNew, From: prior.RunID}
	threshold := policy.Routing.RoundThresholdOf()
	session := ""
	switch {
	case policy.Routing.RoundsMode() == ReviewRoundsColdSession:
		thread.Reason = "routing.rounds is cold-session"
	case round.Broken != "":
		thread.Reason = "the chain of rounds is broken: " + round.Broken
	case !strings.EqualFold(prior.Vendor, plan.Config.Vendor):
		thread.Reason = "the agent's vendor changed from " + prior.Vendor + " to " + plan.Config.Vendor
	case prior.Adapter != plan.Adapter:
		thread.Reason = "the agent changed from " + prior.Adapter + " to " + plan.Adapter
	case round.Number >= threshold:
		thread.Reason = fmt.Sprintf("round %d reached the round threshold %d", round.Number, threshold)
	case prior.SessionDigest == "" || plan.Config.Resume == nil || len(plan.Config.Resume.Command) == 0:
		thread.Reason = "the previous round's session cannot be resumed: it recorded no session or the agent declares no resume"
	default:
		value, rerr := s.readReviewSession(prior.SessionDigest)
		if rerr != nil {
			thread.Reason = "the previous round's session cannot be resumed: " + rerr.Error()
			break
		}
		thread.Mode, session = ReviewThreadContinued, value
	}
	plan.thread, plan.session = thread, session
	return nil
}

// ReviewRoundMetric is the convergence of one scope's rounds (R7, R10).
type ReviewRoundMetric struct {
	Scope            string                 `json:"scope"`
	Rounds           int                    `json:"rounds"`
	FindingsPerRound []int                  `json:"findings_per_round"`
	RoundsToApproval *int                   `json:"rounds_to_approval"`
	Continued        ReviewRoundConvergence `json:"continued"`
	Cold             ReviewRoundConvergence `json:"cold"`
}

// ReviewRoundConvergence counts how the prior findings of the rounds of one
// kind were accounted for: rounds whose session continued the previous
// round's, or cold ones (R10).
type ReviewRoundConvergence struct {
	Rounds     int      `json:"rounds"`
	Fixed      int      `json:"fixed"`
	Partial    int      `json:"partial"`
	NotFixed   int      `json:"not_fixed"`
	Superseded int      `json:"superseded"`
	FixedShare *float64 `json:"fixed_share"`
}

// reviewRoundMetrics measures each scope's rounds from the completed review
// runs that recorded their round. It never lowers the rounds recorded (R8).
func reviewRoundMetrics(runs []ReviewRun) []ReviewRoundMetric {
	byScope := map[string]map[int]ReviewRun{}
	for _, r := range runs {
		if r.Round == 0 || r.Kind != ReviewBriefKindReview || r.Status != ReviewRunCompleted || r.SupersededBy != "" {
			continue
		}
		if byScope[r.Scope] == nil {
			byScope[r.Scope] = map[int]ReviewRun{}
		}
		byScope[r.Scope][r.Round] = r // the latest run of a round concludes it
	}
	scopes := make([]string, 0, len(byScope))
	for scope := range byScope {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	var out []ReviewRoundMetric
	for _, scope := range scopes {
		rounds := byScope[scope]
		m := ReviewRoundMetric{Scope: scope}
		for n := range rounds {
			if n > m.Rounds {
				m.Rounds = n
			}
		}
		m.FindingsPerRound = make([]int, m.Rounds)
		for n := 1; n <= m.Rounds; n++ {
			r, ok := rounds[n]
			if !ok {
				continue
			}
			draft, err := ParseReviewDraft(r.Output)
			if err != nil || draft == nil {
				continue
			}
			m.FindingsPerRound[n-1] = len(draft.Findings)
			if m.RoundsToApproval == nil && draft.Decision == "approved" && len(r.DraftProblems) == 0 {
				approved := n
				m.RoundsToApproval = &approved
			}
			if n < 2 {
				continue
			}
			kind := &m.Cold
			if r.Thread.continued() {
				kind = &m.Continued
			}
			kind.Rounds++
			for _, p := range draft.PriorFindings {
				switch p.Status {
				case PriorFindingFixed:
					kind.Fixed++
				case PriorFindingPartial:
					kind.Partial++
				case PriorFindingNotFixed:
					kind.NotFixed++
				case PriorFindingSuperseded:
					kind.Superseded++
				}
			}
		}
		for _, kind := range []*ReviewRoundConvergence{&m.Continued, &m.Cold} {
			if total := kind.Fixed + kind.Partial + kind.NotFixed; total > 0 {
				share := float64(kind.Fixed) / float64(total)
				kind.FixedShare = &share
			}
		}
		out = append(out, m)
	}
	return out
}
