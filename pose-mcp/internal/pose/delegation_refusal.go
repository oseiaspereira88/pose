package pose

// Refusals of delegated runs carry a reason code, the same on the CLI and
// over MCP (spec pose-delegation-mcp R5). The message stays the engine's; the
// code names the class a caller acts on.

import (
	"errors"
	"os"
	"path/filepath"
)

// Reason codes of a refused delegated run.
const (
	DelegationReasonRerunWithoutReason = "rerun-without-reason"
	DelegationReasonBudgetExhausted    = "budget-exhausted"
	DelegationReasonContainment        = "containment-incompatible"
	DelegationReasonAgentUnavailable   = "agent-unavailable"
	DelegationReasonNoAgent            = "no-agent"
)

// DelegateRootEnv names the project root a `pose delegate` process runs in
// when its caller resolved it — the MCP server, for the project it
// authorized — so the process never rediscovers another one from the
// directory it starts in (spec pose-delegation-mcp R6, review round 1, B01).
const DelegateRootEnv = "POSE_DELEGATE_ROOT"

// SealedCheckoutEnv marks a process a delegated review launched in its sealed
// checkout, as the path of that checkout. Diagnostic commands that would
// republish a derived record there (the delivery index) read it and leave the checkout as sealed: a review that
// changes its subject is invalid (spec pose-validation-evidence-material-binding R7).
const SealedCheckoutEnv = "POSE_SEALED_CHECKOUT"

// IsSealedCheckout reports whether root is the sealed checkout the environment
// names. The marker carries the checkout's path, so a fixture or another project
// a reviewer's process works in is not mistaken for the sealed one.
func IsSealedCheckout(root string) bool {
	sealed := os.Getenv(SealedCheckoutEnv)
	if sealed == "" || root == "" {
		return false
	}
	resolve := func(p string) string {
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		if real, err := filepath.EvalSymlinks(p); err == nil {
			p = real
		}
		return filepath.Clean(p)
	}
	return resolve(sealed) == resolve(root)
}

// DelegationRefusal is a refusal with its reason code.
type DelegationRefusal struct {
	Code string
	Err  error
}

func (r *DelegationRefusal) Error() string { return r.Err.Error() }
func (r *DelegationRefusal) Unwrap() error { return r.Err }

func refusal(code string, err error) error {
	if err == nil {
		return nil
	}
	return &DelegationRefusal{Code: code, Err: err}
}

// DelegationReasonCode is the reason code of err, or "" when it is not a
// coded refusal.
func DelegationReasonCode(err error) string {
	var r *DelegationRefusal
	if errors.As(err, &r) {
		return r.Code
	}
	var unavailable *ErrNoDelegationAgentAvailable
	if errors.As(err, &unavailable) {
		return DelegationReasonAgentUnavailable
	}
	if errors.Is(err, ErrNoReviewerAdapter) {
		return DelegationReasonNoAgent
	}
	return ""
}
