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
	return DispatchPlan{Adapter: adapter, Config: config, Brief: brief, Bundle: bundle, SealedCommit: commit, PriorRuns: count}, nil
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
	// which is not the content under review (found by the first real run).
	if out, _ := exec.Command("git", "-C", worktree, "status", "--porcelain", "--untracked-files=all").Output(); len(bytes.TrimSpace(out)) > 0 {
		changed := []string{}
		for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
			if len(line) < 4 {
				continue
			}
			path := strings.TrimSpace(line[3:])
			if i := strings.Index(path, " -> "); i >= 0 {
				path = path[i+4:]
			}
			if !derivedReviewPath(path) {
				changed = append(changed, path)
			}
		}
		if len(changed) > 0 {
			return finish(ReviewRunFailed, "the run changed files in the sealed worktree: "+strings.Join(changed, " "))
		}
	}
	if head, _ := exec.Command("git", "-C", worktree, "rev-parse", "HEAD").Output(); strings.TrimSpace(string(head)) != plan.SealedCommit {
		return finish(ReviewRunFailed, "the run moved the worktree off the sealed commit")
	}
	return finish(ReviewRunCompleted, "")
}

// derivedReviewPath reports whether path is state POSE regenerates from the
// sealed content rather than part of it.
func derivedReviewPath(path string) bool {
	for _, prefix := range []string{".pose/assessments/", ".pose/state/", ".pose/indexes/", ".pose/results/", ".pose/review-bundles/", ".pose/review-runs/", ".pose/reports/history/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
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
