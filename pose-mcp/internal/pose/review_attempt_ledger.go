package pose

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const reviewAttemptDir = ".pose/review-attempts"
const reviewRunLedgerDir = ".pose/review-run-ledger"
const reviewLedgerMaxRecords = 10000
const reviewLedgerMaxBytes = 1 << 20

// DelegatedReviewAttempt is an immutable reservation, published before any adapter starts.
// Its ordinal path coordinates contenders; its ID binds its complete content.
type DelegatedReviewAttempt struct {
	SchemaVersion int    `json:"schema_version"`
	AttemptID     string `json:"attempt_id"`
	BundleID      string `json:"bundle_id"`
	BundleDigest  string `json:"bundle_digest"`
	Scope         string `json:"scope"`
	Adapter       string `json:"adapter"`
	Reason        string `json:"reason,omitempty"`
	Ordinal       int    `json:"ordinal"`
	ReservedAt    string `json:"reserved_at"`
	Nonce         string `json:"nonce"`
}
type reviewAttemptReceipt struct {
	SchemaVersion int    `json:"schema_version"`
	AttemptID     string `json:"attempt_id"`
	RunID         string `json:"run_id"`
	RunDigest     string `json:"run_digest"`
}
type reviewRunWitness struct {
	SchemaVersion    int    `json:"schema_version"`
	RunID            string `json:"run_id"`
	BundleID         string `json:"bundle_id"`
	Scope            string `json:"scope"`
	RecordDigest     string `json:"record_digest"`
	TranscriptDigest string `json:"transcript_digest"`
}

func reviewAttemptKey(bundle string) string {
	return strings.TrimPrefix(digestBytes([]byte(bundle)), "sha256:")[:24]
}
func reviewAttemptPath(root string, a DelegatedReviewAttempt) string {
	return filepath.Join(root, reviewAttemptDir, reviewAttemptKey(a.BundleID), fmt.Sprintf("%06d.json", a.Ordinal))
}
func attemptReceiptPath(root string, a DelegatedReviewAttempt) string {
	return strings.TrimSuffix(reviewAttemptPath(root, a), ".json") + ".result.json"
}
func attemptIdentity(a DelegatedReviewAttempt) string {
	a.AttemptID = ""
	raw, _ := json.Marshal(a)
	return "rat-" + strings.TrimPrefix(digestBytes(raw), "sha256:")
}

func readLedgerRaw(path string) ([]byte, error) {
	return readBoundedRegular(path, reviewLedgerMaxBytes)
}

// readBoundedRegular reads a regular file of at most limit bytes, refusing a
// link or a file replaced while it was opened.
func readBoundedRegular(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("ledger record is not a bounded regular file: %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, statErr := f.Stat()
	if statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		f.Close()
		return nil, fmt.Errorf("ledger file changed while opening %s", path)
	}
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	closeErr := f.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("ledger record exceeds bound")
	}
	return raw, nil
}

var errReviewLedgerBudget = errors.New("review ledger exceeds 32 MiB inventory read budget")

func readLedgerRawBudget(path string, remaining *int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() < 0 || info.Size() > *remaining {
		return nil, errReviewLedgerBudget
	}
	*remaining -= info.Size()
	raw, err := readLedgerRaw(path)
	if err != nil {
		return nil, err
	}
	if extra := int64(len(raw)) - info.Size(); extra > 0 {
		*remaining -= extra
		if *remaining < 0 {
			return nil, errReviewLedgerBudget
		}
	}
	return raw, nil
}
func readLedgerJSONBudget(path string, v any, remaining *int64) error {
	raw, err := readLedgerRawBudget(path, remaining)
	if err != nil {
		return err
	}
	return strictJSONBytes(raw, v)
}
func reviewReadBudget(provided []*int64) *int64 {
	if len(provided) > 0 {
		return provided[0]
	}
	budget := int64(32 << 20)
	return &budget
}

func readLedgerJSON(path string, v any) error {
	raw, err := readLedgerRaw(path)
	if err != nil {
		return err
	}
	if err := strictJSONBytes(raw, v); err != nil {
		return fmt.Errorf("ledger record %s: %w", path, err)
	}
	return nil
}
func ledgerEntries(root, rel string) ([]os.DirEntry, error) {
	path, err := ensureReviewArtifactDir(root, rel, false)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	entries, err := f.ReadDir(reviewLedgerMaxRecords + 1)
	closeErr := f.Close()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(entries) > reviewLedgerMaxRecords {
		return nil, fmt.Errorf("ledger inventory exceeds %d records", reviewLedgerMaxRecords)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}
func publishLedgerJSON(root, rel string, v any) error {
	dir, err := ensureReviewArtifactDir(root, filepath.ToSlash(filepath.Dir(rel)), true)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if len(raw) > reviewLedgerMaxBytes {
		return fmt.Errorf("ledger record exceeds bound")
	}
	return PublishFile(filepath.Join(dir, filepath.Base(rel)), append(raw, '\n'), 0644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent})
}

func (s Store) readDelegatedReviewAttempts(bundle string, budgets ...*int64) ([]DelegatedReviewAttempt, error) {
	budget := reviewReadBudget(budgets)
	rel := filepath.ToSlash(filepath.Join(reviewAttemptDir, reviewAttemptKey(bundle)))
	entries, err := ledgerEntries(s.Root, rel)
	if err != nil {
		return nil, err
	}
	out := []DelegatedReviewAttempt{}
	for _, e := range entries {
		if IsPublisherStaging(e.Name()) {
			continue
		}
		if strings.HasSuffix(e.Name(), ".result.json") {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".json") {
			return nil, fmt.Errorf("unknown attempt artifact %s", e.Name())
		}
		var a DelegatedReviewAttempt
		if err := readLedgerJSONBudget(filepath.Join(s.Root, rel, e.Name()), &a, budget); err != nil {
			return nil, err
		}
		if a.SchemaVersion != 1 || a.BundleID != bundle || a.Ordinal != len(out)+1 || a.AttemptID != attemptIdentity(a) || e.Name() != fmt.Sprintf("%06d.json", a.Ordinal) {
			return nil, fmt.Errorf("invalid attempt reservation %s", e.Name())
		}
		if _, err := time.Parse(time.RFC3339Nano, a.ReservedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	// Orphan receipts are an unknown history, never ignored.
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".result.json") {
			found := false
			for _, a := range out {
				if filepath.Base(attemptReceiptPath(s.Root, a)) == e.Name() {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("attempt receipt has no reservation: %s", e.Name())
			}
		}
	}
	return out, nil
}
func (s Store) readAttemptReceipt(a DelegatedReviewAttempt, budgets ...*int64) (reviewAttemptReceipt, error) {
	budget := reviewReadBudget(budgets)
	var r reviewAttemptReceipt
	if err := readLedgerJSONBudget(attemptReceiptPath(s.Root, a), &r, budget); err != nil {
		return r, fmt.Errorf("attempt %s incomplete or unreadable; inspect explicitly: %w", a.AttemptID, err)
	}
	if r.SchemaVersion != 1 || r.AttemptID != a.AttemptID || !strings.HasPrefix(r.RunID, "rrn-") || strings.ContainsAny(r.RunID, "/\\") {
		return r, fmt.Errorf("invalid attempt receipt %s", a.AttemptID)
	}
	var run ReviewRun
	if err := readLedgerJSONBudget(filepath.Join(s.Root, reviewRunsDir, r.RunID+".json"), &run, budget); err != nil {
		return r, err
	}
	raw, _ := json.Marshal(run)
	if run.AttemptID != a.AttemptID || run.BundleID != a.BundleID || run.BundleDigest != a.BundleDigest || run.Adapter != a.Adapter || digestBytes(raw) != r.RunDigest {
		return r, fmt.Errorf("attempt receipt does not bind its final run")
	}
	return r, nil
}
func (s Store) reviewAttemptAdmission(plan DispatchPlan) ([]DelegatedReviewAttempt, int, error) {
	if len(plan.Reason) > 2048 || strings.ContainsAny(plan.Reason, "\x00\r\n") {
		return nil, 0, fmt.Errorf("rerun reason must be one line of at most 2048 bytes")
	}
	runs, lost, err := s.ReadReviewRuns(plan.Bundle.BundleID)
	if err != nil {
		return nil, 0, err
	}
	if len(lost) > 0 {
		return nil, 0, fmt.Errorf("review run coverage is unknown: %v", lost)
	}
	attempts, err := s.readDelegatedReviewAttempts(plan.Bundle.BundleID)
	if err != nil {
		return nil, 0, err
	}
	knownAttempts := map[string]bool{}
	count := 0
	receiptBudget := int64(32 << 20)
	for _, a := range attempts {
		knownAttempts[a.AttemptID] = true
		if _, err := s.readAttemptReceipt(a, &receiptBudget); err != nil {
			return nil, 0, err
		}
		if a.Adapter == plan.Adapter {
			count++
		}
	}
	completed := false
	for _, run := range runs {
		if run.AttemptID != "" && !knownAttempts[run.AttemptID] {
			return nil, 0, fmt.Errorf("missing attempt reservation %s", run.AttemptID)
		}
		if run.Status == ReviewRunCompleted {
			completed = true
		}
		if run.AttemptID == "" && run.Adapter == plan.Adapter {
			count++
		}
	}
	if count >= effectiveReviewMaxRuns(plan.Config) {
		return nil, count, fmt.Errorf("budget exhausted: %d attempt(s) of %s on this bundle, at most %d", count, plan.Adapter, effectiveReviewMaxRuns(plan.Config))
	}
	if completed && strings.TrimSpace(plan.Reason) == "" {
		return nil, count, fmt.Errorf("completed run exists on unchanged bundle; an explicit --reason is required")
	}
	return attempts, count, nil
}
func (s Store) reserveDelegatedReviewAttempt(plan DispatchPlan, at time.Time) (DelegatedReviewAttempt, error) {
	attempts, _, err := s.reviewAttemptAdmission(plan)
	if err != nil {
		return DelegatedReviewAttempt{}, err
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return DelegatedReviewAttempt{}, err
	}
	a := DelegatedReviewAttempt{SchemaVersion: 1, BundleID: plan.Bundle.BundleID, BundleDigest: plan.Bundle.BundleDigest, Scope: plan.Brief.Scope, Adapter: plan.Adapter, Reason: strings.TrimSpace(plan.Reason), Ordinal: len(attempts) + 1, ReservedAt: at.UTC().Format(time.RFC3339Nano), Nonce: hex.EncodeToString(nonce)}
	a.AttemptID = attemptIdentity(a)
	rel, _ := filepath.Rel(s.Root, reviewAttemptPath(s.Root, a))
	if err := publishLedgerJSON(s.Root, filepath.ToSlash(rel), a); err != nil {
		return DelegatedReviewAttempt{}, fmt.Errorf("attempt reservation refused; another dispatch may hold the slot: %w", err)
	}
	return a, nil
}
func (s Store) finishDelegatedReviewAttempt(a DelegatedReviewAttempt, run ReviewRun) error {
	raw, _ := json.Marshal(run)
	r := reviewAttemptReceipt{SchemaVersion: 1, AttemptID: a.AttemptID, RunID: run.RunID, RunDigest: digestBytes(raw)}
	rel, _ := filepath.Rel(s.Root, attemptReceiptPath(s.Root, a))
	return publishLedgerJSON(s.Root, filepath.ToSlash(rel), r)
}

// ReviewRunLedgerFindings detects loss using immutable witnesses. No witness
// exists for legacy records; their content identity remains independently checked.
func (s Store) ReviewRunLedgerFindings(bundle string) ([]string, error) {
	budget := int64(32 << 20)
	return s.reviewRunLedgerFindings(bundle, &budget)
}
func (s Store) reviewRunLedgerFindings(bundle string, budget *int64) ([]string, error) {
	entries, err := ledgerEntries(s.Root, reviewRunLedgerDir)
	if err != nil {
		return nil, err
	}
	findings := []string{}
	var checkedBytes int64
	for _, e := range entries {
		if IsPublisherStaging(e.Name()) {
			continue
		}
		var w reviewRunWitness
		if err := readLedgerJSONBudget(filepath.Join(s.Root, reviewRunLedgerDir, e.Name()), &w, budget); err != nil {
			findings = append(findings, err.Error())
			if errors.Is(err, errReviewLedgerBudget) {
				break
			}
			continue
		}
		if w.SchemaVersion != 1 || e.Name() != w.RunID+".json" || strings.ContainsAny(w.RunID, "/\\") {
			findings = append(findings, "invalid run witness "+e.Name())
			continue
		}
		var run ReviewRun
		raw, readErr := readLedgerRawBudget(filepath.Join(s.Root, reviewRunsDir, w.RunID+".json"), budget)
		if readErr != nil {
			findings = append(findings, "witnessed run "+w.RunID+": "+readErr.Error())
			if errors.Is(readErr, errReviewLedgerBudget) {
				break
			}
			continue
		}
		if err := strictJSONBytes(raw, &run); err != nil {
			findings = append(findings, "unreadable witnessed run "+w.RunID)
			continue
		}
		if validateReviewRunRecord(w.RunID+".json", run) != nil || digestBytes(raw) != w.RecordDigest || run.TranscriptDigest != w.TranscriptDigest || run.BundleID != w.BundleID || run.Scope != w.Scope {
			findings = append(findings, "rewritten witnessed run "+w.RunID)
			continue
		}
		if bundle != "" && run.BundleID != bundle {
			continue
		}
		info, statErr := os.Lstat(filepath.Join(s.Root, run.TranscriptPath))
		if statErr == nil {
			checkedBytes += info.Size()
		}
		if checkedBytes > 512<<20 {
			findings = append(findings, "run transcript verification exceeds 512 MiB operation budget")
			break
		}
		if why := s.reviewRunTranscriptIntact(run); why != "" {
			findings = append(findings, w.RunID+": "+why)
		}
	}
	return findings, nil
}

type reviewHistoryBuffer struct{ bytes.Buffer }

func (b *reviewHistoryBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 8<<20 {
		return 0, fmt.Errorf("review history Git output exceeds 8 MiB")
	}
	return b.Buffer.Write(p)
}
func reviewHistoryGit(root string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	base := gitCommand(append([]string{"-C", root}, args...)...)
	cmd := exec.CommandContext(ctx, "git", base.Args[1:]...)
	cmd.Env = base.Env
	var out reviewHistoryBuffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("review history Git observation unavailable: %w", err)
	}
	return out.Bytes(), nil
}

// ReviewRunHistoryFindings adds the Git anchor: edits and deletions of tracked
// immutable evidence are violations even when staged or committed.
func (s Store) ReviewRunHistoryFindings() ([]string, error) {
	runs, losses, err := s.ReadReviewRuns("")
	findings := append(losses, s.reviewContinuationFindings(runs)...)
	if err != nil {
		return nil, err
	}
	roots := []string{reviewRunsDir, reviewRunLedgerDir, reviewAttemptDir, reviewContinuationsDir, reviewRunProofDir}
	if _, headErr := snapshotGit(s.Root, "rev-parse", "--verify", "HEAD"); headErr != nil {
		// An unborn repository without delegated data has no delegated history
		// contract to verify. Preserve the report-history gate's existing behavior.
		present := false
		for _, root := range roots {
			if _, err := os.Lstat(filepath.Join(s.Root, root)); !os.IsNotExist(err) {
				present = true
				break
			}
		}
		tracked, trackedErr := reviewHistoryGit(s.Root, append([]string{"ls-files", "--"}, roots...)...)
		if !present && trackedErr == nil && len(bytes.TrimSpace(tracked)) == 0 {
			return uniqueSorted(findings), nil
		}
		return findings, fmt.Errorf("delegated history Git anchor unavailable: %w", headErr)
	}

	args := append([]string{"diff", "HEAD", "--name-status", "--no-renames", "--diff-filter=MDT", "--"}, roots...)
	pending, err := reviewHistoryGit(s.Root, args...)
	if err != nil {
		return nil, err
	}
	args = append([]string{"log", "--format=", "--name-status", "--no-renames", "--diff-filter=MDT", "--"}, roots...)
	committed, err := reviewHistoryGit(s.Root, args...)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(append(pending, committed...)), "\n") {
		if strings.TrimSpace(line) != "" {
			findings = append(findings, "immutable review evidence changed: "+line)
		}
	}
	runs, lost, err := s.ReadReviewRuns("")
	if err != nil {
		return nil, err
	}
	_ = runs
	findings = append(findings, lost...)
	attempts, losses := s.allDelegatedReviewAttempts()
	findings = append(findings, losses...)
	receiptBudget := int64(32 << 20)
	for _, a := range attempts {
		if _, err := s.readAttemptReceipt(a, &receiptBudget); err != nil {
			findings = append(findings, err.Error())
		}
	}
	return uniqueSorted(findings), nil
}

// DraftDecision is telemetry about an adapter's text, not verified authority.
func draftReviewDecision(output string) string {
	decision := ""
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok || key != "decision" {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "approved" && value != "changes-requested" && value != "rejected" {
			return "unknown"
		}
		if decision != "" {
			return "unknown"
		}
		decision = value
	}
	if decision == "" {
		return "unknown"
	}
	return decision
}

type ReviewRunSummary struct {
	Safety        *ReviewExecutionSafety `json:"safety,omitempty"`
	ResumedBy     string                 `json:"resumed_by,omitempty"`
	Resumes       string                 `json:"resumes,omitempty"`
	LimitOf       string                 `json:"limit_of,omitempty"`
	Partial       bool                   `json:"partial,omitempty"`
	Interruption  *ReviewInterruption    `json:"interruption,omitempty"`
	SupersededBy  string                 `json:"superseded_by,omitempty"`
	RunID         string                 `json:"run_id"`
	AttemptID     string                 `json:"attempt_id,omitempty"`
	BundleID      string                 `json:"bundle_id"`
	Adapter       string                 `json:"adapter"`
	Status        string                 `json:"status"`
	DraftDecision string                 `json:"draft_decision"`
	Reason        string                 `json:"reason,omitempty"`
	Assurance     string                 `json:"assurance"`
}

func (s Store) reviewRunSummaries(scope string) ([]ReviewRunSummary, []string) {
	runs, lost, err := s.ReadReviewRuns("")
	if err != nil {
		lost = append(lost, err.Error())
	}
	out := []ReviewRunSummary{}
	for _, run := range runs {
		if run.Scope == scope {
			decision := "unknown"
			if run.Status == ReviewRunCompleted && reviewSafetyConflict(run) == "" {
				decision = draftReviewDecision(run.Output)
			}
			out = append(out, ReviewRunSummary{Safety: run.Safety, Resumes: run.Resumes, LimitOf: run.LimitOf, Partial: run.Status == ReviewRunInterrupted, Interruption: run.Interruption, SupersededBy: run.SupersededBy, RunID: run.RunID, AttemptID: run.AttemptID, BundleID: run.BundleID, Adapter: run.Adapter, Status: run.Status, DraftDecision: decision, Reason: run.Reason, Assurance: "declared-draft-not-attestation"})
		}
	}
	attempts, losses := s.allDelegatedReviewAttempts()
	lost = append(lost, losses...)
	receiptBudget := int64(32 << 20)
	for _, a := range attempts {
		if a.Scope == scope {
			if _, err := s.readAttemptReceipt(a, &receiptBudget); err != nil {
				lost = append(lost, err.Error())
			}
		}
	}
	return out, uniqueSorted(lost)
}

type DelegatedReviewMetric struct {
	Resumes             int      `json:"resumes"`
	Key                 string   `json:"key"`
	Runs                int      `json:"runs"`
	Attempts            int      `json:"attempts"`
	ApprovedDrafts      int      `json:"approved_drafts"`
	RejectingDrafts     int      `json:"rejecting_drafts"`
	UnknownDrafts       int      `json:"unknown_drafts"`
	Reruns              int      `json:"reruns_without_bundle_change"`
	RerunsWithoutReason int      `json:"reruns_without_reason"`
	ApprovalRate        *float64 `json:"approval_rate"`
}
type DelegatedReviewInsights struct {
	ByBundle         []DelegatedReviewMetric `json:"by_bundle"`
	ByAdapter        []DelegatedReviewMetric `json:"by_adapter"`
	CoverageComplete bool                    `json:"coverage_complete"`
	Losses           []string                `json:"losses,omitempty"`
	Assurance        string                  `json:"assurance"`
}

// allDelegatedReviewAttempts bounds the whole inventory, not just each bundle.
func (s Store) allDelegatedReviewAttempts() ([]DelegatedReviewAttempt, []string) {
	dirs, err := ledgerEntries(s.Root, reviewAttemptDir)
	if err != nil {
		return nil, []string{err.Error()}
	}
	out := []DelegatedReviewAttempt{}
	losses := []string{}
	visited := 0
	budget := int64(32 << 20)
	for _, dir := range dirs {
		if IsPublisherStaging(dir.Name()) {
			continue
		}
		if !dir.IsDir() || len(dir.Name()) != 24 {
			losses = append(losses, "invalid attempt directory "+dir.Name())
			continue
		}
		entries, err := ledgerEntries(s.Root, filepath.ToSlash(filepath.Join(reviewAttemptDir, dir.Name())))
		if err != nil {
			losses = append(losses, err.Error())
			continue
		}
		visited += len(entries)
		if visited > reviewLedgerMaxRecords {
			return out, append(losses, "global attempt inventory exceeds bound")
		}
		bundle := ""
		for _, e := range entries {
			if IsPublisherStaging(e.Name()) || strings.HasSuffix(e.Name(), ".result.json") {
				continue
			}
			var a DelegatedReviewAttempt
			if err := readLedgerJSONBudget(filepath.Join(s.Root, reviewAttemptDir, dir.Name(), e.Name()), &a, &budget); err != nil {
				losses = append(losses, err.Error())
				continue
			}
			bundle = a.BundleID
			break
		}
		if bundle == "" {
			if len(entries) > 0 {
				losses = append(losses, "attempt receipts without reservation in "+dir.Name())
			}
			continue
		}
		if reviewAttemptKey(bundle) != dir.Name() {
			losses = append(losses, "attempt bundle directory identity mismatch")
			continue
		}
		attempts, err := s.readDelegatedReviewAttempts(bundle, &budget)
		if err != nil {
			losses = append(losses, err.Error())
			continue
		}
		out = append(out, attempts...)
	}
	return out, losses
}

func (s Store) delegatedReviewInsights(sinceDays int) *DelegatedReviewInsights {
	runs, lost, err := s.ReadReviewRuns("")
	if err != nil {
		lost = append(lost, err.Error())
	}
	attempts, attemptLoss := s.allDelegatedReviewAttempts()
	lost = append(lost, attemptLoss...)
	receiptBudget := int64(32 << 20)
	for _, a := range attempts {
		if _, err := s.readAttemptReceipt(a, &receiptBudget); err != nil {
			lost = append(lost, err.Error())
		}
	}
	if len(runs) == 0 && len(attempts) == 0 && len(lost) == 0 {
		return nil
	}
	result := &DelegatedReviewInsights{CoverageComplete: len(lost) == 0, Losses: uniqueSorted(lost), Assurance: "declared-draft-outcomes-not-verified-approvals"}
	bundles, adapters := map[string]*DelegatedReviewMetric{}, map[string]*DelegatedReviewMetric{}
	row := func(group map[string]*DelegatedReviewMetric, key string) *DelegatedReviewMetric {
		r := group[key]
		if r == nil {
			r = &DelegatedReviewMetric{Key: key}
			group[key] = r
		}
		return r
	}
	inWindow := func(at string) bool {
		t, _ := time.Parse(time.RFC3339Nano, at)
		return sinceDays == 0 || !t.Before(time.Now().AddDate(0, 0, -sinceDays))
	}
	type event struct{ id, bundle, adapter, reason, at string }
	events := []event{}
	known := map[string]bool{}
	for _, a := range attempts {
		known[a.AttemptID] = true
		events = append(events, event{a.AttemptID, a.BundleID, a.Adapter, a.Reason, a.ReservedAt})
	}
	for _, run := range runs {
		if run.AttemptID == "" {
			events = append(events, event{run.RunID, run.BundleID, run.Adapter, run.Reason, run.StartedAt})
		} else if !known[run.AttemptID] {
			known[run.AttemptID] = true
			events = append(events, event{run.AttemptID, run.BundleID, run.Adapter, run.Reason, run.StartedAt})
			result.CoverageComplete = false
			result.Losses = append(result.Losses, "missing attempt reservation "+run.AttemptID)
		}
		if !inWindow(run.StartedAt) {
			continue
		}
		for _, r := range []*DelegatedReviewMetric{row(bundles, run.BundleID), row(adapters, run.Adapter)} {
			r.Runs++
			if run.Resumes != "" && run.SupersededBy == "" {
				r.Resumes++
			}
			decision := "unknown"
			if run.Status == ReviewRunCompleted && run.SupersededBy == "" && reviewSafetyConflict(run) == "" {
				decision = draftReviewDecision(run.Output)
			}
			switch decision {
			case "approved":
				r.ApprovedDrafts++
			case "changes-requested", "rejected":
				r.RejectingDrafts++
			default:
				r.UnknownDrafts++
			}
		}
	}
	sort.Slice(events, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339Nano, events[i].at)
		b, _ := time.Parse(time.RFC3339Nano, events[j].at)
		if !a.Equal(b) {
			return a.Before(b)
		}
		return events[i].id < events[j].id
	})
	seenBundle := map[string]bool{}
	for _, e := range events {
		if inWindow(e.at) {
			for _, r := range []*DelegatedReviewMetric{row(bundles, e.bundle), row(adapters, e.adapter)} {
				r.Attempts++
				if seenBundle[e.bundle] {
					r.Reruns++
					if e.reason == "" {
						r.RerunsWithoutReason++
					}
				}
			}
		}
		seenBundle[e.bundle] = true
	}
	finish := func(group map[string]*DelegatedReviewMetric) []DelegatedReviewMetric {
		out := []DelegatedReviewMetric{}
		for _, r := range group {
			n := r.ApprovedDrafts + r.RejectingDrafts
			if n > 0 {
				rate := float64(r.ApprovedDrafts) / float64(n)
				r.ApprovalRate = &rate
			}
			out = append(out, *r)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
		return out
	}
	result.ByBundle = finish(bundles)
	result.ByAdapter = finish(adapters)
	return result
}
