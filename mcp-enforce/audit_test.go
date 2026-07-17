package mcpenforce

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

// TestSlogAuditor_Golden pins the audit-event schema (allow + deny) emitted by
// the default SlogAuditor. The slog timestamp is stripped so the golden is
// deterministic.
func TestSlogAuditor_Golden(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})
	a := NewSlogAuditor(slog.New(h), "pose-mcp")

	a.Record(context.Background(), PolicyDecision{
		Allow: true, Principal: "alice", ProjectID: "proj.a", ToolName: "pose_get_spec",
	})
	a.Record(context.Background(), PolicyDecision{
		Allow: false, Principal: "eve", ProjectID: "proj.a", ToolName: "pose_get_spec",
		Violations: []string{"principal_not_authorized"},
	})

	compareGolden(t, "testdata/audit_events.jsonl", buf.Bytes())
}

func TestNopAuditor_Discards(t *testing.T) {
	// Must not panic and must accept any decision.
	NopAuditor{}.Record(context.Background(), PolicyDecision{Allow: false, Violations: []string{"x"}})
}

func TestNewSlogAuditor_Defaults(t *testing.T) {
	a := NewSlogAuditor(nil, "")
	if a.logger == nil {
		t.Error("nil logger should fall back to slog.Default()")
	}
	if a.component != "mcp" {
		t.Errorf("component = %q, want default \"mcp\"", a.component)
	}
}
