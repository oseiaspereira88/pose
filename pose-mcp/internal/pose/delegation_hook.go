package pose

// The on_terminal hook (spec pose-delegated-review-completion-signal R7). A
// command the policy declares for an agent or a purpose runs once after a run
// ends, with the run identifier and its state in its environment and nothing
// else — no credential, no draft, no path outside the run directory, which is
// its working directory. It runs in a process group of its own, killed with
// it, within a bound. Its outcome is recorded beside the run in the engine's
// private state: the run's record is immutable and its state never changes
// because of a hook.

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

const delegationHookDir = reviewPrivateDir + "/hooks"

// DelegationHookOutcome is what a run's hook did.
type DelegationHookOutcome struct {
	SchemaVersion int    `json:"schema_version"`
	RunID         string `json:"run_id"`
	State         string `json:"state"`
	ExitCode      int    `json:"exit_code"`
	Failure       string `json:"failure,omitempty"`
	RanAt         string `json:"ran_at"`
}

// RunDelegationTerminalHook runs the hook of run's agent, or else of its
// purpose; nil when neither declares one.
func (s Store) RunDelegationTerminalHook(run ReviewRun) *DelegationHookOutcome {
	policy, _, found, err := s.LoadDelegationPolicy()
	if err != nil || !found || run.RunID == "" {
		return nil
	}
	hook := policy.Agents[run.Adapter].OnTerminal
	if hook == nil {
		purpose := run.Purpose
		if purpose == "" {
			purpose = DelegationPurposeForKind(run.Kind)
		}
		hook = policy.Purposes[purpose].OnTerminal
	}
	if hook == nil {
		return nil
	}
	// Once per run: a claim published before the hook starts makes a repeated
	// request for the same run — an idempotent resume returning the recorded
	// continuation — run nothing again.
	dir, err := s.delegationHookDir()
	if err != nil || !validAttemptFileName(run.RunID) {
		return nil
	}
	if PublishFile(filepath.Join(dir, run.RunID+".claim"), []byte(time.Now().UTC().Format(time.RFC3339Nano)+"\n"), 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}) != nil {
		return nil
	}
	outcome := &DelegationHookOutcome{SchemaVersion: 1, RunID: run.RunID, State: run.Status, ExitCode: -1, RanAt: time.Now().UTC().Format(time.RFC3339Nano)}
	bound := delegationHookDefaultBound
	if hook.Timeout != "" {
		if d, err := time.ParseDuration(hook.Timeout); err == nil && d > 0 {
			bound = d
		}
	}
	executable, err := exec.LookPath(hook.Command[0])
	if err != nil {
		outcome.Failure = "the hook's command is not found: " + err.Error()
		s.recordDelegationHook(*outcome)
		return outcome
	}
	ctx, cancel := context.WithTimeout(context.Background(), bound)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, hook.Command[1:]...)
	cmd.Args[0] = hook.Command[0]
	cmd.Dir = filepath.Join(s.Root, filepath.FromSlash(reviewRunsDir))
	cmd.Env = []string{"POSE_DELEGATE_RUN=" + run.RunID, "POSE_DELEGATE_STATE=" + run.Status}
	isolateProcessGroup(cmd)
	cmd.WaitDelay = time.Second
	runErr := cmd.Run()
	_ = killProcessGroup(cmd)
	if cmd.ProcessState != nil {
		outcome.ExitCode = cmd.ProcessState.ExitCode()
	}
	switch {
	case ctx.Err() == context.DeadlineExceeded:
		outcome.Failure = fmt.Sprintf("the hook did not end within %s", bound)
	case runErr != nil:
		outcome.Failure = "the hook failed: " + runErr.Error()
	}
	s.recordDelegationHook(*outcome)
	return outcome
}

// delegationHookDir is the hooks directory, reached without following a link.
func (s Store) delegationHookDir() (string, error) {
	if _, err := s.delegationActiveDir(true); err != nil {
		return "", err
	}
	return ensureReviewArtifactDir(s.Root, delegationHookDir, true)
}

func (s Store) recordDelegationHook(outcome DelegationHookOutcome) {
	dir, err := s.delegationHookDir()
	if err != nil || !validAttemptFileName(outcome.RunID) {
		return
	}
	raw, _ := json.MarshalIndent(outcome, "", "  ")
	_ = PublishFile(filepath.Join(dir, outcome.RunID+".json"), append(raw, '\n'), 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAny})
}

// ReadDelegationHook reads the hook outcome recorded for runID, if any.
func (s Store) ReadDelegationHook(runID string) *DelegationHookOutcome {
	if !validAttemptFileName(runID) {
		return nil
	}
	dir, err := ensureReviewArtifactDir(s.Root, delegationHookDir, false)
	if err != nil {
		return nil
	}
	raw, err := readLedgerRaw(filepath.Join(dir, runID+".json"))
	if err != nil {
		return nil
	}
	var outcome DelegationHookOutcome
	if json.Unmarshal(raw, &outcome) != nil || outcome.SchemaVersion != 1 || outcome.RunID != runID {
		return nil
	}
	return &outcome
}
