package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
	SchemaVersion int                         `json:"schema_version"`
	Scope         string                      `json:"scope"`
	Digest        string                      `json:"digest"`
	Inputs        CloseoutPlanInputs          `json:"inputs"`
	Steps         []CloseoutPlanStep          `json:"steps"`
	BundleID      string                      `json:"bundle_id,omitempty"`
	Pending       []ReviewAttestationPendency `json:"pending,omitempty"`
	Blockers      []string                    `json:"blockers,omitempty"`
	Terminal      bool                        `json:"terminal"`
}

// CloseoutPlanInputs bind a plan to the state it was computed from.
type CloseoutPlanInputs struct {
	SpecDigest   string `json:"spec_digest"`
	Revision     string `json:"revision"`
	PolicyDigest string `json:"policy_digest"`
	VerifyState  string `json:"verify_state"`
}

// CloseoutPlanStep is one step and its state: done, pending, blocked or
// waiting (on a reviewer).
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
	closeoutCheckpointDir     = ".pose/closeout-plans"
	CloseoutPlanSchemaVersion = 1
)

// generatedCloseoutPaths are written by the plan's own steps; a dirty one is
// expected while closing and does not block the context step.
var generatedCloseoutPaths = []string{".pose/indexes/", ".pose/results/", ".pose/state/", ".pose/review-bundles/", ".pose/review-attestations/", closeoutCheckpointDir + "/"}

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
	plan.Inputs = CloseoutPlanInputs{SpecDigest: digestHex(raw), Revision: gitHeadAtRoot(s.Root), PolicyDigest: directoryDigest(filepath.Join(s.Root, ".pose", "policy"))}
	step := func(id, state, reason string) {
		plan.Steps = append(plan.Steps, CloseoutPlanStep{ID: id, State: state, Reason: reason})
	}

	if spec.Status == "done" {
		for _, id := range []string{CloseoutStepContext, CloseoutStepTrace, CloseoutStepEvidence, CloseoutStepIndex, CloseoutStepSeal, CloseoutStepMechanical, CloseoutStepJudgment, CloseoutStepVerify, CloseoutStepTransition} {
			step(id, CloseoutStateDone, "the spec is done")
		}
		plan.Terminal = true
		plan.Digest = closeoutPlanDigest(plan)
		return plan, nil
	}
	if spec.Status != "in-progress" {
		step(CloseoutStepContext, CloseoutStateBlocked, "the spec is "+spec.Status+"; only an in-progress spec is closed")
		plan.Blockers = append(plan.Blockers, "spec status is "+spec.Status)
		plan.Digest = closeoutPlanDigest(plan)
		return plan, nil
	}
	if dirty := s.ungeneratedChanges(); len(dirty) > 0 {
		step(CloseoutStepContext, CloseoutStateBlocked, "uncommitted changes outside generated paths ("+strings.Join(dirty, ", ")+"): evidence sealed now would not describe a commit")
		plan.Blockers = append(plan.Blockers, "commit or discard the working-tree changes before closing")
		plan.Digest = closeoutPlanDigest(plan)
		return plan, nil
	}
	step(CloseoutStepContext, CloseoutStateDone, "")
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
	plan.Digest = ""
	digest, _ := digestJSON(plan)
	return digest
}

// NextCloseoutStep is the first step that is not done.
func (p CloseoutPlan) NextCloseoutStep() (CloseoutPlanStep, bool) {
	for _, st := range p.Steps {
		if st.State != CloseoutStateDone {
			return st, true
		}
	}
	return CloseoutPlanStep{}, false
}

// ungeneratedChanges lists working-tree changes outside the paths the plan
// itself writes.
func (s Store) ungeneratedChanges() []string {
	out, err := exec.Command("git", "-C", s.Root, "status", "--porcelain", "--untracked-files=all").Output()
	if err != nil {
		return nil
	}
	var dirty []string
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		generated := false
		for _, prefix := range generatedCloseoutPaths {
			generated = generated || strings.HasPrefix(path, prefix)
		}
		if !generated {
			dirty = append(dirty, path)
		}
	}
	return dirty
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

// ReadCloseoutCheckpoint returns the checkpoint of an interrupted apply.
func (s Store) ReadCloseoutCheckpoint(ref string) (CloseoutCheckpoint, bool) {
	raw, err := os.ReadFile(closeoutCheckpointPath(s.Root, ref))
	if err != nil {
		return CloseoutCheckpoint{}, false
	}
	var cp CloseoutCheckpoint
	if json.Unmarshal(raw, &cp) != nil {
		return CloseoutCheckpoint{}, false
	}
	return cp, true
}

// WriteCloseoutCheckpoint persists progress; ClearCloseoutCheckpoint removes
// it when the closeout completes.
func (s Store) WriteCloseoutCheckpoint(cp CloseoutCheckpoint) error {
	if _, err := ensureReviewArtifactDir(s.Root, closeoutCheckpointDir, true); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(closeoutCheckpointPath(s.Root, cp.Scope), append(raw, '\n'), 0o644)
}

func (s Store) ClearCloseoutCheckpoint(ref string) {
	_ = os.Remove(closeoutCheckpointPath(s.Root, ref))
	_ = os.Remove(filepath.Join(s.Root, filepath.FromSlash(closeoutCheckpointDir)))
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
