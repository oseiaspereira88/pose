package mcpenforce

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// defaultOPAPath is the OPA policy path used when PolicyConfig.OPAPath is empty.
// Consumers should set a domain-specific path (e.g. "pose/mcp/allow").
const defaultOPAPath = "mcp/allow"

// PolicyConfig configures the authorization gate.
// Zero value → allow-all (dev/single-node mode, no OPA required).
type PolicyConfig struct {
	// OPAURL is the base URL of an OPA server, e.g. "http://opa:8181".
	// Empty = no OPA enforcement (dev mode — allow all requests).
	OPAURL string

	// OPAPath is the OPA policy path under /v1/data/. Defaults to defaultOPAPath.
	OPAPath string

	// Timeout for a single OPA evaluation. Defaults to 2s. Policy evaluation
	// exceeding this budget is treated as a denial (default-deny).
	Timeout time.Duration

	// RequirePrincipal, when true, denies any request without an authenticated
	// principal — independent of OPA, and including dev/allow-all mode. Closes
	// the anonymous-caller gap for multi-tenant exposure.
	RequirePrincipal bool

	// RequireIdentity, when true, denies any request without a bound Execution
	// Identity (PolicyInput.RunID empty) — independent of OPA. Enforces ADR-007
	// run-bound identity at the tool-call boundary.
	RequireIdentity bool

	// HTTPClient overrides the HTTP client used for OPA queries (for testing).
	HTTPClient *http.Client

	// Clock supplies the current time for the identity time-box check. Nil =
	// time.Now. Injectable for deterministic tests and skew handling.
	Clock func() time.Time
}

// PolicyGate evaluates per-request authorization via OPA when configured.
type PolicyGate struct {
	cfg PolicyConfig
	hc  *http.Client
	now func() time.Time
}

// NewPolicyGate builds a PolicyGate. Empty config → allow-all dev mode.
func NewPolicyGate(cfg PolicyConfig) *PolicyGate {
	if cfg.OPAPath == "" {
		cfg.OPAPath = defaultOPAPath
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Second
	}
	hc := cfg.HTTPClient
	if hc == nil {
		// Generous transport timeout so context cancellation is the real
		// deadline; OPA should be colocated and fast.
		hc = &http.Client{Timeout: cfg.Timeout + 500*time.Millisecond}
	}
	now := cfg.Clock
	if now == nil {
		now = time.Now
	}
	return &PolicyGate{cfg: cfg, hc: hc, now: now}
}

// Evaluate checks whether the request described by input is authorized.
//
// Order: RequirePrincipal (anonymous) → RequireIdentity (no run-bound identity)
// → time-box (expired identity) → dev allow-all when no OPAURL → OPA REST query.
// The identity checks are local and cheap, applied before any OPA call. On any
// OPA error it returns the error so the caller can apply DenyDecision
// (default-deny contract).
func (g *PolicyGate) Evaluate(ctx context.Context, input PolicyInput) (PolicyDecision, error) {
	d := PolicyDecision{
		Principal:  input.Principal,
		ProjectID:  input.ProjectID,
		ProjectIDs: input.ProjectIDs,
		ToolName:   input.ToolName,
		RunID:      input.RunID,
	}
	if input.InvalidProjectScope {
		d.Violations = []string{"invalid_project_scope"}
		return d, nil
	}
	if g.cfg.RequirePrincipal && strings.TrimSpace(input.Principal) == "" {
		d.Violations = []string{"missing_principal"}
		return d, nil
	}
	if g.cfg.RequireIdentity && strings.TrimSpace(input.RunID) == "" {
		d.Violations = []string{"missing_identity"}
		return d, nil
	}
	// Time-box: a presented identity that has expired is denied regardless of
	// OPA or RequireIdentity (a single clock governs expiry, ADR-007).
	if !input.ExpiresAt.IsZero() && g.now().After(input.ExpiresAt) {
		d.Violations = []string{"identity_expired"}
		return d, nil
	}
	if g.cfg.OPAURL == "" {
		d.Allow = true
		return d, nil
	}
	evalCtx, cancel := context.WithTimeout(ctx, g.cfg.Timeout)
	defer cancel()
	allow, violations, err := g.queryOPA(evalCtx, input)
	if err != nil {
		return d, err
	}
	d.Allow = allow
	d.Violations = violations
	return d, nil
}
