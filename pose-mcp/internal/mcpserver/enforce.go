package mcpserver

import mcpenforce "github.com/crisol/mcp-enforce"

// The authorization gate, its types and the auditor are provided by the shared
// mcp-enforce module (ADR-021, single source of truth). They are re-exported
// here under the package's local vocabulary so the server and its tests read
// unchanged; the implementation no longer lives in this package.
type (
	PolicyGate     = mcpenforce.PolicyGate
	PolicyConfig   = mcpenforce.PolicyConfig
	PolicyInput    = mcpenforce.PolicyInput
	PolicyDecision = mcpenforce.PolicyDecision
)

// NewPolicyGate builds a gate from the shared module.
func NewPolicyGate(cfg PolicyConfig) *PolicyGate { return mcpenforce.NewPolicyGate(cfg) }

// defaultAuditor routes policy decisions (allow and deny) to slog under the
// "pose-mcp" component, preserving the prior audit event schema.
var defaultAuditor mcpenforce.Auditor = mcpenforce.NewSlogAuditor(nil, "pose-mcp")
