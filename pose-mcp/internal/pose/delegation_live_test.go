package pose

import (
	"os"
	"strings"
	"testing"
)

// R9: live events are normalized line by line, redacted with the credentials
// and every session the stream has declared, read incrementally, and a line
// still being written is not read.
func TestDelegationLiveEventsAreRedactedAndIncremental(t *testing.T) {
	s := Store{Root: t.TempDir()}
	w := s.startDelegationLive("att-live", DelegationEventsClaudeStreamJSON, []string{"synthetic-token"}, `"session_id":"([^"]+)"`, "")
	if w == nil {
		t.Fatal("no live writer")
	}
	lines := []string{
		`{"type":"system","subtype":"init","session_id":"sess-live-123"}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"echo synthetic-token sess-live-123"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"ok"}]}}`,
	}
	for _, line := range lines {
		w.Write([]byte(line + "\n"))
	}
	w.Write([]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"half`))
	events, offset, err := s.ReadDelegationLiveEvents("att-live", 0)
	if err != nil || len(events) != 3 {
		t.Fatalf("events: %+v %v", events, err)
	}
	raw, _ := os.ReadFile(s.delegationLivePath("att-live"))
	if strings.Contains(string(raw), "synthetic-token") || strings.Contains(string(raw), "sess-live-123") {
		t.Fatalf("a secret reached the live events: %s", raw)
	}
	if events[2].Kind != DelegationEventToolOutput || events[2].Tool != "shell" {
		t.Fatalf("a tool's output lost its tool across lines: %+v", events[2])
	}
	w.Write([]byte(` done"}]}}` + "\n"))
	more, _, err := s.ReadDelegationLiveEvents("att-live", offset)
	if err != nil || len(more) != 1 || more[0].Text != "half done" || more[0].Seq != 4 {
		t.Fatalf("incremental read: %+v %v", more, err)
	}
	w.Close()
	if none, _, err := s.ReadDelegationLiveEvents("att-gone", 0); err != nil || len(none) != 0 {
		t.Fatalf("a missing file: %+v %v", none, err)
	}
}
