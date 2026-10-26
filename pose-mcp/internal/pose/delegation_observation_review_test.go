package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func recordedRun(t *testing.T, s Store, attempt, output string) ReviewRun {
	t.Helper()
	r := activeRun(attempt)
	r.SchemaVersion, r.Status, r.Output, r.EndedAt = 1, ReviewRunCompleted, output, time.Now().UTC().Format(time.RFC3339Nano)
	published, err := s.recordReviewRun(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	return published
}

// B02 of the first review: a reader that started before a run published its
// record and removed its marker still finds the attempt, as running or as
// recorded, never neither.
func TestAReaderNeverLosesAnEndingAttempt(t *testing.T) {
	s := Store{Root: t.TempDir()}
	for i := 0; i < 120; i++ {
		recordedRun(t, s, "", strings.Repeat("x", 16384)+time.Now().String())
	}
	for round := 0; round < 10; round++ {
		stop, _, err := s.markDelegationActive(activeRun("att-gap"), func() {})
		if err != nil {
			t.Fatal(err)
		}
		read := make(chan []DelegationAttempt, 1)
		go func() { a, _, _ := s.ListDelegationAttempts(); read <- a }()
		time.Sleep(2 * time.Millisecond)
		r := recordedRun(t, s, "att-gap", "decision: approved")
		stop()
		found := false
		for _, a := range <-read {
			found = found || a.AttemptID == "att-gap"
		}
		if !found {
			t.Fatalf("round %d: the reader returned neither the marker nor the record", round)
		}
		os.Remove(filepath.Join(s.Root, reviewRunsDir, r.RunID+".json"))
		os.Remove(filepath.Join(s.Root, reviewRunLedgerDir, r.RunID+".json"))
	}
}

// B05 and B06 of the first review: a line longer than the events' bound is
// read whole live, so the events keep the record's numbering and a session it
// declares is redacted in every later line.
func TestLiveEventsReadLongLinesWhole(t *testing.T) {
	s := Store{Root: t.TempDir()}
	w := s.startDelegationLive("att-long", DelegationEventsCodexJSONL, nil, `"thread_id":"([^"]+)"`, "", 0)
	defer w.Close()
	stream := `{"type":"thread.started","thread_id":"sess-long-123","pad":"` + strings.Repeat("x", delegationEventsMaxBytes+65536) + `"}` + "\n" +
		`{"type":"item.completed","item":{"type":"agent_message","text":"a long line before me"}}` + "\n" +
		`{"type":"item.completed","item":{"type":"agent_message","text":"I used sess-long-123"}}` + "\n"
	for left := []byte(stream); len(left) > 0; {
		n := min(32768, len(left))
		w.Write(left[:n])
		left = left[n:]
	}
	live, _, _ := s.ReadDelegationLiveEvents("att-long", 0)
	final := NormalizeDelegationEvents(DelegationEventsCodexJSONL, []byte(stream), "")
	if len(live) != len(final) || live[len(live)-1].Seq != final[len(final)-1].Seq {
		t.Fatalf("live %d events, record %d: numbering differs", len(live), len(final))
	}
	if raw, _ := os.ReadFile(s.delegationLivePath("att-long")); strings.Contains(string(raw), "sess-long-123") {
		t.Fatal("a session declared in a long line reached the live events")
	}
	limited := s.startDelegationLive("att-bounded", DelegationEventsCodexJSONL, nil, "", "", 10)
	limited.Write([]byte(stream))
	if events, _, _ := s.ReadDelegationLiveEvents("att-bounded", 0); len(events) != 0 {
		t.Fatalf("the live events read past the stream budget: %+v", events)
	}
}

// B09 of the first review: a linked live directory is refused, and a linked
// live file is replaced, never written through.
func TestLiveEventsNeverFollowALink(t *testing.T) {
	s := Store{Root: t.TempDir()}
	outside := t.TempDir()
	victim := filepath.Join(outside, "victim.ndjson")
	os.WriteFile(victim, []byte(`{"v":1,"seq":1,"kind":"message","text":"outside"}`+"\n"), 0o600)
	if _, err := s.delegationActiveDir(true); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(s.Root, filepath.FromSlash(delegationLiveDir))); err != nil {
		t.Fatal(err)
	}
	if w := s.startDelegationLive("victim", DelegationEventsCodexJSONL, nil, "", "", 0); w != nil {
		t.Fatal("a linked live directory was written through")
	}
	if events, _, err := s.ReadDelegationLiveEvents("victim", 0); err == nil || len(events) != 0 {
		t.Fatalf("a linked live directory was read: %+v %v", events, err)
	}
	os.Remove(filepath.Join(s.Root, filepath.FromSlash(delegationLiveDir)))
	os.MkdirAll(filepath.Join(s.Root, filepath.FromSlash(delegationLiveDir)), 0o700)
	os.Symlink(victim, s.delegationLivePath("victim"))
	w := s.startDelegationLive("victim", DelegationEventsCodexJSONL, nil, "", "", 0)
	if w == nil {
		t.Fatal("no live writer over a linked file")
	}
	w.Close()
	if raw, _ := os.ReadFile(victim); !strings.Contains(string(raw), "outside") {
		t.Fatal("a linked live file was written through")
	}
}

// B04 of the first review: a run's hook runs once, however often the run is
// returned.
func TestTheHookRunsOncePerRun(t *testing.T) {
	s := Store{Root: t.TempDir()}
	os.MkdirAll(filepath.Join(s.Root, filepath.FromSlash(reviewRunsDir)), 0o755)
	os.MkdirAll(filepath.Join(s.Root, ".pose", "policy"), 0o755)
	os.WriteFile(filepath.Join(s.Root, filepath.FromSlash(delegationPolicyPath)), []byte(`{"schema_version":2,"agents":{"a":{"vendor":"v","model":"m","commands":{"read":["sh"]},"on_terminal":{"command":["sh","-c","echo once >> calls.txt"]}}}}`), 0o644)
	run := ReviewRun{RunID: "rrn-0123456789abcdef", Adapter: "a", Status: ReviewRunCompleted}
	if s.RunDelegationTerminalHook(run) == nil {
		t.Fatal("the hook did not run")
	}
	if again := s.RunDelegationTerminalHook(run); again != nil {
		t.Fatalf("the hook ran again: %+v", again)
	}
	if raw, _ := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(reviewRunsDir), "calls.txt")); strings.Count(string(raw), "once") != 1 {
		t.Fatalf("calls: %q", raw)
	}
}

// N01 of the second review: a hook whose outcome could not be recorded is
// reported, not read as no hook.
func TestAnUnrecordedHookOutcomeIsReported(t *testing.T) {
	s := Store{Root: t.TempDir()}
	dir, err := s.delegationHookDir()
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "rrn-0123456789abcdef.claim"), []byte("x\n"), 0o600)
	if outcome := s.ReadDelegationHook("rrn-0123456789abcdef"); outcome == nil || !strings.Contains(outcome.Failure, "not recorded") {
		t.Fatalf("an unrecorded outcome: %+v", outcome)
	}
}

// F-R3-01 and F-R3-06 of the third review: a hook still within its bound is
// not reported as lost, and a relative hook command names a file of the
// project.
func TestHookBoundAndRelativeCommand(t *testing.T) {
	s := Store{Root: t.TempDir()}
	dir, err := s.delegationHookDir()
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "rrn-aaaaaaaaaaaaaaaa.claim"), []byte(time.Now().UTC().Format(time.RFC3339Nano)+"\n"), 0o600)
	if outcome := s.ReadDelegationHook("rrn-aaaaaaaaaaaaaaaa"); outcome != nil {
		t.Fatalf("a hook within its bound read as lost: %+v", outcome)
	}
	os.MkdirAll(filepath.Join(s.Root, filepath.FromSlash(reviewRunsDir)), 0o755)
	os.MkdirAll(filepath.Join(s.Root, ".pose", "policy"), 0o755)
	os.WriteFile(filepath.Join(s.Root, "hook.sh"), []byte("#!/bin/sh\necho ran > relative-hook.txt\n"), 0o755)
	os.WriteFile(filepath.Join(s.Root, filepath.FromSlash(delegationPolicyPath)), []byte(`{"schema_version":2,"agents":{"a":{"vendor":"v","model":"m","commands":{"read":["sh"]},"on_terminal":{"command":["./hook.sh"]}}}}`), 0o644)
	outcome := s.RunDelegationTerminalHook(ReviewRun{RunID: "rrn-bbbbbbbbbbbbbbbb", Adapter: "a", Status: ReviewRunCompleted})
	if outcome == nil || outcome.Failure != "" {
		t.Fatalf("a relative hook: %+v", outcome)
	}
}
