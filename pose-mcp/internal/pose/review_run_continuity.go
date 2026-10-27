package pose

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// All vendor recognition is explicit policy; the engine has no vendor vocabulary.
type ReviewerLimit struct {
	ExitCodes      []int    `json:"exit_codes,omitempty"`
	OutputPatterns []string `json:"output_patterns,omitempty"`
	ResetPattern   string   `json:"reset_pattern,omitempty"`
}
type ReviewerSession struct {
	Pattern string `json:"pattern"`
}
type ReviewerResume struct {
	Command     []string `json:"command"`
	MaxResumes  int      `json:"max_resumes,omitempty"`
	MaxWallTime string   `json:"max_wall_time,omitempty"`
}
type ReviewInterruption struct {
	Cause     string `json:"cause"`
	Resumable bool   `json:"resumable"`
	ResetAt   string `json:"reset_at,omitempty"`
}

const reviewContinuationsDir = ".pose/review-continuations"
const reviewPrivateDir = ".pose/review-private"

type reviewContinuation struct {
	SchemaVersion     int    `json:"schema_version"`
	Predecessor       string `json:"predecessor"`
	AttemptID         string `json:"attempt_id"`
	PredecessorDigest string `json:"predecessor_digest"`
	ClaimedAt         string `json:"claimed_at"`
	Nonce             string `json:"nonce"`
}

func validateContinuityAdapter(a ReviewerAdapter) error {
	patterns := []string{}
	if a.Timeout != "" {
		d, e := time.ParseDuration(a.Timeout)
		if e != nil || d <= 0 {
			return fmt.Errorf("timeout must be positive")
		}
	}
	if a.Limit != nil {
		patterns = append(patterns, a.Limit.OutputPatterns...)
		if a.Limit.ResetPattern != "" {
			patterns = append(patterns, a.Limit.ResetPattern)
		}
	}
	if a.Session != nil {
		if a.Session.Pattern == "" {
			return fmt.Errorf("session needs a capture pattern")
		}
		patterns = append(patterns, a.Session.Pattern)
	}
	for _, p := range patterns {
		if len(p) > 4096 {
			return fmt.Errorf("recognition pattern exceeds bound")
		}
		if _, e := regexp.Compile(p); e != nil {
			return e
		}
	}
	for _, p := range []string{sessionPattern(a), resetPattern(a)} {
		if p != "" && regexp.MustCompile(p).NumSubexp() != 1 {
			return fmt.Errorf("session/reset pattern needs exactly one capture group")
		}
	}
	if a.Resume != nil {
		if len(a.Resume.Command) == 0 || a.Resume.MaxResumes < 0 || a.Resume.MaxResumes > 100 {
			return fmt.Errorf("resume needs command and bounded nonnegative max_resumes")
		}
		hasSession := false
		for i, arg := range a.Resume.Command {
			if arg == "{session}" {
				if i == 0 || i > 0 && (a.Resume.Command[i-1] == "-c" || a.Resume.Command[i-1] == "-e" || a.Resume.Command[i-1] == "--eval" || a.Resume.Command[i-1] == "--command") {
					return fmt.Errorf("session reference cannot be executable code")
				}
				hasSession = true
			} else if strings.Contains(arg, "{session}") {
				return fmt.Errorf("{session} must be a separate argv element, never interpolated into code")
			}
		}
		if !hasSession {
			return fmt.Errorf("resume command needs {session}")
		}
		if a.Session == nil {
			return fmt.Errorf("resume needs session declaration")
		}
		if a.Resume.MaxWallTime != "" {
			d, e := time.ParseDuration(a.Resume.MaxWallTime)
			if e != nil || d <= 0 {
				return fmt.Errorf("resume max_wall_time must be positive")
			}
		}
	}
	return nil
}
func sessionPattern(a ReviewerAdapter) string {
	if a.Session != nil {
		return a.Session.Pattern
	}
	return ""
}
func resetPattern(a ReviewerAdapter) string {
	if a.Limit != nil {
		return a.Limit.ResetPattern
	}
	return ""
}
func captureReviewValue(pattern, text string) (string, error) {
	if pattern == "" {
		return "", nil
	}
	r, e := regexp.Compile(pattern)
	if e != nil {
		return "", e
	}
	matches := r.FindAllStringSubmatch(text, 3)
	if len(matches) == 0 {
		return "", nil
	}
	if len(matches) != 1 || len(matches[0]) != 2 {
		return "", fmt.Errorf("ambiguous adapter capture")
	}
	value := matches[0][1]
	if len(value) == 0 || len(value) > 2048 || strings.ContainsAny(value, "\x00\r\n") {
		return "", fmt.Errorf("adapter capture is not a bounded reference")
	}
	return value, nil
}
func (s Store) saveReviewSession(value string) (string, error) {
	digest := digestBytes([]byte(value))
	dir, e := ensureReviewArtifactDir(s.Root, reviewPrivateDir, true)
	if e != nil {
		return "", e
	}
	if e = os.Chmod(dir, 0700); e != nil {
		return "", e
	}
	// Private session files are runtime state, never published evidence.
	ignore := filepath.Join(dir, ".gitignore")
	if e = PublishFile(ignore, []byte("*\n"), 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); e != nil {
		return "", e
	}
	e = PublishFile(filepath.Join(dir, strings.TrimPrefix(digest, "sha256:")+".session"), []byte(value), 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent})
	return digest, e
}
func (s Store) readReviewSession(digest string) (string, error) {
	if len(digest) != 71 || !strings.HasPrefix(digest, "sha256:") || strings.ContainsAny(digest, "/\\") {
		return "", fmt.Errorf("invalid session digest")
	}
	dir, e := ensureReviewArtifactDir(s.Root, reviewPrivateDir, false)
	if e != nil {
		return "", e
	}
	raw, e := readLedgerRaw(filepath.Join(dir, strings.TrimPrefix(digest, "sha256:")+".session"))
	if e != nil {
		return "", e
	}
	if digestBytes(raw) != digest || len(raw) == 0 || len(raw) > 2048 {
		return "", fmt.Errorf("private session digest differs")
	}
	return string(raw), nil
}
func (s Store) classifyReviewInterruption(run *ReviewRun, plan DispatchPlan, transcript []byte, runErr, ctxErr error) (string, string, error) {
	// Inspect both supported output channels before any public publication.
	value, e := captureReviewChannels(sessionPattern(plan.Config), string(transcript), run.Output)
	if e != nil {
		return "", "", e
	}
	if value != "" && plan.session != "" && value != plan.session {
		return "", "", fmt.Errorf("adapter changed the captured session")
	}
	if value == "" {
		value = plan.session
	}
	if value != "" {
		run.sessionForRedaction = value
		run.SessionDigest, e = s.saveReviewSession(value)
		if e != nil {
			return "", "", e
		}
	}
	cause := ""
	if ctxErr != nil {
		cause = "timeout"
		if errors.Is(ctxErr, context.Canceled) {
			cause = "cancelled"
		}
	} else if runErr != nil && run.ExitCode < 0 {
		cause = "process-lost"
	}
	limit := false
	if p := plan.Config.Limit; p != nil {
		for _, code := range p.ExitCodes {
			if code == run.ExitCode {
				limit = true
			}
		}
		for _, pattern := range p.OutputPatterns {
			if regexp.MustCompile(pattern).Match(transcript) || regexp.MustCompile(pattern).MatchString(run.Output) {
				limit = true
			}
		}
	}
	if limit {
		cause = "usage-limit"
	}
	supervisorCause := cause
	if cause == "" {
		if runErr != nil {
			return ReviewRunFailed, "the reviewer command failed: " + runErr.Error(), nil
		}
		return "", "", nil
	}
	// Without any continuity declaration, interruption classification is unknown.
	if plan.Config.Limit == nil && plan.Config.Session == nil && plan.Config.Resume == nil {
		cause = "unknown"
	}
	interruption := &ReviewInterruption{Cause: cause, Resumable: run.SessionDigest != "" && plan.Config.Resume != nil}
	if cause == "usage-limit" {
		reset, e := captureReviewChannels(resetPattern(plan.Config), string(transcript), run.Output)
		if e != nil {
			return "", "", e
		}
		if reset != "" {
			t, e := time.Parse(time.RFC3339Nano, reset)
			if e != nil {
				return "", "", fmt.Errorf("adapter reset instant is not RFC3339")
			}
			interruption.ResetAt = t.UTC().Format(time.RFC3339Nano)
		}
	}
	run.Interruption = interruption
	return ReviewRunInterrupted, "review interrupted: " + cause + " (supervisor: " + supervisorCause + ")", nil
}
func validateContinuityRun(r ReviewRun) error {
	if r.Status == ReviewRunInterrupted {
		if r.Interruption == nil {
			return fmt.Errorf("interrupted run has no cause")
		}
		switch r.Interruption.Cause {
		case "usage-limit", "timeout", "process-lost", "cancelled", "unknown":
		default:
			return fmt.Errorf("invalid interruption cause")
		}
		if r.Interruption.ResetAt != "" {
			if _, e := time.Parse(time.RFC3339Nano, r.Interruption.ResetAt); e != nil {
				return e
			}
		}
		if r.Interruption.Resumable && (r.SessionDigest == "" || r.TreeDigest == "" || r.AttemptID == "") {
			return fmt.Errorf("resumable run lacks bound evidence")
		}
		if r.PartialDigest != digestBytes([]byte(r.Output)) {
			return fmt.Errorf("partial draft digest differs")
		}
	} else if r.Interruption != nil {
		return fmt.Errorf("terminal run contains interruption metadata")
	}
	return nil
}
func continuationPath(run string, result bool) string {
	suffix := ".json"
	if result {
		suffix = ".result.json"
	}
	return filepath.ToSlash(filepath.Join(reviewContinuationsDir, run+suffix))
}
func (s Store) claimReviewContinuation(r ReviewRun, at time.Time) error {
	raw, _ := json.Marshal(r)
	// Clear reader-only fields before deriving the predecessor digest.
	r.Path, r.SupersededBy = "", ""
	raw, _ = json.Marshal(r)
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	return publishLedgerJSON(s.Root, continuationPath(r.RunID, false), reviewContinuation{1, r.RunID, r.AttemptID, digestBytes(raw), at.UTC().Format(time.RFC3339Nano), hex.EncodeToString(nonce)})
}
func (s Store) finishReviewContinuation(r ReviewRun) error {
	raw, _ := json.Marshal(r)
	receipt := reviewAttemptReceipt{1, r.AttemptID, r.RunID, digestBytes(raw)}
	return publishLedgerJSON(s.Root, continuationPath(r.continues(), true), receipt)
}
func (s Store) existingReviewContinuation(r ReviewRun, runs []ReviewRun) (*ReviewRun, error) {
	var claim reviewContinuation
	e := readLedgerJSON(filepath.Join(s.Root, continuationPath(r.RunID, false)), &claim)
	if errors.Is(e, os.ErrNotExist) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	copy := r
	copy.Path, copy.SupersededBy = "", ""
	raw, _ := json.Marshal(copy)
	if claim.SchemaVersion != 1 || claim.Predecessor != r.RunID || claim.AttemptID != r.AttemptID || claim.PredecessorDigest != digestBytes(raw) {
		return nil, fmt.Errorf("continuation predecessor changed")
	}
	var receipt reviewAttemptReceipt
	if e = readLedgerJSON(filepath.Join(s.Root, continuationPath(r.RunID, true)), &receipt); e != nil {
		return nil, &PendingReviewContinuation{ID: "resume-of:" + r.RunID, Predecessor: r.RunID, AttemptID: r.AttemptID}
	}
	for _, next := range runs {
		if next.RunID == receipt.RunID {
			copy := next
			copy.Path, copy.SupersededBy = "", ""
			raw, _ := json.Marshal(copy)
			if receipt.SchemaVersion != 1 || receipt.AttemptID != r.AttemptID || next.AttemptID != r.AttemptID || next.continues() != r.RunID || receipt.RunDigest != digestBytes(raw) {
				return nil, fmt.Errorf("invalid continuation receipt")
			}
			return &next, nil
		}
	}
	return nil, fmt.Errorf("continuation result is missing")
}

// PlanReviewResume validates original evidence, policy and all accumulated budgets.
func (s Store) PlanReviewResume(id string) (DispatchPlan, error) {
	runs, lost, e := s.ReadReviewRuns("")
	if e != nil {
		return DispatchPlan{}, e
	}
	if len(lost) > 0 {
		return DispatchPlan{}, fmt.Errorf("resume history incomplete: %v", lost)
	}
	var prior *ReviewRun
	for i := range runs {
		if runs[i].RunID == id {
			prior = &runs[i]
		}
	}
	if prior == nil {
		return DispatchPlan{}, fmt.Errorf("resume predecessor not found")
	}
	if prior.Status != ReviewRunInterrupted || prior.Interruption == nil || !prior.Interruption.Resumable {
		return DispatchPlan{}, fmt.Errorf("run is %s or not resumable", prior.Status)
	}
	if why := s.reviewRunTranscriptIntact(*prior); why != "" {
		return DispatchPlan{}, fmt.Errorf("resume: %s", why)
	}
	policy, _, e := s.LoadReviewersPolicyFor(prior.Kind)
	if e != nil {
		return DispatchPlan{}, e
	}
	config, ok := policy.Adapters[prior.Adapter]
	if !ok || config.Resume == nil {
		return DispatchPlan{}, fmt.Errorf("resume adapter declaration missing")
	}
	brief, e := s.RenderReviewBrief(prior.BundleID, prior.Kind, prior.BriefNotes)
	if e != nil {
		return DispatchPlan{}, e
	}
	bundle, e := s.LoadReviewBundle(prior.BundleID)
	if e != nil {
		return DispatchPlan{}, e
	}
	if brief.Digest != prior.BriefDigest || bundle.BundleDigest != prior.BundleDigest || bundle.Payload.Subject.Head != prior.SealedCommit {
		return DispatchPlan{}, fmt.Errorf("resume bundle or brief digest changed")
	}
	origin, e := dispatchWorktreeIdentity(s.Root)
	if e != nil {
		return DispatchPlan{}, e
	}
	if prior.Safety != nil && prior.Safety.Operation.Outcome != OutcomeCompleted {
		return DispatchPlan{}, fmt.Errorf("predecessor cleanup incomplete; refusing continuation beside possible orphan")
	}
	plan := DispatchPlan{predecessor: prior, notes: prior.BriefNotes, Reason: prior.Reason, PolicyDigest: prior.PolicyDigest, OriginWorktree: origin, Adapter: prior.Adapter, Config: config, Brief: brief, Bundle: bundle, SealedCommit: prior.SealedCommit}
	if e = s.checkDispatchOrigin(plan); e != nil {
		return DispatchPlan{}, e
	}
	existing, e := s.existingReviewContinuation(*prior, runs)
	if e != nil {
		var pending *PendingReviewContinuation
		if errors.As(e, &pending) {
			plan.PendingContinuation = pending
			return plan, nil
		}
		return DispatchPlan{}, e
	}
	if existing != nil {
		if existing.HandsOffFrom == prior.RunID {
			return DispatchPlan{}, fmt.Errorf("pose: run %s was handed off to run %s; it is not resumed", prior.RunID, existing.RunID)
		}
		plan.Existing = existing
		return plan, nil
	}
	// The attempt's budget, not this agent's: after a handoff the resumed
	// agent may declare another (spec pose-delegation-handoff R2).
	count, spent, limit, total, e := attemptContinuationBudget(runs, *prior, policy)
	if e != nil {
		return DispatchPlan{}, e
	}
	if count >= limit || spent >= total {
		return DispatchPlan{}, fmt.Errorf("resume budget exhausted: %d resumes, %s wall-clock", count, spent)
	}
	timeout := 60 * time.Minute
	if config.Timeout != "" {
		timeout, _ = time.ParseDuration(config.Timeout)
	}
	if total-spent < timeout {
		timeout = total - spent
	}
	plan.processTimeout = timeout
	// A continuation runs the same agent the attempt was routed to.
	plan.Routing = prior.Routing
	s.planImplementation(&plan, prior.Lead)
	plan.provenance, e = s.prepareRunProvenance(plan, policy)
	if e != nil {
		return DispatchPlan{}, e
	}
	plan.session, e = s.readReviewSession(prior.SessionDigest)
	if e != nil {
		return DispatchPlan{}, e
	}
	return plan, nil
}

// ReviewResumeDelay computes a bounded, non-polling wait; callers revalidate afterwards.
func ReviewResumeDelay(r ReviewRun, now time.Time, max time.Duration, jitter time.Duration) (time.Duration, error) {
	if r.Interruption == nil || r.Interruption.ResetAt == "" {
		return 0, fmt.Errorf("no recorded reset instant")
	}
	if max <= 0 || max > 24*time.Hour || jitter < 0 || jitter > time.Second {
		return 0, fmt.Errorf("wait bound must be positive and at most 24h; jitter at most 1s")
	}
	reset, e := time.Parse(time.RFC3339Nano, r.Interruption.ResetAt)
	if e != nil {
		return 0, e
	}
	delay := reset.Sub(now)
	if delay < 0 {
		delay = 0
	}
	delay += jitter
	if delay > max {
		return 0, fmt.Errorf("recorded reset exceeds maximum wait")
	}
	return delay, nil
}

// Continuation inventories use the same bounds as the attempt ledger. Pending
// claims are observable operations, not proof of a completed review.
func (s Store) reviewContinuationFindings(runs []ReviewRun) []string {
	entries, e := ledgerEntries(s.Root, reviewContinuationsDir)
	if e != nil {
		return []string{e.Error()}
	}
	byID := map[string]ReviewRun{}
	for _, r := range runs {
		byID[r.RunID] = r
	}
	claims := map[string]reviewContinuation{}
	receipts := map[string]reviewAttemptReceipt{}
	losses := []string{}
	budget := int64(32 << 20)
	for _, entry := range entries {
		if IsPublisherStaging(entry.Name()) {
			continue
		}
		full := filepath.Join(s.Root, reviewContinuationsDir, entry.Name())
		if strings.HasSuffix(entry.Name(), ".result.json") {
			var receipt reviewAttemptReceipt
			if e := readLedgerJSONBudget(full, &receipt, &budget); e != nil {
				losses = append(losses, e.Error())
				continue
			}
			receipts[strings.TrimSuffix(entry.Name(), ".result.json")] = receipt
		} else if strings.HasSuffix(entry.Name(), ".json") {
			var claim reviewContinuation
			if e := readLedgerJSONBudget(full, &claim, &budget); e != nil {
				losses = append(losses, e.Error())
				continue
			}
			id := strings.TrimSuffix(entry.Name(), ".json")
			prior, ok := byID[id]
			prior.Path, prior.SupersededBy = "", ""
			raw, _ := json.Marshal(prior)
			if !ok || claim.SchemaVersion != 1 || claim.Predecessor != id || claim.AttemptID != prior.AttemptID || claim.PredecessorDigest != digestBytes(raw) {
				losses = append(losses, "missing or rewritten continuation predecessor "+id)
				continue
			}
			if nonce, e := hex.DecodeString(claim.Nonce); e != nil || len(nonce) != 16 {
				losses = append(losses, "invalid continuation nonce")
				continue
			}
			if _, e := time.Parse(time.RFC3339Nano, claim.ClaimedAt); e != nil {
				losses = append(losses, "invalid continuation claim time")
				continue
			}
			claims[id] = claim
		} else {
			losses = append(losses, "unknown continuation artifact "+entry.Name())
		}
		if budget <= 0 {
			losses = append(losses, "continuation inventory read budget exhausted")
			break
		}
	}
	for id, receipt := range receipts {
		claim, ok := claims[id]
		next, exists := byID[receipt.RunID]
		copy := next
		copy.Path, copy.SupersededBy = "", ""
		raw, _ := json.Marshal(copy)
		if !ok || !exists || receipt.SchemaVersion != 1 || receipt.AttemptID != claim.AttemptID || next.AttemptID != claim.AttemptID || next.continues() != id || receipt.RunDigest != digestBytes(raw) {
			losses = append(losses, "missing or conflicting continuation result "+id)
		}
	}
	children := map[string][]ReviewRun{}
	for _, r := range runs {
		if r.Resumes != "" && r.HandsOffFrom != "" {
			losses = append(losses, "invalid continuation chain "+r.RunID+": both a resume and a handoff")
		}
		if parentID := r.continues(); parentID != "" {
			prior, ok := byID[parentID]
			_, claimed := claims[parentID]
			same := ok && claimed && r.AttemptID == prior.AttemptID && r.BundleDigest == prior.BundleDigest && r.BriefDigest == prior.BriefDigest && r.SealedCommit == prior.SealedCommit
			if r.Resumes != "" {
				// A resume continues the same agent and session, of a resumable stop.
				same = same && prior.Status == ReviewRunInterrupted && prior.Interruption != nil && prior.Interruption.Resumable && r.Adapter == prior.Adapter && r.TreeDigest == prior.TreeDigest
			} else {
				// A handoff continues another agent, of any stop short of a conclusion.
				same = same && handoffable(prior) == "" && r.Adapter != prior.Adapter && r.HandoffDigest != "" && (prior.TreeDigest == "" || r.TreeDigest == prior.TreeDigest)
			}
			if !same {
				losses = append(losses, "invalid continuation chain "+r.RunID)
			}
			if _, ok := receipts[parentID]; !ok {
				losses = append(losses, "missing continuation result receipt "+parentID)
			}
			children[parentID] = append(children[parentID], r)
		}
	}
	for id, group := range children {
		endpoints := 0
		for _, child := range group {
			if child.SupersededBy == "" {
				endpoints++
				continue
			}
			next, ok := byID[child.SupersededBy]
			if !ok || next.continues() != id || next.Supersedes != child.RunID {
				losses = append(losses, "invalid continuation supersession "+id)
			}
		}
		if endpoints != 1 {
			losses = append(losses, "multiple continuation endpoints "+id)
		}
	}
	return uniqueSorted(losses)
}

// PendingReviewInterruptions returns only the latest interrupted endpoint of each
// attempt; the public summaries deliberately contain no vendor session material.
func (s Store) PendingReviewInterruptions() ([]ReviewRunSummary, []string) {
	return s.reviewInterruptedRuns(false)
}

func (s Store) ReviewInterruptedRuns() ([]ReviewRunSummary, []string) {
	return s.reviewInterruptedRuns(true)
}

func (s Store) reviewInterruptedRuns(includeContinued bool) ([]ReviewRunSummary, []string) {
	runs, lost, e := s.ReadReviewRuns("")
	if e != nil {
		lost = append(lost, e.Error())
	}
	continued := map[string]string{}
	for _, r := range runs {
		if r.continues() != "" {
			continued[r.continues()] = r.RunID
		}
	}
	out := []ReviewRunSummary{}
	for _, r := range runs {
		if r.Status == ReviewRunInterrupted && (includeContinued || continued[r.RunID] == "") {
			out = append(out, ReviewRunSummary{ResumedBy: continued[r.RunID], RunID: r.RunID, AttemptID: r.AttemptID, BundleID: r.BundleID, Adapter: r.Adapter, Status: r.Status, Partial: true, Interruption: r.Interruption, DraftDecision: "unknown", Assurance: "partial-draft-not-review"})
		}
	}
	return out, lost
}

func captureReviewChannels(pattern, transcript, output string) (string, error) {
	a, e := captureReviewValue(pattern, transcript)
	if e != nil {
		return "", e
	}
	b, e := captureReviewValue(pattern, output)
	if e != nil {
		return "", e
	}
	if a != "" && b != "" && a != b {
		return "", fmt.Errorf("conflicting adapter capture channels")
	}
	if a != "" {
		return a, nil
	}
	return b, nil
}

// PendingReviewContinuation identifies an existing reserved continuation without
// inventing a completed run or launching a second process after a crash.
type PendingReviewContinuation struct {
	ID          string `json:"id"`
	Predecessor string `json:"predecessor"`
	AttemptID   string `json:"attempt_id"`
}

func (p *PendingReviewContinuation) Error() string {
	return "continuation " + p.ID + " already reserved; result pending, no second process launched"
}
