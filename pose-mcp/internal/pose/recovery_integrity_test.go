package pose

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Spec pose-journal-checkpoint-and-lock-recovery: journals, checkpoints and
// start locks are read by the evidence they hold, never around it.

func openedJournal(t *testing.T) (Store, string, []byte) {
	t.Helper()
	store := actionFixture(t)
	view, err := store.OpenActionRequest(decisionRequest(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	path := actionJournalPath(store.Root, view.Request.ID)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return store, view.Request.ID, raw
}

// V1 — every line is read whole; nothing a journal holds is skipped.
func TestAJournalIsReadWholeOrNotAtAll(t *testing.T) {
	_, _, opening := openedJournal(t)
	first := bytes.TrimSuffix(opening, []byte("\n"))
	for name, c := range map[string]struct {
		journal []byte
		kind    string
		line    int
		offset  int
	}{
		"second value on a line":     {append(append([]byte{}, first...), []byte(" {\"unconsumed\":\"tail\"}\n")...), JournalCorrupt, 1, 0},
		"duplicate field":            {append(bytes.Replace(first, []byte(`"seq":1,`), []byte(`"seq":1,"seq":1,`), 1), '\n'), JournalCorrupt, 1, 0},
		"unsupported schema":         {append(bytes.Replace(first, []byte(`"schema_version":1`), []byte(`"schema_version":2`), 1), '\n'), JournalCorrupt, 1, 0},
		"empty line":                 {append(append([]byte{}, opening...), '\n'), JournalCorrupt, 2, len(opening)},
		"sequence gap":               {append(append([]byte{}, opening...), bytes.Replace(opening, []byte(`"seq":1,`), []byte(`"seq":3,`), 1)...), JournalCorrupt, 2, len(opening)},
		"unterminated unparseable":   {append(append([]byte{}, opening...), []byte(`{"schema_version":1,"seq":2,`)...), JournalInterruptedTail, 2, len(opening)},
		"unterminated but parseable": {append(append([]byte{}, opening...), bytes.Replace(first, []byte(`"seq":1,`), []byte(`"seq":2,`), 1)...), JournalInterruptedTail, 2, len(opening)},
		"oversized line":             {append(append([]byte{}, opening...), append(bytes.Repeat([]byte(" "), maxActionJournalLine+1), '\n')...), JournalCorrupt, 2, len(opening)},
	} {
		events, err := parseActionJournal(c.journal)
		var invalid *ActionJournalError
		if !errors.As(err, &invalid) || !errors.Is(err, ErrActionJournalInvalid) || events != nil {
			t.Fatalf("%s: read as %d event(s), err %v", name, len(events), err)
		}
		end := bytes.IndexByte(c.journal[c.offset:], '\n')
		if end < 0 {
			end = len(c.journal) - c.offset
		}
		sum := sha256.Sum256(c.journal[c.offset : c.offset+end])
		if invalid.Kind != c.kind || invalid.Line != c.line || invalid.Offset != c.offset || invalid.Length != end || invalid.SHA256 != hex.EncodeToString(sum[:]) {
			t.Fatalf("%s: located as %+v", name, invalid)
		}
		if (invalid.Kind == JournalInterruptedTail) != (invalid.RepairProposal != "") {
			t.Fatalf("%s: a repair is proposed only for an interrupted tail: %+v", name, invalid)
		}
	}
}

// Escaped control characters are text; the journals already in use read.
func TestAJournalKeepsTextWithEscapedControls(t *testing.T) {
	store := actionFixture(t)
	request := decisionRequest()
	request.Question = "Keep reading v1?\nAnswer\tfor the next delivery."
	view, err := store.OpenActionRequest(request, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	again, err := store.LoadActionRequest(view.Request.ID)
	if err != nil || again.Request.Question != request.Question || again.Revision != 1 {
		t.Fatalf("an escaped newline broke the journal: %+v %v", again, err)
	}
}

// V1/V8 — an interrupted tail is located and proposed, never repaired: the
// bytes stay, and no write is appended over them.
func TestAnInterruptedTailIsProposedNeverRepaired(t *testing.T) {
	store, id, opening := openedJournal(t)
	path := actionJournalPath(store.Root, id)
	damaged := append(append([]byte{}, opening...), []byte(`{"schema_version":1,"seq":2,"type":"answered"`)...)
	if err := os.WriteFile(path, damaged, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := store.LoadActionRequest(id)
	var invalid *ActionJournalError
	if !errors.As(err, &invalid) || invalid.Request != id || invalid.Path != ".pose/actions/"+id+".jsonl" || invalid.ValidEvents != 1 ||
		!strings.Contains(invalid.RepairProposal, "truncating the journal to "+strconv.Itoa(len(opening))+" bytes") {
		t.Fatalf("the interrupted tail: %+v %v", invalid, err)
	}
	if _, err := store.appendActionEvent(id, ActionEvent{Type: ActionEventCancelled, Actor: "human:maintainer", Reason: "probe"}, -1, time.Now()); !errors.Is(err, ErrActionJournalInvalid) {
		t.Fatalf("an append went over a journal that does not read: %v", err)
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(after, damaged) {
		t.Fatal("the journal was changed")
	}
	if _, err := store.ListActionRequests(); !errors.Is(err, ErrActionJournalInvalid) {
		t.Fatalf("the listing read around a journal that does not read: %v", err)
	}
}

// V2 — a checkpoint is absent, valid, corrupt, unsupported or unavailable,
// never one of them read as another.
func TestACheckpointSaysWhatItIs(t *testing.T) {
	store := Store{Root: t.TempDir()}
	ref := "spec:work"
	path := closeoutCheckpointPath(store.Root, ref)
	if read := store.InspectCloseoutCheckpoint(ref); read.State != CheckpointAbsent {
		t.Fatalf("absent: %+v", read)
	}
	valid := `{"schema_version":1,"scope":"spec:work","started_at":"2026-10-12T12:00:00Z","plan_digest":"sha256:x","completed":["context","evidence"]}`
	for name, c := range map[string]struct{ body, state string }{
		"valid":         {valid, CheckpointValid},
		"truncated":     {`{"schema_version":1,`, CheckpointCorrupt},
		"other schema":  {strings.Replace(valid, `"schema_version":1`, `"schema_version":2`, 1), CheckpointUnsupported},
		"other scope":   {strings.Replace(valid, `spec:work`, `spec:other`, 1), CheckpointCorrupt},
		"unknown field": {strings.Replace(valid, `"completed"`, `"extra":1,"completed"`, 1), CheckpointCorrupt},
		"duplicate":     {strings.Replace(valid, `"completed"`, `"scope":"spec:work","completed"`, 1), CheckpointCorrupt},
		"unknown step":  {strings.Replace(valid, `"evidence"`, `"publish"`, 1), CheckpointCorrupt},
		"repeated step": {strings.Replace(valid, `"evidence"`, `"context"`, 1), CheckpointCorrupt},
		"no digest":     {strings.Replace(valid, `"sha256:x"`, `""`, 1), CheckpointCorrupt},
		"bad time":      {strings.Replace(valid, `2026-10-12T12:00:00Z`, `yesterday`, 1), CheckpointCorrupt},
		"trailing":      {valid + `{}`, CheckpointCorrupt},
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body(c.body)), 0o644); err != nil {
			t.Fatal(err)
		}
		read := store.InspectCloseoutCheckpoint(ref)
		if read.State != c.state || (c.state == CheckpointValid) != (read.Checkpoint != nil) || (c.state != CheckpointValid && read.Reason == "") {
			t.Fatalf("%s: %+v", name, read)
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if read := store.InspectCloseoutCheckpoint(ref); read.State != CheckpointCorrupt {
		t.Fatalf("a directory in its place: %+v", read)
	}
}

func body(s string) string { return s + "\n" }

// V3 — the checkpoint is replaced whole, and a removal that fails is
// reported, never ignored.
func TestACheckpointIsReplacedWholeAndItsRemovalIsReported(t *testing.T) {
	store := Store{Root: t.TempDir()}
	cp := CloseoutCheckpoint{SchemaVersion: CloseoutCheckpointSchemaVersion, Scope: "spec:work", StartedAt: "2026-10-12T12:00:00Z", PlanDigest: "sha256:x", Completed: []string{"context"}}
	if err := store.WriteCloseoutCheckpoint(cp); err != nil {
		t.Fatal(err)
	}
	cp.Completed = append(cp.Completed, "evidence")
	if err := store.WriteCloseoutCheckpoint(cp); err != nil {
		t.Fatal(err)
	}
	read := store.InspectCloseoutCheckpoint("spec:work")
	if read.State != CheckpointValid || strings.Join(read.Checkpoint.Completed, ",") != "context,evidence" {
		t.Fatalf("the replaced checkpoint: %+v", read)
	}
	dir := filepath.Join(store.Root, filepath.FromSlash(closeoutCheckpointDir))
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("a temporary file was left: %v", entries)
	}
	if os.Getuid() != 0 {
		if err := os.Chmod(dir, 0o555); err != nil {
			t.Fatal(err)
		}
		err := store.ClearCloseoutCheckpoint("spec:work")
		_ = os.Chmod(dir, 0o755)
		if err == nil {
			t.Fatal("a removal that failed was not reported")
		}
		if err := store.WriteCloseoutCheckpoint(cp); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.ClearCloseoutCheckpoint("spec:work"); err != nil {
		t.Fatal(err)
	}
	if err := store.ClearCloseoutCheckpoint("spec:work"); err != nil {
		t.Fatalf("clearing an absent checkpoint: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("the empty checkpoint directory was left")
	}
}
