package observability

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// Logger emits one structured JSON line per record, correlated with the
// active span's trace_id/span_id (R3: logs shall share trace context) and
// with every string field redacted (R3: redact paths, tokens and
// payloads). A nil/no-op Logger (writer == nil) is a true no-op — Emit
// returns immediately without allocating or touching the writer, so a
// disabled instance costs nothing on the hot path.
type Logger struct {
	mu     sync.Mutex
	writer io.Writer
}

func newLogger(w io.Writer) *Logger {
	return &Logger{writer: w}
}

// record is the fixed, deliberately narrow shape every log line takes —
// never a raw payload/argument dump (Non-goal: never export source
// content, repo names or command output by default). Fields are the same
// low-cardinality categories the metrics use, plus a redacted message.
type record struct {
	Time      string `json:"time"`
	TraceID   string `json:"trace_id,omitempty"`
	SpanID    string `json:"span_id,omitempty"`
	Tool      string `json:"tool,omitempty"`
	RiskClass string `json:"risk_class,omitempty"`
	Outcome   string `json:"outcome,omitempty"`
	Message   string `json:"message,omitempty"`
}

// Emit writes one redacted, trace-correlated record. message is passed
// through Message() (paths + secrets redacted) before being stored — call
// sites must not redact it themselves first.
func (l *Logger) Emit(ctx context.Context, tool, riskClass, outcome, message string) {
	if l == nil || l.writer == nil {
		return
	}
	sc := trace.SpanContextFromContext(ctx)
	rec := record{
		Time:      time.Now().UTC().Format(time.RFC3339Nano),
		Tool:      tool,
		RiskClass: riskClass,
		Outcome:   outcome,
		Message:   Message(message),
	}
	if sc.HasTraceID() {
		rec.TraceID = sc.TraceID().String()
	}
	if sc.HasSpanID() {
		rec.SpanID = sc.SpanID().String()
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return
	}
	b = append(b, '\n')
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.writer.Write(b)
}
