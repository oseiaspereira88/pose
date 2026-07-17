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

// Record implements Auditor. The run_id attribute is included only when the
// decision carries an Execution Identity, keeping anonymous-call logs unchanged.
func (a *SlogAuditor) Record(ctx context.Context, d PolicyDecision) {
	args := []any{"principal", d.Principal, "project_id", d.ProjectID, "tool", d.ToolName}
	if len(d.ProjectIDs) > 0 {
		args = append(args, "project_ids", d.ProjectIDs)
	}
	if d.RunID != "" {
		args = append(args, "run_id", d.RunID)
	}
	if d.Allow {
		a.logger.InfoContext(ctx, a.component+": policy allowed", append(args, "event_type", "policy.decided")...)
		return
	}
	a.logger.WarnContext(ctx, a.component+": policy denied",
		append(args, "violations", d.Violations, "event_type", "policy.violation")...)
}
