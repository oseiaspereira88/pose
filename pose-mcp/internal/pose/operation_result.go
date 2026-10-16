package pose

import (
	"context"
	"errors"
	"fmt"
)

// OperationResult is the result of an operation that runs steps or can wait
// (spec pose-cli-mcp-operation-results, ADR 2026-10-16 "Operation results
// separate execution from verdict"). It names what the call executed and what
// the domain confirmed; it is never a gate verdict. Adapters render it and map
// it to an exit code; the human and the JSON renderers read the same value.
type OperationResult struct {
	SchemaVersion int                  `json:"schema_version"`
	Operation     string               `json:"operation"`
	Scope         string               `json:"scope,omitempty"`
	Outcome       string               `json:"outcome"`
	Step          string               `json:"step,omitempty"`
	StepsRun      []OperationStep      `json:"steps_run"`
	Pending       []OperationPending   `json:"pending,omitempty"`
	NextAction    string               `json:"next_action,omitempty"`
	Effects       []OperationEffect    `json:"effects"`
	Unconfirmed   []OperationUncertain `json:"unconfirmed,omitempty"`
	Resumable     bool                 `json:"resumable"`
	Error         *OperationError      `json:"error,omitempty"`
	Data          any                  `json:"data,omitempty"`
}

// OperationResultSchemaVersion is the schema of OperationResult.
const OperationResultSchemaVersion = 1

// Outcomes of an operation. They are disjoint and describe the execution.
const (
	// OutcomeCompleted: every step this call ran succeeded; for a query, the
	// query ran (its payload says whether the scope is ready).
	OutcomeCompleted = "completed"
	// OutcomeAlreadySatisfied: no step ran in this call because the stored
	// state already says done. It is not a fresh verification of any gate.
	OutcomeAlreadySatisfied = "already-satisfied"
	// OutcomeWaiting: stopped on another actor; Pending and NextAction say
	// who owes what and how to continue.
	OutcomeWaiting = "waiting"
	// OutcomeBlocked: the plan refuses the next step; nothing ran for it.
	OutcomeBlocked = "blocked"
	// OutcomeConflict: the input digest no longer matches the repository.
	OutcomeConflict = "conflict"
	// OutcomeFailed: a step ran and failed.
	OutcomeFailed = "failed"
	// OutcomeInconclusive: a deadline or a cancellation ended the execution;
	// there is no verdict.
	OutcomeInconclusive = "inconclusive"
)

// OperationStep is one step this call executed and how it ended.
type OperationStep struct {
	ID     string `json:"id"`
	Result string `json:"result"`
}

// OperationPending is one item the operation waits on.
type OperationPending struct {
	ID     string `json:"id"`
	Kind   string `json:"kind,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// OperationEffect is an effect the domain confirmed: the step returned
// without error and the repository reflects it.
type OperationEffect struct {
	Step   string `json:"step"`
	Effect string `json:"effect"`
	Ref    string `json:"ref,omitempty"`
}

// OperationUncertain names a step that failed after it may have written. The
// absence of an effect in Effects never proves a rollback.
type OperationUncertain struct {
	Step   string `json:"step"`
	Reason string `json:"reason"`
}

// OperationError is the typed cause of an outcome other than completed,
// already-satisfied or waiting.
type OperationError struct {
	Cause   string `json:"cause"`
	Message string `json:"message"`
}

// NewOperationResult starts a result with the empty lists a consumer can
// range over.
func NewOperationResult(operation, scope string) OperationResult {
	return OperationResult{SchemaVersion: OperationResultSchemaVersion, Operation: operation, Scope: scope, StepsRun: []OperationStep{}, Effects: []OperationEffect{}}
}

// Fail ends the result with an outcome and its typed cause.
func (r *OperationResult) Fail(outcome, step, cause, message string) {
	r.Outcome, r.Step = outcome, step
	r.Error = &OperationError{Cause: cause, Message: message}
}

// Execution causes of an inconclusive result.
const (
	ExecutionCauseDeadline = "deadline-exceeded"
	ExecutionCauseCanceled = "canceled"
)

// GateExecutionError is a gate subprocess that did not finish: its deadline
// passed or its caller cancelled it. It carries no verdict. PartialOutput is
// what the process wrote before it was stopped and is never complete.
type GateExecutionError struct {
	Command       string
	Cause         string
	PartialOutput string
	Cleanup       string
}

func (e *GateExecutionError) Error() string {
	return fmt.Sprintf("%s did not finish (%s): no verdict; %s", e.Command, e.Cause, e.Cleanup)
}

// executionCause classifies a finished context: the deadline and the
// cancellation are execution causes, checked before an exit status is read
// as a verdict.
func executionCause(ctx context.Context) string {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return ExecutionCauseDeadline
	case errors.Is(ctx.Err(), context.Canceled):
		return ExecutionCauseCanceled
	default:
		return ""
	}
}
