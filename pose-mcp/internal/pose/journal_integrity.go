package pose

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ErrActionJournalInvalid marks a journal that does not read as a complete,
// ordered sequence of events (spec pose-journal-checkpoint-and-lock-recovery
// R1). Errors that carry it are *ActionJournalError.
var ErrActionJournalInvalid = errors.New("action-journal-invalid")

// Kinds of an invalid journal.
const (
	// JournalInterruptedTail: every line but the last reads, and the last has
	// no line terminator — an append that did not finish. It is never read
	// as an event, even when its bytes happen to parse.
	JournalInterruptedTail = "interrupted-tail"
	// JournalCorrupt: a terminated line that does not read as the next
	// event, wherever it is.
	JournalCorrupt = "corrupt"
)

// maxActionJournalLine bounds one event; a longer line is corrupt.
const maxActionJournalLine = 1 << 20

// ActionJournalError locates the first bytes of a journal that do not read
// as the next event (spec pose-journal-checkpoint-and-lock-recovery R2). It
// names the line, its byte offset and length and the SHA-256 of those bytes,
// and the events before it that do read. It proposes nothing it would apply:
// RepairProposal, when set, is text for a person to review.
type ActionJournalError struct {
	Request        string `json:"request,omitempty"`
	Path           string `json:"path,omitempty"`
	Kind           string `json:"kind"`
	Line           int    `json:"line"`
	Offset         int    `json:"offset"`
	Length         int    `json:"length"`
	SHA256         string `json:"sha256"`
	ValidEvents    int    `json:"valid_events"`
	Reason         string `json:"reason"`
	RepairProposal string `json:"repair_proposal,omitempty"`
}

func (e *ActionJournalError) Error() string {
	return fmt.Sprintf("%s: journal %s at line %d (bytes %d+%d, sha256 %s): %s", ErrActionJournalInvalid, e.Kind, e.Line, e.Offset, e.Length, e.SHA256, e.Reason)
}

func (e *ActionJournalError) Unwrap() error { return ErrActionJournalInvalid }

// parseActionJournal reads every byte of a journal: each line is one event,
// terminated by a newline, holding exactly one JSON object with no duplicate
// field, of the supported schema and with the next sequence number. Nothing
// is skipped: an empty line, a second value on a line or an unterminated last
// line is an *ActionJournalError, never part of a view.
func parseActionJournal(raw []byte) ([]ActionEvent, error) {
	var events []ActionEvent
	offset := 0
	for line := 1; offset < len(raw); line++ {
		end := bytes.IndexByte(raw[offset:], '\n')
		terminated := end >= 0
		if !terminated {
			end = len(raw) - offset
		}
		text := raw[offset : offset+end]
		invalid := func(kind, reason string) error {
			sum := sha256.Sum256(text)
			e := &ActionJournalError{Kind: kind, Line: line, Offset: offset, Length: len(text), SHA256: hex.EncodeToString(sum[:]), ValidEvents: len(events), Reason: reason}
			if kind == JournalInterruptedTail {
				e.RepairProposal = fmt.Sprintf("the last %d bytes are an append that did not finish; after confirming no writer is running and that no answer or waiver is lost, truncating the journal to %d bytes keeps its %d complete events", len(text), offset, len(events))
			}
			return e
		}
		if !terminated {
			return nil, invalid(JournalInterruptedTail, "the last line has no terminator")
		}
		if len(text) > maxActionJournalLine {
			return nil, invalid(JournalCorrupt, fmt.Sprintf("the line exceeds %d bytes", maxActionJournalLine))
		}
		event, err := decodeActionEvent(text)
		if err != nil {
			return nil, invalid(JournalCorrupt, err.Error())
		}
		if event.SchemaVersion != ActionRequestSchemaVersion {
			return nil, invalid(JournalCorrupt, fmt.Sprintf("schema_version %d is not supported", event.SchemaVersion))
		}
		if event.Seq != len(events)+1 {
			return nil, invalid(JournalCorrupt, fmt.Sprintf("sequence %d breaks the journal order", event.Seq))
		}
		events = append(events, event)
		offset += end + 1
	}
	return events, nil
}

// decodeActionEvent decodes exactly one JSON object from a line.
func decodeActionEvent(text []byte) (ActionEvent, error) {
	var event ActionEvent
	if len(bytes.TrimSpace(text)) == 0 {
		return event, fmt.Errorf("empty line")
	}
	// Escaped control characters are valid text in an answer or a reason;
	// a duplicate field is not, because only one of its values would count.
	if err := rejectDuplicateJSONKeys(text, false); err != nil {
		return event, err
	}
	decoder := json.NewDecoder(bytes.NewReader(text))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		return event, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return event, fmt.Errorf("content after the event on the same line")
	}
	return event, nil
}

// ActionJournalNextAction is the supported next step for a journal that does
// not read. Nothing is repaired by the engine: a person restores the journal
// from version control or, for an interrupted tail, reviews the proposal.
func ActionJournalNextAction(e *ActionJournalError) string {
	where := "the journal"
	if e.Path != "" {
		where = e.Path
	}
	if e.Kind == JournalInterruptedTail {
		return "inspect " + where + " from byte " + fmt.Sprint(e.Offset) + "; " + e.RepairProposal
	}
	return "inspect " + where + " at line " + fmt.Sprint(e.Line) + " and restore it from version control; an event in the middle of a journal is never dropped to make it read"
}
