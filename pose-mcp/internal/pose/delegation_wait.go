package pose

// The outcome a waiter on a delegated run reads (spec
// pose-delegated-review-completion-signal R2, R5, R8). A run is terminal once
// its record is published and no process runs its attempt: the record is
// published before the marker is removed, so a reader that finds no marker
// finds the record. The exit codes are a public contract (D1).

// Exit codes of pose delegate wait.
const (
	DelegationWaitApproved    = 0
	DelegationWaitChanges     = 10
	DelegationWaitFailed      = 20
	DelegationWaitInterrupted = 30
	DelegationWaitStalled     = 40
	DelegationWaitTimeout     = 50
)

// DelegationWaitResult is the one line a waiter prints.
type DelegationWaitResult struct {
	Attempt   string `json:"attempt"`
	Run       string `json:"run,omitempty"`
	State     string `json:"state"`
	Decision  string `json:"decision,omitempty"`
	Resumable bool   `json:"resumable"`
	ResetAt   string `json:"reset_at,omitempty"`
	Code      int    `json:"exit_code"`
}

// DelegationWaitOutcome reads attempt a: terminal is false while a process
// still runs it.
func DelegationWaitOutcome(a DelegationAttempt) (DelegationWaitResult, bool) {
	result := DelegationWaitResult{Attempt: a.AttemptID, Run: a.LatestRun, State: a.State}
	switch {
	case a.Active:
		return result, false
	case a.State == DelegationStalled:
		result.Code = DelegationWaitStalled
		return result, true
	}
	latest := a.Latest()
	if latest == nil {
		// Neither a process nor a record: the marker was removed without a
		// record, which only a lost host leaves behind.
		result.State, result.Code = DelegationStalled, DelegationWaitStalled
		return result, true
	}
	switch latest.Status {
	case ReviewRunCompleted:
		// A task concludes nothing (spec pose-delegation-task R7): it succeeded
		// when its agent did.
		if latest.Purpose == DelegationPurposeTask {
			if latest.ExitCode != 0 {
				result.Code = DelegationWaitFailed
			}
			break
		}
		result.Decision = draftReviewDecision(latest.Output)
		if len(latest.DraftProblems) > 0 {
			// A draft that does not account for its round decides nothing
			// (spec pose-delegated-review-round-protocol R4).
			result.Decision = "invalid"
		}
		switch result.Decision {
		case "approved":
			result.Code = DelegationWaitApproved
		case "changes-requested", "rejected":
			result.Code = DelegationWaitChanges
		default:
			// A completed run with no decision answers nothing a waiter can act
			// on as a review: it is counted with the failures.
			result.Code = DelegationWaitFailed
		}
	case ReviewRunInterrupted:
		result.Code = DelegationWaitInterrupted
		if i := latest.Interruption; i != nil {
			result.Resumable, result.ResetAt = i.Resumable, i.ResetAt
		}
	default:
		result.Code = DelegationWaitFailed
	}
	return result, true
}
