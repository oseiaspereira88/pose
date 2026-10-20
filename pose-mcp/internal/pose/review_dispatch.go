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
	OriginWorktree string
	Adapter        string
	Config         ReviewerAdapter
	Brief          ReviewBrief
	Bundle         ReviewBundle
	SealedCommit   string
	PriorRuns      int
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
	origin, err := dispatchWorktreeIdentity(s.Root)
	if err != nil {
		return DispatchPlan{}, err
	}
	return DispatchPlan{OriginWorktree: origin, Adapter: adapter, Config: config, Brief: brief, Bundle: bundle, SealedCommit: commit, PriorRuns: count}, nil
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
	// Recompute semantic freshness at execution time, not just preview time.
	if err := s.checkDispatchOrigin(plan); err != nil {
		return finish(ReviewRunFailed, err.Error())
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

	if err := verifyDispatchWorktree(worktree, plan.SealedCommit); err != nil {
		return finish(ReviewRunFailed, err.Error())
	}
	if err := verifyBundleMaterialization(worktree, plan.Bundle); err != nil {
		return finish(ReviewRunFailed, err.Error())
	}
	diskBefore, err := dispatchDiskSnapshot(worktree)
	if err != nil {
		return finish(ReviewRunFailed, err.Error())
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
	if err := verifyDispatchWorktree(worktree, plan.SealedCommit); err != nil {
		return finish(ReviewRunFailed, err.Error())
	}
	if err := s.checkDispatchOrigin(plan); err != nil {
		return finish(ReviewRunFailed, err.Error())
	}
	return finish(ReviewRunCompleted, "")
}

// The adapter has no authenticated producer/output declaration. Until such a
// declaration exists, a directory name cannot authorize regenerated evidence.
func verifyDispatchWorktree(root, commit string) error {
	head, err := snapshotGit(root, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(head)) != commit {
		return fmt.Errorf("the run moved the worktree off the sealed commit")
	}
	paths, err := gitChangedPaths(root)
	if err != nil {
		return err
	}
	materialized, err := gitMaterializedChanges(root, commit)
	if err != nil {
		return err
	}
	paths = append(paths, materialized...)
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
	bundle, err := s.CurrentReviewBundle(plan.Brief.Scope)
	if err != nil {
		return fmt.Errorf("dispatch origin unavailable: %w", err)
	}
	if bundle == nil || bundle.BundleID != plan.Bundle.BundleID || bundle.BundleDigest != plan.Bundle.BundleDigest {
		return fmt.Errorf("dispatch origin stale: sealed bundle changed or was superseded")
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
