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
	"os"
	"os/exec"
	"path/filepath"
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
	Command          []string `json:"command"`
	Vendor           string   `json:"vendor"`
	Model            string   `json:"model"`
	Timeout          string   `json:"timeout,omitempty"`
	MaxRunsPerBundle int      `json:"max_runs_per_bundle,omitempty"`
}

// ReviewersPolicy is .pose/policy/reviewers.json.
type ReviewersPolicy struct {
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
	for name, adapter := range policy.Adapters {
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
	SchemaVersion    int    `json:"schema_version"`
	RunID            string `json:"run_id"`
	Scope            string `json:"scope"`
	BundleID         string `json:"bundle_id"`
	BundleDigest     string `json:"bundle_digest"`
	SealedCommit     string `json:"sealed_commit"`
	Kind             string `json:"kind"`
	BriefDigest      string `json:"brief_digest"`
	Adapter          string `json:"adapter"`
	Vendor           string `json:"vendor"`
	Model            string `json:"model"`
	StartedAt        string `json:"started_at"`
	EndedAt          string `json:"ended_at"`
	ExitCode         int    `json:"exit_code"`
	Status           string `json:"status"`
	Failure          string `json:"failure,omitempty"`
	TranscriptDigest string `json:"transcript_digest"`
	TranscriptPath   string `json:"transcript_path"`
	Output           string `json:"output"`
	Path             string `json:"-"`
}

// Review run states.
const (
	ReviewRunCompleted = "completed"
	ReviewRunFailed    = "failed"
	// ReviewRunStale is a run that finished while the bundle, the policy or the inputs it
	// was dispatched against no longer governed the scope: kept as history, not as the
	// current fulfilment.
	ReviewRunStale = "stale"
)

// ListReviewRuns returns the recorded runs for bundleID ("" for all).
func (s Store) ListReviewRuns(bundleID string) ([]ReviewRun, error) {
	dir := filepath.Join(s.Root, filepath.FromSlash(reviewRunsDir))
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []ReviewRun{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []ReviewRun{}
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var run ReviewRun
		if json.Unmarshal(raw, &run) != nil || run.RunID == "" {
			continue
		}
		if bundleID == "" || run.BundleID == bundleID {
			run.Path = filepath.ToSlash(filepath.Join(reviewRunsDir, e.Name()))
			out = append(out, run)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt < out[j].StartedAt })
	return out, nil
}

// DispatchPlan is what a dispatch will do.
type DispatchPlan struct {
	Adapter      string
	Config       ReviewerAdapter
	Brief        ReviewBrief
	Bundle       ReviewBundle
	SealedCommit string
	PriorRuns    int
	// PolicyDigest is the digest of .pose/policy when the dispatch was planned.
	PolicyDigest string
}

// PlanReviewDispatch resolves the bundle, brief and adapter without running.
func (s Store) PlanReviewDispatch(target, kind, notes, adapter string) (DispatchPlan, error) {
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
	policyDigest := directoryDigest(filepath.Join(s.Root, ".pose", "policy"))
	if !directoryDigestKnown(policyDigest) {
		// Nothing to compare the policy with later: a run planned without a baseline
		// could not tell a changed policy from an unchanged one.
		return DispatchPlan{}, fmt.Errorf("pose: the policy could not be read whole (%s); a review dispatch needs a baseline to revalidate against", policyDigest)
	}
	prior, err := s.ListReviewRuns(bundle.BundleID)
	if err != nil {
		return DispatchPlan{}, err
	}
	count := 0
	for _, run := range prior {
		if run.Adapter == adapter {
			count++
		}
	}
	return DispatchPlan{Adapter: adapter, Config: config, Brief: brief, Bundle: bundle, SealedCommit: commit, PriorRuns: count,
		PolicyDigest: policyDigest}, nil
}

// ErrNoReviewerAdapter means the project configured no reviewer.
var ErrNoReviewerAdapter = errors.New("pose: no reviewer adapter is configured in .pose/policy/reviewers.json")

// RunReviewDispatch executes plan and records the run, failed or completed.
func (s Store) RunReviewDispatch(plan DispatchPlan, now func() time.Time) (ReviewRun, error) {
	run := ReviewRun{SchemaVersion: 1, Scope: plan.Brief.Scope, BundleID: plan.Bundle.BundleID, BundleDigest: plan.Bundle.BundleDigest,
		SealedCommit: plan.SealedCommit, Kind: plan.Brief.Kind, BriefDigest: plan.Brief.Digest, Adapter: plan.Adapter,
		Vendor: plan.Config.Vendor, Model: plan.Config.Model, StartedAt: now().UTC().Format(time.RFC3339Nano), ExitCode: -1}
	transcript := []byte{}
	finish := func(status, failure string) (ReviewRun, error) {
		run.Status, run.Failure = status, failure
		run.EndedAt = now().UTC().Format(time.RFC3339Nano)
		return s.recordReviewRun(run, transcript)
	}
	if plan.Config.MaxRunsPerBundle > 0 && plan.PriorRuns >= plan.Config.MaxRunsPerBundle {
		return finish(ReviewRunFailed, fmt.Sprintf("budget exhausted: %d run(s) of %s on this bundle, at most %d", plan.PriorRuns, plan.Adapter, plan.Config.MaxRunsPerBundle))
	}
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
	// The adapter reviews what was sealed: the materialized worktree is compared with the
	// sealed commit before it is handed over (a checkout filter or a hook can change it).
	submodules := map[string]bool{}
	for _, entry := range plan.Bundle.Payload.Subject.Entries {
		if entry.Class == "submodule" {
			submodules[reviewBundleEntryPath(entry)] = true
		}
	}
	if diverged, err := WorktreeDivergenceFor(worktree, plan.SealedCommit, nil, func(p string) bool { return submodules[p] }); err != nil {
		return finish(ReviewRunFailed, "the sealed worktree could not be verified before the run: "+err.Error())
	} else if len(diverged) > 0 {
		return finish(ReviewRunFailed, "the sealed worktree does not hold the sealed content: "+strings.Join(diverged, " "))
	}

	// The checkout is the commit; the bundle is what was sealed. A bundle can hold
	// inputs the commit does not (a requirement, a setting, a submodule pin that was
	// only in the working tree when it was sealed): the reviewer would then read
	// something else than the bundle says it reads.
	checkout := s
	checkout.Root = worktree
	checkout.indexPin = nil
	if prepared, perr := checkout.PrepareReviewBundle(plan.Brief.Scope); perr != nil {
		return finish(ReviewRunFailed, "the inputs the bundle sealed could not be checked in the sealed worktree: "+perr.Error())
	} else if differing := sealedInputsThatDiffer(plan.Bundle, prepared); len(differing) > 0 {
		return finish(ReviewRunFailed, "the sealed worktree does not hold the inputs the bundle sealed: "+strings.Join(differing, " "))
	}

	// And the repository must still be what the bundle sealed before the reviewer is
	// started: running against inputs that already moved reviews nothing current.
	if reason := s.dispatchNoLongerGoverns(plan); reason != "" {
		return finish(ReviewRunFailed, "the run was not started: "+reason)
	}

	outputFile := filepath.Join(tmp, "output.txt")
	argv := make([]string, len(plan.Config.Command))
	for i, arg := range plan.Config.Command {
		argv[i] = strings.ReplaceAll(arg, "{output}", outputFile)
	}
	timeout := 60 * time.Minute
	if plan.Config.Timeout != "" {
		timeout, _ = time.ParseDuration(plan.Config.Timeout)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
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
	runErr := cmd.Run()
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
	} else {
		run.Output = tail(string(transcript), 20000)
	}
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return finish(ReviewRunFailed, "timeout after "+timeout.String())
	case runErr != nil:
		return finish(ReviewRunFailed, "the reviewer command failed: "+runErr.Error())
	}
	// A reviewer that changed the sealed content did not review it. The tools
	// a plan requires regenerate derived state (assessments, indexes, results),
	// which is not the content under review (found by the first real run). The
	// content is compared with the sealed commit's own tree, so a path Git was told to
	// ignore, a link or a replaced type cannot hide a change, and a tree that cannot be
	// read is not a tree that is unchanged (spec pose-git-and-dispatch-snapshot-integrity).
	changed, divergeErr := WorktreeDivergence(worktree, plan.SealedCommit, IsDerivedRegeneration)
	if divergeErr != nil {
		return finish(ReviewRunFailed, "the sealed worktree could not be verified after the run: "+divergeErr.Error())
	}
	if len(changed) > 0 {
		return finish(ReviewRunFailed, "the run changed files in the sealed worktree: "+strings.Join(changed, " "))
	}
	head, headErr := ReadGitHead(worktree)
	if headErr != nil {
		return finish(ReviewRunFailed, "the sealed worktree's HEAD could not be read after the run: "+headErr.Error())
	}
	if head != plan.SealedCommit {
		return finish(ReviewRunFailed, "the run moved the worktree off the sealed commit")
	}
	// What was reviewed must still be what governs the scope: a bundle superseded or a
	// policy changed while the run was going leaves the run as history, never as the
	// current fulfilment.
	if reason := s.dispatchNoLongerGoverns(plan); reason != "" {
		return finish(ReviewRunStale, reason)
	}
	return finish(ReviewRunCompleted, "")
}

// sealedInputsThatDiffer lists the inputs the bundle sealed that a preparation made from
// another checkout reads differently: the sections of the scope (the requirements and
// the rest of the spec) and the configuration, profiles and rules the plan consumed.
// Generated state (evidence, indexes) is not compared: it is not carried by a commit.
func sealedInputsThatDiffer(sealed, other ReviewBundle) []string {
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
	sort.Strings(differing)
	return differing
}

// dispatchNoLongerGoverns says why the bundle a dispatch was planned against no longer
// governs its scope, or "" when it still does.
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
	if now := directoryDigest(filepath.Join(s.Root, ".pose", "policy")); now != plan.PolicyDigest {
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
	return ""
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func (s Store) recordReviewRun(run ReviewRun, transcript []byte) (ReviewRun, error) {
	sum := sha256.Sum256(transcript)
	run.TranscriptDigest = "sha256:" + hex.EncodeToString(sum[:])
	identity, _ := json.Marshal(run)
	idSum := sha256.Sum256(identity)
	run.RunID = "rrn-" + hex.EncodeToString(idSum[:])[:16]
	dir, err := ensureReviewArtifactDir(s.Root, reviewRunsDir, true)
	if err != nil {
		return ReviewRun{}, err
	}
	run.TranscriptPath = filepath.ToSlash(filepath.Join(reviewRunsDir, run.RunID+".transcript"))
	if err := writeNewFile(filepath.Join(dir, run.RunID+".transcript"), transcript); err != nil {
		return ReviewRun{}, err
	}
	raw, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return ReviewRun{}, err
	}
	if err := writeNewFile(filepath.Join(dir, run.RunID+".json"), append(raw, '\n')); err != nil {
		return ReviewRun{}, err
	}
	run.Path = filepath.ToSlash(filepath.Join(reviewRunsDir, run.RunID+".json"))
	return run, nil
}

func writeNewFile(path string, raw []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(raw); err != nil {
		file.Close()
		os.Remove(path)
		return err
	}
	return file.Close()
}
