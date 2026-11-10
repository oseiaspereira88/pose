package pose

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const governanceReadViewGitParserVersion = "pose-governance-git-inputs-v1"

type governanceReadViewGitObservation struct {
	root           string
	args           []string
	files          Files
	ctx            context.Context
	deadline       time.Time
	hasDeadline    bool
	maxOutputBytes int64
	stdout         []byte
	stderr         []byte
	err            error
	errText        string
	producers      map[string]bool
	// pure is set when the output cannot change while the commit it was read at stands: it is a
	// function of object ids, or of headAt, the commit HEAD named before and after the command.
	// The final audit then reruns it only if HEAD moved.
	pure   bool
	headAt string
}

// governanceReadViewGitState captures exact, bounded Git command results for
// one read-view operation. Repeated commands replay the first result; audit
// reruns each exact command and reports any changed or unprovable result.
type governanceReadViewGitState struct {
	op           *governanceReadViewOperation
	mu           sync.Mutex
	observations map[string]*governanceReadViewGitObservation
}

func newGovernanceReadViewGitState(op *governanceReadViewOperation) *governanceReadViewGitState {
	return &governanceReadViewGitState{op: op, observations: map[string]*governanceReadViewGitObservation{}}
}

// captureGovernanceGit runs a Git command through the read-view generation
// when Files belongs to one. Native and preview operations retain their
// existing behavior when there is no governance view.
func captureGovernanceGit(files Files, root, producer string, args ...string) ([]byte, error) {
	return captureGovernanceGitBounded(files, root, producer, governanceReadViewMaxInputBytes, args...)
}

// CaptureGovernanceGitIn executes one bounded raw-history Git command through
// the operation's captured read view when Files carries one. maxOutputBytes
// bounds combined stdout and stderr before bytes are retained. Git commands
// use gitCommand, which disables replacement refs and grafts.
func CaptureGovernanceGitIn(files Files, root, producer string, maxOutputBytes int64, args ...string) ([]byte, error) {
	if maxOutputBytes <= 0 || maxOutputBytes > governanceReadViewMaxInputBytes {
		return nil, fmt.Errorf("pose: captured Git output limit must be between 1 and %d bytes", governanceReadViewMaxInputBytes)
	}
	return captureGovernanceGitBounded(files, root, producer, maxOutputBytes, args...)
}

// CaptureGovernanceRawHistoryGitIn names the history semantics explicitly for
// callers whose result depends on raw commit ancestry or object contents.
func CaptureGovernanceRawHistoryGitIn(files Files, root, producer string, maxOutputBytes int64, args ...string) ([]byte, error) {
	return CaptureGovernanceGitIn(files, root, producer, maxOutputBytes, args...)
}

// CaptureGovernanceGitContextIn is CaptureGovernanceGitIn under ctx. The same
// context is retained for the final audit rerun.
func CaptureGovernanceGitContextIn(files Files, ctx context.Context, root, producer string, maxOutputBytes int64, args ...string) ([]byte, error) {
	if maxOutputBytes <= 0 || maxOutputBytes > governanceReadViewMaxInputBytes {
		return nil, fmt.Errorf("pose: captured Git output limit must be between 1 and %d bytes", governanceReadViewMaxInputBytes)
	}
	return captureGovernanceGitBoundedContext(files, ctx, root, producer, maxOutputBytes, args...)
}

// CaptureGovernanceRawHistoryGitContextIn names raw history semantics for
// context-bound callers that inspect commit ancestry or object contents.
func CaptureGovernanceRawHistoryGitContextIn(files Files, ctx context.Context, root, producer string, maxOutputBytes int64, args ...string) ([]byte, error) {
	return CaptureGovernanceGitContextIn(files, ctx, root, producer, maxOutputBytes, args...)
}

func captureGovernanceGitBounded(files Files, root, producer string, maxOutputBytes int64, args ...string) ([]byte, error) {
	return captureGovernanceGitBoundedContext(files, context.Background(), root, producer, maxOutputBytes, args...)
}

func captureGovernanceGitBoundedContext(files Files, ctx context.Context, root, producer string, maxOutputBytes int64, args ...string) ([]byte, error) {
	files = FilesOrNative(files)
	view, viewFiles := governanceReadViewFromFiles(files)
	if view == nil {
		return runGovernanceGitCommandContext(ctx, files, root, args, maxOutputBytes)
	}
	if view.operation == nil {
		view.markUnknown(producer, "Git input has no operation-scoped generation vector")
		return runGovernanceGitCommandContext(ctx, files, root, args, maxOutputBytes)
	}
	if viewFiles != nil {
		project := viewFiles.project
		if target := view.route(root, producer, project); target == nil {
			view.markUnknown(producer, "Git input is outside the authorized root generation vector")
			return nil, errGovernanceReadViewOutsideRoot
		} else {
			view = target
		}
	}
	return view.operation.captureFilesBoundedContext(files, ctx, root, producer, maxOutputBytes, args...)
}

func captureGovernanceSourceRevision(files Files, root, producer string) (string, error) {
	producer = governanceProducerFromFiles(files, producer)
	raw, err := captureGovernanceGit(files, root, producer, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return "", err
	}
	revision := strings.TrimSpace(string(raw))
	if revision == "" {
		if view, _ := governanceReadViewFromFiles(files); view != nil {
			view.markUnknown(producer, "source revision binding is unavailable")
		}
		return "", fmt.Errorf("pose: Git returned no source revision")
	}
	return revision, nil
}

func readGitStatusGovernedIn(files Files, root, producer string, pathspecs ...string) ([]GitStatusEntry, error) {
	producer = governanceProducerFromFiles(files, producer)
	args := []string{"status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=none"}
	if len(pathspecs) > 0 {
		args = append(args, "--")
		args = append(args, pathspecs...)
	}
	out, err := captureGovernanceGit(files, root, producer, args...)
	if err != nil {
		if hasGitMetadataIn(FilesOrNative(files), root) {
			markGovernanceReadUnknownForProject(files, root, "", producer, "Git status could not be read in this repository")
		} else {
			return nil, fmt.Errorf("%w: %v", ErrGitUnavailable, err)
		}
		return nil, err
	}
	entries, err := parseGitStatusEntries(out)
	if err != nil {
		return nil, err
	}
	if preview := FilesOrNative(files).Preview(); preview != nil {
		return projectGitStatus(preview, root, entries, pathspecs)
	}
	return entries, nil
}

func gitChangedPathsGovernedIn(files Files, root, producer string) ([]string, error) {
	entries, err := readGitStatusGovernedIn(files, root, producer)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(entries)*2)
	for _, entry := range entries {
		if untrackedAuthorityTransferLock(entry) {
			continue
		}
		paths = append(paths, entry.Path)
		if entry.OrigPath != "" {
			paths = append(paths, entry.OrigPath)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func commitChangedPathsGovernedIn(files Files, root, producer, commit string) (map[string]bool, bool) {
	producer = governanceProducerFromFiles(files, producer)
	// Each governed read is captured and audited, so it is not served from a cache:
	// the read is bounded instead, in output bytes and in paths (spec
	// pose-bounded-isolated-graph-caches R4).
	raw, err := captureGovernanceGitBounded(files, root, producer, commitPathsMaxOutput, "diff-tree", "--root", "--no-commit-id", "--name-only", "--no-renames", "-r", "-z", commit)
	var paths map[string]bool
	if err == nil {
		paths, err = parseCommitPaths(raw)
	}
	if err != nil {
		if hasGitMetadataIn(FilesOrNative(files), root) {
			markGovernanceReadUnknownForProject(files, root, "", producer, "commit path inventory could not be read from Git")
		}
		return nil, false
	}
	return paths, true
}

func dispatchWorktreeIdentityGoverned(files Files, root, producer string) (string, error) {
	producer = governanceProducerFromFiles(files, producer)
	top, err := captureGovernanceGit(files, root, producer, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	abs, err = FilesOrNative(files).EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	topPath, err := FilesOrNative(files).EvalSymlinks(strings.TrimSpace(string(top)))
	if err != nil || filepath.Clean(abs) != filepath.Clean(topPath) {
		return "", fmt.Errorf("dispatch origin conflict: Git selected a different worktree")
	}
	gitDir, err := captureGovernanceGit(files, root, producer, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", err
	}
	return string(top) + "\x00" + string(gitDir), nil
}

func governanceReadViewFromFiles(files Files) (*governanceReadView, *governanceReadViewFiles) {
	for depth := 0; depth < maxGovernanceFilesWrappers; depth++ {
		files = FilesOrNative(files)
		if viewFiles, ok := files.(*governanceReadViewFiles); ok {
			return viewFiles.view, viewFiles
		}
		if provider, ok := files.(interface{ GovernanceReadView() *governanceReadView }); ok {
			if view := provider.GovernanceReadView(); view != nil {
				return view, nil
			}
		}
		underlying, ok := files.(interface{ UnderlyingFiles() Files })
		if !ok {
			return nil, nil
		}
		files = underlying.UnderlyingFiles()
	}
	return nil, nil
}

// capture is also used while an authorized child root is being opened, before
// a producer Files wrapper for that root exists. Root membership is checked
// against the operation's already-bound roots.
func (op *governanceReadViewOperation) capture(root, producer string, args ...string) ([]byte, error) {
	return op.captureFiles(NativeFiles(), root, producer, args...)
}

func (op *governanceReadViewOperation) captureFiles(files Files, root, producer string, args ...string) ([]byte, error) {
	return op.captureFilesBounded(files, root, producer, governanceReadViewMaxInputBytes, args...)
}

func (op *governanceReadViewOperation) captureFilesBounded(files Files, root, producer string, maxOutputBytes int64, args ...string) ([]byte, error) {
	return op.captureFilesBoundedContext(files, context.Background(), root, producer, maxOutputBytes, args...)
}

func (op *governanceReadViewOperation) captureFilesBoundedContext(files Files, ctx context.Context, root, producer string, maxOutputBytes int64, args ...string) ([]byte, error) {
	if op == nil {
		return runGovernanceGitCommandContext(ctx, files, root, args, maxOutputBytes)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		op.primary.markUnknown(producer, "Git root identity could not be resolved")
		return nil, err
	}
	absolute = filepath.Clean(absolute)
	op.mu.Lock()
	authorized := false
	var target *governanceReadView
	for _, entry := range op.roots {
		if sameLexicalRoot(entry.root, absolute) {
			authorized = true
			target = entry.view
			break
		}
	}
	state := op.git
	if state == nil {
		state = newGovernanceReadViewGitState(op)
		op.git = state
	}
	op.mu.Unlock()
	if !authorized {
		op.primary.markUnknown(producer, "Git command root is outside the authorized root generation vector")
		return nil, errGovernanceReadViewOutsideRoot
	}
	if target == nil {
		target = op.primary
	}
	if err := target.begin(); err != nil {
		return nil, err
	}
	defer target.end()
	return state.capture(files, ctx, absolute, producer, maxOutputBytes, args...)
}

func (state *governanceReadViewGitState) capture(files Files, ctx context.Context, root, producer string, maxOutputBytes int64, args ...string) ([]byte, error) {
	if state == nil || state.op == nil {
		return runGovernanceGitCommandContext(ctx, files, root, args, maxOutputBytes)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	deadline, hasDeadline := ctx.Deadline()
	auditContext := context.WithoutCancel(ctx)
	key := governanceGitObservationKey(root, args, maxOutputBytes)
	state.mu.Lock()
	if prior := state.observations[key]; prior != nil {
		prior.producers[producer] = true
		stdout, err := append([]byte(nil), prior.stdout...), prior.err
		state.mu.Unlock()
		return stdout, err
	}
	metadataCost := int64(len(root) + len(key) + len(producer) + 128)
	if !state.op.reserveRecord(metadataCost) {
		state.mu.Unlock()
		state.markUnknownForRoot(root, producer, "read view exceeded its shared record or metadata budget while capturing Git")
		return nil, errGovernanceReadViewBudget
	}
	limit := maxOutputBytes
	if state.op.remainingInput() < limit {
		limit = state.op.remainingInput()
	}
	if limit <= 0 {
		state.mu.Unlock()
		state.markUnknownForRoot(root, producer, "read view exceeded its Git input byte budget")
		return nil, errGovernanceReadViewBudget
	}
	byObject, byHead := governancePureGitRead(args)
	headBefore := ""
	if byHead {
		headBefore = governanceHeadAt(ctx, files, root)
	}
	// A process that turned the resident memo on answers a pure read from it; the bytes are
	// admitted against the operation's budget as a fresh read's are.
	var stdout, stderr []byte
	var runErr error
	memoKey, memoized := "", false
	memo := residentGit.Load()
	if memo != nil && (byObject || byHead && headBefore != "") {
		if key, ok := residentGitKey(root, headBefore, args); ok {
			memoKey = key
			if hit, found := memo.get(key); found {
				if int64(len(hit)) > limit || !state.op.reserveInput(int64(len(hit))) {
					state.mu.Unlock()
					state.markUnknownForRoot(root, producer, "Git command output exceeded the read-view input byte budget")
					return nil, errGovernanceReadViewBudget
				}
				stdout, memoized = hit, true
			}
		}
	}
	if !memoized {
		stdout, stderr, runErr = runGovernanceGitCommandOutputAdmittedContext(ctx, files, root, args, limit, state.op.reserveInput)
	}
	if errors.Is(runErr, errGovernanceReadViewBudget) {
		state.mu.Unlock()
		state.markUnknownForRoot(root, producer, "Git command output exceeded the read-view input byte budget")
		return nil, errGovernanceReadViewBudget
	}
	observation := &governanceReadViewGitObservation{
		root: root, args: append([]string(nil), args...), files: files, ctx: auditContext, deadline: deadline, hasDeadline: hasDeadline, maxOutputBytes: maxOutputBytes,
		stdout: append([]byte(nil), stdout...), stderr: append([]byte(nil), stderr...), err: runErr, errText: errorText(runErr),
		producers: map[string]bool{producer: true},
	}
	if runErr == nil && len(stderr) == 0 {
		switch {
		case byObject:
			observation.pure = true
		case memoized && byHead:
			// The commit is the one the memo's key names; nothing ran, so there is no second read.
			observation.pure, observation.headAt = true, headBefore
		case byHead && headBefore != "" && governanceHeadAt(ctx, files, root) == headBefore:
			observation.pure, observation.headAt = true, headBefore
		}
		if memoKey != "" && !memoized && observation.pure {
			memo.put(memoKey, stdout)
		}
	}
	state.observations[key] = observation
	state.mu.Unlock()
	if isGovernanceGitHeadCommand(args) {
		if runErr != nil {
			if !state.op.mayOmitPrimaryRevision(root) {
				state.markUnknownForRoot(root, producer, "source revision could not be read from Git")
			}
		} else {
			revision := strings.TrimSpace(string(stdout))
			if revision == "" && state.op.mayOmitPrimaryRevision(root) {
				return append([]byte(nil), stdout...), runErr
			}
			state.bindRevision(root, revision)
		}
	}
	return append([]byte(nil), stdout...), runErr
}

func isGovernanceGitHeadCommand(args []string) bool {
	return len(args) == 3 && args[0] == "rev-parse" && args[1] == "--verify" && args[2] == "HEAD"
}

func governanceGitObservationKey(root string, args []string, maxOutputBytes int64) string {
	encoded, _ := json.Marshal(args)
	return filepath.Clean(root) + "\x00" + string(encoded) + "\x00" + fmt.Sprint(maxOutputBytes)
}

func (state *governanceReadViewGitState) bindRevision(root, revision string) {
	if revision == "" || state == nil || state.op == nil {
		state.markUnknownForRoot(root, "projection", "source revision binding is unavailable")
		return
	}
	state.op.mu.Lock()
	for _, entry := range state.op.roots {
		if sameLexicalRoot(entry.root, root) {
			entry.revision = revision
			state.op.mu.Unlock()
			return
		}
	}
	state.op.mu.Unlock()
	state.markUnknownForRoot(root, "projection", "source revision root is outside the authorized vector")
}

func (state *governanceReadViewGitState) markUnknownForRoot(root, producer, reason string) {
	if state == nil || state.op == nil {
		return
	}
	state.op.mu.Lock()
	var view *governanceReadView
	for _, entry := range state.op.roots {
		if sameLexicalRoot(entry.root, root) {
			view = entry.view
			break
		}
	}
	state.op.mu.Unlock()
	if view == nil {
		view = state.op.primary
	}
	view.markUnknown(producer, reason)
}

func (state *governanceReadViewGitState) audit() governanceReadViewAudit {
	result := governanceReadViewAudit{ChangedProducers: map[string]bool{}, UnknownProducers: map[string][]string{}}
	if state == nil {
		return result
	}
	state.mu.Lock()
	observations := make([]governanceReadViewGitObservation, 0, len(state.observations))
	for _, item := range state.observations {
		copy := *item
		copy.args = append([]string(nil), item.args...)
		copy.stdout = append([]byte(nil), item.stdout...)
		copy.stderr = append([]byte(nil), item.stderr...)
		copy.producers = cloneBoolMap(item.producers)
		observations = append(observations, copy)
	}
	state.mu.Unlock()
	sort.Slice(observations, func(i, j int) bool {
		left := governanceGitObservationKey(observations[i].root, observations[i].args, observations[i].maxOutputBytes)
		right := governanceGitObservationKey(observations[j].root, observations[j].args, observations[j].maxOutputBytes)
		return left < right
	})
	remaining := int64(governanceReadViewMaxInputBytes)
	heads := map[string]string{}
	for _, item := range observations {
		if item.pure {
			if item.headAt == "" {
				continue
			}
			now, read := heads[item.root]
			if !read {
				now = governanceHeadAt(item.ctx, item.files, item.root)
				heads[item.root] = now
			}
			if now == item.headAt {
				continue
			}
		}
		if remaining <= 0 {
			for producer := range item.producers {
				result.UnknownProducers[producer] = appendUnique(result.UnknownProducers[producer], "Git final audit exceeded its shared byte budget")
			}
			continue
		}
		limit := item.maxOutputBytes
		if remaining < limit {
			limit = remaining
		}
		auditContext := item.ctx
		cancelAuditContext := func() {}
		if item.hasDeadline {
			auditContext, cancelAuditContext = context.WithDeadline(auditContext, item.deadline)
		}
		stdout, stderr, err := governanceAuditGit(auditContext, item.files, item.root, item.args, limit)
		cancelAuditContext()
		if errors.Is(err, errGovernanceReadViewBudget) {
			command := "unknown"
			if len(item.args) > 0 {
				command = item.args[0]
			}
			for producer := range item.producers {
				result.UnknownProducers[producer] = appendUnique(result.UnknownProducers[producer], "Git command "+command+" exceeded its bounded final-audit budget")
			}
			continue
		}
		remaining -= int64(len(stdout) + len(stderr) + 1)
		if errorText(err) != item.errText || !bytes.Equal(stdout, item.stdout) || !bytes.Equal(stderr, item.stderr) {
			for producer := range item.producers {
				result.ChangedProducers[producer] = true
			}
			rootDigest := digestBytes([]byte(filepath.Clean(item.root)))
			argsDigest := digestBytes([]byte(strings.Join(item.args, "\x00")))
			result.ChangedPaths = append(result.ChangedPaths, "git:"+strings.TrimPrefix(rootDigest, "sha256:")+":"+strings.TrimPrefix(argsDigest, "sha256:"))
		}
	}
	result.ChangedPaths = uniqueSorted(result.ChangedPaths)
	for producer := range result.UnknownProducers {
		result.UnknownProducers[producer] = uniqueSorted(result.UnknownProducers[producer])
	}
	return result
}

func (state *governanceReadViewGitState) identityDigest() string {
	if state == nil {
		return ""
	}
	type identity struct {
		RootDigest   string   `json:"root_digest"`
		ArgsDigest   string   `json:"args_digest"`
		OutputLimit  int64    `json:"output_limit"`
		OutputDigest string   `json:"output_digest"`
		ErrorDigest  string   `json:"error_digest,omitempty"`
		Producers    []string `json:"producers"`
	}
	state.mu.Lock()
	items := make([]identity, 0, len(state.observations))
	for _, item := range state.observations {
		args, _ := json.Marshal(item.args)
		items = append(items, identity{
			RootDigest: digestBytes([]byte(filepath.Clean(item.root))), ArgsDigest: digestBytes(args),
			OutputLimit:  item.maxOutputBytes,
			OutputDigest: digestBytes(append(append([]byte(nil), item.stdout...), item.stderr...)),
			ErrorDigest:  errorDigest(item.errText), Producers: sortedBoolKeys(item.producers),
		})
	}
	state.mu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		if items[i].RootDigest != items[j].RootDigest {
			return items[i].RootDigest < items[j].RootDigest
		}
		if items[i].ArgsDigest != items[j].ArgsDigest {
			return items[i].ArgsDigest < items[j].ArgsDigest
		}
		return items[i].OutputLimit < items[j].OutputLimit
	})
	raw, _ := json.Marshal(items)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func runGovernanceGitCommand(files Files, root string, args []string, limit int64) ([]byte, error) {
	stdout, _, err := runGovernanceGitCommandOutput(files, root, args, limit)
	return stdout, err
}

func runGovernanceGitCommandContext(ctx context.Context, files Files, root string, args []string, limit int64) ([]byte, error) {
	stdout, _, err := runGovernanceGitCommandOutputContext(ctx, files, root, args, limit)
	return stdout, err
}

func runGovernanceGitCommandOutput(files Files, root string, args []string, limit int64) ([]byte, []byte, error) {
	return runGovernanceGitCommandOutputAdmitted(files, root, args, limit, nil)
}

func runGovernanceGitCommandOutputContext(ctx context.Context, files Files, root string, args []string, limit int64) ([]byte, []byte, error) {
	return runGovernanceGitCommandOutputAdmittedContext(ctx, files, root, args, limit, nil)
}

// runGovernanceGitCommandOutputAdmitted admits each stream chunk before the
// bounded output buffer retains it. The operation callback is shared with
// Files reads, so concurrent producers cannot spend the same remaining bytes.
func runGovernanceGitCommandOutputAdmitted(files Files, root string, args []string, limit int64, admit func(int64) bool) ([]byte, []byte, error) {
	return runGovernanceGitCommandOutputAdmittedContext(context.Background(), files, root, args, limit, admit)
}

// runGovernanceGitCommandOutputAdmittedContext admits each stream chunk before
// the bounded output buffer retains it and executes Git under ctx.
func runGovernanceGitCommandOutputAdmittedContext(ctx context.Context, files Files, root string, args []string, limit int64, admit func(int64) bool) ([]byte, []byte, error) {
	if limit < 0 {
		return nil, nil, errGovernanceReadViewBudget
	}
	files = FilesOrNative(files)
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := gitCommandContext(ctx, append([]string{"--no-replace-objects", "-C", root}, args...)...)
	cmd.WaitDelay = 2 * time.Second
	cmd.Env = append(cmd.Env, "GIT_OPTIONAL_LOCKS=0")
	output := &boundedGovernanceGitOutput{limit: limit, admit: admit}
	cmd.Stdout = governanceGitStream{output: output, stderr: false}
	cmd.Stderr = governanceGitStream{output: output, stderr: true}
	err := cmd.Run()
	stdout, stderr, exceeded := output.values()
	if exceeded {
		return nil, nil, errGovernanceReadViewBudget
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			exit.Stderr = append([]byte(nil), stderr...)
		}
		return stdout, stderr, &GitSnapshotError{Operation: strings.Join(args, " "), Cause: err}
	}
	if len(stderr) > 0 {
		detail := strings.TrimSpace(string(stderr))
		if detail == "" {
			detail = "Git wrote unclassified stderr output"
		}
		return stdout, stderr, &GitSnapshotError{Operation: strings.Join(args, " "), Cause: errors.New(detail)}
	}
	return stdout, stderr, nil
}

type boundedGovernanceGitOutput struct {
	mu       sync.Mutex
	limit    int64
	used     int64
	admit    func(int64) bool
	stdout   bytes.Buffer
	stderr   bytes.Buffer
	exceeded bool
}

type governanceGitStream struct {
	output *boundedGovernanceGitOutput
	stderr bool
}

func (stream governanceGitStream) Write(data []byte) (int, error) {
	output := stream.output
	output.mu.Lock()
	defer output.mu.Unlock()
	if output.exceeded || int64(len(data)) > output.limit-output.used {
		output.exceeded = true
		return 0, errGovernanceReadViewBudget
	}
	if output.admit != nil && !output.admit(int64(len(data))) {
		output.exceeded = true
		return 0, errGovernanceReadViewBudget
	}
	var err error
	if stream.stderr {
		_, err = output.stderr.Write(data)
	} else {
		_, err = output.stdout.Write(data)
	}
	if err == nil {
		output.used += int64(len(data))
	}
	return len(data), err
}

func (output *boundedGovernanceGitOutput) values() ([]byte, []byte, bool) {
	output.mu.Lock()
	defer output.mu.Unlock()
	return append([]byte(nil), output.stdout.Bytes()...), append([]byte(nil), output.stderr.Bytes()...), output.exceeded
}
