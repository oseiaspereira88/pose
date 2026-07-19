package mcpenforce

import (
	"context"
	"log/slog"
)

// Auditor records the outcome of every policy evaluation — both allow and deny —
// producing a complete trail of who invoked what. Implementations MUST NOT log
// payload content, only authorization metadata.
type Auditor interface {
	Record(ctx context.Context, d PolicyDecision)
}

// NopAuditor discards every decision. Useful in tests or when audit is wired
// through a different sink.
type NopAuditor struct{}

// Record implements Auditor.
func (NopAuditor) Record(context.Context, PolicyDecision) {}

// SlogAuditor emits one structured slog line per decision:
//
//	allow → Info  "<component>: policy allowed"  event_type=policy.decided
//	deny  → Warn  "<component>: policy denied"    event_type=policy.violation
type SlogAuditor struct {
	logger    *slog.Logger
	component string
}

// NewSlogAuditor builds a SlogAuditor. A nil logger uses slog.Default(); an empty
// component falls back to "mcp". The component prefixes each log message so an
// operator can tell which MCP server emitted the decision.
func NewSlogAuditor(logger *slog.Logger, component string) *SlogAuditor {
	if logger == nil {
		logger = slog.Default()
	}
	if component == "" {
		component = "mcp"
	}
	return &SlogAuditor{logger: logger, component: component}
}

// maxAuditFieldLen bounds every caller-supplied identity field before it
// reaches a log line. Principal/ProjectID arrive verbatim from request
// headers with no format validation — they are authorization identifiers
// by design (this Auditor's entire purpose is recording who invoked
// what), not secrets, but an unbounded value is still a log-injection /
// volume-abuse surface worth closing regardless of intent.
const maxAuditFieldLen = 128

func truncateForAudit(s string) string {
	if len(s) <= maxAuditFieldLen {
		return s
	}
	return s[:maxAuditFieldLen] + "...(truncated)"
}

// Record implements Auditor. The run_id attribute is included only when the
// decision carries an Execution Identity, keeping anonymous-call logs unchanged.
//
// principal/project_id are authorization identifiers from the caller's own
// request (X-MCP-Principal / project_id), not secrets — recording them is
// this Auditor's documented purpose (see the package doc: "a complete
// trail of who invoked what"). Bounded via truncateForAudit as defense in
// depth against an oversized value.
func (a *SlogAuditor) Record(ctx context.Context, d PolicyDecision) {
	args := []any{"principal", truncateForAudit(d.Principal), "project_id", truncateForAudit(d.ProjectID), "tool", d.ToolName}
	if len(d.ProjectIDs) > 0 {
		args = append(args, "project_ids", d.ProjectIDs)
	}
	if d.RunID != "" {
		args = append(args, "run_id", d.RunID)
	}
	if d.Allow {
		a.logger.InfoContext(ctx, a.component+": policy allowed", append(args, "event_type", "policy.decided")...) // codeql[go/clear-text-logging] see Record doc comment above
		return
	}
	a.logger.WarnContext(ctx, a.component+": policy denied", append(args, "violations", d.Violations, "event_type", "policy.violation")...) // codeql[go/clear-text-logging] see Record doc comment above
}
