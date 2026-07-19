// Package mcpenforce is the canonical authorization-enforcement layer for Harne8
// MCP servers: a per-call policy gate (OPA-backed, default-deny), an auditor for
// allow and deny decisions, and the request-extraction helpers shared by every
// MCP server. It is consumed in-process by Harne8-org Go MCP servers (e.g.
// pose-mcp) and embedded in the enforcement sidecar that fronts foreign servers
// (e.g. graphforge/mcp-server), per ADR-021.
//
// Standard library only; the single network dependency is the OPA REST API.
package mcpenforce

import "time"

// PolicyInput carries the authorization context of one MCP tools/call request.
type PolicyInput struct {
	Principal           string   // identity from X-MCP-Principal (or X-Principal)
	ProjectID           string   // project scope resolved from argument or header
	ProjectIDs          []string // aggregate project scope from project_ids
	InvalidProjectScope bool     // malformed project scope must be denied before OPA
	Method              string   // JSON-RPC method, e.g. "tools/call"
	ToolName            string   // MCP tool name, e.g. "pose_get_spec"

	// Execution Identity scope fields (ADR-007). Reserved and INERT: they are
	// serialized into the OPA input document only when non-zero, and no shipped
	// Rego evaluates them yet. They exist so binding the conductor-issued
	// Execution Identity (spec mcp-execution-identity-scope-binding) is additive
	// and never breaks the OPA wire contract.
	Scopes    []string  // least-privilege scopes granted to the run
	RunID     string    // correlation id of the agent run
	ExpiresAt time.Time // hard expiry of the run's identity (time-box)
}

// PolicyDecision is the outcome of PolicyGate.Evaluate.
type PolicyDecision struct {
	Allow      bool
	Principal  string
	ProjectID  string
	ProjectIDs []string
	ToolName   string
	RunID      string   // Execution Identity run correlation (ADR-007), when present
	Violations []string // non-empty when Allow is false
}

// Metadata returns a map suitable for structured JSON-RPC error data and audit
// logs. Violations are never nil to ease JSON consumers.
func (d PolicyDecision) Metadata() map[string]any {
	v := d.Violations
	if v == nil {
		v = []string{}
	}
	return map[string]any{
		"principal":   d.Principal,
		"project_id":  d.ProjectID,
		"project_ids": d.ProjectIDs,
		"tool":        d.ToolName,
		"violations":  v,
	}
}

// DenyDecision builds a denied decision that carries the request identity and a
// single machine-readable reason code.
func DenyDecision(input PolicyInput, reason string) PolicyDecision {
	return PolicyDecision{
		Allow:      false,
		Principal:  input.Principal,
		ProjectID:  input.ProjectID,
		ProjectIDs: input.ProjectIDs,
		ToolName:   input.ToolName,
		Violations: []string{reason},
	}
}
