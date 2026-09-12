package cliout

// The machine channel (spec pose-cli-output-rendering-system R8, R9).
//
// `--json` used to mean two different things — a boolean printing to stdout in
// most commands, a file path in `validate` — and eight gates had no machine
// channel at all, so an agent had to parse prose to learn what a gate decided.
//
// The renderer already sees every result event, so it is what records them. A
// command emits its verdict, findings and fields exactly as it does for a human;
// with recording on, stdout carries one JSON document instead, and the two
// channels cannot disagree because there is only one source.

import (
	"encoding/json"
	"fmt"
	"io"
)

// RecordSchemaVersion is the machine document's contract version.
const RecordSchemaVersion = 1

// RecordedFinding is one finding as a machine reads it. The severity is the
// state's key, untranslated: the human line localises its word, this does not.
type RecordedFinding struct {
	Severity    string `json:"severity"`
	Code        string `json:"code,omitempty"`
	Path        string `json:"path,omitempty"`
	Message     string `json:"message"`
	Remediation string `json:"remediation,omitempty"`
	MessageID   string `json:"message_id,omitempty"`
}

// RecordedStep is one unit of work with what it decided and how long it took.
type RecordedStep struct {
	Label   string  `json:"label"`
	Detail  string  `json:"detail,omitempty"`
	Outcome string  `json:"outcome"`
	Seconds float64 `json:"seconds"`
	Note    string  `json:"note,omitempty"`
}

// Record is the document `--json` prints.
type Record struct {
	SchemaVersion int               `json:"schema_version"`
	Command       string            `json:"command"`
	Outcome       string            `json:"outcome"`
	Verdict       string            `json:"verdict,omitempty"`
	Findings      []RecordedFinding `json:"findings"`
	Fields        map[string]string `json:"fields,omitempty"`
	Steps         []RecordedStep    `json:"steps,omitempty"`
	Counts        map[string]int    `json:"counts,omitempty"`
}

// RecordJSON turns the result channel into a JSON document for command. Human
// result output is suppressed from here on — stdout carries the document and
// nothing else — while progress keeps going to stderr, where it never was part
// of the result.
func (r *Renderer) RecordJSON(command string) {
	r.record = &Record{SchemaVersion: RecordSchemaVersion, Command: command, Outcome: StatePass.Key(), Findings: []RecordedFinding{}}
}

// Recording reports whether the result channel is a JSON document.
func (r *Renderer) Recording() bool { return r.record != nil }

// RecordField adds a machine-only field that has no human line of its own.
func (r *Renderer) RecordField(name, value string) {
	if r.record == nil {
		return
	}
	if r.record.Fields == nil {
		r.record.Fields = map[string]string{}
	}
	r.record.Fields[name] = value
}

// RecordCount adds a counter to the document.
func (r *Renderer) RecordCount(name string, value int) {
	if r.record == nil {
		return
	}
	if r.record.Counts == nil {
		r.record.Counts = map[string]int{}
	}
	r.record.Counts[name] = value
}

// FlushJSON writes the document. A command calls it once, last, whatever its
// exit code: a gate that failed still owes the machine its findings.
func (r *Renderer) FlushJSON() error {
	if r.record == nil {
		return nil
	}
	raw, err := json.MarshalIndent(r.record, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(r.out, string(raw))
	return err
}

// WriteJSONTo writes the document to an explicit destination — what
// `--json-out <path>` needs, for a caller that also wants the human output.
func (r *Renderer) WriteJSONTo(w io.Writer) error {
	if r.record == nil {
		return nil
	}
	raw, err := json.MarshalIndent(r.record, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(raw))
	return err
}
