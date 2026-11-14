package pose

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Streamed reads (spec pose-read-view-streamed-object-digests): a Git object
// read whose bytes a producer only hashes is streamed into the producer and
// the read view retains its digest and length, never its bytes. Streamed
// bytes have their own budget, apart from the retained captured input.
const (
	// governanceReadViewMaxStreamedBytes bounds the bytes one operation
	// streams, and separately the bytes its final audit streams again.
	governanceReadViewMaxStreamedBytes = 8 * 1024 * 1024 * 1024
	// governanceReadViewMaxStreamedRead bounds one streamed read.
	governanceReadViewMaxStreamedRead = 256 * 1024 * 1024
	governanceStreamStderrLimit       = 64 * 1024
)

type governanceReadViewStreamObservation struct {
	root        string
	args        []string
	files       Files
	ctx         context.Context
	deadline    time.Time
	hasDeadline bool
	maxBytes    int64
	digest      string
	length      int64
	// errText is the digest of the read's error text, never the text: a
	// warning on stderr can be up to the stderr bound, and a retained copy
	// per read would escape every budget (review round 2, N2).
	errText   string
	producers map[string]bool
}

// StreamGovernanceRawHistoryGitIn runs a raw-history Git read of an immutable
// object and streams its output into sink. Inside a governance read view the
// view records the output's digest and length as the observation and keeps
// no bytes; reading the same command again in the operation streams it again
// and leaves the producer unknown if it differs. Outside a view it is a
// bounded streaming run.
func StreamGovernanceRawHistoryGitIn(files Files, ctx context.Context, root, producer string, maxBytes int64, sink io.Writer, args ...string) error {
	if maxBytes <= 0 || maxBytes > governanceReadViewMaxStreamedRead {
		return fmt.Errorf("pose: streamed Git output limit must be between 1 and %d bytes", governanceReadViewMaxStreamedRead)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	files = FilesOrNative(files)
	view, viewFiles := governanceReadViewFromFiles(files)
	if view == nil {
		_, _, _, err := runGovernanceGitStreamContext(ctx, files, root, args, maxBytes, nil, sink)
		return err
	}
	if view.operation == nil {
		view.markUnknown(producer, "Git input has no operation-scoped generation vector")
		_, _, _, err := runGovernanceGitStreamContext(ctx, files, root, args, maxBytes, nil, sink)
		return err
	}
	if viewFiles != nil {
		if target := view.route(root, producer, viewFiles.project); target == nil {
			view.markUnknown(producer, "Git input is outside the authorized root generation vector")
			return errGovernanceReadViewOutsideRoot
		} else {
			view = target
		}
	}
	op := view.operation
	absolute, target, state, err := op.gitRoot(root, producer)
	if err != nil {
		return err
	}
	if err := target.begin(); err != nil {
		return err
	}
	defer target.end()
	return state.stream(files, ctx, absolute, producer, maxBytes, sink, args...)
}

func (op *governanceReadViewOperation) reserveStreamed(size int64) bool {
	if op == nil {
		return size >= 0 && size <= governanceReadViewMaxStreamedBytes
	}
	if size < 0 {
		return false
	}
	op.budgetMu.Lock()
	defer op.budgetMu.Unlock()
	if size > governanceReadViewMaxStreamedBytes-op.streamedBytes {
		return false
	}
	op.streamedBytes += size
	return true
}

func (state *governanceReadViewGitState) stream(files Files, ctx context.Context, root, producer string, maxBytes int64, sink io.Writer, args ...string) error {
	deadline, hasDeadline := ctx.Deadline()
	key := governanceGitObservationKey(root, args, maxBytes)
	state.mu.Lock()
	if state.streams == nil {
		state.streams = map[string]*governanceReadViewStreamObservation{}
	}
	prior := state.streams[key]
	if prior == nil {
		metadataCost := int64(len(root) + len(key) + len(producer) + 128)
		if !state.op.reserveRecord(metadataCost) {
			state.mu.Unlock()
			state.markUnknownForRoot(root, producer, "read view exceeded its shared record or metadata budget while streaming Git")
			return errGovernanceReadViewBudget
		}
	}
	state.mu.Unlock()
	digest, length, stderr, runErr := governanceStreamRun(ctx, files, root, args, maxBytes, state.op.reserveStreamed, sink)
	if errors.Is(runErr, errGovernanceReadViewBudget) {
		state.markUnknownForRoot(root, producer, "Git streamed output exceeded the read-view streamed byte budget")
		return errGovernanceReadViewBudget
	}
	_ = stderr
	state.mu.Lock()
	if prior = state.streams[key]; prior != nil {
		prior.producers[producer] = true
		same := prior.digest == digest && prior.length == length && prior.errText == streamErrorDigest(runErr)
		state.mu.Unlock()
		if !same {
			state.markUnknownForRoot(root, producer, "a streamed Git object read differed within the operation")
			return errGovernanceReadViewBudget
		}
		return runErr
	}
	state.streams[key] = &governanceReadViewStreamObservation{
		root: root, args: append([]string(nil), args...), files: files, ctx: context.WithoutCancel(ctx), deadline: deadline, hasDeadline: hasDeadline,
		maxBytes: maxBytes, digest: digest, length: length, errText: streamErrorDigest(runErr), producers: map[string]bool{producer: true},
	}
	state.mu.Unlock()
	return runErr
}

// auditStreams streams every observation again, under its own audit budget,
// and reports a producer changed when the digest, length or error differs.
func (state *governanceReadViewGitState) auditStreams(result *governanceReadViewAudit) {
	state.mu.Lock()
	streams := make([]governanceReadViewStreamObservation, 0, len(state.streams))
	for _, item := range state.streams {
		copy := *item
		copy.args = append([]string(nil), item.args...)
		copy.producers = cloneBoolMap(item.producers)
		streams = append(streams, copy)
	}
	state.mu.Unlock()
	sort.Slice(streams, func(i, j int) bool {
		return governanceGitObservationKey(streams[i].root, streams[i].args, streams[i].maxBytes) < governanceGitObservationKey(streams[j].root, streams[j].args, streams[j].maxBytes)
	})
	var budgetMu sync.Mutex
	remaining := int64(governanceReadViewMaxStreamedBytes)
	admit := func(size int64) bool {
		budgetMu.Lock()
		defer budgetMu.Unlock()
		if size < 0 || size > remaining {
			return false
		}
		remaining -= size
		return true
	}
	// Every streamed read is streamed again: its rendering depends on Git
	// configuration and attributes the operation does not capture, so no
	// read is proven unchanged without it (review round 1, F1 and F3).
	for _, item := range streams {
		auditContext := item.ctx
		cancel := func() {}
		if item.hasDeadline {
			auditContext, cancel = context.WithDeadline(auditContext, item.deadline)
		}
		digest, length, _, err := governanceAuditGitStream(auditContext, item.files, item.root, item.args, item.maxBytes, admit, io.Discard)
		cancel()
		if errors.Is(err, errGovernanceReadViewBudget) {
			for producer := range item.producers {
				result.UnknownProducers[producer] = appendUnique(result.UnknownProducers[producer], "Git streamed final audit exceeded its byte budget")
			}
			continue
		}
		if digest != item.digest || length != item.length || streamErrorDigest(err) != item.errText {
			for producer := range item.producers {
				result.ChangedProducers[producer] = true
			}
			rootDigest := digestBytes([]byte(filepath.Clean(item.root)))
			argsDigest := digestBytes([]byte(strings.Join(item.args, "\x00")))
			result.ChangedPaths = append(result.ChangedPaths, "git:"+strings.TrimPrefix(rootDigest, "sha256:")+":"+strings.TrimPrefix(argsDigest, "sha256:"))
		}
	}
}

// governanceStreamRun and governanceAuditGitStream run streamed reads during
// the operation and in its final audit.
var (
	governanceStreamRun      = runGovernanceGitStreamContext
	governanceAuditGitStream = runGovernanceGitStreamContext
)

// runGovernanceGitStreamContext runs Git like the captured reads do and
// writes its stdout to sink as it arrives, admitting each chunk before it is
// written and hashing it; stderr is kept bounded. It returns the stdout's
// digest and length. A non-empty stderr is an error, as for captured reads.
func runGovernanceGitStreamContext(ctx context.Context, files Files, root string, args []string, limit int64, admit func(int64) bool, sink io.Writer) (string, int64, []byte, error) {
	if limit <= 0 {
		return "", 0, nil, errGovernanceReadViewBudget
	}
	_ = FilesOrNative(files)
	if ctx == nil {
		ctx = context.Background()
	}
	if sink == nil {
		sink = io.Discard
	}
	cmd := gitCommandContext(ctx, append([]string{"--no-replace-objects", "-C", root}, args...)...)
	cmd.WaitDelay = 2 * time.Second
	cmd.Env = append(cmd.Env, "GIT_OPTIONAL_LOCKS=0")
	out := &governanceStreamWriter{limit: limit, admit: admit, sink: sink, hash: sha256.New()}
	var stderr bytes.Buffer
	cmd.Stdout = out
	cmd.Stderr = &limitedStderr{buf: &stderr, limit: governanceStreamStderrLimit}
	err := cmd.Run()
	if out.exceeded {
		return "", 0, nil, errGovernanceReadViewBudget
	}
	digest := "sha256:" + hex.EncodeToString(out.hash.Sum(nil))
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			exit.Stderr = append([]byte(nil), stderr.Bytes()...)
		}
		return digest, out.written, stderr.Bytes(), &GitSnapshotError{Operation: strings.Join(args, " "), Cause: err}
	}
	if stderr.Len() > 0 {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = "Git wrote unclassified stderr output"
		}
		return digest, out.written, stderr.Bytes(), &GitSnapshotError{Operation: strings.Join(args, " "), Cause: errors.New(detail)}
	}
	return digest, out.written, nil, nil
}

type governanceStreamWriter struct {
	mu       sync.Mutex
	limit    int64
	written  int64
	admit    func(int64) bool
	sink     io.Writer
	hash     hash.Hash
	exceeded bool
}

func (w *governanceStreamWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.exceeded || int64(len(data)) > w.limit-w.written || w.admit != nil && !w.admit(int64(len(data))) {
		w.exceeded = true
		return 0, errGovernanceReadViewBudget
	}
	if _, err := w.sink.Write(data); err != nil {
		return 0, err
	}
	w.hash.Write(data)
	w.written += int64(len(data))
	return len(data), nil
}

type limitedStderr struct {
	buf   *bytes.Buffer
	limit int
}

func (w *limitedStderr) Write(data []byte) (int, error) {
	if room := w.limit - w.buf.Len(); room > 0 {
		if len(data) > room {
			w.buf.Write(data[:room])
		} else {
			w.buf.Write(data)
		}
	}
	return len(data), nil
}

// streamErrorDigest is the fixed-size identity of a streamed read's error.
func streamErrorDigest(err error) string {
	if err == nil {
		return ""
	}
	return digestBytes([]byte(errorText(err)))
}
