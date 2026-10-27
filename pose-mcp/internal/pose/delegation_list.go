package pose

// Listing and naming delegated runs (spec pose-delegation-surface R11, and the
// listing pose-delegated-review-completion-signal extends).
//
// A person and an agent follow a task, not a process: an attempt and its
// continuations are one row, shown by their latest run. A reference names a
// row by attempt, by any of its runs, by label or by a prefix of its session.

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// DelegationAttempt is one attempt and its continuations.
type DelegationAttempt struct {
	AttemptID    string              `json:"attempt_id"`
	Runs         []string            `json:"runs"`
	LatestRun    string              `json:"latest_run,omitempty"`
	State        string              `json:"state"`
	Purpose      string              `json:"purpose"`
	Agent        string              `json:"agent"`
	Vendor       string              `json:"vendor,omitempty"`
	Model        string              `json:"model,omitempty"`
	Label        string              `json:"label,omitempty"`
	Lead         string              `json:"lead,omitempty"`
	Access       string              `json:"access"`
	Scope        string              `json:"scope"`
	BundleID     string              `json:"bundle_id"`
	StartedAt    string              `json:"started_at"`
	EndedAt      string              `json:"ended_at,omitempty"`
	Turns        int                 `json:"turns"`
	LastStep     string              `json:"last_step,omitempty"`
	Failure      string              `json:"failure,omitempty"`
	Interruption *ReviewInterruption `json:"interruption,omitempty"`
	// Partial says an interrupted run kept a partial draft; TimeLeft is the
	// wait until its recorded reset, when there is one.
	Partial         bool   `json:"partial,omitempty"`
	TimeLeft        string `json:"time_left,omitempty"`
	Active          bool   `json:"active"`
	Liveness        string `json:"liveness,omitempty"`
	CancelRequested bool   `json:"cancel_requested,omitempty"`
	// HeartbeatAge is how long ago the running process last beat; a stalled
	// run is not active (spec pose-delegated-review-completion-signal R4, R6).
	HeartbeatAge string `json:"heartbeat_age,omitempty"`
	StalledAt    string `json:"stalled_at,omitempty"`
	// HookFailure is why the latest run's on_terminal hook failed (R7).
	HookFailure string `json:"hook_failure,omitempty"`
	sessions    []string
	latest      *ReviewRun
}

// Latest is the latest recorded run of the attempt, if any.
func (a DelegationAttempt) Latest() *ReviewRun { return a.latest }

// DelegationRunning is the state of an attempt whose process is still there;
// DelegationStalled, of one whose process is gone without a record.
const (
	DelegationRunning = "running"
	DelegationStalled = "stalled"
)

// ListDelegationAttempts groups every recorded run by attempt, adds the runs
// in progress, and returns the losses the run history reports.
func (s Store) ListDelegationAttempts() ([]DelegationAttempt, []string, error) {
	// The markers are read before the runs: a run publishes its record before
	// it removes its marker, so a reader that finds no marker for an attempt
	// it saw running finds its record (spec
	// pose-delegated-review-completion-signal R5).
	active, activeLost, activeErr := s.ListDelegationActive()
	runs, lost, err := s.ReadReviewRuns("")
	if err != nil {
		return nil, nil, err
	}
	// Task runs share the lifecycle, never the review history (spec
	// pose-delegation-task R7, R8).
	tasks, taskLost := s.ReadDelegationTaskRuns()
	runs, lost = append(runs, tasks...), append(lost, taskLost...)
	byAttempt := map[string]*DelegationAttempt{}
	order := []string{}
	for i := range runs {
		run := runs[i]
		key := run.AttemptID
		if key == "" {
			key = run.RunID // a run recorded before attempts existed is its own attempt
		}
		row, ok := byAttempt[key]
		if !ok {
			row = &DelegationAttempt{AttemptID: key, StartedAt: run.StartedAt}
			byAttempt[key] = row
			order = append(order, key)
		}
		row.Runs = append(row.Runs, run.RunID)
		row.latest = &runs[i]
		if run.SessionDigest != "" {
			if value, err := s.readReviewSession(run.SessionDigest); err == nil {
				row.sessions = append(row.sessions, value)
			}
		}
		events, err := s.ReadDelegationEvents(run)
		if err != nil {
			// A turn count read from nothing would report a whole history.
			lost = append(lost, "the events of run "+run.RunID+" could not be read: "+err.Error())
		} else {
			for _, e := range events {
				if e.Kind == DelegationEventTurn {
					row.Turns++
				}
			}
			if len(events) > 0 {
				row.LastStep = delegationStepText(events[len(events)-1])
			}
		}
	}
	for _, key := range order {
		row := byAttempt[key]
		latest := row.latest
		row.LatestRun, row.State, row.Agent, row.Vendor, row.Model = latest.RunID, latest.Status, latest.Adapter, latest.Vendor, latest.Model
		row.Label, row.Scope, row.BundleID, row.EndedAt, row.Failure, row.Interruption = latest.Label, latest.Scope, latest.BundleID, latest.EndedAt, latest.Failure, latest.Interruption
		row.Purpose, row.Lead, row.Access = latest.Purpose, latest.Lead, DelegationAccessRead
		if latest.Access != "" {
			row.Access = latest.Access
		}
		if hook := s.ReadDelegationHook(latest.RunID); hook != nil {
			row.HookFailure = hook.Failure
		}
		if row.Purpose == "" {
			row.Purpose = DelegationPurposeForKind(latest.Kind)
		}
		if latest.Status == ReviewRunInterrupted {
			row.Partial = strings.TrimSpace(latest.Output) != ""
			if latest.Interruption != nil && latest.Interruption.ResetAt != "" {
				if at, err := time.Parse(time.RFC3339Nano, latest.Interruption.ResetAt); err == nil {
					left := time.Until(at)
					if left < 0 {
						left = 0
					}
					row.TimeLeft = left.Round(time.Second).String()
				}
			}
		}
	}
	if activeErr != nil {
		lost = append(lost, "active runs could not be read: "+activeErr.Error())
	}
	lost = append(lost, activeLost...)
	for _, marker := range active {
		row, ok := byAttempt[marker.AttemptID]
		if !ok {
			row = &DelegationAttempt{AttemptID: marker.AttemptID, Runs: []string{}, StartedAt: marker.StartedAt, Purpose: marker.Purpose, Access: DelegationAccessRead, Agent: marker.Agent, Label: marker.Label, Scope: marker.Scope, BundleID: marker.BundleID}
			byAttempt[marker.AttemptID] = row
			order = append(order, marker.AttemptID)
		}
		row.State, row.Active, row.Liveness, row.CancelRequested = DelegationRunning, true, marker.Liveness, marker.CancelRequested
		row.EndedAt, row.Failure, row.Interruption, row.Partial, row.TimeLeft = "", "", nil, false, ""
		if beat, err := time.Parse(time.RFC3339Nano, marker.HeartbeatAt); err == nil {
			row.HeartbeatAge = time.Since(beat).Round(time.Second).String()
		}
		if marker.StalledAt != "" {
			row.State, row.Active, row.StalledAt = DelegationStalled, false, marker.StalledAt
		}
		// A running run's progress comes from its live events (R12).
		if live, _, err := s.ReadDelegationLiveEvents(marker.AttemptID, 0); err == nil && len(live) > 0 {
			for _, e := range live {
				if e.Kind == DelegationEventTurn {
					row.Turns++
				}
			}
			row.LastStep = delegationStepText(live[len(live)-1])
		}
	}
	out := make([]DelegationAttempt, 0, len(order))
	for _, key := range order {
		out = append(out, *byAttempt[key])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartedAt < out[j].StartedAt })
	return out, lost, nil
}

func delegationStepText(e DelegationEvent) string {
	text := strings.Join(strings.Fields(e.Text), " ")
	if len(text) > 80 {
		cut := 77
		for cut > 0 && !utf8RuneStart(text[cut]) {
			cut--
		}
		text = text[:cut] + "..."
	}
	switch e.Kind {
	case DelegationEventToolCall:
		return "▶ " + text
	case DelegationEventToolOutput:
		if e.ExitCode != nil && *e.ExitCode != 0 {
			return fmt.Sprintf("✗ exit %d", *e.ExitCode)
		}
		return "✓ exit 0"
	case DelegationEventTurn:
		return "turn"
	}
	if text == "" {
		return e.Kind
	}
	return e.Kind + ": " + text
}

// ErrDelegationRefAmbiguous carries the candidates of an ambiguous reference.
// Neither reference error carries the reference: it may be a session, which
// no output names.
type ErrDelegationRefAmbiguous struct {
	Candidates []string
}

func (e *ErrDelegationRefAmbiguous) Error() string {
	return "pose: the reference names more than one delegated run: " + strings.Join(e.Candidates, ", ")
}

// ErrDelegationRefUnknown means no attempt matches.
type ErrDelegationRefUnknown struct{}

func (e *ErrDelegationRefUnknown) Error() string {
	return "pose: no delegated run matches the reference (a run id, an attempt id, a label or a session prefix of six characters or more)"
}

// ResolveDelegationRef finds the one attempt ref names (R11).
func ResolveDelegationRef(attempts []DelegationAttempt, ref string) (DelegationAttempt, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return DelegationAttempt{}, &ErrDelegationRefUnknown{}
	}
	// An identifier is exact and wins over a label or a session prefix.
	for _, a := range attempts {
		if a.AttemptID == ref || containsString(a.Runs, ref) {
			return a, nil
		}
	}
	var matched []DelegationAttempt
	for _, a := range attempts {
		hit := a.Label == ref
		if !hit && len(ref) >= 6 {
			for _, session := range a.sessions {
				if strings.HasPrefix(session, ref) {
					hit = true
				}
			}
		}
		if hit {
			matched = append(matched, a)
		}
	}
	switch len(matched) {
	case 0:
		return DelegationAttempt{}, &ErrDelegationRefUnknown{}
	case 1:
		return matched[0], nil
	}
	candidates := make([]string, 0, len(matched))
	for _, a := range matched {
		name := a.AttemptID
		if a.Label != "" {
			name += " (" + a.Label + ", " + a.State + ")"
		}
		candidates = append(candidates, name)
	}
	return DelegationAttempt{}, &ErrDelegationRefAmbiguous{Candidates: candidates}
}
