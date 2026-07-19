package mcpserver

// Safe validation orchestration (spec pose-safe-validate-orchestration):
// pose-mcp resolves an immutable, digest-pinned check plan and enforces
// approval + explicit Execution Identity authorization before handing it to
// a pluggable Harness executor. pose-mcp never runs the plan itself — MCP
// requests, the Harness executes, POSE owns the plan/result state machine.
// Local `pose validate` is completely unaffected (a separate command path).
//
// State machine (single-writer-per-request via the registry mutex):
//
//	pending_approval --approve--> approved --submit--> submitted
//	pending_approval --reject-->  rejected  (terminal)
//	{pending_approval,approved}   --cancel--> cancelled (terminal)
//	submitted                     --cancel--> cancelled (best-effort local
//	                                          marker only; propagating
//	                                          cancellation to a running
//	                                          Harness execution is the
//	                                          executor's responsibility)

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type callerIdentityCtxKeyType struct{}

var callerIdentityCtxKey = callerIdentityCtxKeyType{}

type callerIdentity struct {
	RunID  string
	Scopes []string
}

// withCallerIdentity carries the caller's verified Execution Identity (if
// any) into ctx, downstream of PolicyGate.Evaluate. runID is empty when no
// identity was presented — a plain-anonymous or dev-mode call.
func withCallerIdentity(ctx context.Context, runID string, scopes []string) context.Context {
	return context.WithValue(ctx, callerIdentityCtxKey, callerIdentity{RunID: runID, Scopes: scopes})
}

func callerIdentityFromContext(ctx context.Context) callerIdentity {
	id, _ := ctx.Value(callerIdentityCtxKey).(callerIdentity)
	return id
}

// HarnessExecutor is satisfied by a real Harness client. pose-mcp defines
// and enforces the request/approval contract; Submit is the only point
// where control passes outside this process, and only after every R2 gate
// (project scope, policy allow, bound Execution Identity) has passed.
type HarnessExecutor interface {
	Submit(ctx context.Context, req ApprovedValidationRequest) (executionID string, err error)
}

// ApprovedValidationRequest is the immutable material handed to the
// Harness: exactly what was approved, nothing a later mutation could widen.
type ApprovedValidationRequest struct {
	RequestID      string
	Plan           ValidationPlan
	ApproverRunID  string
	ApproverScopes []string
}

// ValidationPlan is the versioned, digest-pinned selection an approval
// binds to (R1, R3). Digest covers every field below plus the exact bytes
// of the validation matrix at resolution time — any drift invalidates it.
type ValidationPlan struct {
	SchemaVersion int    `json:"schema_version"`
	ProjectID     string `json:"project_id"`
	GitHead       string `json:"git_head,omitempty"`
	StackFilter   string `json:"stack_filter,omitempty"`
	ModuleFilter  string `json:"module_filter,omitempty"`
	ChangedFrom   string `json:"changed_from,omitempty"`
	ChangedTo     string `json:"changed_to,omitempty"`
	MatrixSHA256  string `json:"matrix_sha256"`
	Digest        string `json:"digest"` // sha256 over every field above
}

const validationPlanSchema = 1

func computePlanDigest(p ValidationPlan) string {
	p.Digest = ""
	b, _ := json.Marshal(p)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

type validationRequestState string

const (
	statePendingApproval validationRequestState = "pending_approval"
	stateApproved        validationRequestState = "approved"
	stateRejected        validationRequestState = "rejected"
	stateSubmitted       validationRequestState = "submitted"
	stateCancelled       validationRequestState = "cancelled"
)

type validationRequest struct {
	ID             string                 `json:"request_id"`
	Plan           ValidationPlan         `json:"plan"`
	State          validationRequestState `json:"state"`
	CreatedAt      string                 `json:"created_at"`
	ApproverRunID  string                 `json:"approver_run_id,omitempty"`
	ApproverScopes []string               `json:"approver_scopes,omitempty"`
	Decision       string                 `json:"decision,omitempty"`
	Rationale      string                 `json:"rationale,omitempty"`
	DecidedAt      string                 `json:"decided_at,omitempty"`
	ExecutionID    string                 `json:"execution_id,omitempty"`
	SubmittedAt    string                 `json:"submitted_at,omitempty"`
	CancelledAt    string                 `json:"cancelled_at,omitempty"`
	CancelReason   string                 `json:"cancel_reason,omitempty"`
}

// orchestrator is an in-process reference implementation of the request
// registry. It is intentionally not "central" storage — a real multi-replica
// deployment centralizes run state in Conductor (like conductor_run_*
// already does) and plugs execution in via HarnessExecutor; this registry
// exists so the state machine is real, local, and directly testable without
// requiring that external system.
type orchestrator struct {
	mu   sync.Mutex
	byID map[string]*validationRequest
	seq  int
}

func newOrchestrator() *orchestrator { return &orchestrator{byID: map[string]*validationRequest{}} }

func (o *orchestrator) nextID() string {
	o.seq++
	var buf [4]byte
	_, _ = rand.Read(buf[:])
	return fmt.Sprintf("vreq_%d_%s", o.seq, hex.EncodeToString(buf[:]))
}

var errValidationRequestNotFound = fmt.Errorf("validation request not found")

func (o *orchestrator) request(root, projectID, stackFilter, moduleFilter, changedFrom, changedTo string) (*validationRequest, error) {
	matrixPath := filepath.Join(root, ".pose", "indexes", "validation-matrix.json")
	matrixBytes, err := os.ReadFile(matrixPath)
	if err != nil {
		return nil, fmt.Errorf("validation matrix not found at %s", matrixPath)
	}
	digestSum := sha256.Sum256(matrixBytes)
	head := ""
	if out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output(); err == nil {
		head = strings.TrimSpace(string(out))
	}
	plan := ValidationPlan{
		SchemaVersion: validationPlanSchema,
		ProjectID:     projectID,
		GitHead:       head,
		StackFilter:   stackFilter,
		ModuleFilter:  moduleFilter,
		ChangedFrom:   changedFrom,
		ChangedTo:     changedTo,
		MatrixSHA256:  hex.EncodeToString(digestSum[:]),
	}
	plan.Digest = computePlanDigest(plan)

	o.mu.Lock()
	defer o.mu.Unlock()
	req := &validationRequest{
		ID: o.nextID(), Plan: plan, State: statePendingApproval,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	o.byID[req.ID] = req
	return req, nil
}

// approve requires a bound, verified Execution Identity (runID non-empty —
// callers with no identity are rejected before this method ever runs) and
// binds the decision to the exact plan digest the caller confirms (R2/R3):
// a digest mismatch is substitution — the plan changed since the request
// was resolved — and is rejected, not silently re-approved against drift.
func (o *orchestrator) approve(id, planDigest, decision, rationale, runID string, scopes []string) (*validationRequest, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	req, ok := o.byID[id]
	if !ok {
		return nil, errValidationRequestNotFound
	}
	if req.State != statePendingApproval {
		return nil, fmt.Errorf("validation request %s is %s, not pending_approval (no replay of a decided request)", id, req.State)
	}
	if planDigest != req.Plan.Digest {
		return nil, fmt.Errorf("plan digest mismatch: approval must reference the exact resolved plan (substitution rejected)")
	}
	req.ApproverRunID, req.ApproverScopes = runID, scopes
	req.Decision, req.Rationale = decision, rationale
	req.DecidedAt = time.Now().UTC().Format(time.RFC3339)
	switch decision {
	case "approve":
		req.State = stateApproved
	case "reject":
		req.State = stateRejected
	default:
		return nil, fmt.Errorf("decision must be %q or %q", "approve", "reject")
	}
	return req, nil
}

// submit is idempotent: resubmitting an already-submitted request returns
// the same execution_id without invoking the executor a second time.
func (o *orchestrator) submit(ctx context.Context, id string, executor HarnessExecutor) (*validationRequest, error) {
	o.mu.Lock()
	req, ok := o.byID[id]
	if !ok {
		o.mu.Unlock()
		return nil, errValidationRequestNotFound
	}
	if req.State == stateSubmitted {
		o.mu.Unlock()
		return req, nil // idempotent replay: same execution_id, no re-submit
	}
	if req.State != stateApproved {
		o.mu.Unlock()
		return nil, fmt.Errorf("validation request %s is %s, not approved", id, req.State)
	}
	approved := ApprovedValidationRequest{RequestID: req.ID, Plan: req.Plan, ApproverRunID: req.ApproverRunID, ApproverScopes: req.ApproverScopes}
	o.mu.Unlock() // never hold the lock across an external Submit call

	execID, err := executor.Submit(ctx, approved)
	if err != nil {
		return nil, fmt.Errorf("harness submit: %w", err)
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	// Re-check under lock: a concurrent submit could have already landed.
	if req.State == stateSubmitted {
		return req, nil
	}
	req.ExecutionID = execID
	req.State = stateSubmitted
	req.SubmittedAt = time.Now().UTC().Format(time.RFC3339)
	return req, nil
}

func (o *orchestrator) status(id string) (*validationRequest, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	req, ok := o.byID[id]
	if !ok {
		return nil, errValidationRequestNotFound
	}
	cp := *req
	return &cp, nil
}

func (o *orchestrator) cancel(id, reason string) (*validationRequest, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	req, ok := o.byID[id]
	if !ok {
		return nil, errValidationRequestNotFound
	}
	switch req.State {
	case stateRejected, stateCancelled:
		return nil, fmt.Errorf("validation request %s is already terminal (%s)", id, req.State)
	}
	req.State = stateCancelled
	req.CancelledAt = time.Now().UTC().Format(time.RFC3339)
	req.CancelReason = reason
	return req, nil
}
