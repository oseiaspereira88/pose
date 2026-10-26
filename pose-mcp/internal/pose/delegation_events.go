package pose

// Normalized delegation events (spec pose-delegation-surface R9).
//
// An agent's `events` field selects the parser that turns its stream into one
// versioned event schema, the same for every vendor, kept beside the raw
// transcript and bound to the run record by digest. The parsers read the raw
// stream, where a JSON escape no longer hides a value once decoded; every
// decoded value is redacted whole before a parser trims, unquotes or joins it,
// and every field again before it is cut to its bound, so an event never holds
// what the transcript withheld, nor a part of it.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DelegationEventSchemaVersion versions the normalized event.
const DelegationEventSchemaVersion = 1

// Normalized event kinds.
const (
	DelegationEventTurn       = "turn"
	DelegationEventMessage    = "message"
	DelegationEventToolCall   = "tool_call"
	DelegationEventToolOutput = "tool_output"
	DelegationEventUsageLimit = "usage_limit"
	DelegationEventFinal      = "final"
	DelegationEventError      = "error"
)

const (
	delegationEventTextBytes = 4096
	delegationEventsMaxBytes = 8 << 20
)

// DelegationEvent is one normalized event.
type DelegationEvent struct {
	Version  int    `json:"v"`
	Seq      int    `json:"seq"`
	Kind     string `json:"kind"`
	Tool     string `json:"tool,omitempty"`
	Text     string `json:"text,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	ResetAt  string `json:"reset_at,omitempty"`
	// tail bounds Text by its end, for a tool's output.
	tail bool
}

// NormalizeDelegationEvents parses transcript with the parser format selects
// and appends the final message. An unknown or empty format reads as plain:
// only the final message is an event.
func NormalizeDelegationEvents(format string, transcript []byte, final string) []DelegationEvent {
	events, _ := normalizeDelegationEvents(format, transcript, final, func(v string) (string, error) { return v, nil })
	return events
}

// normalizeDelegationEvents passes every decoded value through clean before a
// parser transforms it, and every field again before it is bounded: a trim, an
// unquote or a join can split a value the transcript redacts whole, or join one
// it never held.
func normalizeDelegationEvents(format string, transcript []byte, final string, clean func(string) (string, error)) ([]DelegationEvent, error) {
	var cleanErr error
	field := func(v any) string {
		if cleanErr != nil {
			return ""
		}
		cleaned, err := clean(str(v))
		if err != nil {
			cleanErr = err
		}
		return cleaned
	}
	var events []DelegationEvent
	switch format {
	case DelegationEventsCodexJSONL:
		events = codexEvents(transcript, field)
	case DelegationEventsClaudeStreamJSON:
		events = claudeEvents(transcript, field)
	}
	if cleanErr != nil {
		return nil, cleanErr
	}
	if strings.TrimSpace(final) != "" {
		events = append(events, DelegationEvent{Kind: DelegationEventFinal, Text: final})
	}
	for i := range events {
		e := &events[i]
		for _, field := range []*string{&e.Tool, &e.Text, &e.ResetAt} {
			cleaned, err := clean(*field)
			if err != nil {
				return nil, err
			}
			*field = cleaned
		}
		if e.tail {
			e.Text = boundEventTail(e.Text)
		} else {
			e.Text = boundEventText(e.Text)
		}
		e.Version, e.Seq = DelegationEventSchemaVersion, i+1
	}
	return events, nil
}

// EncodeDelegationEvents renders events as NDJSON.
func EncodeDelegationEvents(events []DelegationEvent) []byte {
	var buf bytes.Buffer
	for _, e := range events {
		raw, _ := json.Marshal(e)
		buf.Write(raw)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func jsonLines(transcript []byte, each func(map[string]any)) {
	// The transcript is already bounded by the containment's stream budget, so
	// a line is read whole whatever its length.
	for _, line := range bytes.Split(transcript, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] != '{' {
			continue // a vendor's log line on the same channel
		}
		var obj map[string]any
		if json.Unmarshal(line, &obj) == nil {
			each(obj)
		}
	}
}

// field yields a decoded value an event may carry; str reads one that only
// classifies.
func codexEvents(transcript []byte, field func(any) string) []DelegationEvent {
	var out []DelegationEvent
	jsonLines(transcript, func(e map[string]any) {
		item, _ := e["item"].(map[string]any)
		itemType, _ := item["type"].(string)
		switch str(e["type"]) {
		case "turn.started":
			out = append(out, DelegationEvent{Kind: DelegationEventTurn})
		case "item.started":
			if itemType == "command_execution" {
				out = append(out, DelegationEvent{Kind: DelegationEventToolCall, Tool: "shell", Text: unwrapShell(field(item["command"]))})
			}
		case "item.completed":
			switch itemType {
			case "agent_message":
				out = append(out, DelegationEvent{Kind: DelegationEventMessage, Text: field(item["text"])})
			case "command_execution":
				code := intOf(item["exit_code"])
				out = append(out, DelegationEvent{Kind: DelegationEventToolOutput, Tool: "shell", ExitCode: &code, Text: field(item["aggregated_output"]), tail: true})
			case "file_change":
				paths := []string{}
				changes, _ := item["changes"].([]any)
				for _, c := range changes {
					if m, ok := c.(map[string]any); ok {
						paths = append(paths, strings.TrimSpace(field(m["kind"])+" "+field(m["path"])))
					}
				}
				out = append(out, DelegationEvent{Kind: DelegationEventToolCall, Tool: "file_change", Text: strings.Join(paths, ", ")})
			}
		case "turn.failed":
			errObj, _ := e["error"].(map[string]any)
			out = append(out, limitOrError(field(errObj["message"]), ""))
		case "error":
			out = append(out, limitOrError(field(e["message"]), ""))
		}
	})
	return out
}

func claudeEvents(transcript []byte, field func(any) string) []DelegationEvent {
	var out []DelegationEvent
	tools := map[string]string{} // tool_use id -> normalized tool name
	jsonLines(transcript, func(e map[string]any) {
		switch str(e["type"]) {
		case "system":
			if str(e["subtype"]) == "init" {
				out = append(out, DelegationEvent{Kind: DelegationEventTurn})
			}
		case "assistant", "user":
			message, _ := e["message"].(map[string]any)
			content, _ := message["content"].([]any)
			for _, c := range content {
				part, _ := c.(map[string]any)
				switch str(part["type"]) {
				case "text":
					if text := strings.TrimSpace(field(part["text"])); text != "" {
						out = append(out, DelegationEvent{Kind: DelegationEventMessage, Text: text})
					}
				case "tool_use":
					input, _ := part["input"].(map[string]any)
					tool, text := field(part["name"]), field(input["command"])
					if tool == "Bash" {
						tool = "shell"
					}
					if text == "" {
						text = field(input["file_path"])
					}
					if text == "" {
						text = field(part["name"])
					}
					tools[str(part["id"])] = tool
					out = append(out, DelegationEvent{Kind: DelegationEventToolCall, Tool: tool, Text: text})
				case "tool_result":
					code := 0
					if b, _ := part["is_error"].(bool); b {
						code = 1
					}
					out = append(out, DelegationEvent{Kind: DelegationEventToolOutput, Tool: tools[str(part["tool_use_id"])], ExitCode: &code, Text: toolResultText(part["content"], field), tail: true})
				}
			}
		case "rate_limit_event":
			info, _ := e["rate_limit_info"].(map[string]any)
			if str(info["status"]) == "rejected" {
				out = append(out, DelegationEvent{Kind: DelegationEventUsageLimit, Text: "usage limit reached", ResetAt: unixInstant(info["resetsAt"])})
			}
		case "result":
			if b, _ := e["is_error"].(bool); b {
				out = append(out, limitOrError(field(e["result"]), ""))
			}
		}
	})
	return out
}

// limitOrError classifies a vendor's failure text: both vendors word a usage
// limit with these phrases (recorded streams, 2026-10).
func limitOrError(text, reset string) DelegationEvent {
	lower := strings.ToLower(text)
	for _, phrase := range []string{"usage limit", "hit your limit", "session limit", "weekly limit", "rate limit"} {
		if strings.Contains(lower, phrase) {
			return DelegationEvent{Kind: DelegationEventUsageLimit, Text: text, ResetAt: reset}
		}
	}
	return DelegationEvent{Kind: DelegationEventError, Text: text}
}

func toolResultText(content any, field func(any) string) string {
	switch v := content.(type) {
	case string:
		return field(v)
	case []any:
		parts := []string{}
		for _, c := range v {
			if m, ok := c.(map[string]any); ok {
				parts = append(parts, field(m["text"]))
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func unwrapShell(command string) string {
	for _, prefix := range []string{"/usr/bin/bash -lc ", "/usr/bin/zsh -lc ", "/usr/bin/sh -lc ", "/bin/bash -lc ", "bash -lc "} {
		if strings.HasPrefix(command, prefix) {
			rest := strings.TrimPrefix(command, prefix)
			if unquoted, err := strconv.Unquote(rest); err == nil {
				return unquoted
			}
			if len(rest) > 1 && rest[0] == '\'' && rest[len(rest)-1] == '\'' {
				return rest[1 : len(rest)-1]
			}
			return rest
		}
	}
	return command
}

func unixInstant(v any) string {
	f, ok := v.(float64)
	if !ok || f <= 0 {
		return ""
	}
	return time.Unix(int64(f), 0).UTC().Format(time.RFC3339Nano)
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func intOf(v any) int {
	f, _ := v.(float64)
	return int(f)
}

func boundEventText(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= delegationEventTextBytes {
		return s
	}
	cut := delegationEventTextBytes
	for cut > 0 && !utf8RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + " …"
}

func boundEventTail(s string) string {
	s = strings.TrimRight(s, "\n")
	if len(s) <= delegationEventTextBytes {
		return s
	}
	start := len(s) - delegationEventTextBytes
	for start < len(s) && !utf8RuneStart(s[start]) {
		start++
	}
	return "… " + s[start:]
}

func utf8RuneStart(b byte) bool { return b&0xC0 != 0x80 }

func delegationEventsPath(runID string) string {
	return filepath.ToSlash(filepath.Join(reviewRunsDir, runID+".events.ndjson"))
}

// ReadDelegationEvents reads the normalized events of run, verified against
// the digest its record binds.
func (s Store) ReadDelegationEvents(run ReviewRun) ([]DelegationEvent, error) {
	if run.EventsDigest == "" {
		return nil, nil
	}
	raw, err := readBoundedRegular(filepath.Join(s.Root, filepath.FromSlash(delegationEventsPath(run.RunID))), delegationEventsMaxBytes)
	if err != nil {
		return nil, err
	}
	if digestBytes(raw) != run.EventsDigest {
		return nil, fmt.Errorf("pose: the events of run %s are not the ones its record binds", run.RunID)
	}
	events := []DelegationEvent{}
	for _, line := range bytes.Split(bytes.TrimRight(raw, "\n"), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var e DelegationEvent
		if err := json.Unmarshal(line, &e); err != nil || e.Version != DelegationEventSchemaVersion {
			return nil, fmt.Errorf("pose: run %s has an event this engine does not read", run.RunID)
		}
		events = append(events, e)
	}
	return events, nil
}

func publishDelegationEvents(dir, runID string, events []byte) error {
	if len(events) > delegationEventsMaxBytes {
		return fmt.Errorf("normalized events exceed %d-byte bound; nothing published", delegationEventsMaxBytes)
	}
	return PublishFile(filepath.Join(dir, runID+".events.ndjson"), events, 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent})
}

// DelegationStreamFinal is the final message an agent's stream carries, for
// an agent that writes no output file: Claude's successful result, or Codex's
// last agent message. A plain stream carries none.
func DelegationStreamFinal(format string, transcript []byte) string {
	final := ""
	switch format {
	case DelegationEventsClaudeStreamJSON:
		jsonLines(transcript, func(e map[string]any) {
			if str(e["type"]) == "result" {
				if failed, _ := e["is_error"].(bool); !failed {
					final = str(e["result"])
				}
			}
		})
	case DelegationEventsCodexJSONL:
		jsonLines(transcript, func(e map[string]any) {
			item, _ := e["item"].(map[string]any)
			if str(e["type"]) == "item.completed" && str(item["type"]) == "agent_message" {
				final = str(item["text"])
			}
		})
	}
	return final
}

// redactedDelegationEvents normalizes the raw stream and redacts every field
// with the run's credentials and session after JSON decoding.
func redactedDelegationEvents(format string, raw []byte, final string, secrets []string, session string) ([]byte, error) {
	events, err := normalizeDelegationEvents(format, raw, final, func(v string) (string, error) {
		// A replacement is at most 17 bytes for a value of one or more, so this
		// budget never refuses a field the stream budget already admitted.
		red, err := redactReviewCapture([]byte(v), secrets, session, 17*int64(len(v))+1)
		return string(red), err
	})
	if err != nil {
		return nil, err
	}
	return EncodeDelegationEvents(events), nil
}
