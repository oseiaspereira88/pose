package pose

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

// The same session as each vendor records it: one turn, one shell command
// that succeeds, one agent message. Synthetic, shaped after recorded streams.
const codexStreamFixture = `{"type":"thread.started","thread_id":"0199-aaaa"}
{"type":"turn.started"}
{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"/usr/bin/bash -lc 'git status --short'","status":"in_progress"}}
{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"/usr/bin/bash -lc 'git status --short'","aggregated_output":"","exit_code":0,"status":"completed"}}
{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"decision: approved"}}
{"type":"turn.completed","usage":{"input_tokens":10,"output_tokens":2}}
`

const claudeStreamFixture = `{"type":"system","subtype":"init","session_id":"5f0c-bbbb","model":"claude-opus-5-5"}
{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"git status --short"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"","is_error":false}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"decision: approved"}]}}
{"type":"result","subtype":"success","is_error":false,"result":"decision: approved","session_id":"5f0c-bbbb"}
`

func eventShape(events []DelegationEvent) []string {
	out := []string{}
	for _, e := range events {
		code := ""
		if e.ExitCode != nil {
			code = "exit=" + string(rune('0'+*e.ExitCode))
		}
		out = append(out, strings.Join(strings.Fields(e.Kind+" "+e.Tool+" "+e.Text+" "+code), " "))
	}
	return out
}

// R9: recorded Codex and Claude streams normalize to the same event sequence.
func TestDelegationEventsAreTheSameForBothVendors(t *testing.T) {
	codex := NormalizeDelegationEvents(DelegationEventsCodexJSONL, []byte(codexStreamFixture), DelegationStreamFinal(DelegationEventsCodexJSONL, []byte(codexStreamFixture)))
	claude := NormalizeDelegationEvents(DelegationEventsClaudeStreamJSON, []byte(claudeStreamFixture), DelegationStreamFinal(DelegationEventsClaudeStreamJSON, []byte(claudeStreamFixture)))
	want := []string{"turn", "tool_call shell git status --short", "tool_output shell exit=0", "message decision: approved", "final decision: approved"}
	if got := eventShape(codex); !reflect.DeepEqual(got, want) {
		t.Fatalf("codex: %q", got)
	}
	if got := eventShape(claude); !reflect.DeepEqual(got, want) {
		t.Fatalf("claude: %q", got)
	}
	for i, e := range codex {
		if e.Version != DelegationEventSchemaVersion || e.Seq != i+1 {
			t.Fatalf("event %d is not versioned and sequenced: %+v", i, e)
		}
	}
}

// R9: a usage limit is an event of its own in both streams, with the reset
// instant when the stream carries one; a stream event never carries a session.
func TestDelegationEventsReportUsageLimits(t *testing.T) {
	codex := "{\"type\":\"turn.started\"}\n{\"type\":\"turn.failed\",\"error\":{\"message\":\"You've hit your usage limit. Try again at 10:45 AM.\"}}\n"
	claude := "{\"type\":\"rate_limit_event\",\"rate_limit_info\":{\"status\":\"rejected\",\"resetsAt\":1792198200},\"session_id\":\"secret-session\"}\n" +
		"{\"type\":\"result\",\"is_error\":true,\"result\":\"You've hit your session limit · resets 2:10pm\",\"session_id\":\"secret-session\"}\n"
	c := NormalizeDelegationEvents(DelegationEventsCodexJSONL, []byte(codex), "")
	if len(c) != 2 || c[1].Kind != DelegationEventUsageLimit {
		t.Fatalf("codex limit: %+v", c)
	}
	a := NormalizeDelegationEvents(DelegationEventsClaudeStreamJSON, []byte(claude), "")
	if len(a) != 2 || a[0].Kind != DelegationEventUsageLimit || a[0].ResetAt != "2026-10-17T00:50:00Z" || a[1].Kind != DelegationEventUsageLimit {
		t.Fatalf("claude limit: %+v", a)
	}
	if strings.Contains(string(EncodeDelegationEvents(a)), "secret-session") {
		t.Fatal("a normalized event carries a session reference")
	}
	errs := NormalizeDelegationEvents(DelegationEventsCodexJSONL, []byte("{\"type\":\"turn.failed\",\"error\":{\"message\":\"Selected model is at capacity.\"}}\n"), "")
	if len(errs) != 1 || errs[0].Kind != DelegationEventError {
		t.Fatalf("an error that is not a limit: %+v", errs)
	}
}

// R9: plain and unknown formats yield only the final message; log lines and
// malformed JSON on the same channel are skipped; texts are bounded.
func TestDelegationEventsArePlainAndBounded(t *testing.T) {
	if got := NormalizeDelegationEvents(DelegationEventsPlain, []byte("anything\n"), "done"); len(got) != 1 || got[0].Kind != DelegationEventFinal {
		t.Fatalf("plain: %+v", got)
	}
	if got := NormalizeDelegationEvents("", nil, ""); len(got) != 0 {
		t.Fatalf("no format and no final: %+v", got)
	}
	noisy := "2026-10-26 WARN something\n{not json\n" + codexStreamFixture
	if got := NormalizeDelegationEvents(DelegationEventsCodexJSONL, []byte(noisy), ""); len(got) != 4 {
		t.Fatalf("noise was not skipped: %+v", got)
	}
	long := strings.Repeat("é", delegationEventTextBytes)
	got := NormalizeDelegationEvents(DelegationEventsPlain, nil, long)
	if len(got[0].Text) > delegationEventTextBytes+8 || !strings.HasSuffix(got[0].Text, "…") {
		t.Fatalf("an unbounded text: %d bytes", len(got[0].Text))
	}
	if !strings.HasPrefix(boundEventTail(long), "… ") {
		t.Fatal("a tool output tail is not marked as cut")
	}
}

// R9: the events file is bound to its run; a changed file is refused.
func TestDelegationEventsAreBoundToTheRecord(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root}
	dir, err := ensureReviewArtifactDir(root, reviewRunsDir, true)
	if err != nil {
		t.Fatal(err)
	}
	events := EncodeDelegationEvents(NormalizeDelegationEvents(DelegationEventsCodexJSONL, []byte(codexStreamFixture), "decision: approved"))
	if err := publishDelegationEvents(dir, "rrn-0000000000000001", events); err != nil {
		t.Fatal(err)
	}
	run := ReviewRun{RunID: "rrn-0000000000000001", EventsDigest: digestBytes(events)}
	read, err := s.ReadDelegationEvents(run)
	if err != nil || len(read) != 5 {
		t.Fatalf("read: %v %+v", err, read)
	}
	if err := os.WriteFile(dir+"/rrn-0000000000000001.events.ndjson", append(events, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadDelegationEvents(run); err == nil {
		t.Fatal("events that are not the bound bytes were read")
	}
}

// B02 of the first review: a value hidden by a JSON escape is redacted after
// decoding, so the events never re-expose what the transcript withheld.
func TestDelegationEventsRedactDecodedValues(t *testing.T) {
	stream := `{"type":"item.started","item":{"type":"command_execution","command":"echo synthetic-token s3ssion-ref"}}` + "\n"
	out, err := redactedDelegationEvents(DelegationEventsCodexJSONL, []byte(stream), "", []string{"synthetic-token"}, "s3ssion-ref")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "synthetic-token") || strings.Contains(string(out), "s3ssion-ref") || !strings.Contains(string(out), "[credential]") {
		t.Fatalf("decoded values leaked: %s", out)
	}
}

// F02 and F03: the last step is cut on a rune boundary and a long line is
// read whole.
func TestDelegationEventsKeepRunesAndLongLines(t *testing.T) {
	step := delegationStepText(DelegationEvent{Kind: DelegationEventMessage, Text: strings.Repeat("a", 66) + "é" + strings.Repeat("b", 30)})
	if !utf8.ValidString(step) {
		t.Fatalf("invalid UTF-8: %q", step)
	}
	long := `{"type":"item.completed","item":{"type":"command_execution","aggregated_output":"` + strings.Repeat("x", 17<<20) + `","exit_code":0}}` + "\n" +
		`{"type":"item.completed","item":{"type":"agent_message","text":"decision: approved"}}` + "\n"
	if got := DelegationStreamFinal(DelegationEventsCodexJSONL, []byte(long)); got != "decision: approved" {
		t.Fatalf("a line over 16 MiB hid the final message: %q", got)
	}
}
