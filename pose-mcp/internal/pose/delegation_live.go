package pose

// Live events of a running delegated run (spec
// pose-delegated-review-completion-signal R9). The process running the
// attempt normalizes each complete line of the agent's stream as it arrives,
// redacts it with the run's credentials and every session reference the
// stream has declared so far, and appends the events to a private file that
// pose delegate watch follows. The file is removed when the run's record is
// published: from then on the events bound to the record are the ones read.
// A session reference printed before the line that declares it cannot be
// recognised yet; the published events are derived again from the whole
// transcript and redacted with it.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const delegationLiveDir = reviewPrivateDir + "/live"

// delegationLiveWriter is a sink beside the bounded capture; it never fails
// the run, it only stops writing.
type delegationLiveWriter struct {
	mu       sync.Mutex
	file     *os.File
	format   string
	secrets  []string
	sessions []string
	pattern  *regexp.Regexp
	tools    map[string]string
	pending  []byte
	seq      int
	written  int64
}

func (s Store) delegationLivePath(attempt string) string {
	return filepath.Join(s.Root, filepath.FromSlash(delegationLiveDir), attempt+".ndjson")
}

// startDelegationLive opens the live file of attempt; nil when the agent's
// stream has no parser or the file cannot be made.
func (s Store) startDelegationLive(attempt, format string, secrets []string, sessionPattern, knownSession string) *delegationLiveWriter {
	if format != DelegationEventsCodexJSONL && format != DelegationEventsClaudeStreamJSON || !validAttemptFileName(attempt) {
		return nil
	}
	if _, err := s.delegationActiveDir(true); err != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Join(s.Root, filepath.FromSlash(delegationLiveDir)), 0o700); err != nil {
		return nil
	}
	file, err := os.OpenFile(s.delegationLivePath(attempt), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return nil
	}
	w := &delegationLiveWriter{file: file, format: format, secrets: secrets, tools: map[string]string{}}
	if knownSession != "" {
		w.sessions = append(w.sessions, knownSession)
	}
	if sessionPattern != "" {
		w.pattern, _ = regexp.Compile(sessionPattern)
	}
	return w
}

func (w *delegationLiveWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return len(p), nil
	}
	w.pending = append(w.pending, p...)
	for {
		i := bytes.IndexByte(w.pending, '\n')
		if i < 0 {
			// A line longer than the events' own bound is dropped live; the
			// published events read it whole from the transcript.
			if len(w.pending) > delegationEventsMaxBytes {
				w.pending = nil
			}
			return len(p), nil
		}
		line := w.pending[:i+1]
		w.pending = w.pending[i+1:]
		w.feed(line)
	}
}

func (w *delegationLiveWriter) feed(line []byte) {
	if w.pattern != nil {
		if m := w.pattern.FindSubmatch(line); len(m) == 2 && len(m[1]) > 0 {
			w.sessions = append(w.sessions, string(m[1]))
		}
	}
	clean := func(v string) (string, error) {
		out := []byte(v)
		var err error
		for _, session := range append([]string{""}, w.sessions...) {
			if out, err = redactReviewCapture(out, w.secrets, session, 17*int64(len(out))+1); err != nil {
				return "", err
			}
		}
		return string(out), nil
	}
	var cleanErr error
	field := func(v any) string {
		cleaned, err := clean(str(v))
		if err != nil {
			cleanErr = err
		}
		return cleaned
	}
	var events []DelegationEvent
	switch w.format {
	case DelegationEventsCodexJSONL:
		events = codexEvents(line, field)
	case DelegationEventsClaudeStreamJSON:
		events = claudeEventsWith(line, field, w.tools)
	}
	if cleanErr != nil || len(events) == 0 {
		return
	}
	events, err := finishDelegationEvents(events, w.seq+1, clean)
	if err != nil {
		return
	}
	w.seq += len(events)
	raw := EncodeDelegationEvents(events)
	if w.written+int64(len(raw)) > delegationEventsMaxBytes {
		w.closeLocked()
		return
	}
	if n, err := w.file.Write(raw); err != nil {
		w.closeLocked()
	} else {
		w.written += int64(n)
	}
}

func (w *delegationLiveWriter) closeLocked() {
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
}

// Close stops writing; the file stays until the run's record is published.
func (w *delegationLiveWriter) Close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closeLocked()
}

// ReadDelegationLiveEvents reads the live events of a running attempt from
// the given byte offset, returning the complete events and the offset after
// them; a missing file reads as none.
func (s Store) ReadDelegationLiveEvents(attempt string, offset int64) ([]DelegationEvent, int64, error) {
	if !validAttemptFileName(attempt) {
		return nil, offset, nil
	}
	f, err := os.Open(s.delegationLivePath(attempt))
	if os.IsNotExist(err) {
		return nil, offset, nil
	}
	if err != nil {
		return nil, offset, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, offset, err
	}
	if info.Size() < offset {
		offset = 0 // a continuation of the attempt started the file again
	}
	if info.Size()-offset > delegationEventsMaxBytes {
		offset = info.Size() - delegationEventsMaxBytes
	}
	buf := make([]byte, info.Size()-offset)
	n, _ := f.ReadAt(buf, offset)
	buf = buf[:n]
	end := bytes.LastIndexByte(buf, '\n')
	if end < 0 {
		return nil, offset, nil
	}
	var events []DelegationEvent
	for _, line := range strings.Split(string(buf[:end]), "\n") {
		var e DelegationEvent
		if json.Unmarshal([]byte(line), &e) == nil && e.Version == DelegationEventSchemaVersion {
			events = append(events, e)
		}
	}
	return events, offset + int64(end) + 1, nil
}
