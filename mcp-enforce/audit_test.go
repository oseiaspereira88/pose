package mcpenforce

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
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

// TestSlogAuditor_TruncatesOversizedIdentityFields proves an abusive,
// oversized X-MCP-Principal/project_id header (a client controls both,
// unvalidated) cannot inflate the audit log without bound — defense in
// depth on top of these fields being legitimate, by-design audit data.
func TestSlogAuditor_TruncatesOversizedIdentityFields(t *testing.T) {
	var buf bytes.Buffer
	a := NewSlogAuditor(slog.New(slog.NewJSONHandler(&buf, nil)), "pose-mcp")

	huge := strings.Repeat("x", 10_000)
	a.Record(context.Background(), PolicyDecision{Allow: true, Principal: huge, ProjectID: huge, ToolName: "pose_get_spec"})

	out := buf.String()
	if strings.Contains(out, huge) {
		t.Fatal("an oversized identity field was logged verbatim, unbounded")
	}
	if !strings.Contains(out, "(truncated)") {
		t.Errorf("expected a truncation marker in the audit line: %s", out)
	}
	if len(out) > 2*maxAuditFieldLen+512 {
		t.Errorf("audit line length = %d, expected it bounded near maxAuditFieldLen, not proportional to the 10,000-byte input", len(out))
	}
}
