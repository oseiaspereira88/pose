package pose

// Atomic start (spec pose-abm-atomic-start): `pose start` records, before
// managed execution begins, the R/A/D baseline an amendment is measured
// against, and moves the spec from draft to in-progress as one recoverable
// operation. It records observable precedence — what was written before the
// managed run — and never claims when anyone decided anything.
//
// Preview is always available and read-only. Apply requires the review
// policy's atomic_start_version capability, so adopting it is explicit.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	StartSchemaVersion       = 1
	AtomicStartPolicyVersion = 1
)

// StartPlan is the digest-bound preview of a start.
type StartPlan struct {
	SchemaVersion int          `json:"schema_version"`
	Spec          string       `json:"spec"`
	SpecPath      string       `json:"spec_path"`
	SpecDigest    string       `json:"spec_digest"`
	Revision      string       `json:"revision"`
	FromStatus    string       `json:"from_status"`
	Ready         bool         `json:"ready"`
	WaitingOn     []WaitingRef `json:"waiting_on"`
	Reason        string       `json:"reason,omitempty"`
	Obligations   []string     `json:"obligations"`
	// ObligationItems types Obligations: each declared review criterion the
	// start leaves owed at closeout (spec pose-typed-producer-diagnostics).
	ObligationItems []Diagnostic            `json:"obligation_items,omitempty"`
	Baseline        ContractNodesProjection `json:"baseline"`
	Digest          string                  `json:"digest"`
}

// StartRecord is the journal of one applied start.
type StartRecord struct {
	SchemaVersion    int                     `json:"schema_version"`
	Spec             string                  `json:"spec"`
	PlanDigest       string                  `json:"plan_digest"`
	Revision         string                  `json:"revision"`
	SpecDigestBefore string                  `json:"spec_digest_before"`
	SpecDigestAfter  string                  `json:"spec_digest_after"`
	Baseline         ContractNodesProjection `json:"baseline"`
	Phase            string                  `json:"phase"` // baseline-recorded | started
	RecordedAt       string                  `json:"recorded_at"`
}

// NodeOrigin classifies one current node against the recorded baseline.
type NodeOrigin struct {
	ID     string `json:"id"`
	Origin string `json:"origin"` // recorded-before-managed-execution | introduced-during-execution | legacy-unbaselined
}

// StartStatus reports a spec's start state without writing anything.
type StartStatus struct {
	SchemaVersion  int          `json:"schema_version"`
	Spec           string       `json:"spec"`
	Status         string       `json:"status"`
	Phase          string       `json:"phase"` // not-started | baseline-recorded | started
	Nodes          []NodeOrigin `json:"nodes"`
	Removed        []string     `json:"removed,omitempty"`
	Obligations    []string     `json:"obligations"`
	Reconciliation []string     `json:"reconciliation,omitempty"`
	// Notes explain a state that needs no action, such as a spec that was
	// already in progress before atomic start was adopted.
	Notes           []string `json:"notes,omitempty"`
	CapabilityAdopt bool     `json:"capability_adopted"`
	// Lock is the start lock another call holds, with its owner's state
	// (spec pose-journal-checkpoint-and-lock-recovery R9).
	Lock *StartLockError `json:"lock,omitempty"`
}

// StartBaselineWriteError is a baseline that could not be published. The
// baseline is published whole or not at all, so nothing was recorded (spec
// pose-writer-coordination-and-atomic-publication R1/R6).
type StartBaselineWriteError struct{ Err error }

func (e *StartBaselineWriteError) Error() string { return e.Err.Error() }
func (e *StartBaselineWriteError) Unwrap() error { return e.Err }

// StartFailureHook is called after each phase apply reaches: tests return an
// error to interrupt it, and adapters observe which phases the call reached.
type StartFailureHook func(phase string) error

func startError(code string) error { return fmt.Errorf("start: %s", code) }

func startRecordPath(root, slug string) string {
	return filepath.Join(root, ".pose", "starts", slug+".json")
}

// AtomicStartAdopted reads the review policy's capability.
func (s Store) AtomicStartAdopted() (bool, error) {
	policy, err := s.GetReviewPolicy()
	if err != nil {
		return false, err
	}
	return policy.AtomicStartVersion == AtomicStartPolicyVersion, nil
}

// atomicStartCutoff is the policy's atomic_start_adopted_at, or "".
func (s Store) atomicStartCutoff() string {
	policy, err := s.GetReviewPolicy()
	if err != nil {
		return ""
	}
	return policy.AtomicStartAdoptedAt
}

// PreviewStart builds the plan. It reads the spec, its readiness, the declared
// review plan and the contract-node baseline, and writes nothing.
func (s Store) PreviewStart(slug string) (StartPlan, error) {
	plan := StartPlan{SchemaVersion: StartSchemaVersion, Spec: slug, WaitingOn: []WaitingRef{}, Obligations: []string{}}
	spec, err := s.GetSpec(slug)
	if err != nil {
		return plan, err
	}
	raw, err := os.ReadFile(spec.Path)
	if err != nil {
		return plan, err
	}
	rel, err := filepath.Rel(s.Root, spec.Path)
	if err != nil {
		return plan, err
	}
	plan.SpecPath = filepath.ToSlash(rel)
	plan.SpecDigest = digestHex(raw)
	plan.Revision = gitHeadAtRoot(s.Root)
	plan.FromStatus = spec.Status
	if readiness, err := s.SpecReadiness(slug); err == nil {
		plan.Ready, plan.WaitingOn, plan.Reason = readiness.Ready, readiness.WaitingOn, readiness.Reason
	} else {
		plan.Reason = "readiness unavailable: " + err.Error()
	}
	// Governed effects: an unsatisfied action request restricting start
	// keeps the plan not ready, and apply recomputes the plan, so a request
	// opened after the preview still refuses the start.
	if adopted, adoptErr := s.AgencyReadinessAdopted(); adoptErr == nil && adopted {
		if restrictions, restrictErr := s.GovernedEffectRestrictions(PhaseStart, []string{slug}); restrictErr == nil {
			for _, o := range restrictions {
				plan.WaitingOn = append(plan.WaitingOn, WaitingRef{Ref: "action:" + o.Source.Detail, Reason: o.Message, Code: "action-request-pending"})
				plan.Ready = false
				if plan.Reason == "" || plan.Reason == "ready" {
					plan.Reason = "an action request restricts start"
				}
			}
		} else {
			plan.Ready = false
			plan.Reason = "governed effects could not be read: " + restrictErr.Error()
		}
	}
	if spec.Status != "draft" {
		plan.Ready = false
		if plan.Reason == "" {
			plan.Reason = fmt.Sprintf("spec status is %q; start moves a draft", spec.Status)
		}
	}
	plan.Obligations = s.startObligations(slug)
	for _, criterion := range plan.Obligations {
		plan.ObligationItems = append(plan.ObligationItems, NewDiagnostic("review-criterion-owed", "review criterion "+criterion+" is answered at closeout", "spec:"+slug+"#criterion:"+criterion))
	}
	plan.Baseline = ProjectContractNodes(slug, spec.Body)
	plan.Digest = startPlanDigest(plan)
	return plan, nil
}

// startObligations lists the criteria the declared review plan will ask for.
// It needs no future structural delta; once a subject exists the same call
// recomputes them from what is observed.
func (s Store) startObligations(slug string) []string {
	if s.indexPin == nil {
		s = s.PinIndexes()
	}
	plan, err := s.ReviewPlan("spec:" + slug)
	if err != nil {
		return []string{}
	}
	out := []string{}
	for _, criterion := range plan.Criteria {
		out = append(out, criterion.ID)
	}
	sort.Strings(out)
	return out
}

func startPlanDigest(plan StartPlan) string {
	plan.Digest = ""
	raw, _ := json.Marshal(plan)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// ApplyStart records the baseline and moves the spec to in-progress. The
// plan must still describe the repository; the same plan again is a no-op or
// resumes an interrupted run, and any other plan for a started spec fails.
func (s Store) ApplyStart(plan StartPlan, expectedDigest string, failAfter StartFailureHook) (StartRecord, error) {
	adopted, err := s.AtomicStartAdopted()
	if err != nil {
		return StartRecord{}, err
	}
	if !adopted {
		return StartRecord{}, startError("atomic-start-capability-not-adopted")
	}
	if expectedDigest == "" || plan.Digest != expectedDigest || startPlanDigest(plan) != plan.Digest {
		return StartRecord{}, startError("plan-digest-mismatch")
	}
	if ValidateSlug(plan.Spec) != nil {
		return StartRecord{}, startError("invalid-spec")
	}
	unlock, err := lockStart(s.Root, plan.Spec, "start.apply")
	if err != nil {
		return StartRecord{}, err
	}
	defer unlock()

	recordPath := startRecordPath(s.Root, plan.Spec)
	specPath := filepath.Join(s.Root, filepath.FromSlash(plan.SpecPath))
	record, exists, err := readStartRecord(recordPath)
	if err != nil {
		return StartRecord{}, err
	}
	if exists && record.PlanDigest != plan.Digest {
		return record, startError("spec-already-started-with-another-plan")
	}
	if !exists {
		current, err := s.PreviewStart(plan.Spec)
		if err != nil {
			return StartRecord{}, err
		}
		if current.Digest != plan.Digest {
			return StartRecord{}, startError("stale-plan")
		}
		if !plan.Ready {
			return StartRecord{}, startError("spec-not-ready")
		}
		raw, err := os.ReadFile(specPath)
		if err != nil {
			return StartRecord{}, err
		}
		after := setSpecLifecycle(string(raw), plan.Spec, "in-progress")
		record = StartRecord{SchemaVersion: StartSchemaVersion, Spec: plan.Spec, PlanDigest: plan.Digest, Revision: plan.Revision,
			SpecDigestBefore: plan.SpecDigest, SpecDigestAfter: digestHex([]byte(after)), Baseline: plan.Baseline,
			Phase: "baseline-recorded", RecordedAt: time.Now().UTC().Format(time.RFC3339)}
		if err := writeStartRecord(recordPath, record, true); err != nil {
			return StartRecord{}, &StartBaselineWriteError{Err: err}
		}
	}
	return finishStart(recordPath, specPath, plan.Spec, record, failAfter)
}

// ResumeStart finishes an apply that recorded its baseline for digest and
// already moved the spec, then stopped before the record said started. Once
// the spec moved, a fresh preview has another digest, so the persisted record
// — not a new plan — names what to finish (spec
// pose-cli-mcp-operation-results R6). It refuses with
// transition-not-published when the spec has not moved. failAfter observes
// the phases it reaches, as in ApplyStart.
func (s Store) ResumeStart(slug, digest string, failAfter StartFailureHook) (StartRecord, error) {
	adopted, err := s.AtomicStartAdopted()
	if err != nil {
		return StartRecord{}, err
	}
	if !adopted {
		return StartRecord{}, startError("atomic-start-capability-not-adopted")
	}
	if ValidateSlug(slug) != nil {
		return StartRecord{}, startError("invalid-spec")
	}
	current, err := s.PreviewStart(slug)
	if err != nil {
		return StartRecord{}, err
	}
	unlock, err := lockStart(s.Root, slug, "start.resume")
	if err != nil {
		return StartRecord{}, err
	}
	defer unlock()
	recordPath := startRecordPath(s.Root, slug)
	record, exists, err := readStartRecord(recordPath)
	if err != nil {
		return StartRecord{}, err
	}
	if !exists || digest == "" || record.PlanDigest != digest {
		return record, startError("plan-digest-mismatch")
	}
	if record.Phase == "started" {
		return record, nil
	}
	// Only a transition that was already published is recovered here. A
	// spec still in its recorded "before" state has published nothing: its
	// start is judged again by the current plan, with today's readiness and
	// governed restrictions, never authorized by the persisted baseline.
	specPath := filepath.Join(s.Root, filepath.FromSlash(current.SpecPath))
	raw, err := os.ReadFile(specPath)
	if err != nil {
		return record, err
	}
	switch digestHex(raw) {
	case record.SpecDigestAfter:
		return finishStart(recordPath, specPath, slug, record, failAfter)
	case record.SpecDigestBefore:
		return record, startError("transition-not-published")
	default:
		return record, startError("compare-and-swap-conflict")
	}
}

// Recovery states of a start that recorded its baseline and did not finish.
const (
	StartRecoveryStarted   = "started"              // nothing to recover
	StartRecoveryPublished = "transition-published" // the spec moved; the same digest finishes it
	StartRecoveryRetry     = "retry"                // nothing published and the plan is unchanged; the same digest applies it
	StartRecoveryStale     = "stale-baseline"       // nothing published, but the repository moved on: the digest can never apply
	StartRecoveryConflict  = "spec-conflict"        // the spec is in neither recorded state
	StartRecoveryNone      = ""                     // no record for this digest
)

// StartRecovery classifies what a start recorded for digest can still do,
// so a failure never promises a retry that cannot finish (spec
// pose-cli-mcp-operation-results R6).
func (s Store) StartRecovery(slug, digest string) (string, error) {
	record, ok, err := s.GetStartRecord(slug)
	if err != nil || !ok || record.PlanDigest != digest {
		return StartRecoveryNone, err
	}
	if record.Phase == "started" {
		return StartRecoveryStarted, nil
	}
	current, err := s.PreviewStart(slug)
	if err != nil {
		return StartRecoveryNone, err
	}
	raw, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(current.SpecPath)))
	if err != nil {
		return StartRecoveryNone, err
	}
	switch digestHex(raw) {
	case record.SpecDigestAfter:
		return StartRecoveryPublished, nil
	case record.SpecDigestBefore:
		if current.Digest == digest {
			return StartRecoveryRetry, nil
		}
		return StartRecoveryStale, nil
	default:
		return StartRecoveryConflict, nil
	}
}

// finishStart moves the spec to in-progress, reconciling a transition an
// earlier run already wrote, and records the start as started.
func finishStart(recordPath, specPath, slug string, record StartRecord, failAfter StartFailureHook) (StartRecord, error) {
	if record.Phase == "started" {
		return record, nil
	}
	if failAfter != nil {
		if err := failAfter("baseline-recorded"); err != nil {
			return record, err
		}
	}
	raw, err := os.ReadFile(specPath)
	if err != nil {
		return record, err
	}
	switch digestHex(raw) {
	case record.SpecDigestBefore:
		after := []byte(setSpecLifecycle(string(raw), slug, "in-progress"))
		if digestHex(after) != record.SpecDigestAfter {
			return record, startError("plan-digest-mismatch")
		}
		if err := writeTransferFile(specPath, after, record.SpecDigestBefore); err != nil {
			return record, err
		}
	case record.SpecDigestAfter:
		// A previous run already moved the spec.
	default:
		return record, startError("compare-and-swap-conflict")
	}
	if failAfter != nil {
		if err := failAfter("transitioned"); err != nil {
			return record, err
		}
	}
	record.Phase = "started"
	if err := writeStartRecord(recordPath, record, false); err != nil {
		return record, err
	}
	return record, nil
}

// CancelStart undoes a start that recorded its baseline but never moved the
// spec, so no half transition is left behind.
func (s Store) CancelStart(slug string) error {
	if ValidateSlug(slug) != nil {
		return startError("invalid-spec")
	}
	unlock, err := lockStart(s.Root, slug, "start.cancel")
	if err != nil {
		return err
	}
	defer unlock()
	recordPath := startRecordPath(s.Root, slug)
	record, exists, err := readStartRecord(recordPath)
	if err != nil {
		return err
	}
	if !exists {
		return startError("no-start-to-cancel")
	}
	spec, err := s.GetSpec(slug)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(spec.Path)
	if err != nil {
		return err
	}
	if record.Phase != "baseline-recorded" || digestHex(raw) != record.SpecDigestBefore {
		return startError("start-already-transitioned")
	}
	return os.Remove(recordPath)
}

// GetStartStatus classifies the current nodes against the recorded baseline
// and lists what needs reconciliation. It never writes.
func (s Store) GetStartStatus(slug string) (StartStatus, error) {
	status := StartStatus{SchemaVersion: StartSchemaVersion, Spec: slug, Phase: "not-started", Nodes: []NodeOrigin{}}
	spec, err := s.GetSpec(slug)
	if err != nil {
		return status, err
	}
	status.Status = spec.Status
	status.Obligations = s.startObligations(slug)
	adopted, err := s.AtomicStartAdopted()
	if err != nil {
		return status, err
	}
	status.CapabilityAdopt = adopted
	if lock, err := s.StartLockState(slug); err != nil {
		status.Reconciliation = append(status.Reconciliation, "start lock is unreadable: "+err.Error())
	} else {
		status.Lock = lock
	}
	current := ProjectContractNodes(slug, spec.Body)
	record, exists, err := readStartRecord(startRecordPath(s.Root, slug))
	if err != nil {
		status.Reconciliation = append(status.Reconciliation, "start record is unreadable: "+err.Error())
		exists = false
	}
	if !exists {
		for _, node := range current.Nodes {
			status.Nodes = append(status.Nodes, NodeOrigin{ID: node.ID, Origin: "legacy-unbaselined"})
		}
		if adopted && spec.Status != "draft" && !terminalStatuses[spec.Status] {
			// A spec created before the adoption cutoff predates the
			// capability: it is reported, never blocked (spec
			// pose-atomic-start-adoption-cutoff). Without a cutoff the
			// capability applies to every spec, as before.
			if cutoff := s.atomicStartCutoff(); cutoff != "" && spec.CreatedAt != "" && spec.CreatedAt < cutoff {
				status.Notes = append(status.Notes, fmt.Sprintf("spec was created on %s, before atomic start was adopted on %s; its nodes stay legacy-unbaselined and nothing is required", spec.CreatedAt, cutoff))
			} else {
				status.Reconciliation = append(status.Reconciliation, fmt.Sprintf("spec is %s without a recorded start; its nodes stay legacy-unbaselined until a baseline is recorded explicitly", spec.Status))
			}
		}
		return status, nil
	}
	status.Phase = record.Phase
	baseline := CurrentNodeStates(record.Baseline)
	for _, node := range current.Nodes {
		origin := "introduced-during-execution"
		if recorded, ok := baseline[node.ID]; ok && recorded.Hash == node.Hash && recorded.State == node.State {
			origin = "recorded-before-managed-execution"
		}
		status.Nodes = append(status.Nodes, NodeOrigin{ID: node.ID, Origin: origin})
	}
	present := CurrentNodeStates(current)
	for id := range baseline {
		if _, ok := present[id]; !ok {
			status.Removed = append(status.Removed, id)
		}
	}
	sort.Strings(status.Removed)
	if adopted {
		if contractNodesDigest(record.Baseline) != record.Baseline.Digest {
			status.Reconciliation = append(status.Reconciliation, "the recorded baseline was edited after the start")
		}
		if record.Phase == "started" && spec.Status == "draft" {
			status.Reconciliation = append(status.Reconciliation, "the spec was moved back to draft by hand after the start")
		}
		if record.Phase == "baseline-recorded" {
			status.Reconciliation = append(status.Reconciliation, "a start was interrupted after recording its baseline; rerun apply with the same plan or cancel it")
		}
	}
	return status, nil
}

// GetStartRecord returns the recorded start of a spec, if any.
func (s Store) GetStartRecord(slug string) (StartRecord, bool, error) {
	if ValidateSlug(slug) != nil {
		return StartRecord{}, false, startError("invalid-spec")
	}
	return readStartRecord(startRecordPath(s.Root, slug))
}

func readStartRecord(path string) (StartRecord, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return StartRecord{}, false, nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || info.Size() > 4<<20 {
		return StartRecord{}, false, startError("invalid-start-record")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return StartRecord{}, false, err
	}
	var record StartRecord
	if err := strictJSONBytes(raw, &record); err != nil || record.SchemaVersion != StartSchemaVersion {
		return StartRecord{}, false, startError("invalid-start-record")
	}
	return record, true, nil
}

func writeStartRecord(path string, record StartRecord, exclusive bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if exclusive {
		// The baseline is immutable: it is published whole or not at all, and
		// never over another record (spec
		// pose-writer-coordination-and-atomic-publication R1/R2).
		err := PublishFile(path, raw, 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent})
		if errors.Is(err, ErrPublishConflict) {
			return startError("start-record-exists")
		}
		return err
	}
	// The update runs under the spec's lifecycle lock; the record is an
	// authoritative artifact, published whole and synced.
	return PublishFile(path, raw, 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAny})
}
