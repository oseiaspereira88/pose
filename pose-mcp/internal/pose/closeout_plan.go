package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CloseoutPlan is the ordered, recoverable closeout of one spec (spec
// pose-recoverable-closeout-plan). It is computed from the repository every
// time, so resuming after an interruption is re-planning: steps already
// satisfied by the current state read as done, and nothing is repeated.
//
// The plan automates only mechanical steps — regenerating evidence, indexing,
// sealing, recording criteria answered by sealed evidence, the guarded
// transition. A judgment criterion stays pending until a reviewer answers
// it; the plan never writes that answer. It promises a recoverable
// operation, not a transaction over Git and external processes.
type CloseoutPlan struct {
	LifecycleDone bool                        `json:"lifecycle_done"`
	Verification  *CloseoutState              `json:"verification,omitempty"`
	NextAction    string                      `json:"next_action,omitempty"`
	SchemaVersion int                         `json:"schema_version"`
	Scope         string                      `json:"scope"`
	Digest        string                      `json:"digest"`
	Inputs        CloseoutPlanInputs          `json:"inputs"`
	Steps         []CloseoutPlanStep          `json:"steps"`
	BundleID      string                      `json:"bundle_id,omitempty"`
	Pending       []ReviewAttestationPendency `json:"pending,omitempty"`
	Blockers      []string                    `json:"blockers,omitempty"`
	Terminal      bool                        `json:"terminal"`
	// Checkpoint is how the checkpoint of an earlier apply reads, set by the
	// adapter that shows the plan; it is outside the digest.
	Checkpoint *CloseoutCheckpointRead `json:"checkpoint,omitempty"`
}

// CloseoutPlanInputs bind a plan to the state it was computed from.
type CloseoutPlanInputs struct {
	SpecDigest   string `json:"spec_digest"`
	Revision     string `json:"revision"`
	PolicyDigest string `json:"policy_digest"`
	VerifyState  string `json:"verify_state"`
}

// CloseoutPlanStep is one step and its state: done, pending, blocked,
// waiting (on a reviewer), or not-run (no historical execution inferred).
type CloseoutPlanStep struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

// Closeout step ids, in execution order.
const (
	CloseoutStepContext       = "context"
	CloseoutStepTrace         = "trace"
	CloseoutStepEvidence      = "evidence"
	CloseoutStepIndex         = "index"
	CloseoutStepSeal          = "seal"
	CloseoutStepMechanical    = "mechanical-attestation"
	CloseoutStepJudgment      = "judgment"
	CloseoutStepVerify        = "verify"
	CloseoutStepTransition    = "transition"
	CloseoutStateDone         = "done"
	CloseoutStatePending      = "pending"
	CloseoutStateBlocked      = "blocked"
	CloseoutStateWaiting      = "waiting"
	CloseoutStateNotRun       = "not-run"
	closeoutCheckpointDir     = ".pose/closeout-plans"
	CloseoutPlanSchemaVersion = 1
)

// PlanCloseout computes the plan for a spec scope.
func (s Store) PlanCloseout(ref string) (CloseoutPlan, error) {
	scope, err := ParseScopeRef(ref)
	if err != nil {
		return CloseoutPlan{}, err
	}
	if scope.Kind != "spec" {
		return CloseoutPlan{}, fmt.Errorf("pose: the closeout plan covers spec scopes; use `pose close %s` for %s scopes", ref, scope.Kind)
	}
	spec, err := s.GetSpec(scope.Slug)
	if err != nil {
		return CloseoutPlan{}, err
	}
	raw, err := os.ReadFile(spec.Path)
	if err != nil {
		return CloseoutPlan{}, err
	}
	plan := CloseoutPlan{SchemaVersion: CloseoutPlanSchemaVersion, Scope: ref}
	head, gitErr := snapshotGit(s.Root, "rev-parse", "--verify", "HEAD")
	if gitErr != nil {
		return CloseoutPlan{}, gitErr
	}
	plan.Inputs = CloseoutPlanInputs{SpecDigest: digestHex(raw), Revision: strings.TrimSpace(string(head)), PolicyDigest: directoryDigestIn(s.fsys(), filepath.Join(s.Root, ".pose", "policy"))}
	step := func(id, state, reason string) {
		plan.Steps = append(plan.Steps, CloseoutPlanStep{ID: id, State: state, Reason: reason})
	}

	if spec.Status == "done" {
		state, err := s.GetCloseoutState(ref)
		if err != nil {
			return plan, err
		}
		plan.LifecycleDone = true
		plan.Verification = &state
		plan.Terminal = state.Terminal
		plan.Blockers = append([]string{}, state.Blockers...)
		plan.NextAction = state.NextAction
		plan.BundleID = state.Review.BundleID
		step(CloseoutStepContext, CloseoutStateDone, "current scope and applicable contract read")
		for _, id := range []string{CloseoutStepTrace, CloseoutStepEvidence, CloseoutStepIndex, CloseoutStepSeal, CloseoutStepMechanical, CloseoutStepJudgment} {
			step(id, CloseoutStateNotRun, "no historical execution is inferred from lifecycle done; see verification basis and limitations")
		}
		if state.Terminal {
			plan.Inputs.VerifyState = "terminal-verified"
			step(CloseoutStepVerify, CloseoutStateDone, "applicable terminal verifier passed in this query")
		} else {
			plan.Inputs.VerifyState = "terminal-blocked"
			step(CloseoutStepVerify, CloseoutStateBlocked, strings.Join(state.Blockers, "; ")+"; "+state.NextAction)
		}
		step(CloseoutStepTransition, CloseoutStateDone, "lifecycle is already done; no transition executed")
		plan.Digest = closeoutPlanDigest(plan)
		return plan, nil
	}
	if spec.Status != "in-progress" {
		step(CloseoutStepContext, CloseoutStateBlocked, "the spec is "+spec.Status+"; only an in-progress spec is closed")
		plan.Blockers = append(plan.Blockers, "spec status is "+spec.Status)
		plan.Digest = closeoutPlanDigest(plan)
		return plan, nil
	}
	dirty, gitErr := s.ungeneratedChanges(scope.Slug, plan.Inputs.Revision)
	if gitErr != nil {
		return CloseoutPlan{}, gitErr
	}
	if len(dirty) > 0 {
		step(CloseoutStepContext, CloseoutStateBlocked, "uncommitted changes outside generated paths ("+strings.Join(dirty, ", ")+"): evidence sealed now would not describe a commit")
		plan.Blockers = append(plan.Blockers, "commit or discard the working-tree changes before closing")
		plan.Digest = closeoutPlanDigest(plan)
		return plan, nil
	}
	step(CloseoutStepContext, CloseoutStateDone, "")
	if _, findings, historyErr := s.InspectAmendmentHistory(*spec); historyErr != nil || len(findings) > 0 {
		reason := strings.Join(findings, "; ")
		if historyErr != nil {
			reason = historyErr.Error()
		}
		reason = "amendment history: " + reason
		step(CloseoutStepTrace, CloseoutStateBlocked, reason)
		plan.Blockers = append(plan.Blockers, reason)
		plan.NextAction = "inspect amendment history and explicitly acknowledge or repair it before closing " + ref
		plan.Digest = closeoutPlanDigest(plan)
		return plan, nil
	}

	// The exit gate comes before sealing: completing the trace edits the spec,
	// which would supersede a bundle sealed first (spec
	// pose-quickstart-real-lifecycle).
	if blockers := RequirementTraceCloseoutBlockers(string(raw)); len(blockers) > 0 {
		reason := "requirement trace incomplete: " + strings.Join(blockers, "; ") + " — declare each under `### Requirement trace`, e.g. `- R1 [satisfied] test:<TestName>`, or [waived: <reason>] / [withdrawn: <reason>], and commit"
		step(CloseoutStepTrace, CloseoutStateBlocked, reason)
		plan.Blockers = append(plan.Blockers, reason)
		plan.Digest = closeoutPlanDigest(plan)
		return plan, nil
	}
	step(CloseoutStepTrace, CloseoutStateDone, "")

	verification, err := s.VerifyReviewBundle(ref)
	if err != nil {
		return plan, err
	}
	plan.Inputs.VerifyState = verification.State
	if verification.Bundle != nil {
		plan.BundleID = verification.Bundle.BundleID
	}
	sealedOK := func(reason string) {
		step(CloseoutStepEvidence, CloseoutStateDone, reason)
		step(CloseoutStepIndex, CloseoutStateDone, reason)
		step(CloseoutStepSeal, CloseoutStateDone, reason)
	}
	switch verification.State {
	case "ready-to-close", "closed":
		sealedOK("a current sealed bundle exists")
		step(CloseoutStepMechanical, CloseoutStateDone, "an approved attestation exists")
		step(CloseoutStepJudgment, CloseoutStateDone, "an approved attestation exists")
		step(CloseoutStepVerify, CloseoutStateDone, "")
	case "ready-for-review":
		sealedOK("a current sealed bundle exists")
		prepared, prepErr := s.PrepareReviewAttestation(plan.BundleID, "agent:closeout-plan", time.Now())
		switch {
		case prepErr != nil:
			step(CloseoutStepMechanical, CloseoutStateBlocked, prepErr.Error())
			plan.Blockers = append(plan.Blockers, prepErr.Error())
		case len(prepared.Pending) > 0:
			plan.Pending = prepared.Pending
			step(CloseoutStepMechanical, CloseoutStateWaiting, "mechanical criteria are recorded together with the judgment in one attestation")
			step(CloseoutStepJudgment, CloseoutStateWaiting, fmt.Sprintf("%d criterion/criteria need a reviewer's conclusion; the plan never writes it", len(prepared.Pending)))
		default:
			step(CloseoutStepMechanical, CloseoutStatePending, "every required criterion is answered by sealed evidence")
			step(CloseoutStepJudgment, CloseoutStateDone, "no judgment criterion is pending")
		}
		step(CloseoutStepVerify, CloseoutStatePending, "")
	case "ready-to-seal":
		step(CloseoutStepEvidence, CloseoutStateDone, "the prepared bundle has no blocker")
		step(CloseoutStepIndex, CloseoutStateDone, "the prepared bundle has no blocker")
		step(CloseoutStepSeal, CloseoutStatePending, "")
		step(CloseoutStepMechanical, CloseoutStatePending, "")
		step(CloseoutStepJudgment, CloseoutStatePending, "known after sealing")
		step(CloseoutStepVerify, CloseoutStatePending, "")
	case "changes-requested":
		sealedOK("")
		step(CloseoutStepMechanical, CloseoutStateBlocked, "the latest review requested changes; remediate and seal a superseding bundle")
		plan.Blockers = append(plan.Blockers, verification.Blockers...)
	default:
		// needs-validation, superseded, or a prepared bundle with blockers:
		// regenerate the evidence, index it and seal.
		reason := "the prepared bundle reports: " + strings.Join(verification.Blockers, "; ")
		step(CloseoutStepEvidence, CloseoutStatePending, reason)
		step(CloseoutStepIndex, CloseoutStatePending, "change sets and evidence must be in the index before sealing")
		step(CloseoutStepSeal, CloseoutStatePending, "")
		step(CloseoutStepMechanical, CloseoutStatePending, "")
		step(CloseoutStepJudgment, CloseoutStatePending, "known after sealing")
		step(CloseoutStepVerify, CloseoutStatePending, "")
	}
	step(CloseoutStepTransition, CloseoutStatePending, "")
	plan.Digest = closeoutPlanDigest(plan)
	return plan, nil
}

func closeoutPlanDigest(plan CloseoutPlan) string {
	plan.Digest, plan.Checkpoint = "", nil
	digest, _ := digestJSON(plan)
	return digest
}

// NextCloseoutStep is the first step that is not done.
func (p CloseoutPlan) NextCloseoutStep() (CloseoutPlanStep, bool) {
	for _, st := range p.Steps {
		if st.State != CloseoutStateDone && st.State != CloseoutStateNotRun {
			return st, true
		}
	}
	return CloseoutPlanStep{}, false
}

// ungeneratedChanges lists working-tree changes outside the paths the plan
// itself writes.
func (s Store) ungeneratedChanges(slug, head string) ([]string, error) {
	paths, err := gitChangedPaths(s.Root)
	if err != nil {
		return nil, err
	}
	hidden, err := gitMaterializedChanges(s.Root, head)
	if err != nil {
		return nil, err
	}
	paths = append(paths, hidden...)
	trackedChanges := map[string]bool{}
	for _, path := range hidden {
		trackedChanges[path] = true
	}
	policy, err := LoadDeliveryPolicyIn(s.fsys(), s.Root)
	if err != nil {
		return nil, err
	}
	dirty := []string{}
	seen := map[string]bool{}
	for _, path := range paths {
		// The lifecycle owner file is operational state of this exact spec.
		if path == ".pose/starts/."+slug+".lock" {
			continue
		}
		if IsPublisherStaging(path) {
			continue
		}
		generated := closeoutProducedPath(path, slug, policy.ResultsPath, trackedChanges[path])
		if generated {
			_, err := dispatchRegularInputPath(s.Root, path)
			generated = err == nil
		}
		if generated && strings.HasPrefix(path, ".pose/indexes/generations/") {
			view := s.ReadIndexGeneration()
			generated = false
			if view.Status == IndexGenerationCurrent && strings.HasPrefix(path, ".pose/indexes/generations/"+view.Descriptor.GenerationID+"/") {
				name := filepath.Base(path)
				if file, found := view.Descriptor.file(name); found {
					raw, readErr := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(path)))
					generated = readErr == nil && sha256Prefixed(raw) == file.SHA256
				}
			}
		}
		if !generated && !seen[path] {
			dirty = append(dirty, path)
			seen[path] = true
		}
	}
	observedHead, err := snapshotGit(s.Root, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(observedHead)) != head {
		return nil, &GitSnapshotError{"HEAD coherence", fmt.Errorf("HEAD changed during closeout observation; retry with the current inputs")}
	}
	return dirty, nil
}

// Each exception names the actual producer's output, never an entire tree.
// Evidence and attestations already in HEAD are immutable even while closing.
func closeoutProducedPath(path, slug, resultsPath string, trackedChange bool) bool {
	if path == filepath.ToSlash(filepath.Join(closeoutCheckpointDir, "spec-"+slug+".json")) {
		return true
	}
	if resultsPath != "" && path == resultsPath {
		return true
	}
	for _, name := range append(append([]string{}, IndexGenerationFiles...), indexDescriptorName) {
		if path == ".pose/indexes/"+name {
			return true
		}
	}
	parts := strings.Split(path, "/")
	if len(parts) == 5 && parts[0] == ".pose" && parts[1] == "indexes" && parts[2] == indexGenerationsDir && strings.HasPrefix(parts[3], "gen-") {
		id := strings.TrimPrefix(parts[3], "gen-")
		valid := len(id) == 16
		for _, c := range id {
			valid = valid && strings.ContainsRune("0123456789abcdef", c)
		}
		if valid && !trackedChange {
			for _, name := range IndexGenerationFiles {
				if parts[4] == name {
					return true
				}
			}
		}
	}
	if trackedChange {
		return false
	}
	for prefix, idPrefix := range map[string]string{".pose/review-bundles/": "rvb-", ".pose/review-attestations/": "rva-"} {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		name := strings.TrimPrefix(path, prefix)
		if !strings.HasPrefix(name, idPrefix) || !strings.HasSuffix(name, ".json") {
			return false
		}
		id := strings.TrimSuffix(strings.TrimPrefix(name, idPrefix), ".json")
		if len(id) != 16 {
			return false
		}
		for _, c := range id {
			if !strings.ContainsRune("0123456789abcdef", c) {
				return false
			}
		}
		return true
	}
	return false
}

// CloseoutCheckpoint records how far an apply got, so an interrupted run is
// visible and its resume reports where it continues.
type CloseoutCheckpoint struct {
	SchemaVersion int      `json:"schema_version"`
	Scope         string   `json:"scope"`
	StartedAt     string   `json:"started_at"`
	PlanDigest    string   `json:"plan_digest"`
	Completed     []string `json:"completed"`
	LastError     string   `json:"last_error,omitempty"`
}

func closeoutCheckpointPath(root, ref string) string {
	return filepath.Join(root, filepath.FromSlash(closeoutCheckpointDir), strings.ReplaceAll(ref, ":", "-")+".json")
}

// States of a closeout checkpoint (spec pose-journal-checkpoint-and-lock-recovery R3).
const (
	CheckpointAbsent      = "absent"      // no apply was interrupted
	CheckpointValid       = "valid"       // reads, supported, of this scope
	CheckpointCorrupt     = "corrupt"     // does not read as a checkpoint of this scope
	CheckpointUnsupported = "unsupported" // another schema version
	CheckpointUnavailable = "unavailable" // could not be read
)

// CloseoutCheckpointRead is how a checkpoint reads. Only a valid one is
// progress; the others are never taken as absent, because an apply started
// over them would hide what the earlier run left.
type CloseoutCheckpointRead struct {
	State      string              `json:"state"`
	Path       string              `json:"path"`
	Reason     string              `json:"reason,omitempty"`
	Checkpoint *CloseoutCheckpoint `json:"recorded,omitempty"`
}

// maxCloseoutCheckpointBytes bounds a checkpoint.
const maxCloseoutCheckpointBytes = 1 << 20

// InspectCloseoutCheckpoint reads the checkpoint of an interrupted apply and
// says whether it is absent, valid, corrupt, unsupported or unavailable. A
// valid checkpoint names this scope, a plan digest, a start time and only
// known steps, each once. It is a record of progress, not a receipt: the plan
// is computed from the repository, and a completed step the repository does
// not reflect is a conflict for the adapter to report (R5).
func (s Store) InspectCloseoutCheckpoint(ref string) CloseoutCheckpointRead {
	path := closeoutCheckpointPath(s.Root, ref)
	read := CloseoutCheckpointRead{Path: filepath.ToSlash(filepath.Join(closeoutCheckpointDir, filepath.Base(path)))}
	fail := func(state, reason string) CloseoutCheckpointRead {
		read.State, read.Reason = state, reason
		return read
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		read.State = CheckpointAbsent
		return read
	}
	if err != nil {
		return fail(CheckpointUnavailable, err.Error())
	}
	if !info.Mode().IsRegular() {
		return fail(CheckpointCorrupt, "not a regular file")
	}
	if info.Size() > maxCloseoutCheckpointBytes {
		return fail(CheckpointCorrupt, fmt.Sprintf("larger than %d bytes", maxCloseoutCheckpointBytes))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fail(CheckpointUnavailable, err.Error())
	}
	var version struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &version); err != nil {
		return fail(CheckpointCorrupt, err.Error())
	}
	if version.SchemaVersion != CloseoutCheckpointSchemaVersion {
		return fail(CheckpointUnsupported, fmt.Sprintf("schema_version %d is not supported", version.SchemaVersion))
	}
	var cp CloseoutCheckpoint
	if err := rejectDuplicateJSONKeys(raw, false, true); err != nil {
		return fail(CheckpointCorrupt, err.Error())
	}
	if err := strictJSONBytes(raw, &cp); err != nil {
		return fail(CheckpointCorrupt, err.Error())
	}
	if cp.Scope != ref {
		return fail(CheckpointCorrupt, fmt.Sprintf("it records scope %q", cp.Scope))
	}
	if cp.PlanDigest == "" {
		return fail(CheckpointCorrupt, "it records no plan digest")
	}
	if _, err := time.Parse(time.RFC3339, cp.StartedAt); err != nil {
		return fail(CheckpointCorrupt, "started_at is not an RFC 3339 time")
	}
	seen := map[string]bool{}
	for _, id := range cp.Completed {
		if !closeoutStepIDs[id] || seen[id] {
			return fail(CheckpointCorrupt, fmt.Sprintf("completed step %q is unknown or repeated", id))
		}
		seen[id] = true
	}
	read.State, read.Checkpoint = CheckpointValid, &cp
	return read
}

// CloseoutCheckpointSchemaVersion is the schema of CloseoutCheckpoint.
const CloseoutCheckpointSchemaVersion = 1

var closeoutStepIDs = map[string]bool{
	CloseoutStepContext: true, CloseoutStepTrace: true, CloseoutStepEvidence: true, CloseoutStepIndex: true,
	CloseoutStepSeal: true, CloseoutStepMechanical: true, CloseoutStepJudgment: true, CloseoutStepVerify: true,
	CloseoutStepTransition: true,
}

// WriteCloseoutCheckpoint replaces the checkpoint through an exclusive
// temporary file and a rename, so a reader sees the previous checkpoint or
// this one, never part of either. It is not synced: the checkpoint is a
// record of progress the plan does not depend on, not an authoritative
// artifact, so a crash may lose the last update but never presents a torn
// one (R4).
func (s Store) WriteCloseoutCheckpoint(cp CloseoutCheckpoint) error {
	dir, err := ensureReviewArtifactDirIn(s.fsys(), s.Root, closeoutCheckpointDir, true)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".checkpoint-*.tmp")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), closeoutCheckpointPath(s.Root, cp.Scope)); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// ClearCloseoutCheckpoint removes the checkpoint when the closeout completes
// and reports a removal that failed, so a stale checkpoint is never left
// behind silently. The directory is removed only when it is empty.
func (s Store) ClearCloseoutCheckpoint(ref string) error {
	if err := os.Remove(closeoutCheckpointPath(s.Root, ref)); err != nil && !os.IsNotExist(err) {
		return err
	}
	dir := filepath.Join(s.Root, filepath.FromSlash(closeoutCheckpointDir))
	if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
		_ = os.Remove(dir)
	}
	return nil
}

// RequirementTraceCloseoutBlockers is the requirement-trace gate lint applies
// to a done spec, for a spec about to become done.
func RequirementTraceCloseoutBlockers(text string) []string {
	trace := ParseRequirementTrace(text)
	blockers := append([]string{}, trace.Errors...)
	for _, id := range trace.Orphans {
		blockers = append(blockers, id+" is traced but not declared in Requirements")
	}
	if trace.HasSection {
		for _, id := range trace.Missing {
			blockers = append(blockers, id+" has no trace entry")
		}
	} else if len(trace.Requirements) > 0 {
		blockers = append(blockers, "no `### Requirement trace` subsection in Validation")
	}
	return blockers
}
