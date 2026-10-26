package pose

import "testing"

// R2: each terminal state maps to its documented exit code; a running attempt
// is not terminal.
func TestDelegationWaitOutcomeCodes(t *testing.T) {
	run := func(status, output string, interruption *ReviewInterruption) DelegationAttempt {
		r := &ReviewRun{RunID: "rrn-1", Status: status, Output: output, Interruption: interruption}
		return DelegationAttempt{AttemptID: "att", LatestRun: r.RunID, State: status, latest: r}
	}
	cases := []struct {
		name     string
		attempt  DelegationAttempt
		terminal bool
		code     int
	}{
		{"running", DelegationAttempt{AttemptID: "att", State: DelegationRunning, Active: true}, false, 0},
		{"approved", run(ReviewRunCompleted, "decision: approved", nil), true, DelegationWaitApproved},
		{"changes", run(ReviewRunCompleted, "decision: changes-requested", nil), true, DelegationWaitChanges},
		{"rejected", run(ReviewRunCompleted, "decision: rejected", nil), true, DelegationWaitChanges},
		{"no decision", run(ReviewRunCompleted, "nothing", nil), true, DelegationWaitFailed},
		{"failed", run(ReviewRunFailed, "", nil), true, DelegationWaitFailed},
		{"interrupted", run(ReviewRunInterrupted, "", &ReviewInterruption{Cause: "usage-limit", Resumable: true, ResetAt: "2099-01-01T00:00:00Z"}), true, DelegationWaitInterrupted},
		{"stalled", DelegationAttempt{AttemptID: "att", State: DelegationStalled}, true, DelegationWaitStalled},
	}
	for _, c := range cases {
		result, terminal := DelegationWaitOutcome(c.attempt)
		if terminal != c.terminal || terminal && result.Code != c.code {
			t.Fatalf("%s: %+v %t", c.name, result, terminal)
		}
		if c.name == "interrupted" && (!result.Resumable || result.ResetAt == "") {
			t.Fatalf("interrupted lost its reset: %+v", result)
		}
	}
}
