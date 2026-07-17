package sidecar

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"
)

// Exchange is one recorded live MCP tools/call (ADR-022 / ADR-017): the request,
// the authorization outcome, and the response (when not streamed). Correlated to
// the run via RunID so a run is replayable post-hoc from its exchanges.
type Exchange struct {
	At           time.Time       `json:"at"`
	RunID        string          `json:"run_id,omitempty"`
	Principal    string          `json:"principal,omitempty"`
	ProjectID    string          `json:"project_id,omitempty"`
	Tool         string          `json:"tool"`
	Allowed      bool            `json:"allowed"`
	Violations   []string        `json:"violations,omitempty"`
	RequestArgs  json.RawMessage `json:"request_args,omitempty"`
	ResponseBody json.RawMessage `json:"response_body,omitempty"` // absent when denied or streamed
	Streamed     bool            `json:"streamed,omitempty"`
}

// ExchangeRecorder receives every gated live MCP exchange. Implementations must
// be safe for concurrent use and should not block the proxy meaningfully.
type ExchangeRecorder interface {
	Record(ctx context.Context, ex Exchange)
}

// NopRecorder discards exchanges (default).
type NopRecorder struct{}

// Record implements ExchangeRecorder.
func (NopRecorder) Record(context.Context, Exchange) {}

// JSONLRecorder appends one JSON object per exchange to w (e.g. the run dir's
// exchange log, wired to the ADR-017 recorder). Safe for concurrent use.
type JSONLRecorder struct {
	mu sync.Mutex
	w  io.Writer
}

// NewJSONLRecorder builds a JSONLRecorder writing to w.
func NewJSONLRecorder(w io.Writer) *JSONLRecorder { return &JSONLRecorder{w: w} }

// Record implements ExchangeRecorder.
func (r *JSONLRecorder) Record(_ context.Context, ex Exchange) {
	line, err := json.Marshal(ex)
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = r.w.Write(append(line, '\n'))
}
