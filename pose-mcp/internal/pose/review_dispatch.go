package pose

// Delegated review dispatch (spec pose-delegated-review-dispatch, roadmap
// delegated-review).
//
// A run hands the brief of a sealed bundle to a configured reviewer command —
// another vendor's agent, typically — on a disposable worktree at the sealed
// commit, and records what happened. It never records an attestation: turning
// a run's conclusion into one is the engine's verified step (spec
// pose-delegated-review-capability). A run that changed files, exceeded its
// time or budget, or failed is recorded as failed, never as a review.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"
)

// ReviewersPolicySchemaVersion versions .pose/policy/reviewers.json.
const ReviewersPolicySchemaVersion = 1

const reviewRunsDir = ".pose/review-runs"

// ReviewerAdapter is one configured reviewer command.
type ReviewerAdapter struct {
	// Command is the argv; "{output}" is replaced by a file the reviewer may
	// write its final message to. The brief arrives on stdin and the working
	// directory is the disposable worktree.
	Executor         string           `json:"executor,omitempty"`
	Command          []string         `json:"command"`
	Vendor           string           `json:"vendor"`
	Model            string           `json:"model"`
	Timeout          string           `json:"timeout,omitempty"`
	MaxRunsPerBundle int              `json:"max_runs_per_bundle,omitempty"`
	Limit            *ReviewerLimit   `json:"limit,omitempty"`
	Session          *ReviewerSession `json:"session,omitempty"`
	Resume           *ReviewerResume  `json:"resume,omitempty"`
}

// ReviewersPolicy is .pose/policy/reviewers.json.
type ReviewersPolicy struct {
	Provenance    *ReviewRunProvenancePolicy `json:"provenance,omitempty"`
	SchemaVersion int                        `json:"schema_version"`
	Adapters      map[string]ReviewerAdapter `json:"adapters"`
}

// ExampleReviewersPolicy is what POSE suggests and never enables by itself.
func ExampleReviewersPolicy() ReviewersPolicy {
	return ReviewersPolicy{SchemaVersion: ReviewersPolicySchemaVersion, Adapters: map[string]ReviewerAdapter{
		"codex": {Command: []string{"codex", "exec", "-m", "gpt-6.1-sol", "-c", `model_reasoning_effort="medium"`, "-s", "read-only", "--skip-git-repo-check", "-o", "{output}", "-"},
			Vendor: "openai", Model: "gpt-6.1-sol", Timeout: "60m", MaxRunsPerBundle: 3},
		"claude": {Command: []string{"claude", "-p", "--model", "claude-opus-5-5", "--permission-mode", "plan"},
			Vendor: "anthropic", Model: "claude-opus-5-5", Timeout: "60m", MaxRunsPerBundle: 3},
	}}
}

// LoadReviewersPolicy reads the adapters; an absent file means none.
func (s Store) LoadReviewersPolicy() (ReviewersPolicy, bool, error) {
	raw, err := os.ReadFile(filepath.Join(s.Root, ".pose", "policy", "reviewers.json"))
	if errors.Is(err, os.ErrNotExist) {
		return ReviewersPolicy{}, false, nil
	}
	if err != nil {
		return ReviewersPolicy{}, false, err
	}
	var policy ReviewersPolicy
	if err := json.Unmarshal(raw, &policy); err != nil {
		return ReviewersPolicy{}, false, fmt.Errorf("pose: invalid .pose/policy/reviewers.json: %w", err)
	}
	if policy.SchemaVersion != ReviewersPolicySchemaVersion {
		return ReviewersPolicy{}, false, fmt.Errorf("pose: .pose/policy/reviewers.json schema_version %d is not supported", policy.SchemaVersion)
	}
	if err := validateRunProvenancePolicy(policy); err != nil {
		return ReviewersPolicy{}, false, err
	}
	for name, adapter := range policy.Adapters {
		if err := validateContinuityAdapter(adapter); err != nil {
			return ReviewersPolicy{}, false, fmt.Errorf("pose: reviewer adapter %s: %w", name, err)
		}
		if len(adapter.Command) == 0 || adapter.Vendor == "" || adapter.Model == "" {
			return ReviewersPolicy{}, false, fmt.Errorf("pose: reviewer adapter %s needs command, vendor and model", name)
		}
		if adapter.Timeout != "" {
			if _, err := time.ParseDuration(adapter.Timeout); err != nil {
				return ReviewersPolicy{}, false, fmt.Errorf("pose: reviewer adapter %s timeout: %w", name, err)
			}
		}
	}
	return policy, true, nil
}

// ReviewRun is one recorded dispatch.
type ReviewRun struct {
	Provenance          *ReviewRunProvenance `json:"provenance,omitempty"`
	sessionForRedaction string
	Resumes             string              `json:"resumes,omitempty"`
	Interruption        *ReviewInterruption `json:"interruption,omitempty"`
	SessionDigest       string              `json:"session_digest,omitempty"`
	TreeDigest          string              `json:"tree_digest,omitempty"`
	BriefNotes          string              `json:"brief_notes,omitempty"`
	PartialDigest       string              `json:"partial_digest,omitempty"`
	LimitOf             string              `json:"limit_of,omitempty"`
	LedgerVersion       int                 `json:"ledger_version,omitempty"`
	AttemptID           string              `json:"attempt_id,omitempty"`
	Reason              string              `json:"reason,omitempty"`
	SchemaVersion       int                 `json:"schema_version"`
	RunID               string              `json:"run_id"`
	Scope               string              `json:"scope"`
	BundleID            string              `json:"bundle_id"`
	BundleDigest        string              `json:"bundle_digest"`
	SealedCommit        string              `json:"sealed_commit"`
	Kind                string              `json:"kind"`
	BriefDigest         string              `json:"brief_digest"`
	Adapter             string              `json:"adapter"`
	Vendor              string              `json:"vendor"`
	Model               string              `json:"model"`
	StartedAt           string              `json:"started_at"`
	EndedAt             string              `json:"ended_at"`
	ExitCode            int                 `json:"exit_code"`
	Status              string              `json:"status"`
	Failure             string              `json:"failure,omitempty"`
	TranscriptDigest    string              `json:"transcript_digest"`
	TranscriptPath      string              `json:"transcript_path"`
	Output              string              `json:"output"`
	PolicyDigest        string              `json:"policy_digest,omitempty"`
	Supersedes          string              `json:"supersedes,omitempty"`
	SupersededBy        string              `json:"superseded_by,omitempty"`
	Path                string              `json:"-"`
}

// Review run states.
const (
	ReviewRunCompleted   = "completed"
	ReviewRunFailed      = "failed"
	ReviewRunInterrupted = "interrupted"
	ReviewRunStale       = "stale"
)

// ListReviewRuns returns the recorded runs for bundleID ("" for all).
func (s Store) ListReviewRuns(bundleID string) ([]ReviewRun, error) {
	runs, _, err := s.ReadReviewRuns(bundleID)
	return runs, err
}

func (s Store) ReadReviewRuns(bundleID string) ([]ReviewRun, []string, error) {
	dir := filepath.Join(s.Root, filepath.FromSlash(reviewRunsDir))
	// The storage is read where the writer writes it: a link in its place is another place.
	if info, lerr := os.Lstat(dir); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, nil, fmt.Errorf("pose: the review runs directory is a link, not the directory the runs are written to")
	}
	entries, err := ledgerEntries(s.Root, reviewRunsDir)
	if errors.Is(err, os.ErrNotExist) {
		if _, lerr := os.Lstat(dir); lerr == nil {
			return nil, nil, fmt.Errorf("pose: the review runs directory is a link whose target cannot be read")
		}
		return []ReviewRun{}, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	out := []ReviewRun{}
	var lost []string
	readBudget := int64(32 << 20)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if !e.Type().IsRegular() {
			lost = append(lost, e.Name()+" (not a regular file)")
			continue
		}
		info, infoErr := e.Info()
		if infoErr != nil || info.Size() > reviewLedgerMaxBytes {
			lost = append(lost, e.Name()+" (record exceeds bound or is unreadable)")
			continue
		}
		raw, err := readLedgerRawBudget(filepath.Join(dir, e.Name()), &readBudget)
		if err != nil {
			lost = append(lost, e.Name()+" ("+err.Error()+")")
			if errors.Is(err, errReviewLedgerBudget) {
				break
			}
			continue
		}
		var run ReviewRun
		if json.Unmarshal(raw, &run) != nil || run.RunID == "" {
			lost = append(lost, e.Name()+" (not a review run)")
			continue
		}
		if verr := validateReviewRunRecord(e.Name(), run); verr != nil {
			lost = append(lost, e.Name()+" ("+verr.Error()+")")
			continue
		}
		run.Path = filepath.ToSlash(filepath.Join(reviewRunsDir, e.Name()))
		out = append(out, run)
	}
	sort.Slice(out, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339Nano, out[i].StartedAt)
		b, _ := time.Parse(time.RFC3339Nano, out[j].StartedAt)
		if !a.Equal(b) {
			return a.Before(b)
		}
		return out[i].RunID < out[j].RunID
	})
	superseded := map[string]string{}
	for _, run := range out {
		if run.Supersedes != "" {
			superseded[run.Supersedes] = run.RunID
		}
	}
	for i := range out {
		out[i].SupersededBy = superseded[out[i].RunID]
	}
	checked := map[string][]DelegatedReviewAttempt{}
	for _, run := range out {
		if run.LedgerVersion == 1 {
			if _, err := os.Lstat(filepath.Join(s.Root, reviewRunLedgerDir, run.RunID+".json")); err != nil {
				lost = append(lost, "missing run witness "+run.RunID)
			}
		}
		if run.AttemptID != "" {
			attempts, ok := checked[run.BundleID]
			if !ok {
				var aerr error
				attempts, aerr = s.readDelegatedReviewAttempts(run.BundleID, &readBudget)
				if aerr != nil {
					lost = append(lost, aerr.Error())
				}
				checked[run.BundleID] = attempts
			}
			found := false
			for _, a := range attempts {
				if a.AttemptID == run.AttemptID && a.BundleDigest == run.BundleDigest && a.Scope == run.Scope && a.Adapter == run.Adapter && a.Reason == run.Reason {
					found = true
					break
				}
			}
			if !found {
				lost = append(lost, "missing or conflicting attempt reservation "+run.AttemptID)
			}
		}
	}
	findings, ledgerErr := s.reviewRunLedgerFindings("", &readBudget)
	if ledgerErr != nil {
		lost = append(lost, ledgerErr.Error())
	}
	lost = append(lost, findings...)
	lost = append(lost, s.reviewContinuationFindings(out)...)
	if bundleID != "" {
		filtered := []ReviewRun{}
		for _, run := range out {
			if run.BundleID == bundleID {
				filtered = append(filtered, run)
			}
		}
		out = filtered
	}
	return out, uniqueSorted(lost), nil
}

// DispatchPlan is what a dispatch will do.
type DispatchPlan struct {
	provenance          *ReviewRunProvenance
	PendingContinuation *PendingReviewContinuation
	processTimeout      time.Duration
	predecessor         *ReviewRun
	session             string
	notes               string
	Existing            *ReviewRun
	LimitOf             string
	Reason              string
	PolicyDigest        string
	OriginWorktree      string
	Adapter             string
	Config              ReviewerAdapter
	Brief               ReviewBrief
	Bundle              ReviewBundle
	SealedCommit        string
	PriorRuns           int
}

// PlanReviewDispatch resolves the bundle, brief and adapter without running.
func (s Store) PlanReviewDispatch(target, kind, notes, adapter string, reason ...string) (DispatchPlan, error) {
	if len(reason) > 1 {
		return DispatchPlan{}, fmt.Errorf("pose: one rerun reason is accepted")
	}
	why := ""
	if len(reason) == 1 {
		why = strings.TrimSpace(reason[0])
	}
	brief, err := s.RenderReviewBrief(target, kind, notes)
	if err != nil {
		return DispatchPlan{}, err
	}
	bundle, err := s.LoadReviewBundle(brief.BundleID)
	if err != nil {
		return DispatchPlan{}, err
	}
	policy, found, err := s.LoadReviewersPolicy()
	if err != nil {
		return DispatchPlan{}, err
	}
	if !found || len(policy.Adapters) == 0 {
		return DispatchPlan{Brief: brief, Bundle: bundle}, ErrNoReviewerAdapter
	}
	config, ok := policy.Adapters[adapter]
	if !ok {
		names := make([]string, 0, len(policy.Adapters))
		for name := range policy.Adapters {
			names = append(names, name)
		}
		sort.Strings(names)
		return DispatchPlan{}, fmt.Errorf("pose: no reviewer adapter %q; configured: %s", adapter, strings.Join(names, ", "))
	}
	commit := bundle.Payload.Subject.Head
	if commit == "" || exec.Command("git", "-C", s.Root, "cat-file", "-e", commit+"^{commit}").Run() != nil {
		return DispatchPlan{}, fmt.Errorf("pose: the bundle's sealed commit %q is not in this repository", commit)
	}
	origin, err := dispatchWorktreeIdentity(s.Root)
	if err != nil {
		return DispatchPlan{}, err
	}
	plan := DispatchPlan{notes: notes, Reason: why, PolicyDigest: directoryDigestIn(s.fsys(), filepath.Join(s.Root, ".pose", "policy")), OriginWorktree: origin, Adapter: adapter, Config: config, Brief: brief, Bundle: bundle, SealedCommit: commit, PriorRuns: 0}
	runs, losses, err := s.ReadReviewRuns(bundle.BundleID)
	if err != nil {
		return DispatchPlan{}, err
	}
	if len(losses) > 0 {
		return DispatchPlan{}, fmt.Errorf("review history incomplete: %v", losses)
	}
	continued := map[string]bool{}
	for _, r := range runs {
		if r.Resumes != "" {
			continued[r.Resumes] = true
		}
	}
	for _, prior := range runs {
		if prior.Status == ReviewRunInterrupted && prior.SupersededBy == "" && !continued[prior.RunID] && prior.Interruption != nil && prior.Interruption.Cause == "usage-limit" && prior.Adapter != adapter {
			plan.LimitOf, plan.Reason = prior.RunID, "limit-of:"+prior.RunID
		}
	}
	_, currentCount, err := s.reviewAttemptAdmission(plan)
	if err != nil {
		return DispatchPlan{}, err
	}
	plan.PriorRuns = currentCount
	plan.provenance, err = s.prepareRunProvenance(plan, policy)
	if err != nil {
		return DispatchPlan{}, err
	}
	return plan, nil
}

// ErrNoReviewerAdapter means the project configured no reviewer.
var ErrNoReviewerAdapter = errors.New("pose: no reviewer adapter is configured in .pose/policy/reviewers.json")

// RunReviewDispatch executes plan and records the run, failed or completed.
func (s Store) RunReviewDispatch(plan DispatchPlan, now func() time.Time) (result ReviewRun, resultErr error) {
	if plan.PendingContinuation != nil {
		return ReviewRun{}, plan.PendingContinuation
	}
	if plan.Existing != nil {
		if plan.predecessor == nil {
			return ReviewRun{}, fmt.Errorf("existing continuation lacks predecessor")
		}
		fresh, err := s.PlanReviewResume(plan.predecessor.RunID)
		if err != nil {
			return ReviewRun{}, err
		}
		if fresh.Existing == nil {
			return ReviewRun{}, fmt.Errorf("existing continuation disappeared")
		}
		return *fresh.Existing, nil
	}
	run := ReviewRun{Provenance: plan.provenance, BriefNotes: plan.notes, LimitOf: plan.LimitOf, SchemaVersion: 1, Scope: plan.Brief.Scope, BundleID: plan.Bundle.BundleID, BundleDigest: plan.Bundle.BundleDigest,
		SealedCommit: plan.SealedCommit, Kind: plan.Brief.Kind, BriefDigest: plan.Brief.Digest, Adapter: plan.Adapter,
		Vendor: plan.Config.Vendor, Model: plan.Config.Model, Reason: plan.Reason, PolicyDigest: plan.PolicyDigest, StartedAt: now().UTC().Format(time.RFC3339Nano), ExitCode: -1}
	transcript := []byte{}
	finish := func(status, failure string) (ReviewRun, error) {
		run.sessionForRedaction = ""
		run.Status, run.Failure = status, failure
		if status != ReviewRunInterrupted {
			run.Interruption, run.PartialDigest = nil, ""
		}
		run.EndedAt = now().UTC().Format(time.RFC3339Nano)
		return s.recordReviewRun(run, transcript)
	}
	// Recompute semantic freshness at execution time, not just preview time.
	if err := s.checkDispatchOrigin(plan); err != nil {
		return finish(ReviewRunFailed, "the run was not started: "+err.Error())
	}
	var attempt DelegatedReviewAttempt
	if plan.predecessor == nil {
		var err error
		attempt, err = s.reserveDelegatedReviewAttempt(plan, now())
		if err != nil {
			return ReviewRun{}, err
		}
		run.AttemptID, run.Reason = attempt.AttemptID, attempt.Reason
	} else {
		// Re-plan at execution, before the exclusive predecessor claim.
		fresh, err := s.PlanReviewResume(plan.predecessor.RunID)
		if err != nil {
			return ReviewRun{}, err
		}
		if fresh.PendingContinuation != nil {
			return ReviewRun{}, fresh.PendingContinuation
		}
		if fresh.Existing != nil {
			return *fresh.Existing, nil
		}
		plan = fresh
		run.Provenance = fresh.provenance
		run.AttemptID, run.Reason, run.Resumes = fresh.predecessor.AttemptID, fresh.predecessor.Reason, fresh.predecessor.RunID
		run.TreeDigest = fresh.predecessor.TreeDigest
		if err := s.claimReviewContinuation(*fresh.predecessor, now()); err != nil {
			return ReviewRun{}, err
		}
	}
	defer func() {
		if result.RunID != "" {
			var err error
			if plan.predecessor == nil {
				err = s.finishDelegatedReviewAttempt(attempt, result)
			} else {
				err = s.finishReviewContinuation(result)
			}
			if err != nil {
				resultErr = errors.Join(resultErr, fmt.Errorf("attempt result not confirmed: %w", err))
			}
		}
	}()
	tmp, err := os.MkdirTemp("", "pose-review-run-*")
	if err != nil {
		return ReviewRun{}, err
	}
	defer os.RemoveAll(tmp)
	worktree := filepath.Join(tmp, "worktree")
	if out, err := exec.Command("git", "-C", s.Root, "worktree", "add", "--detach", worktree, plan.SealedCommit).CombinedOutput(); err != nil {
		return ReviewRun{}, fmt.Errorf("pose: creating the review worktree: %v: %s", err, strings.TrimSpace(string(out)))
	}
	defer exec.Command("git", "-C", s.Root, "worktree", "remove", "--force", worktree).Run()

	submodules := map[string]bool{}
	for _, entry := range plan.Bundle.Payload.Subject.Entries {
		if entry.Class == "submodule" {
			submodules[reviewBundleEntryPath(entry)] = true
		}
	}
	checkout := s
	checkout.Root, checkout.indexPin = worktree, nil
	if prepared, err := checkout.PrepareReviewBundle(plan.Brief.Scope); err != nil {
		return finish(ReviewRunFailed, "sealed checkout inputs unavailable: "+err.Error())
	} else if blockers := checkoutBlockers(prepared.Blockers); len(blockers) > 0 {
		return finish(ReviewRunFailed, fmt.Sprintf("sealed checkout blocked: %v", blockers))
	} else if differing := sealedInputsThatDiffer(plan.Bundle, prepared, worktree); len(differing) > 0 {
		return finish(ReviewRunFailed, fmt.Sprintf("sealed checkout inputs differ: %v", differing))
	}
	if err := verifyDispatchWorktreeFor(worktree, plan.SealedCommit, submodules); err != nil {
		return finish(ReviewRunFailed, err.Error())
	}
	if err := verifyBundleMaterialization(worktree, plan.Bundle); err != nil {
		return finish(ReviewRunFailed, err.Error())
	}
	diskBefore, err := dispatchDiskSnapshot(worktree)
	if err != nil {
		return finish(ReviewRunFailed, err.Error())
	}

	rawTree, err := json.Marshal(diskBefore)
	if err != nil {
		return finish(ReviewRunFailed, err.Error())
	}
	if plan.predecessor != nil && digestBytes(rawTree) != plan.predecessor.TreeDigest {
		return finish(ReviewRunFailed, "recreated worktree tree digest differs")
	}
	run.TreeDigest = digestBytes(rawTree)
	outputFile := filepath.Join(tmp, "output.txt")
	command := plan.Config.Command
	if plan.predecessor != nil {
		command = plan.Config.Resume.Command
	}
	argv := make([]string, len(command))
	for i, arg := range command {
		argv[i] = strings.ReplaceAll(strings.ReplaceAll(arg, "{output}", outputFile), "{session}", plan.session)
	}
	timeout := 60 * time.Minute
	if plan.Config.Timeout != "" {
		timeout, _ = time.ParseDuration(plan.Config.Timeout)
	}
	execPath := argv[0]
	if run.Provenance != nil {
		copy := *run.Provenance
		run.Provenance = &copy
		resolved, err := resolveProvenanceExecutable(argv[0], worktree)
		if err != nil {
			return finish(ReviewRunFailed, "provenance executable could not be resolved")
		}
		digest, err := reviewExecutableDigest(resolved)
		if err != nil {
			return finish(ReviewRunFailed, err.Error())
		}
		run.Provenance.ExecutableDigest = digest
		execPath = resolved
		raw, _ := json.Marshal(argv)
		run.Provenance.ArgvDigest = digestBytes(raw)
	}
	if plan.processTimeout > 0 {
		started, err := time.Parse(time.RFC3339Nano, run.StartedAt)
		if err != nil {
			return finish(ReviewRunFailed, "invalid execution start")
		}
		setup := now().Sub(started)
		remaining := plan.processTimeout - setup
		if setup < 0 || remaining <= 0 {
			return finish(ReviewRunFailed, "accumulated wall-clock budget spent before adapter launch")
		}
		if remaining < timeout {
			timeout = remaining
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, execPath, argv[1:]...)
	cmd.Args[0] = argv[0]
	cmd.Dir = worktree
	cmd.Stdin = strings.NewReader(plan.Brief.Text)
	var combined bytes.Buffer
	cmd.Stdout, cmd.Stderr = &combined, &combined
	// Adapters such as `codex exec` and `claude -p` start children; killing
	// only the parent left a child holding stdout, so the timeout never fired
	// (found in review). The run gets its own process group, the whole group
	// is killed on timeout and after the run, and pipes are released at most
	// a few seconds after the kill.
	isolateProcessGroup(cmd)
	cmd.WaitDelay = 5 * time.Second
	var runErr error
	if run.Provenance == nil {
		runErr = cmd.Run()
	} else {
		digest, n, complete, started, err := runProvenanceCommand(cmd, plan.Brief.Text)
		runErr = err
		run.Provenance.WireDigest, run.Provenance.WireBytes, run.Provenance.WireComplete = digest, n, complete
		if started {
			run.Provenance.ExecutionState = "started"
		}
	}
	killProcessGroup(cmd)
	// The adapter exited and a child it left still held the output when
	// WaitDelay expired: the run finished, its leftover is killed above, and
	// the exit status decides (found in review).
	if errors.Is(runErr, exec.ErrWaitDelay) && cmd.ProcessState != nil && cmd.ProcessState.Success() {
		runErr = nil
	}
	transcript = combined.Bytes()
	if cmd.ProcessState != nil {
		run.ExitCode = cmd.ProcessState.ExitCode()
	}
	if raw, err := os.ReadFile(outputFile); err == nil && len(bytes.TrimSpace(raw)) > 0 {
		run.Output = string(raw)
		if run.Provenance != nil {
			run.Provenance.OutputComplete = true
			run.Provenance.OutputSource = "file"
		}
	} else {
		run.Output = tail(string(transcript), 20000)
		if run.Provenance != nil {
			run.Provenance.OutputComplete = len(transcript) > 0 && len(transcript) <= 20000
			run.Provenance.OutputSource = "transcript-tail"
		}
	}
	if run.Provenance != nil {
		run.Provenance.RawTranscriptDigest = digestBytes(transcript)
		run.Provenance.RawOutputDigest = digestBytes([]byte(run.Output))
	}
	if runErr != nil && cmd.ProcessState == nil {
		run.Output, transcript = "", nil
		if run.Provenance != nil {
			run.Provenance.OutputComplete = false
		}
		return finish(ReviewRunFailed, "reviewer command could not start")
	}
	status, failure, captureErr := s.classifyReviewInterruption(&run, plan, transcript, runErr, ctx.Err())
	if captureErr != nil {
		run.Output, transcript = "", nil
		if run.Provenance != nil {
			run.Provenance.OutputComplete = false
		}
		return finish(ReviewRunFailed, "adapter capture failed; unredacted output withheld")
	}
	if session := run.sessionForRedaction; session != "" {
		transcript = []byte(strings.ReplaceAll(string(transcript), session, "[private-session]"))
		run.Output = strings.ReplaceAll(run.Output, session, "[private-session]")
	}
	run.sessionForRedaction = ""
	if status != "" {
		// Partial drafts are evidence only after the same containment checks.
		after, err := dispatchDiskSnapshot(worktree)
		if err != nil || !reflect.DeepEqual(diskBefore, after) {
			return finish(ReviewRunFailed, "interrupted adapter changed or obscured the sealed worktree")
		}
		if err := verifyDispatchWorktreeFor(worktree, plan.SealedCommit, submodules); err != nil {
			return finish(ReviewRunFailed, err.Error())
		}
		if err := s.checkDispatchOrigin(plan); err != nil {
			return finish(ReviewRunFailed, "origin changed during interruption: "+err.Error())
		}
		if status == ReviewRunInterrupted {
			run.PartialDigest = digestBytes([]byte(run.Output))
		}
		return finish(status, failure)
	}
	diskAfter, err := dispatchDiskSnapshot(worktree)
	if err != nil {
		return finish(ReviewRunFailed, err.Error())
	}
	if !reflect.DeepEqual(diskBefore, diskAfter) {
		changed := []string{}
		for path, digest := range diskBefore {
			if diskAfter[path] != digest {
				changed = append(changed, path)
			}
		}
		for path := range diskAfter {
			if _, exists := diskBefore[path]; !exists {
				changed = append(changed, path)
			}
		}
		sort.Strings(changed)
		return finish(ReviewRunFailed, fmt.Sprintf("the run changed files in the sealed worktree (including ignored materialization): %q", changed))
	}
	if err := verifyDispatchWorktreeFor(worktree, plan.SealedCommit, submodules); err != nil {
		return finish(ReviewRunFailed, err.Error())
	}
	if err := s.checkDispatchOrigin(plan); err != nil {
		return finish(ReviewRunFailed, "origin or policy changed: "+err.Error())
	}

	published, err := finish(ReviewRunCompleted, "")
	if err != nil {
		return published, err
	}
	if why := s.checkDispatchOrigin(plan); why != nil {
		late := published
		late.RunID, late.Path, late.TranscriptDigest, late.TranscriptPath = "", "", "", ""
		late.Status, late.Failure, late.Supersedes = ReviewRunFailed, why.Error(), published.RunID
		late.EndedAt = now().UTC().Format(time.RFC3339Nano)
		return s.recordReviewRun(late, transcript)
	}
	return published, nil
}

// The adapter has no authenticated producer/output declaration. Until such a
// declaration exists, a directory name cannot authorize regenerated evidence.
func verifyDispatchWorktree(root, commit string) error {
	return verifyDispatchWorktreeFor(root, commit, nil)
}
func verifyDispatchWorktreeFor(root, commit string, submodules map[string]bool) error {
	head, err := snapshotGit(root, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(head)) != commit {
		return fmt.Errorf("the run moved the worktree off the sealed commit")
	}
	paths, err := WorktreeDivergenceFor(root, commit, nil, func(p string) bool { return submodules == nil || submodules[p] })
	if err != nil {
		return err
	}

	if len(paths) > 0 {
		return fmt.Errorf("the run changed files in the sealed worktree: %q", paths)
	}
	return nil
}

func (s Store) checkDispatchOrigin(plan DispatchPlan) error {
	origin, err := dispatchWorktreeIdentity(s.Root)
	if err != nil {
		return err
	}
	if plan.OriginWorktree == "" || origin != plan.OriginWorktree {
		return fmt.Errorf("dispatch origin conflict: worktree identity changed")
	}
	if _, err := dispatchRegularInputPath(s.Root, ".pose/policy/reviewers.json"); err != nil {
		return err
	}
	policy, found, err := s.LoadReviewersPolicy()
	if err != nil {
		return fmt.Errorf("dispatch adapter policy unavailable: %w", err)
	}
	if !found || !reflect.DeepEqual(policy.Adapters[plan.Adapter], plan.Config) {
		return fmt.Errorf("dispatch origin stale: reviewer adapter policy changed")
	}
	if plan.SealedCommit != plan.Bundle.Payload.Subject.Head || plan.Brief.BundleID != plan.Bundle.BundleID || plan.Brief.BundleDigest != plan.Bundle.BundleDigest {
		return fmt.Errorf("dispatch input conflict: plan does not identify its sealed bundle")
	}
	body, ok := strings.CutSuffix(plan.Brief.Text, "\n<!-- brief-digest: "+plan.Brief.Digest+" -->\n")
	if !ok || digestBytes([]byte(body)) != plan.Brief.Digest {
		return fmt.Errorf("dispatch input conflict: brief digest changed")
	}
	if _, err := snapshotGit(s.Root, "rev-parse", "--verify", "HEAD"); err != nil {
		return err
	}
	if _, err := gitChangedPaths(s.Root); err != nil {
		return err
	}
	digest, err := reviewBundlePayloadDigest(plan.Bundle.Payload)
	if err != nil || digest != plan.Bundle.BundleDigest {
		return fmt.Errorf("dispatch input conflict: bundle payload no longer matches its sealed digest")
	}
	// Inspect types and bytes before a freshness reader can follow an input.
	if err := verifyBundleMaterialization(s.Root, plan.Bundle); err != nil {
		return err
	}
	if mismatch := sourceMaterializationDiffers(s.Root, plan.SealedCommit, plan.Bundle.Payload.Subject.Entries); mismatch != "" {
		return fmt.Errorf("source materialization differs: %s", mismatch)
	}
	if reason := s.dispatchNoLongerGoverns(plan); reason != "" {
		return fmt.Errorf("%s", reason)
	}

	return nil
}

func dispatchWorktreeIdentity(root string) (string, error) {
	top, err := snapshotGit(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	topPath, err := filepath.EvalSymlinks(strings.TrimSpace(string(top)))
	if err != nil || abs != topPath {
		return "", fmt.Errorf("dispatch origin conflict: Git selected a different worktree")
	}
	dir, err := snapshotGit(root, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", err
	}
	return string(top) + "\x00" + string(dir), nil
}

func verifyBundleMaterialization(root string, bundle ReviewBundle) error {
	for _, entry := range bundle.Payload.Subject.Entries {
		path := entry.Path
		if entry.NewPath != "" {
			path = entry.NewPath
		}
		if entry.Class == "submodule" {
			full, err := dispatchInputPath(root, path)
			if err != nil {
				return err
			}
			if _, err := dispatchWorktreeIdentity(full); err != nil {
				return err
			}
			head, err := snapshotGit(full, "rev-parse", "--verify", "HEAD")
			if err != nil {
				return err
			}
			if digestBytes([]byte(strings.TrimSpace(string(head)))) != entry.Digest {
				return fmt.Errorf("dispatch input conflict: submodule %q moved", path)
			}
			if err := verifyDispatchWorktree(full, strings.TrimSpace(string(head))); err != nil {
				return err
			}
			continue
		}
		if entry.Action == "removed" {
			if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("dispatch input conflict: removed path %q exists or is unreadable", path)
			}
			continue
		}
		if err := verifyDispatchInput(root, path, entry.Digest, true); err != nil {
			return err
		}
	}
	for _, input := range bundle.Payload.ConsumedInputs {
		switch input.Kind {
		case "governed-config", "review-profile", "rule":
			if err := verifyDispatchInput(root, input.Path, input.Digest, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func dispatchInputPath(root, path string) (string, error) {
	var err error
	root, err = filepath.Abs(root)
	if err != nil {
		return "", &GitSnapshotError{"root", err}
	}
	if path == "" || filepath.IsAbs(path) || filepath.ToSlash(filepath.Clean(path)) != path || path == ".." || strings.HasPrefix(path, "../") {
		return "", fmt.Errorf("dispatch input conflict: unsafe path %q", path)
	}
	full := filepath.Join(root, filepath.FromSlash(path))
	for parent := full; parent != root; parent = filepath.Dir(parent) {
		info, err := os.Lstat(parent)
		if err != nil {
			return "", &GitSnapshotError{"input " + path, err}
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("dispatch input conflict: symlink %q", path)
		}
	}
	return full, nil
}

func verifyDispatchInput(root, path, digest string, normalize bool) error {
	full, err := dispatchRegularInputPath(root, path)
	if err != nil {
		return err
	}
	file, err := os.Open(full)
	if err != nil {
		return &GitSnapshotError{"input " + path, err}
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, (256<<20)+1))
	closeErr := file.Close()
	if readErr != nil {
		return &GitSnapshotError{"input " + path, readErr}
	}
	if closeErr != nil {
		return &GitSnapshotError{"input " + path, closeErr}
	}
	if len(raw) > 256<<20 {
		return &GitSnapshotError{"input " + path, fmt.Errorf("input exceeds 256 MiB")}
	}
	if normalize {
		raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	}
	if digest == "" || digestBytes(raw) != digest {
		return fmt.Errorf("dispatch input conflict: sealed bytes changed for %q", path)
	}
	return nil
}

func dispatchRegularInputPath(root, path string) (string, error) {
	full, err := dispatchInputPath(root, path)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(full)
	if err != nil {
		return "", &GitSnapshotError{"input " + path, err}
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("dispatch input conflict: %q is not a regular file", path)
	}
	return full, nil
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func (s Store) recordReviewRun(run ReviewRun, transcript []byte) (ReviewRun, error) {
	// Identity covers only persisted content, never fields derived by a reader
	// or a previous write of a copied record.
	run.RunID, run.Path, run.TranscriptPath, run.SupersededBy = "", "", "", ""
	run.LedgerVersion = 1
	sum := sha256.Sum256(transcript)
	run.TranscriptDigest = "sha256:" + hex.EncodeToString(sum[:])
	identity, _ := json.Marshal(run)
	idSum := sha256.Sum256(identity)
	run.RunID = "rrn-" + hex.EncodeToString(idSum[:])[:16]
	dir, err := ensureReviewArtifactDirIn(s.fsys(), s.Root, reviewRunsDir, true)
	if err != nil {
		return ReviewRun{}, err
	}
	run.TranscriptPath = filepath.ToSlash(filepath.Join(reviewRunsDir, run.RunID+".transcript"))
	pretty, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return ReviewRun{}, err
	}
	pretty = append(pretty, '\n')
	if len(pretty) > reviewLedgerMaxBytes {
		return ReviewRun{}, fmt.Errorf("review run record exceeds %d-byte bound; nothing published", reviewLedgerMaxBytes)
	}
	if len(transcript) > reviewTranscriptMaxBytes {
		return ReviewRun{}, fmt.Errorf("review transcript exceeds %d-byte bound; nothing published", reviewTranscriptMaxBytes)
	}

	witness := reviewRunWitness{SchemaVersion: 1, RunID: run.RunID, BundleID: run.BundleID, Scope: run.Scope, RecordDigest: digestBytes(pretty), TranscriptDigest: run.TranscriptDigest}
	if err := publishLedgerJSON(s.Root, filepath.ToSlash(filepath.Join(reviewRunLedgerDir, run.RunID+".json")), witness); err != nil {
		return ReviewRun{}, err
	}
	if err := PublishFile(filepath.Join(dir, run.RunID+".transcript"), transcript, 0644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err != nil {
		return ReviewRun{}, err
	}
	if err := PublishFile(filepath.Join(dir, run.RunID+".json"), pretty, 0644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err != nil {
		return ReviewRun{}, err
	}
	run.Path = filepath.ToSlash(filepath.Join(reviewRunsDir, run.RunID+".json"))
	return run, nil
}

func (s Store) dispatchNoLongerGoverns(plan DispatchPlan) string {
	current, err := s.LoadReviewBundle(plan.Bundle.BundleID)
	if err != nil {
		return "stale: the bundle the run reviewed can no longer be read: " + err.Error()
	}
	if current.BundleDigest != plan.Bundle.BundleDigest {
		return "stale: the bundle the run reviewed changed while it ran"
	}
	// The baseline is a digest of a policy that was read whole, and the policy must still
	// read whole and unchanged: a policy that cannot be read now is not an unchanged one.
	if !directoryDigestKnown(plan.PolicyDigest) {
		return "stale: the dispatch has no readable policy baseline (" + plan.PolicyDigest + ")"
	}
	if now := directoryDigestIn(s.fsys(), filepath.Join(s.Root, ".pose", "policy")); now != plan.PolicyDigest {
		return "stale: the policy changed or can no longer be read whole while the run was going (" + now + ")"
	}
	// Every sealed bundle of the scope must have been read: one that cannot be read may be
	// the one that superseded this.
	read, readErr := s.ReadReviewBundles(plan.Brief.Scope)
	if readErr != nil {
		return "stale: the bundles of the scope could not be read: " + readErr.Error()
	}
	if !read.Coverage.CompleteFor(plan.Brief.Scope) {
		return "stale: a sealed bundle of the scope could not be read, so the run cannot be tied to the one that governs it"
	}
	// The bundle governs while the inputs it sealed are the ones the repository has now:
	// the semantic digest of today's preparation, not the order of identifiers.
	prepared, prepErr := s.PrepareReviewBundle(plan.Brief.Scope)
	if prepErr != nil {
		return "stale: the inputs of the scope could not be prepared again: " + prepErr.Error()
	}
	if prepared.BundleDigest != plan.Bundle.BundleDigest {
		return "stale: the bundle was superseded or its inputs changed while the run was going (the scope now prepares " + prepared.BundleDigest + ")"
	}
	// An equal digest is not an eligible preparation: a preparation that carries blockers
	// (an unreadable Git status among them) did not read what it says it read.
	if len(prepared.Blockers) > 0 {
		return "stale: the scope cannot be prepared for review without blockers: " + strings.Join(prepared.Blockers, "; ")
	}
	// The digest of a file is its bytes; the mode the commit recorded is part of what was
	// sealed, and a status that is told to ignore modes cannot be the witness of it.
	// The revision of the source must be readable: an unreadable HEAD is not a HEAD that did not move.
	if hasGitMetadata(s.Root) {
		if _, headErr := ReadGitHead(s.Root); headErr != nil && !errors.Is(headErr, ErrGitUnavailable) {
			return "stale: the revision of the source could not be read: " + headErr.Error()
		}
	}
	if mismatch := sourceMaterializationDiffers(s.Root, plan.SealedCommit, plan.Bundle.Payload.Subject.Entries); mismatch != "" {
		return "stale: the source file " + mismatch + " differs from what the sealed commit records"
	}
	return ""
}

func (s Store) ReviewRunStillGoverns(run ReviewRun) string {
	if run.Status != ReviewRunCompleted {
		return "the run is " + run.Status + ", not completed"
	}
	runs, lost, err := s.ReadReviewRuns("")
	if err != nil {
		return "the runs of the bundle could not be read: " + err.Error()
	}
	if len(lost) > 0 {
		return "a run record could not be read and may supersede this run: " + strings.Join(lost, ", ")
	}
	var persisted *ReviewRun
	for i, other := range runs {
		if other.Supersedes == run.RunID {
			return "the run was superseded by " + other.RunID
		}
		if other.RunID == run.RunID {
			persisted = &runs[i]
		}
	}
	// The copy in hand is not the run: the record is, and it must be there and say the same.
	if persisted == nil {
		return "the record of the run is not in the review runs"
	}
	a, b := *persisted, run
	a.Path, a.SupersededBy, b.Path, b.SupersededBy = "", "", "", ""
	if !reflect.DeepEqual(a, b) {
		return "the record of the run differs from the copy that is being consulted"
	}
	if why := s.reviewRunTranscriptIntact(*persisted); why != "" {
		return why
	}
	if _, err := dispatchWorktreeIdentity(s.Root); err != nil {
		return "origin Git identity unknown: " + err.Error()
	}
	if _, err := ReadGitHead(s.Root); err != nil {
		return "origin HEAD unknown: " + err.Error()
	}
	bundle, err := s.LoadReviewBundle(run.BundleID)
	if err != nil {
		return "the bundle of the run can no longer be read: " + err.Error()
	}
	// The record says what it reviewed; what it says must be what the bundle is.
	if run.BundleDigest != bundle.BundleDigest || run.Scope != bundle.Payload.Scope.Ref || run.SealedCommit != bundle.Payload.Subject.Head {
		return "the run record does not describe the bundle it names (bundle digest, scope or sealed commit differ)"
	}
	if err := verifyBundleMaterialization(s.Root, bundle); err != nil {
		return "persisted review inputs no longer materialize: " + err.Error()
	}
	return s.dispatchNoLongerGoverns(DispatchPlan{Bundle: bundle, Brief: ReviewBrief{Scope: run.Scope}, SealedCommit: run.SealedCommit,
		PolicyDigest: run.PolicyDigest})
}

func validateReviewRunRecord(name string, run ReviewRun) error {
	if name != run.RunID+".json" {
		return fmt.Errorf("the file is not named after the run it holds")
	}
	if run.LedgerVersion != 0 && run.LedgerVersion != 1 {
		return fmt.Errorf("unknown ledger version %d", run.LedgerVersion)
	}
	if run.SchemaVersion != 1 {
		return fmt.Errorf("schema_version %d is not read by this engine", run.SchemaVersion)
	}
	switch run.Status {
	case ReviewRunCompleted, ReviewRunFailed, ReviewRunStale, ReviewRunInterrupted:
	default:
		return fmt.Errorf("status %q is not a state of a run", run.Status)
	}
	if err := validateContinuityRun(run); err != nil {
		return err
	}
	if run.BundleID == "" || run.Scope == "" {
		return fmt.Errorf("the bundle or scope is missing")
	}
	for _, at := range []string{run.StartedAt, run.EndedAt} {
		if _, err := time.Parse(time.RFC3339Nano, at); err != nil {
			return fmt.Errorf("an instant of the run is not RFC3339: %q", at)
		}
	}
	if run.TranscriptPath != filepath.ToSlash(filepath.Join(reviewRunsDir, run.RunID+".transcript")) {
		return fmt.Errorf("the transcript path is not the one the run is written with")
	}
	unsealed := run
	unsealed.RunID, unsealed.TranscriptPath, unsealed.SupersededBy, unsealed.Path = "", "", "", ""
	identity, _ := json.Marshal(unsealed)
	sum := sha256.Sum256(identity)
	if run.RunID != "rrn-"+hex.EncodeToString(sum[:])[:16] {
		return fmt.Errorf("the identity of the run is not derived from its content")
	}
	return nil
}

const reviewTranscriptMaxBytes = 256 << 20

func (s Store) reviewRunTranscriptIntact(run ReviewRun) string {
	full := filepath.Join(s.Root, filepath.FromSlash(run.TranscriptPath))
	info, err := os.Lstat(full)
	if err != nil {
		return "the transcript of the run cannot be inspected: " + err.Error()
	}
	if !info.Mode().IsRegular() {
		return "the transcript of the run is not a regular file"
	}
	if info.Size() > reviewTranscriptMaxBytes {
		return "the transcript exceeds the 256 MiB verification bound"
	}
	file, err := os.Open(full)
	if err != nil {
		return "the transcript of the run cannot be read: " + err.Error()
	}
	opened, statErr := file.Stat()
	if statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		file.Close()
		return "the transcript changed while opening"
	}
	hash := sha256.New()
	n, readErr := io.Copy(hash, io.LimitReader(file, reviewTranscriptMaxBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return fmt.Sprintf("the transcript could not be verified: %v %v", readErr, closeErr)
	}
	if n > reviewTranscriptMaxBytes {
		return "the transcript exceeds the 256 MiB verification bound"
	}
	if "sha256:"+hex.EncodeToString(hash.Sum(nil)) != run.TranscriptDigest {
		return "the transcript of the run is not the one its digest names"
	}
	return ""
}

func removedSides(entry ReviewBundleSubjectEntry) []string {
	switch entry.Action {
	case "removed":
		if p := reviewBundleEntryPath(entry); p != "" {
			return []string{p}
		}
	case "renamed":
		if entry.OldPath != "" {
			return []string{entry.OldPath}
		}
	}
	return nil
}

func sealedInputsThatDiffer(sealed, other ReviewBundle, checkout string) []string {
	key := func(kind, p string) string { return kind + " " + p }
	collect := func(b ReviewBundle) map[string]string {
		out := map[string]string{}
		for _, in := range b.Payload.Scope.Sections {
			out[key(in.Kind, in.Path)] = in.Digest
		}
		for _, in := range b.Payload.ConsumedInputs {
			switch in.Kind {
			case "governed-config", "review-profile", "rule":
				out[key(in.Kind, in.Path)] = in.Digest
			}
		}
		return out
	}
	want, got := collect(sealed), collect(other)
	var differing []string
	for k, digest := range want {
		if got[k] != digest {
			differing = append(differing, k)
		}
	}
	// The implementation the bundle sealed: each file of its subject, as the checkout holds it.
	for _, entry := range sealed.Payload.Subject.Entries {
		if entry.Digest == "" || entry.Action == "removed" || entry.Class == "submodule" {
			continue
		}
		rel := reviewBundleEntryPath(entry)
		if strings.HasPrefix(filepath.ToSlash(rel), pendingFragmentDir) {
			continue // archived since: the ledger, not the path, holds it
		}
		raw, err := os.ReadFile(filepath.Join(checkout, filepath.FromSlash(rel)))
		if err != nil || digestBytes(bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))) != entry.Digest {
			differing = append(differing, "subject "+rel)
		}
	}
	sort.Strings(differing)
	return differing
}

func checkoutBlockers(blockers []string) []string {
	var out []string
	for _, blocker := range blockers {
		if !strings.Contains(blocker, "delivery integrity is unavailable") {
			out = append(out, blocker)
		}
	}
	return out
}

func sourceMaterializationDiffers(root, commit string, entries []ReviewBundleSubjectEntry) string {
	if commit == "" || !hasGitMetadata(root) {
		// No sealed commit, or no repository at all: nothing to compare with (a dispatch has both).
		return ""
	}
	checkedDirs := map[string]bool{}
	for _, entry := range entries {
		if entry.Class == "submodule" {
			continue
		}
		// The absence the bundle sealed is part of what it reviewed: a removed path, and the old
		// side of a rename, must still be absent, whatever Git was told to ignore.
		for _, gone := range removedSides(entry) {
			if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(gone))); err == nil {
				return gone + " (the bundle sealed it as removed, and it is there)"
			}
		}
		if entry.Action == "removed" {
			continue
		}
		rel := reviewBundleEntryPath(entry)
		if linkAbove(root, rel, checkedDirs) {
			return rel + " (a directory above it is a link, or not a directory)"
		}
		out, err := gitCommand("-C", root, "ls-tree", "-z", commit, "--", rel).Output()
		if err != nil {
			return rel + " (its record in the sealed commit could not be read: " + err.Error() + ")"
		}
		meta, _, ok := strings.Cut(strings.TrimSuffix(string(out), "\x00"), "\t")
		fields := strings.Fields(meta)
		if !ok || len(fields) < 3 {
			continue // not in the sealed commit
		}
		mode, object := fields[0], fields[2]
		info, lerr := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
		if lerr != nil {
			return rel + " (it cannot be inspected: " + lerr.Error() + ")"
		}
		switch mode {
		case "120000":
			target, rerr := os.Readlink(filepath.Join(root, filepath.FromSlash(rel)))
			if info.Mode()&os.ModeSymlink == 0 || rerr != nil || gitBlobID("sha1", []byte(target)) != object && gitBlobID("sha256", []byte(target)) != object {
				return rel + " (it is not the link the sealed commit records)"
			}
		default:
			if !info.Mode().IsRegular() {
				return rel + " (it is not a regular file, and the sealed commit records one)"
			}
			if runtime.GOOS != "windows" && (info.Mode().Perm()&0o111 != 0) != (mode == "100755") {
				return rel + " (its executable bit differs)"
			}
		}
	}
	return ""
}
